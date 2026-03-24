package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// createTestCACert generates a self-signed CA cert PEM file for testing.
func createTestCACert(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "SkyGate CA",
			Organization: []string{"SkyGate"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	path := t.TempDir() + "/root-ca.crt"
	if err := os.WriteFile(path, certPEM, 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	return path
}

func TestHandleMobileConfig(t *testing.T) {
	certPath := createTestCACert(t)
	srv := newTestServer(t)
	srv.cfg.CACertPath = certPath

	req := httptest.NewRequest(http.MethodGet, "/ca.mobileconfig", nil)
	w := httptest.NewRecorder()
	srv.HandleMobileConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/x-apple-aspen-config" {
		t.Errorf("expected Content-Type application/x-apple-aspen-config, got %s", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "PayloadType") {
		t.Error("response body missing PayloadType")
	}
	if !strings.Contains(body, "com.apple.security.root") {
		t.Error("response body missing com.apple.security.root")
	}
	if !strings.Contains(body, "<data>") {
		t.Error("response body missing <data> (base64 cert)")
	}
}

func TestHandleMobileConfig_ContentDisposition(t *testing.T) {
	certPath := createTestCACert(t)
	srv := newTestServer(t)
	srv.cfg.CACertPath = certPath

	req := httptest.NewRequest(http.MethodGet, "/ca.mobileconfig", nil)
	w := httptest.NewRecorder()
	srv.HandleMobileConfig(w, req)

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "SkyGate.mobileconfig") {
		t.Errorf("expected Content-Disposition with SkyGate.mobileconfig, got %s", cd)
	}
}

func TestHandleCertDownloadDER(t *testing.T) {
	certPath := createTestCACert(t)
	srv := newTestServer(t)
	srv.cfg.CACertPath = certPath

	req := httptest.NewRequest(http.MethodGet, "/ca.crt", nil)
	w := httptest.NewRecorder()
	srv.HandleCertDownloadDER(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/x-x509-ca-cert" {
		t.Errorf("expected Content-Type application/x-x509-ca-cert, got %s", ct)
	}

	// Verify we can parse the DER bytes back to a valid certificate
	cert, err := x509.ParseCertificate(w.Body.Bytes())
	if err != nil {
		t.Fatalf("failed to parse DER certificate: %v", err)
	}
	if cert.Subject.CommonName != "SkyGate CA" {
		t.Errorf("expected CN SkyGate CA, got %s", cert.Subject.CommonName)
	}
}

func TestHandleCertDownloadDER_ContentDisposition(t *testing.T) {
	certPath := createTestCACert(t)
	srv := newTestServer(t)
	srv.cfg.CACertPath = certPath

	req := httptest.NewRequest(http.MethodGet, "/ca.crt", nil)
	w := httptest.NewRecorder()
	srv.HandleCertDownloadDER(w, req)

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "skygate-ca.crt") {
		t.Errorf("expected Content-Disposition with skygate-ca.crt, got %s", cd)
	}
}
