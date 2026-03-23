package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dashboard.yaml")
	content := []byte(`port: 8081
poll_interval_sec: 5
pihole_address: "http://localhost:8080"
pihole_password: "secret"
db_path: "/data/skygate/dashboard.db"
categories_file: "/data/skygate/domain-categories.yaml"
static_dir: "/opt/skygate/static"
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Port != 8081 {
		t.Errorf("expected port 8081, got %d", cfg.Port)
	}
	if cfg.PollIntervalSec != 5 {
		t.Errorf("expected poll_interval_sec 5, got %d", cfg.PollIntervalSec)
	}
	if cfg.PiHoleAddress != "http://localhost:8080" {
		t.Errorf("expected pihole_address http://localhost:8080, got %s", cfg.PiHoleAddress)
	}
	if cfg.PiHolePassword != "secret" {
		t.Errorf("expected pihole_password secret, got %s", cfg.PiHolePassword)
	}
	if cfg.DBPath != "/data/skygate/dashboard.db" {
		t.Errorf("expected db_path /data/skygate/dashboard.db, got %s", cfg.DBPath)
	}
	if cfg.CategoriesFile != "/data/skygate/domain-categories.yaml" {
		t.Errorf("expected categories_file, got %s", cfg.CategoriesFile)
	}
	if cfg.StaticDir != "/opt/skygate/static" {
		t.Errorf("expected static_dir /opt/skygate/static, got %s", cfg.StaticDir)
	}
}

func TestLoadConfig_Missing(t *testing.T) {
	_, err := LoadConfig("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	// Write a minimal YAML with no fields -- all should get defaults
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Port != 8081 {
		t.Errorf("expected default port 8081, got %d", cfg.Port)
	}
	if cfg.PollIntervalSec != 5 {
		t.Errorf("expected default poll_interval_sec 5, got %d", cfg.PollIntervalSec)
	}
	if cfg.PiHoleAddress != "http://localhost:8080" {
		t.Errorf("expected default pihole_address, got %s", cfg.PiHoleAddress)
	}
	if cfg.DBPath != "/data/skygate/dashboard.db" {
		t.Errorf("expected default db_path, got %s", cfg.DBPath)
	}
	if cfg.CategoriesFile != "/data/skygate/domain-categories.yaml" {
		t.Errorf("expected default categories_file, got %s", cfg.CategoriesFile)
	}
	if cfg.StaticDir != "/opt/skygate/static" {
		t.Errorf("expected default static_dir, got %s", cfg.StaticDir)
	}
}
