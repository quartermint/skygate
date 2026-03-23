package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := []byte(`interface: wg0
check_interval_s: 5
handshake_timeout_s: 180
recovery_threshold_s: 30
fail_count: 3
recover_count: 3
fwmark: "0x2"
table: 200
priority: 200
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Interface != "wg0" {
		t.Errorf("expected interface wg0, got %s", cfg.Interface)
	}
	if cfg.CheckIntervalS != 5 {
		t.Errorf("expected check_interval_s 5, got %d", cfg.CheckIntervalS)
	}
	if cfg.HandshakeTimeoutS != 180 {
		t.Errorf("expected handshake_timeout_s 180, got %d", cfg.HandshakeTimeoutS)
	}
	if cfg.RecoveryThresholdS != 30 {
		t.Errorf("expected recovery_threshold_s 30, got %d", cfg.RecoveryThresholdS)
	}
	if cfg.FailCount != 3 {
		t.Errorf("expected fail_count 3, got %d", cfg.FailCount)
	}
	if cfg.RecoverCount != 3 {
		t.Errorf("expected recover_count 3, got %d", cfg.RecoverCount)
	}
	if cfg.Fwmark != "0x2" {
		t.Errorf("expected fwmark 0x2, got %s", cfg.Fwmark)
	}
	if cfg.Table != 200 {
		t.Errorf("expected table 200, got %d", cfg.Table)
	}
	if cfg.Priority != 200 {
		t.Errorf("expected priority 200, got %d", cfg.Priority)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	content := []byte("interface: wg0\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error for minimal config: %v", err)
	}
	if cfg.Interface != "wg0" {
		t.Errorf("expected interface wg0, got %s", cfg.Interface)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := []byte("interface: [\ninvalid yaml garbage\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
