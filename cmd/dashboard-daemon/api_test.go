package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleGetDevices(t *testing.T) {
	srv := newTestServer(t)

	// Seed a device
	if err := srv.db.WriteDevice("aa:bb:cc:dd:ee:01", "iPhone", "Apple"); err != nil {
		t.Fatalf("failed to seed device: %v", err)
	}
	if err := srv.db.WriteUsageSnapshot("aa:bb:cc:dd:ee:01", 1048576, 1048576); err != nil {
		t.Fatalf("failed to seed usage: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/stats/devices", nil)
	w := httptest.NewRecorder()
	srv.HandleGetDevices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var devices []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&devices); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(devices) < 1 {
		t.Fatalf("expected at least 1 device, got %d", len(devices))
	}
	if _, ok := devices[0]["mac"]; !ok {
		t.Error("expected device to have 'mac' field")
	}
	if _, ok := devices[0]["bytes_total"]; !ok {
		t.Error("expected device to have 'bytes_total' field")
	}
}

func TestHandleGetDomains(t *testing.T) {
	// Create a mock Pi-hole server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"top_domains": []map[string]interface{}{
				{"domain": "facebook.com", "count": 500},
				{"domain": "google.com", "count": 300},
			},
		})
	}))
	defer ts.Close()

	srv := newTestServer(t)
	srv.pihole = NewPiHoleClient(ts.URL)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/domains", nil)
	w := httptest.NewRecorder()
	srv.HandleGetDomains(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var domains []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&domains); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0]["domain"] != "facebook.com" {
		t.Errorf("expected facebook.com, got %v", domains[0]["domain"])
	}
	if _, ok := domains[0]["category"]; !ok {
		t.Error("expected domain to have 'category' field")
	}
}

func TestHandleGetSavings(t *testing.T) {
	// Create a mock Pi-hole server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domains_being_blocked": 150000,
			"ads_blocked_today":    100,
			"dns_queries_today":    1000,
		})
	}))
	defer ts.Close()

	srv := newTestServer(t)
	srv.pihole = NewPiHoleClient(ts.URL)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/savings", nil)
	w := httptest.NewRecorder()
	srv.HandleGetSavings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := result["blocked_queries"]; !ok {
		t.Error("expected 'blocked_queries' field")
	}
	if _, ok := result["estimated_bytes_saved"]; !ok {
		t.Error("expected 'estimated_bytes_saved' field")
	}
	if _, ok := result["dollar_amount"]; !ok {
		t.Error("expected 'dollar_amount' field")
	}
	if _, ok := result["formatted_amount"]; !ok {
		t.Error("expected 'formatted_amount' field")
	}
}

func TestHandleGetSettings(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	srv.HandleGetSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var settings map[string]string
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if settings["plan_cap_gb"] != "20" {
		t.Errorf("expected plan_cap_gb=20, got %s", settings["plan_cap_gb"])
	}
	if settings["billing_cycle_start"] != "1" {
		t.Errorf("expected billing_cycle_start=1, got %s", settings["billing_cycle_start"])
	}
	if settings["overage_rate_per_mb"] != "0.01" {
		t.Errorf("expected overage_rate_per_mb=0.01, got %s", settings["overage_rate_per_mb"])
	}
}

func TestHandlePutSettings(t *testing.T) {
	srv := newTestServer(t)

	body := `{"plan_cap_gb":"50","overage_rate_per_mb":"0.02"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.HandlePutSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify settings were updated
	getReq := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	getW := httptest.NewRecorder()
	srv.HandleGetSettings(getW, getReq)

	var settings map[string]string
	if err := json.NewDecoder(getW.Body).Decode(&settings); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if settings["plan_cap_gb"] != "50" {
		t.Errorf("expected plan_cap_gb=50 after PUT, got %s", settings["plan_cap_gb"])
	}
	if settings["overage_rate_per_mb"] != "0.02" {
		t.Errorf("expected overage_rate_per_mb=0.02 after PUT, got %s", settings["overage_rate_per_mb"])
	}
}

func TestHandlePutSettings_Invalid(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.HandlePutSettings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", w.Code)
	}
}
