package main

import (
	"crypto"
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

// GenerateRootCA generates or loads a root CA certificate with the appliance SSID
// in the CommonName and 3-year validity. The root CA key never leaves the Pi (D-06, D-08).
// If certPath and keyPath both exist, loads and returns them without regenerating.
// If both missing, generates a new ECDSA P-256 root CA. If only one exists, returns error.
func GenerateRootCA(certPath, keyPath, ssid string) (*x509.Certificate, crypto.PrivateKey, error) {
	certExists := fileExists(certPath)
	keyExists := fileExists(keyPath)

	// If both exist, load and return.
	if certExists && keyExists {
		return loadRootCA(certPath, keyPath)
	}

	// If one exists but not the other, return error (inconsistent state).
	if certExists || keyExists {
		return nil, nil, fmt.Errorf("inconsistent CA files: cert=%v key=%v", certExists, keyExists)
	}

	// Generate new ECDSA P-256 keypair.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating root CA key: %w", err)
	}

	// Random 128-bit serial number.
	serialNumber, err := randomSerial()
	if err != nil {
		return nil, nil, fmt.Errorf("generating serial: %w", err)
	}

	// Create self-signed root CA certificate template per D-07.
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"SkyGate"},
			CommonName:   fmt.Sprintf("SkyGate-%s CA", ssid),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(3 * 365 * 24 * time.Hour), // 3 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Self-sign (template == parent for root CA).
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("creating root CA certificate: %w", err)
	}

	// PEM-encode cert and key.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling root CA key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// Create parent directories if needed.
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return nil, nil, fmt.Errorf("creating cert directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return nil, nil, fmt.Errorf("creating key directory: %w", err)
	}

	// Write cert (0644) and key (0600).
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return nil, nil, fmt.Errorf("writing root CA cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, nil, fmt.Errorf("writing root CA key: %w", err)
	}

	// Parse the DER cert to return *x509.Certificate.
	parsedCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing generated root CA cert: %w", err)
	}

	return parsedCert, privKey, nil
}

// GenerateIntermediateCA generates or loads an intermediate CA certificate signed by
// the root CA. The intermediate has MaxPathLen=0 (can only sign leaf certs) and
// 1-year validity. Used by the remote proxy for on-the-fly leaf cert signing (D-17, D-18).
// If certPath and keyPath both exist, loads via tls.LoadX509KeyPair and returns.
func GenerateIntermediateCA(rootCert *x509.Certificate, rootKey crypto.PrivateKey, certPath, keyPath string) (*tls.Certificate, error) {
	// If both files exist, load and return.
	if fileExists(certPath) && fileExists(keyPath) {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("loading intermediate CA: %w", err)
		}
		return &cert, nil
	}

	// Generate new ECDSA P-256 keypair for the intermediate.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating intermediate CA key: %w", err)
	}

	// Random 128-bit serial number.
	serialNumber, err := randomSerial()
	if err != nil {
		return nil, fmt.Errorf("generating serial: %w", err)
	}

	// Intermediate CA template per Pitfall 4: MaxPathLen=0.
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"SkyGate"},
			CommonName:   "SkyGate Intermediate CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,    // Can only sign leaf certs
		MaxPathLenZero:        true, // Explicitly zero, not unset
	}

	// Sign with root CA key (parent = rootCert).
	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert, &privKey.PublicKey, rootKey)
	if err != nil {
		return nil, fmt.Errorf("creating intermediate CA certificate: %w", err)
	}

	// PEM-encode cert and key.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling intermediate CA key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// Create parent directories if needed.
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return nil, fmt.Errorf("creating cert directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return nil, fmt.Errorf("creating key directory: %w", err)
	}

	// Write cert (0644) and key (0600).
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return nil, fmt.Errorf("writing intermediate CA cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("writing intermediate CA key: %w", err)
	}

	// Reload via tls.LoadX509KeyPair.
	reloaded, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("reloading intermediate CA: %w", err)
	}
	return &reloaded, nil
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// loadRootCA loads an existing root CA cert and key from disk.
func loadRootCA(certPath, keyPath string) (*x509.Certificate, crypto.PrivateKey, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading root CA cert: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode PEM block from %s", certPath)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing root CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading root CA key: %w", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode PEM block from %s", keyPath)
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing root CA key: %w", err)
	}

	return cert, key, nil
}

// randomSerial generates a random 128-bit serial number for certificate generation.
func randomSerial() (*big.Int, error) {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, max)
}
