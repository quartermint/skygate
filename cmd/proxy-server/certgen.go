package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// LoadOrGenerateCA loads an existing CA certificate and key from disk,
// or generates a new self-signed CA if the files don't exist.
// The CA is used for MITM HTTPS interception via goproxy.
// Per D-11: single CA generated at first startup, persisted to Docker volume.
func LoadOrGenerateCA(certPath, keyPath string) (*tls.Certificate, error) {
	// Try loading existing cert and key.
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err == nil {
		return &cert, nil
	}

	// If both files don't exist, generate new CA.
	if !os.IsNotExist(err) {
		// Files exist but are invalid -- return error, don't regenerate.
		// Check if either file exists to distinguish "missing" from "corrupt".
		_, certErr := os.Stat(certPath)
		_, keyErr := os.Stat(keyPath)
		if certErr == nil || keyErr == nil {
			return nil, fmt.Errorf("loading CA certificate: %w", err)
		}
	}

	// Generate new ECDSA P-256 private key.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}

	// Create self-signed CA certificate template.
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"SkyGate Proxy CA"},
			CommonName:   "SkyGate Proxy CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Self-sign the certificate.
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, fmt.Errorf("creating CA certificate: %w", err)
	}

	// PEM-encode the certificate.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// PEM-encode the private key.
	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling CA key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// Create parent directories if needed.
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return nil, fmt.Errorf("creating cert directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return nil, fmt.Errorf("creating key directory: %w", err)
	}

	// Write cert (0644) and key (0600) to disk.
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return nil, fmt.Errorf("writing CA cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("writing CA key: %w", err)
	}

	// Reload from disk to return a properly parsed tls.Certificate.
	reloaded, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("reloading generated CA: %w", err)
	}
	return &reloaded, nil
}
