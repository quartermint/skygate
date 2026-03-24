package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestGenerateRootCA(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "root-ca.crt")
	keyPath := filepath.Join(dir, "root-ca.key")

	rootCert, rootKey, err := GenerateRootCA(certPath, keyPath, "TestPlane")
	if err != nil {
		t.Fatalf("GenerateRootCA error: %v", err)
	}
	if rootCert == nil {
		t.Fatal("expected non-nil root certificate")
	}
	if rootKey == nil {
		t.Fatal("expected non-nil root key")
	}

	// CN contains "SkyGate-TestPlane CA"
	if !strings.Contains(rootCert.Subject.CommonName, "SkyGate-TestPlane CA") {
		t.Errorf("CommonName = %q, want to contain %q", rootCert.Subject.CommonName, "SkyGate-TestPlane CA")
	}

	// IsCA must be true
	if !rootCert.IsCA {
		t.Error("root cert IsCA = false, want true")
	}

	// Validity ~3 years from now (within a day tolerance)
	expectedExpiry := time.Now().Add(3 * 365 * 24 * time.Hour)
	if rootCert.NotAfter.Before(expectedExpiry.Add(-24*time.Hour)) || rootCert.NotAfter.After(expectedExpiry.Add(24*time.Hour)) {
		t.Errorf("NotAfter = %v, want ~%v (3 years)", rootCert.NotAfter, expectedExpiry)
	}

	// Key is ECDSA P-256
	ecKey, ok := rootKey.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("expected ECDSA private key")
	}
	if ecKey.Curve != elliptic.P256() {
		t.Error("expected P-256 curve")
	}

	// Cert file written with 0644 perms
	certInfo, err := os.Stat(certPath)
	if err != nil {
		t.Fatalf("stat cert file: %v", err)
	}
	if perm := certInfo.Mode().Perm(); perm != 0644 {
		t.Errorf("cert file permissions = %o, want 0644", perm)
	}

	// Key file written with 0600 perms
	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if perm := keyInfo.Mode().Perm(); perm != 0600 {
		t.Errorf("key file permissions = %o, want 0600", perm)
	}
}

func TestGenerateRootCA_ExistingFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "root-ca.crt")
	keyPath := filepath.Join(dir, "root-ca.key")

	// Generate first time
	cert1, _, err := GenerateRootCA(certPath, keyPath, "TestPlane")
	if err != nil {
		t.Fatalf("first GenerateRootCA error: %v", err)
	}

	// Call again -- should load existing, not regenerate
	cert2, _, err := GenerateRootCA(certPath, keyPath, "TestPlane")
	if err != nil {
		t.Fatalf("second GenerateRootCA error: %v", err)
	}

	// Compare serial numbers to confirm same cert was loaded
	if cert1.SerialNumber.Cmp(cert2.SerialNumber) != 0 {
		t.Error("serial numbers differ -- cert was regenerated instead of reloaded")
	}
}

func TestGenerateIntermediateCA(t *testing.T) {
	dir := t.TempDir()
	rootCertPath := filepath.Join(dir, "root-ca.crt")
	rootKeyPath := filepath.Join(dir, "root-ca.key")
	intCertPath := filepath.Join(dir, "intermediate-ca.crt")
	intKeyPath := filepath.Join(dir, "intermediate-ca.key")

	// Generate root CA first
	rootCert, rootKey, err := GenerateRootCA(rootCertPath, rootKeyPath, "TestPlane")
	if err != nil {
		t.Fatalf("GenerateRootCA error: %v", err)
	}

	// Generate intermediate CA
	intTLSCert, err := GenerateIntermediateCA(rootCert, rootKey, intCertPath, intKeyPath)
	if err != nil {
		t.Fatalf("GenerateIntermediateCA error: %v", err)
	}
	if intTLSCert == nil {
		t.Fatal("expected non-nil intermediate certificate")
	}

	// Parse the intermediate cert to inspect fields
	intX509, err := x509.ParseCertificate(intTLSCert.Certificate[0])
	if err != nil {
		t.Fatalf("parsing intermediate cert: %v", err)
	}

	// Intermediate's Issuer matches root's Subject
	if intX509.Issuer.CommonName != rootCert.Subject.CommonName {
		t.Errorf("Issuer.CN = %q, want %q", intX509.Issuer.CommonName, rootCert.Subject.CommonName)
	}

	// Is a CA
	if !intX509.IsCA {
		t.Error("intermediate cert IsCA = false, want true")
	}

	// MaxPathLen == 0 and MaxPathLenZero == true
	if intX509.MaxPathLen != 0 {
		t.Errorf("MaxPathLen = %d, want 0", intX509.MaxPathLen)
	}
	if !intX509.MaxPathLenZero {
		t.Error("MaxPathLenZero = false, want true")
	}

	// Validity ~1 year
	expectedExpiry := time.Now().Add(365 * 24 * time.Hour)
	if intX509.NotAfter.Before(expectedExpiry.Add(-24*time.Hour)) || intX509.NotAfter.After(expectedExpiry.Add(24*time.Hour)) {
		t.Errorf("NotAfter = %v, want ~%v (1 year)", intX509.NotAfter, expectedExpiry)
	}

	// Key is ECDSA P-256
	ecKey, ok := intTLSCert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("expected ECDSA private key for intermediate")
	}
	if ecKey.Curve != elliptic.P256() {
		t.Error("expected P-256 curve for intermediate")
	}
}

