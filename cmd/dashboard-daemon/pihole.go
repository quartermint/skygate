package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DomainStat represents a single domain with its query count from Pi-hole.
type DomainStat struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// PiHoleSummary holds aggregate stats from the Pi-hole FTL summary API.
type PiHoleSummary struct {
	DomainsBlocked uint64 `json:"domains_being_blocked"`
	QueriesBlocked uint64 `json:"ads_blocked_today"`
	TotalQueries   uint64 `json:"dns_queries_today"`
}

// PiHoleClient communicates with the Pi-hole FTL v6 REST API.
type PiHoleClient struct {
	BaseURL    string
	SessionID  string
	httpClient *http.Client
}

// NewPiHoleClient creates a new Pi-hole API client with a 5-second timeout.
func NewPiHoleClient(baseURL string) *PiHoleClient {
	return &PiHoleClient{
		BaseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Authenticate performs session-based authentication against the Pi-hole v6 API.
// POST /api/auth with {"password": password} and stores the returned session ID.
func (c *PiHoleClient) Authenticate(password string) error {
	body, err := json.Marshal(map[string]string{"password": password})
	if err != nil {
		return fmt.Errorf("marshaling auth request: %w", err)
	}

	resp, err := c.httpClient.Post(c.BaseURL+"/api/auth", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("Pi-hole auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Pi-hole auth failed with status %d", resp.StatusCode)
	}

	var result struct {
		Session struct {
			Valid bool   `json:"valid"`
			SID   string `json:"sid"`
		} `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing auth response: %w", err)
	}
	if !result.Session.Valid {
		return fmt.Errorf("Pi-hole auth: session not valid")
	}
	c.SessionID = result.Session.SID
	return nil
}

// FetchTopDomains retrieves the top queried domains from Pi-hole.
// GET /api/stats/top_domains?count={limit} with session auth header.
func (c *PiHoleClient) FetchTopDomains(limit int) ([]DomainStat, error) {
	url := fmt.Sprintf("%s/api/stats/top_domains?count=%d", c.BaseURL, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating top domains request: %w", err)
	}
	req.Header.Set("X-FTL-SID", c.SessionID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching top domains: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("Pi-hole API: not authenticated (401)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Pi-hole API: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		TopDomains []DomainStat `json:"top_domains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing top domains response: %w", err)
	}
	return result.TopDomains, nil
}

// FetchBlockedCount retrieves the Pi-hole summary including blocked query count.
// GET /api/stats/summary with session auth header.
func (c *PiHoleClient) FetchBlockedCount() (*PiHoleSummary, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/stats/summary", nil)
	if err != nil {
		return nil, fmt.Errorf("creating summary request: %w", err)
	}
	req.Header.Set("X-FTL-SID", c.SessionID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching summary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Pi-hole summary: unexpected status %d", resp.StatusCode)
	}

	var summary PiHoleSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("parsing summary response: %w", err)
	}
	return &summary, nil
}
