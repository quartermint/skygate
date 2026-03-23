package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandleCaptiveAccept handles the captive portal terms acceptance.
// POST /api/captive/accept -- accepts MAC from form value or attempts ARP lookup.
// Records acceptance in SQLite and adds MAC to nftables allowed_macs set.
func (s *Server) HandleCaptiveAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse MAC from form data or JSON
	mac := r.FormValue("mac")
	if mac == "" {
		// Try to extract from remote address via ARP lookup (best-effort)
		ip := extractIP(r.RemoteAddr)
		mac = lookupMACFromARP(ip)
	}

	if mac == "" {
		http.Error(w, "MAC address required", http.StatusBadRequest)
		return
	}

	// Normalize MAC to lowercase
	mac = strings.ToLower(mac)

	// Add to nftables allowed_macs set
	ip := extractIP(r.RemoteAddr)
	if err := AcceptDevice(mac, ip); err != nil {
		log.Printf("WARN: AcceptDevice failed for %s: %v", mac, err)
		// Continue anyway -- record in DB even if nftables fails
	}

	// Record in SQLite portal_accepted table
	if err := s.recordPortalAcceptance(mac, ip); err != nil {
		log.Printf("ERROR: recording portal acceptance: %v", err)
	}

	// Return HTML redirect page (simple enough for iOS CNA)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Connected</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>body{font-family:system-ui;text-align:center;padding:2em;background:#0f172a;color:#e2e8f0}
a{color:#38bdf8;text-decoration:none;font-size:1.2em}</style>
</head><body>
<h1>Connected!</h1>
<p>Open your browser and visit:</p>
<p><a href="http://192.168.4.1">SkyGate Dashboard</a></p>
</body></html>`)
}

// recordPortalAcceptance writes the captive portal acceptance to the DB.
func (s *Server) recordPortalAcceptance(mac, ip string) error {
	now := time.Now().Unix()
	_, err := s.db.db.Exec(
		"INSERT OR REPLACE INTO portal_accepted (mac_addr, accepted_at, ip_addr) VALUES (?, ?, ?)",
		mac, now, ip,
	)
	if err != nil {
		return fmt.Errorf("recording portal acceptance: %w", err)
	}
	return nil
}

// extractIP extracts the IP address from an http.Request.RemoteAddr (host:port).
func extractIP(remoteAddr string) string {
	// RemoteAddr is typically "IP:port" or "[::1]:port"
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		ip := remoteAddr[:idx]
		// Remove brackets for IPv6
		ip = strings.TrimPrefix(ip, "[")
		ip = strings.TrimSuffix(ip, "]")
		return ip
	}
	return remoteAddr
}

// lookupMACFromARP attempts to find a MAC address for the given IP from the ARP table.
// On Linux, reads /proc/net/arp. On macOS (dev), returns empty string.
func lookupMACFromARP(ip string) string {
	// This is a best-effort lookup. On macOS dev, it will return empty.
	// On Linux production, reads /proc/net/arp.
	return lookupARPTable(ip)
}