func TestIntermediateCAChainValidation(t *testing.T) {
	dir := t.TempDir()
	rootCertPath := filepath.Join(dir, "root-ca.crt")
	rootKeyPath := filepath.Join(dir, "root-ca.key")
	intCertPath := filepath.Join(dir, "intermediate-ca.crt")
	intKeyPath := filepath.Join(dir, "intermediate-ca.key")

	rootCert, rootKey, err := GenerateRootCA(rootCertPath, rootKeyPath, "ChainTest")
	if err != nil {
		t.Fatalf("GenerateRootCA error: %v", err)
	}

	intTLSCert, err := GenerateIntermediateCA(rootCert, rootKey, intCertPath, intKeyPath)
	if err != nil {
		t.Fatalf("GenerateIntermediateCA error: %v", err)
	}

	intX509, err := x509.ParseCertificate(intTLSCert.Certificate[0])
	if err != nil {
		t.Fatalf("parsing intermediate cert: %v", err)
	}

	// Verify intermediate against root CA cert pool
	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCert)

	opts := x509.VerifyOptions{
		Roots: rootPool,
	}

	if _, err := intX509.Verify(opts); err != nil {
		t.Fatalf("intermediate CA chain validation failed: %v", err)
	}
}

func TestIntermediateCALeafSigning(t *testing.T) {
	dir := t.TempDir()
	rootCertPath := filepath.Join(dir, "root-ca.crt")
	rootKeyPath := filepath.Join(dir, "root-ca.key")
	intCertPath := filepath.Join(dir, "intermediate-ca.crt")
	intKeyPath := filepath.Join(dir, "intermediate-ca.key")

	rootCert, rootKey, err := GenerateRootCA(rootCertPath, rootKeyPath, "LeafTest")
	if err != nil {
		t.Fatalf("GenerateRootCA error: %v", err)
	}

	intTLSCert, err := GenerateIntermediateCA(rootCert, rootKey, intCertPath, intKeyPath)
	if err != nil {
		t.Fatalf("GenerateIntermediateCA error: %v", err)
	}

	intX509, err := x509.ParseCertificate(intTLSCert.Certificate[0])
	if err != nil {
		t.Fatalf("parsing intermediate cert: %v", err)
	}

	intKey := intTLSCert.PrivateKey

	// Create a leaf cert for "example.com" signed by the intermediate CA
	leafPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating leaf key: %v", err)
	}

	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(100),
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, intX509, &leafPriv.PublicKey, intKey)
	if err != nil {
		t.Fatalf("creating leaf cert: %v", err)
	}

	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parsing leaf cert: %v", err)
	}

	// Verify leaf cert against the full chain [intermediate, root]
	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCert)

	intermediatePool := x509.NewCertPool()
	intermediatePool.AddCert(intX509)

	opts := x509.VerifyOptions{
		Roots:         rootPool,
		Intermediates: intermediatePool,
		DNSName:       "example.com",
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if _, err := leafCert.Verify(opts); err != nil {
		t.Fatalf("leaf cert chain validation failed: %v", err)
	}

	// Suppress unused import warnings for packages needed in tests
	_ = tls.Certificate{}
	_ = strings.Contains("", "")
	_ = time.Now()
}
