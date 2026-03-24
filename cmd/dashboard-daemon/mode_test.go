package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSetMode_QuickConnect(t *testing.T) {
	srv := newTestServer(t)

	body := `{"mac":"aa:bb:cc:dd:ee:01","mode":"quickconnect"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.HandleSetMode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
	if resp["mode"] != "quickconnect" {
		t.Errorf("expected mode quickconnect, got %s", resp["mode"])
	}
}

func TestHandleSetMode_MaxSavings(t *testing.T) {
	srv := newTestServer(t)

	body := `{"mac":"aa:bb:cc:dd:ee:01","mode":"maxsavings"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.HandleSetMode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp["mode"] != "maxsavings" {
		t.Errorf("expected mode maxsavings, got %s", resp["mode"])
	}
}

func TestHandleSetMode_InvalidMode(t *testing.T) {
	srv := newTestServer(t)

	body := `{"mac":"aa:bb:cc:dd:ee:01","mode":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.HandleSetMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid mode, got %d", w.Code)
	}
}

func TestHandleSetMode_MissingMAC(t *testing.T) {
	srv := newTestServer(t)

	body := `{"mode":"maxsavings"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.HandleSetMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing MAC, got %d", w.Code)
	}
}

func TestHandleGetMode(t *testing.T) {
	srv := newTestServer(t)

	// Set mode first
	setBody := `{"mac":"aa:bb:cc:dd:ee:01","mode":"maxsavings"}`
	setReq := httptest.NewRequest(http.MethodPost, "/api/mode", strings.NewReader(setBody))
	setReq.Header.Set("Content-Type", "application/json")
	setW := httptest.NewRecorder()
	srv.HandleSetMode(setW, setReq)
	if setW.Code != http.StatusOK {
		t.Fatalf("set mode failed: %d", setW.Code)
	}

	// Get mode
	req := httptest.NewRequest(http.MethodGet, "/api/mode?mac=aa:bb:cc:dd:ee:01", nil)
	w := httptest.NewRecorder()
	srv.HandleGetMode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["mode"] != "maxsavings" {
		t.Errorf("expected mode maxsavings, got %s", resp["mode"])
	}
}

func TestHandleGetMode_DefaultQuickConnect(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/mode?mac=ff:ff:ff:ff:ff:ff", nil)
	w := httptest.NewRecorder()
	srv.HandleGetMode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["mode"] != "quickconnect" {
		t.Errorf("expected default mode quickconnect, got %s", resp["mode"])
	}
}

func TestGetDeviceMode_DBPersistence(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	mac := "aa:bb:cc:dd:ee:01"
	err = db.SetDeviceMode(mac, "maxsavings")
	if err != nil {
		t.Fatalf("SetDeviceMode failed: %v", err)
	}

	mode, err := db.GetDeviceMode(mac)
	if err != nil {
		t.Fatalf("GetDeviceMode failed: %v", err)
	}
	if mode != "maxsavings" {
		t.Errorf("expected maxsavings, got %s", mode)
	}

	// Unknown MAC returns quickconnect
	mode2, err := db.GetDeviceMode("ff:ff:ff:ff:ff:ff")
	if err != nil {
		t.Fatalf("GetDeviceMode unknown failed: %v", err)
	}
	if mode2 != "quickconnect" {
		t.Errorf("expected quickconnect for unknown MAC, got %s", mode2)
	}
}

func TestGetMaxSavingsMACs_ReturnsAll(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	// Set multiple devices to maxsavings
	macs := []string{"aa:bb:cc:dd:ee:01", "aa:bb:cc:dd:ee:02", "aa:bb:cc:dd:ee:03"}
	for _, mac := range macs {
		if err := db.SetDeviceMode(mac, "maxsavings"); err != nil {
			t.Fatalf("SetDeviceMode failed: %v", err)
		}
	}
	// Set one device to quickconnect
	if err := db.SetDeviceMode("aa:bb:cc:dd:ee:04", "quickconnect"); err != nil {
		t.Fatalf("SetDeviceMode failed: %v", err)
	}

	result, err := db.GetMaxSavingsMACs()
	if err != nil {
		t.Fatalf("GetMaxSavingsMACs failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 maxsavings MACs, got %d", len(result))
	}
}

func TestGetMaxSavingsIPs_ReturnsIPsFromPortalAccepted(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	// Insert portal_accepted rows
	_, err = db.db.Exec("INSERT INTO portal_accepted (mac_addr, accepted_at, ip_addr) VALUES (?, ?, ?)", "aa:bb:cc:dd:ee:01", 1000, "192.168.4.2")
	if err != nil {
		t.Fatalf("insert portal_accepted failed: %v", err)
	}
	_, err = db.db.Exec("INSERT INTO portal_accepted (mac_addr, accepted_at, ip_addr) VALUES (?, ?, ?)", "aa:bb:cc:dd:ee:02", 1000, "192.168.4.3")
	if err != nil {
		t.Fatalf("insert portal_accepted failed: %v", err)
	}
	_, err = db.db.Exec("INSERT INTO portal_accepted (mac_addr, accepted_at, ip_addr) VALUES (?, ?, ?)", "aa:bb:cc:dd:ee:03", 1000, "192.168.4.4")
	if err != nil {
		t.Fatalf("insert portal_accepted failed: %v", err)
	}

	// Set modes
	if err := db.SetDeviceMode("aa:bb:cc:dd:ee:01", "maxsavings"); err != nil {
		t.Fatalf("SetDeviceMode failed: %v", err)
	}
	if err := db.SetDeviceMode("aa:bb:cc:dd:ee:02", "maxsavings"); err != nil {
		t.Fatalf("SetDeviceMode failed: %v", err)
	}
	if err := db.SetDeviceMode("aa:bb:cc:dd:ee:03", "quickconnect"); err != nil {
		t.Fatalf("SetDeviceMode failed: %v", err)
	}

	ips, err := db.GetMaxSavingsIPs()
	if err != nil {
		t.Fatalf("GetMaxSavingsIPs failed: %v", err)
	}
	if len(ips) != 2 {
		t.Errorf("expected 2 maxsavings IPs, got %d: %v", len(ips), ips)
	}
	// Verify specific IPs
	ipSet := make(map[string]bool)
	for _, ip := range ips {
		ipSet[ip] = true
	}
	if !ipSet["192.168.4.2"] {
		t.Errorf("expected 192.168.4.2 in result")
	}
	if !ipSet["192.168.4.3"] {
		t.Errorf("expected 192.168.4.3 in result")
	}
}

func TestGetMaxSavingsIPs_EmptyWhenNoMaxSavings(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	// Insert portal_accepted but all devices are quickconnect
	_, err = db.db.Exec("INSERT INTO portal_accepted (mac_addr, accepted_at, ip_addr) VALUES (?, ?, ?)", "aa:bb:cc:dd:ee:01", 1000, "192.168.4.2")
	if err != nil {
		t.Fatalf("insert portal_accepted failed: %v", err)
	}
	if err := db.SetDeviceMode("aa:bb:cc:dd:ee:01", "quickconnect"); err != nil {
		t.Fatalf("SetDeviceMode failed: %v", err)
	}

	ips, err := db.GetMaxSavingsIPs()
	if err != nil {
		t.Fatalf("GetMaxSavingsIPs failed: %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("expected 0 IPs, got %d: %v", len(ips), ips)
	}
}

func TestHandleGetMaxSavingsIPs(t *testing.T) {
	srv := newTestServer(t)

	// Insert portal_accepted + set mode
	_, err := srv.db.db.Exec("INSERT INTO portal_accepted (mac_addr, accepted_at, ip_addr) VALUES (?, ?, ?)", "aa:bb:cc:dd:ee:01", 1000, "192.168.4.2")
	if err != nil {
		t.Fatalf("insert portal_accepted failed: %v", err)
	}
	if err := srv.db.SetDeviceMode("aa:bb:cc:dd:ee:01", "maxsavings"); err != nil {
		t.Fatalf("SetDeviceMode failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/mode/ips", nil)
	w := httptest.NewRecorder()
	srv.HandleGetMaxSavingsIPs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp struct {
		MaxSavingsIPs []string `json:"maxsavings_ips"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp.MaxSavingsIPs) != 1 {
		t.Errorf("expected 1 IP, got %d", len(resp.MaxSavingsIPs))
	}
	if len(resp.MaxSavingsIPs) > 0 && resp.MaxSavingsIPs[0] != "192.168.4.2" {
		t.Errorf("expected 192.168.4.2, got %s", resp.MaxSavingsIPs[0])
	}
}
