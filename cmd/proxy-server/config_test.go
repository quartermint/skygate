package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	// Load the actual proxy-config.yaml and verify all expected values.
	cfgPath := filepath.Join("..", "..", "server", "proxy-config.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%s) error: %v", cfgPath, err)
	}

	if cfg.ListenAddr != ":8443" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8443")
	}
	if cfg.CACertPath != "/data/skygate/ca/root-ca.crt" {
		t.Errorf("CACertPath = %q, want %q", cfg.CACertPath, "/data/skygate/ca/root-ca.crt")
	}
	if cfg.CAKeyPath != "/data/skygate/ca/root-ca.key" {
		t.Errorf("CAKeyPath = %q, want %q", cfg.CAKeyPath, "/data/skygate/ca/root-ca.key")
	}
	if cfg.CADownloadAddr != ":8080" {
		t.Errorf("CADownloadAddr = %q, want %q", cfg.CADownloadAddr, ":8080")
	}
	if cfg.DBPath != "/data/skygate/proxy.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/data/skygate/proxy.db")
	}
	if cfg.BypassDomainsFile != "/etc/skygate/bypass-domains.yaml" {
		t.Errorf("BypassDomainsFile = %q, want %q", cfg.BypassDomainsFile, "/etc/skygate/bypass-domains.yaml")
	}
	if cfg.Verbose != false {
		t.Errorf("Verbose = %v, want false", cfg.Verbose)
	}

	// Image config
	if cfg.Image.Quality != 30 {
		t.Errorf("Image.Quality = %d, want 30", cfg.Image.Quality)
	}
	if cfg.Image.MaxWidth != 800 {
		t.Errorf("Image.MaxWidth = %d, want 800", cfg.Image.MaxWidth)
	}
	if cfg.Image.TimeoutMS != 500 {
		t.Errorf("Image.TimeoutMS = %d, want 500", cfg.Image.TimeoutMS)
	}
	if cfg.Image.MaxSizeBytes != 5242880 {
		t.Errorf("Image.MaxSizeBytes = %d, want 5242880", cfg.Image.MaxSizeBytes)
	}
	if cfg.Image.ConcurrentLimit != 4 {
		t.Errorf("Image.ConcurrentLimit = %d, want 4", cfg.Image.ConcurrentLimit)
	}

	// Minify config
	if !cfg.Minify.Enabled {
		t.Error("Minify.Enabled = false, want true")
	}
	if !cfg.Minify.HTML {
		t.Error("Minify.HTML = false, want true")
	}
	if !cfg.Minify.CSS {
		t.Error("Minify.CSS = false, want true")
	}
	if !cfg.Minify.JS {
		t.Error("Minify.JS = false, want true")
	}
	if !cfg.Minify.SVG {
		t.Error("Minify.SVG = false, want true")
	}
	if cfg.Minify.JSON {
		t.Error("Minify.JSON = true, want false")
	}

	// Log config
	if cfg.Log.RetentionDays != 7 {
		t.Errorf("Log.RetentionDays = %d, want 7", cfg.Log.RetentionDays)
	}
	if cfg.Log.BatchIntervalS != 30 {
		t.Errorf("Log.BatchIntervalS = %d, want 30", cfg.Log.BatchIntervalS)
	}

	// Phase 5: Intermediate CA and dashboard API
	if cfg.IntermediateCACertPath != "/data/skygate/ca/intermediate-ca.crt" {
		t.Errorf("IntermediateCACertPath = %q, want %q", cfg.IntermediateCACertPath, "/data/skygate/ca/intermediate-ca.crt")
	}
	if cfg.IntermediateCAKeyPath != "/data/skygate/ca/intermediate-ca.key" {
		t.Errorf("IntermediateCAKeyPath = %q, want %q", cfg.IntermediateCAKeyPath, "/data/skygate/ca/intermediate-ca.key")
	}
	if cfg.DashboardAPIURL != "http://10.0.0.2:8080" {
		t.Errorf("DashboardAPIURL = %q, want %q", cfg.DashboardAPIURL, "http://10.0.0.2:8080")
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("LoadConfig with missing file should return error")
	}
	if got := err.Error(); !contains(got, "reading config") {
		t.Errorf("error = %q, want to contain %q", got, "reading config")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, "bad.yaml")
	// Invalid YAML: tabs mixed with spaces in a way that breaks parsing
	if err := os.WriteFile(badFile, []byte(":\n\t- :\n\t\t[invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(badFile)
	if err == nil {
		t.Fatal("LoadConfig with invalid YAML should return error")
	}
	if got := err.Error(); !contains(got, "parsing config") {
		t.Errorf("error = %q, want to contain %q", got, "parsing config")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// An empty YAML file should parse with zero values, not hidden defaults.
	tmp := t.TempDir()
	emptyFile := filepath.Join(tmp, "empty.yaml")
	if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(emptyFile)
	if err != nil {
		t.Fatalf("LoadConfig with empty file: %v", err)
	}
	if cfg.ListenAddr != "" {
		t.Errorf("ListenAddr = %q, want empty", cfg.ListenAddr)
	}
	if cfg.Image.Quality != 0 {
		t.Errorf("Image.Quality = %d, want 0", cfg.Image.Quality)
	}
	if cfg.Minify.Enabled {
		t.Error("Minify.Enabled = true, want false for empty file")
	}
}

func TestLoadBypassDomains(t *testing.T) {
	bypassPath := filepath.Join("..", "..", "server", "bypass-domains.yaml")
	domains, err := LoadBypassDomains(bypassPath)
	if err != nil {
		t.Fatalf("LoadBypassDomains error: %v", err)
	}
	if len(domains) == 0 {
		t.Fatal("expected non-empty bypass domains list")
	}

	// Check specific expected entries
	want := map[string]bool{
		"*.apple.com":      false,
		"*.chase.com":      false,
		"*.foreflight.com": false,
	}
	for _, d := range domains {
		if _, ok := want[d]; ok {
			want[d] = true
		}
	}
	for domain, found := range want {
		if !found {
			t.Errorf("expected domain %q not found in bypass list", domain)
		}
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
