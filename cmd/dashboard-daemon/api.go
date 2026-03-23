package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

// Server holds shared state across HTTP handlers.
type Server struct {
	cfg          *Config
	db           *DB
	categories   *CategoryMap
	pihole       *PiHoleClient
	prevCounters map[string]uint64 // previous nftables counter snapshot for delta computation
	mu           sync.RWMutex     // protects prevCounters and chartHistory
	chartHistory []chartPoint     // ring buffer of bandwidth readings for chart
}

// chartPoint stores a single bandwidth measurement for the chart history.
type chartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"` // Mbps
}

// DeviceResponse represents a device in the REST API response.
type DeviceResponse struct {
	MAC        string `json:"mac"`
	Name       string `json:"name"`
	BytesTotal uint64 `json:"bytes_total"`
	TopDomain  string `json:"top_domain,omitempty"`
}

// DomainResponse represents a domain in the REST API response.
type DomainResponse struct {
	Domain     string `json:"domain"`
	Category   string `json:"category"`
	QueryCount int    `json:"query_count"`
}

// validSettingsKeys defines the allowed settings keys for PUT requests.
var validSettingsKeys = map[string]bool{
	"plan_cap_gb":        true,
	"billing_cycle_start": true,
	"overage_rate_per_mb": true,
}

// NewServer creates a new Server with the given dependencies.
func NewServer(cfg *Config, db *DB, cats *CategoryMap, pihole *PiHoleClient) *Server {
	return &Server{
		cfg:          cfg,
		db:           db,
		categories:   cats,
		pihole:       pihole,
		prevCounters: make(map[string]uint64),
		chartHistory: make([]chartPoint, 0, 60),
	}
}

// HandleGetDevices returns all known devices with their latest usage.
// GET /api/stats/devices -> JSON array of device objects.
func (s *Server) HandleGetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.db.GetDevices()
	if err != nil {
		log.Printf("ERROR: GetDevices: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var result []DeviceResponse
	for _, d := range devices {
		name := resolveDeviceName(d)
		// Get the latest usage snapshot for total bytes
		snapshots, _ := s.db.GetDeviceUsage(d.MAC, 1)
		var total uint64
		if len(snapshots) > 0 {
			total = snapshots[0].BytesTotal
		}
		result = append(result, DeviceResponse{
			MAC:        d.MAC,
			Name:       name,
			BytesTotal: total,
		})
	}

	// Ensure non-nil slice for JSON encoding ([] vs null)
	if result == nil {
		result = []DeviceResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleGetDomains returns top domains with categories.
// GET /api/stats/domains -> JSON array of domain objects.
func (s *Server) HandleGetDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := s.pihole.FetchTopDomains(20)
	if err != nil {
		log.Printf("WARN: FetchTopDomains: %v", err)
		// Return empty array on failure (Pi-hole may be down)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]DomainResponse{})
		return
	}

	var result []DomainResponse
	for _, d := range domains {
		cat := "Other"
		if s.categories != nil {
			cat = s.categories.Categorize(d.Domain)
		}
		result = append(result, DomainResponse{
			Domain:     d.Domain,
			Category:   cat,
			QueryCount: d.Count,
		})
	}

	if result == nil {
		result = []DomainResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleGetSavings returns estimated bandwidth savings from DNS blocking.
// GET /api/stats/savings -> JSON savings object.
func (s *Server) HandleGetSavings(w http.ResponseWriter, r *http.Request) {
	summary, err := s.pihole.FetchBlockedCount()
	if err != nil {
		log.Printf("WARN: FetchBlockedCount: %v", err)
		// Return zero savings on failure
		result := CalcSavings(0, 0, DefaultSavingsConfig())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Load overage rate from settings
	cfg := DefaultSavingsConfig()
	settings, err := s.db.GetSettings()
	if err == nil {
		if rate, ok := settings["overage_rate_per_mb"]; ok {
			if r, err := strconv.ParseFloat(rate, 64); err == nil {
				cfg.OverageRatePerMB = r
			}
		}
	}

	// Use blocked queries as ad count estimate (conservative: all ads, zero trackers)
	result := CalcSavings(summary.QueriesBlocked, 0, cfg)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleGetSettings returns all settings as a JSON key-value map.
// GET /api/settings -> JSON settings map.
func (s *Server) HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.GetSettings()
	if err != nil {
		log.Printf("ERROR: GetSettings: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// HandlePutSettings updates settings from a JSON body containing key-value pairs.
// PUT /api/settings -> validates keys, writes each to DB, returns 200.
func (s *Server) HandlePutSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(settings) == 0 {
		http.Error(w, "empty settings", http.StatusBadRequest)
		return
	}

	for key, value := range settings {
		if !validSettingsKeys[key] {
			http.Error(w, fmt.Sprintf("invalid setting key: %s", key), http.StatusBadRequest)
			return
		}
		if err := s.db.PutSetting(key, value); err != nil {
			log.Printf("ERROR: PutSetting %s: %v", key, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// resolveDeviceName determines the display name for a device using multi-tier resolution.
// Priority: user-assigned name > DHCP hostname > OUI vendor > truncated MAC.
func resolveDeviceName(d Device) string {
	if d.UserName != "" {
		return d.UserName
	}
	if d.Hostname != "" {
		return d.Hostname
	}
	if d.OUIVendor != "" {
		return d.OUIVendor
	}
	// Truncate MAC for display
	if len(d.MAC) >= 8 {
		return d.MAC[:8] + "..."
	}
	return d.MAC
}

// formatBytes converts a byte count to a human-readable string.
func formatBytes(b uint64) string {
	const (
		_        = iota
		kilobyte = 1 << (10 * iota)
		megabyte
		gigabyte
	)
	switch {
	case b >= gigabyte:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gigabyte))
	case b >= megabyte:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(megabyte))
	case b >= kilobyte:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kilobyte))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
