package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
)

// mobileconfigTemplate is the Apple .mobileconfig profile template for installing
// the SkyGate CA certificate on iOS/macOS devices.
const mobileconfigTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadCertificateFileName</key>
            <string>skygate-ca.cer</string>
            <key>PayloadContent</key>
            <data>{{.CertBase64}}</data>
            <key>PayloadDescription</key>
            <string>Adds SkyGate bandwidth optimization CA certificate</string>
            <key>PayloadDisplayName</key>
            <string>SkyGate CA Certificate</string>
            <key>PayloadIdentifier</key>
            <string>com.skygate.ca.{{.UUID1}}</string>
            <key>PayloadType</key>
            <string>com.apple.security.root</string>
            <key>PayloadUUID</key>
            <string>{{.UUID1}}</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
    <key>PayloadDescription</key>
    <string>SkyGate Max Savings - Enables bandwidth compression for browser traffic</string>
    <key>PayloadDisplayName</key>
    <string>SkyGate Max Savings</string>
    <key>PayloadIdentifier</key>
    <string>com.skygate.profile.{{.UUID2}}</string>
    <key>PayloadOrganization</key>
    <string>SkyGate</string>
    <key>PayloadRemovalDisallowed</key>
    <false/>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>{{.UUID2}}</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`

// mobileconfigData holds the template data for the .mobileconfig profile.
type mobileconfigData struct {
	CertBase64 string
	UUID1      string
	UUID2      string
}

// HandleMobileConfig serves the CA certificate as an Apple .mobileconfig profile.
// GET /ca.mobileconfig -> Content-Type: application/x-apple-aspen-config
func (s *Server) HandleMobileConfig(w http.ResponseWriter, r *http.Request) {
	derBytes, err := readCACertDER(s.cfg.CACertPath)
	if err != nil {
		log.Printf("ERROR: reading CA cert for mobileconfig: %v", err)
		http.Error(w, "CA certificate not available", http.StatusInternalServerError)
		return
	}

	certB64 := base64.StdEncoding.EncodeToString(derBytes)
	uuid1 := generateDeterministicUUID(derBytes, 1)
	uuid2 := generateDeterministicUUID(derBytes, 2)

	tmpl, err := template.New("mobileconfig").Parse(mobileconfigTemplate)
	if err != nil {
		log.Printf("ERROR: parsing mobileconfig template: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, mobileconfigData{
		CertBase64: certB64,
		UUID1:      uuid1,
		UUID2:      uuid2,
	}); err != nil {
		log.Printf("ERROR: executing mobileconfig template: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("Content-Disposition", `attachment; filename="SkyGate.mobileconfig"`)
	w.Write(buf.Bytes())
}

// HandleCertDownloadDER serves the CA certificate as a DER-encoded .crt file.
// GET /ca.crt -> Content-Type: application/x-x509-ca-cert
func (s *Server) HandleCertDownloadDER(w http.ResponseWriter, r *http.Request) {
	derBytes, err := readCACertDER(s.cfg.CACertPath)
	if err != nil {
		log.Printf("ERROR: reading CA cert for DER download: %v", err)
		http.Error(w, "CA certificate not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="skygate-ca.crt"`)
	w.Write(derBytes)
}

// readCACertDER reads a PEM-encoded CA certificate and returns the DER bytes.
func readCACertDER(path string) ([]byte, error) {
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading CA cert file %s: %w", path, err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}

	return block.Bytes, nil
}

// generateDeterministicUUID generates a deterministic UUID-like string from data
// and an index. Uses SHA-256 hash, formatting the first 16 bytes as a UUID string.
func generateDeterministicUUID(data []byte, index int) string {
	input := append(data, byte(index))
	hash := sha256.Sum256(input)

	// Format first 16 bytes as UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		hash[0:4],
		hash[4:6],
		hash[6:8],
		hash[8:10],
		hash[10:16],
	)
}
