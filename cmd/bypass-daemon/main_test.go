package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temp YAML file with 3 domains
	content := `bypass_domains:
  - "foreflight.com"
  - "garmin.com"
  - "aviationweather.gov"
`
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "bypass-domains.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if len(cfg.BypassDomains) != 3 {
		t.Errorf("expected 3 bypass domains, got %d", len(cfg.BypassDomains))
	}
}

func TestLoadConfigMissing(t *testing.T) {
	_, err := loadConfig("/nonexistent/path/bypass-domains.yaml")
	if err == nil {
		t.Error("expected error for nonexistent config file, got nil")
	}
}

func TestResolveDomains(t *testing.T) {
	if testing.Short() {
		// Still run in short mode — these are real DNS lookups but fast
	}

	ips, err := resolveDomains([]string{"google.com", "cloudflare.com"})
	if err != nil {
		t.Fatalf("resolveDomains returned error: %v", err)
	}

	if len(ips) == 0 {
		t.Error("expected at least one resolved IP, got 0")
	}

	// Verify all results are valid IPs
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			t.Errorf("invalid IP address in results: %s", ip)
		}
	}
}

func TestResolveDomainsWildcard(t *testing.T) {
	// Verify that *.foreflight.com strips the prefix and resolves foreflight.com
	ips, err := resolveDomains([]string{"*.foreflight.com"})
	if err != nil {
		t.Fatalf("resolveDomains returned error: %v", err)
	}

	// foreflight.com should resolve to at least one IP
	if len(ips) == 0 {
		t.Error("expected at least one IP for *.foreflight.com (resolved as foreflight.com), got 0")
	}

	// Verify all results are valid IPs
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			t.Errorf("invalid IP address in results: %s", ip)
		}
	}
}

func TestResolveDomainsInvalid(t *testing.T) {
	// Invalid domain should be gracefully skipped, not cause an error
	ips, err := resolveDomains([]string{"nonexistent.invalid.domain.xyz"})
	if err != nil {
		t.Fatalf("resolveDomains returned error for invalid domain: %v", err)
	}

	// May be empty (domain doesn't resolve) — that's fine
	_ = ips
}
