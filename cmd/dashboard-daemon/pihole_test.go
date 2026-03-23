package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewPiHoleClient(t *testing.T) {
	c := NewPiHoleClient("http://localhost:8080")
	if c.BaseURL != "http://localhost:8080" {
		t.Errorf("expected BaseURL http://localhost:8080, got %s", c.BaseURL)
	}
	if c.SessionID != "" {
		t.Errorf("expected empty SessionID, got %s", c.SessionID)
	}
}

func TestAuthenticate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/auth" {
			t.Errorf("expected /api/auth, got %s", r.URL.Path)
		}
		// Verify request body contains password
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["password"] != "testpass" {
			t.Errorf("expected password testpass, got %s", body["password"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session": map[string]interface{}{
				"valid": true,
				"sid":   "test-session-123",
			},
		})
	}))
	defer ts.Close()

	c := NewPiHoleClient(ts.URL)
	err := c.Authenticate("testpass")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if c.SessionID != "test-session-123" {
		t.Errorf("expected session ID test-session-123, got %s", c.SessionID)
	}
}

func TestFetchTopDomains(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stats/top_domains" {
			t.Errorf("expected /api/stats/top_domains, got %s", r.URL.Path)
		}
		if r.Header.Get("X-FTL-SID") != "my-session" {
			t.Errorf("expected X-FTL-SID my-session, got %s", r.Header.Get("X-FTL-SID"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"top_domains": []map[string]interface{}{
				{"domain": "facebook.com", "count": 500},
				{"domain": "youtube.com", "count": 300},
				{"domain": "google.com", "count": 200},
			},
		})
	}))
	defer ts.Close()

	c := NewPiHoleClient(ts.URL)
	c.SessionID = "my-session"
	domains, err := c.FetchTopDomains(10)
	if err != nil {
		t.Fatalf("FetchTopDomains failed: %v", err)
	}
	if len(domains) != 3 {
		t.Fatalf("expected 3 domains, got %d", len(domains))
	}
	if domains[0].Domain != "facebook.com" {
		t.Errorf("expected facebook.com, got %s", domains[0].Domain)
	}
	if domains[0].Count != 500 {
		t.Errorf("expected count 500, got %d", domains[0].Count)
	}
	if domains[1].Domain != "youtube.com" {
		t.Errorf("expected youtube.com, got %s", domains[1].Domain)
	}
}

func TestFetchTopDomains_AuthError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
	}))
	defer ts.Close()

	c := NewPiHoleClient(ts.URL)
	_, err := c.FetchTopDomains(10)
	if err == nil {
		t.Error("expected error for 401 response")
	}
}

func TestFetchBlockedCount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stats/summary" {
			t.Errorf("expected /api/stats/summary, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domains_being_blocked": 150000,
			"ads_blocked_today":    2500,
			"dns_queries_today":    10000,
		})
	}))
	defer ts.Close()

	c := NewPiHoleClient(ts.URL)
	c.SessionID = "test-sid"
	summary, err := c.FetchBlockedCount()
	if err != nil {
		t.Fatalf("FetchBlockedCount failed: %v", err)
	}
	if summary.QueriesBlocked != 2500 {
		t.Errorf("expected 2500 blocked queries, got %d", summary.QueriesBlocked)
	}
	if summary.TotalQueries != 10000 {
		t.Errorf("expected 10000 total queries, got %d", summary.TotalQueries)
	}
	if summary.DomainsBlocked != 150000 {
		t.Errorf("expected 150000 domains blocked, got %d", summary.DomainsBlocked)
	}
}
