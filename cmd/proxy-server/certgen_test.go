package main

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrGenerateCA_NewCert(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	cert, err := LoadOrGenerateCA(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadOrGenerateCA error: %v", err)
	}
	if cert == nil {
		t.Fatal("expected non-nil certificate")
	}

	// Verify cert files were created on disk.
	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("cert file not created: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("key file not created: %v", err)
	}

	// Parse the persisted cert and verify CA properties.
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("reading cert file: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to decode PEM block from cert file")
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parsing x509 cert: %v", err)
	}
	if !x509Cert.IsCA {
		t.Error("certificate IsCA = false, want true")
	}
	if len(x509Cert.Subject.Organization) == 0 || x509Cert.Subject.Organization[0] != "SkyGate Proxy CA" {
		t.Errorf("Organization = %v, want [SkyGate Proxy CA]", x509Cert.Subject.Organization)
	}

	// Verify key file permissions (0600).
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("key file permissions = %o, want 0600", perm)
	}
}

func TestLoadOrGenerateCA_ExistingCert(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	// Generate cert first.
	cert1, err := LoadOrGenerateCA(certPath, keyPath)
	if err != nil {
		t.Fatalf("first LoadOrGenerateCA error: %v", err)
	}

	// Load existing cert -- should not regenerate.
	cert2, err := LoadOrGenerateCA(certPath, keyPath)
	if err != nil {
		t.Fatalf("second LoadOrGenerateCA error: %v", err)
	}

	// Both should return certs with same leaf data.
	if cert1 == nil || cert2 == nil {
		t.Fatal("expected non-nil certificates")
	}

	// Parse and compare serial numbers to confirm same cert was reloaded.
	leaf1, err := x509.ParseCertificate(cert1.Certificate[0])
	if err != nil {
		t.Fatalf("parsing cert1: %v", err)
	}
	leaf2, err := x509.ParseCertificate(cert2.Certificate[0])
	if err != nil {
		t.Fatalf("parsing cert2: %v", err)
	}
	if leaf1.SerialNumber.Cmp(leaf2.SerialNumber) != 0 {
		t.Error("serial numbers differ -- cert was regenerated instead of reloaded")
	}
}

func TestLoadOrGenerateCA_InvalidKey(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	// Generate valid cert first.
	_, err := LoadOrGenerateCA(certPath, keyPath)
	if err != nil {
		t.Fatalf("initial LoadOrGenerateCA error: %v", err)
	}

	// Corrupt the key file.
	if err := os.WriteFile(keyPath, []byte("not a valid key"), 0600); err != nil {
		t.Fatalf("writing corrupt key: %v", err)
	}

	// Should fail to load.
	_, err = LoadOrGenerateCA(certPath, keyPath)
	if err == nil {
		t.Fatal("expected error with corrupt key file")
	}
}
