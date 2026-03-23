package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleCaptiveAccept_POST(t *testing.T) {
	srv := newTestServer(t)

	form := url.Values{"mac": {"aa:bb:cc:dd:ee:01"}}
	req := httptest.NewRequest(http.MethodPost, "/api/captive/accept", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.HandleCaptiveAccept(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify MAC was recorded in DB (portal_accepted table)
	var count int
	err := srv.db.db.QueryRow("SELECT COUNT(*) FROM portal_accepted WHERE mac_addr = ?", "aa:bb:cc:dd:ee:01").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query portal_accepted: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record in portal_accepted, got %d", count)
	}
}

func TestHandleCaptiveAccept_GET(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/captive/accept", nil)
	w := httptest.NewRecorder()
	srv.HandleCaptiveAccept(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", w.Code)
	}
}
