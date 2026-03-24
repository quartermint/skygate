package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// Mode constants for per-device savings level selection.
const (
	ModeQuickConnect = "quickconnect"
	ModeMaxSavings   = "maxsavings"
)

// modeRequest represents the JSON body for POST /api/mode.
type modeRequest struct {
	MAC  string `json:"mac"`
	Mode string `json:"mode"`
}

// HandleSetMode sets the savings mode for a device.
// POST /api/mode with JSON body: {"mac":"xx:xx:xx:xx:xx:xx","mode":"quickconnect"|"maxsavings"}
func (s *Server) HandleSetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req modeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate MAC
	if req.MAC == "" {
		http.Error(w, "mac is required", http.StatusBadRequest)
		return
	}

	// Validate mode
	if req.Mode != ModeQuickConnect && req.Mode != ModeMaxSavings {
		http.Error(w, "mode must be 'quickconnect' or 'maxsavings'", http.StatusBadRequest)
		return
	}

	// Normalize MAC to lowercase
	mac := strings.ToLower(req.MAC)

	// Persist to DB
	if err := s.db.SetDeviceMode(mac, req.Mode); err != nil {
		log.Printf("ERROR: SetDeviceMode: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Update nftables: add/remove from maxsavings_macs set.
	// Log warning on error but don't fail (accept eventual consistency per Pitfall 6).
	if req.Mode == ModeMaxSavings {
		if err := AddMaxSavingsMAC(mac); err != nil {
			log.Printf("WARN: AddMaxSavingsMAC failed for %s: %v", mac, err)
		}
	} else {
		if err := RemoveMaxSavingsMAC(mac); err != nil {
			log.Printf("WARN: RemoveMaxSavingsMAC failed for %s: %v", mac, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"mode":   req.Mode,
	})
}

// HandleGetMode returns the current savings mode for a device.
// GET /api/mode?mac=xx:xx:xx:xx:xx:xx
// Returns "quickconnect" for unknown devices (default per D-02).
func (s *Server) HandleGetMode(w http.ResponseWriter, r *http.Request) {
	mac := r.URL.Query().Get("mac")
	if mac == "" {
		http.Error(w, "mac query parameter required", http.StatusBadRequest)
		return
	}

	mac = strings.ToLower(mac)
	mode, err := s.db.GetDeviceMode(mac)
	if err != nil {
		log.Printf("ERROR: GetDeviceMode: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"mode": mode,
	})
}

// HandleGetMaxSavingsIPs returns source IPs of devices in Max Savings mode.
// GET /api/mode/ips -> JSON: {"maxsavings_ips":["192.168.4.2",...]}
// This endpoint is polled by the remote proxy every 10 seconds to build its
// per-device MITM decision set (per Research Pattern 3).
func (s *Server) HandleGetMaxSavingsIPs(w http.ResponseWriter, r *http.Request) {
	ips, err := s.db.GetMaxSavingsIPs()
	if err != nil {
		log.Printf("ERROR: GetMaxSavingsIPs: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"maxsavings_ips": ips,
	})
}
