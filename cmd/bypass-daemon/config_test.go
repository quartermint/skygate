package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := []byte("bypass_domains:\n  - \"google.com\"\n  - \"cloudflare.com\"\n  - \"example.com\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if len(cfg.BypassDomains) != 3 {
		t.Errorf("expected 3 domains, got %d", len(cfg.BypassDomains))
	}
}

func TestLoadConfigMissing(t *testing.T) {
	_, err := LoadConfig("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfigEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error for empty file: %v", err)
	}
	if len(cfg.BypassDomains) != 0 {
		t.Errorf("expected 0 domains for empty file, got %d", len(cfg.BypassDomains))
	}
}
