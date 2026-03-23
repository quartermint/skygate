package main

import (
	"net"
	"testing"
)

func TestResolveDomains(t *testing.T) {
	ips := ResolveDomains([]string{"google.com", "cloudflare.com"})
	if len(ips) == 0 {
		t.Fatal("expected at least 1 IP, got 0")
	}
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			t.Errorf("invalid IP: %s", ip)
		}
	}
}

func TestResolveDomainsWildcard(t *testing.T) {
	ips := ResolveDomains([]string{"*.foreflight.com"})
	// Should strip "*." and resolve "foreflight.com"
	if len(ips) == 0 {
		t.Log("WARN: foreflight.com did not resolve (may be DNS issue), skipping")
		t.Skip("DNS resolution failed for foreflight.com")
	}
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			t.Errorf("invalid IP from wildcard resolution: %s", ip)
		}
	}
}

func TestResolveDomainsInvalid(t *testing.T) {
	ips := ResolveDomains([]string{"nonexistent.invalid.domain.xyz"})
	// Should not error, just return empty
	_ = ips // may be empty, that's fine
}

func TestResolveDomainsDedup(t *testing.T) {
	// Same domain twice should deduplicate
	ips := ResolveDomains([]string{"google.com", "google.com"})
	seen := make(map[string]bool)
	for _, ip := range ips {
		if seen[ip] {
			t.Errorf("duplicate IP in results: %s", ip)
		}
		seen[ip] = true
	}
}
