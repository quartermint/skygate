package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// maxChartHistory is the number of bandwidth readings to keep for the chart (5 min at 5s = 60).
const maxChartHistory = 60

// BandwidthEvent is the JSON payload for the "bandwidth" SSE event.
type BandwidthEvent struct {
	Timestamp int64             `json:"ts"`
	Devices   map[string]uint64 `json:"devices"` // MAC -> bytes/sec
	TotalBps  uint64            `json:"total_bps"`
}

// ChartData is the JSON payload for the "chart-data" SSE event.
type ChartData struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

// CategoryData is the JSON payload for the "categories" SSE event.
type CategoryData struct {
	Labels []string `json:"labels"`
	Values []int    `json:"values"`
}

// HandleSSE streams real-time events to the dashboard via Server-Sent Events.
// Sends 6 named events every poll interval: bandwidth, chart-data, devices,
// cap-status, savings, and categories.
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	interval := time.Duration(s.cfg.PollIntervalSec) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Send initial data immediately
	s.sendSSEEvents(w, flusher)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			s.sendSSEEvents(w, flusher)
		}
	}
}

// sendSSEEvents collects current state and sends all 6 SSE events.
func (s *Server) sendSSEEvents(w http.ResponseWriter, flusher http.Flusher) {
	// 1. Bandwidth event
	bwEvent := s.collectBandwidthEvent()
	if data, err := json.Marshal(bwEvent); err == nil {
		fmt.Fprintf(w, "event: bandwidth\ndata: %s\n\n", data)
	}

	// 2. Chart-data event
	chartData := s.collectChartData(bwEvent)
	if data, err := json.Marshal(chartData); err == nil {
		fmt.Fprintf(w, "event: chart-data\ndata: %s\n\n", data)
	}

	// 3. Devices event (HTML fragment)
	devicesHTML := s.collectDevicesHTML()
	fmt.Fprintf(w, "event: devices\ndata: %s\n\n", devicesHTML)

	// 4. Cap-status event (HTML fragment)
	capHTML := s.collectCapStatusHTML()
	fmt.Fprintf(w, "event: cap-status\ndata: %s\n\n", capHTML)

	// 5. Savings event (HTML fragment)
	savingsHTML := s.collectSavingsHTML()
	fmt.Fprintf(w, "event: savings\ndata: %s\n\n", savingsHTML)

	// 6. Categories event (JSON for pie chart)
	catData := s.collectCategoryData()
	if data, err := json.Marshal(catData); err == nil {
		fmt.Fprintf(w, "event: categories\ndata: %s\n\n", data)
	}

	// Alert events at thresholds
	alertHTML := s.collectAlertHTML()
	if alertHTML != "" {
		fmt.Fprintf(w, "event: alerts\ndata: %s\n\n", alertHTML)
	}

	flusher.Flush()
}

// collectBandwidthEvent reads nftables counters and computes bandwidth deltas.
func (s *Server) collectBandwidthEvent() BandwidthEvent {
	counters, err := ReadPerMACCounters()
	if err != nil {
		log.Printf("WARN: ReadPerMACCounters: %v", err)
		return BandwidthEvent{Timestamp: time.Now().Unix(), Devices: map[string]uint64{}}
	}

	s.mu.Lock()
	deltas := ComputeDeltas(s.prevCounters, counters)
	s.prevCounters = counters
	s.mu.Unlock()

	// Convert deltas to bytes/sec based on poll interval
	interval := float64(s.cfg.PollIntervalSec)
	if interval <= 0 {
		interval = 5.0
	}

	bps := make(map[string]uint64)
	var totalBps uint64
	for mac, delta := range deltas {
		rate := uint64(float64(delta) / interval)
		bps[mac] = rate
		totalBps += rate
	}

	return BandwidthEvent{
		Timestamp: time.Now().Unix(),
		Devices:   bps,
		TotalBps:  totalBps,
	}
}

// collectChartData builds the chart history data points.
func (s *Server) collectChartData(bw BandwidthEvent) ChartData {
	// Convert total bps to Mbps
	mbps := float64(bw.TotalBps) / (1024 * 1024) * 8

	point := chartPoint{
		Label: time.Now().Format("15:04:05"),
		Value: mbps,
	}

	s.mu.Lock()
	s.chartHistory = append(s.chartHistory, point)
	if len(s.chartHistory) > maxChartHistory {
		s.chartHistory = s.chartHistory[len(s.chartHistory)-maxChartHistory:]
	}
	history := make([]chartPoint, len(s.chartHistory))
	copy(history, s.chartHistory)
	s.mu.Unlock()

	labels := make([]string, len(history))
	values := make([]float64, len(history))
	for i, p := range history {
		labels[i] = p.Label
		values[i] = p.Value
	}

	return ChartData{Labels: labels, Values: values}
}

// collectDevicesHTML generates an HTML table body fragment for the device table.
func (s *Server) collectDevicesHTML() string {
	devices, err := s.db.GetDevices()
	if err != nil {
		log.Printf("WARN: GetDevices for SSE: %v", err)
		return "<tr><td colspan=\"4\">Error loading devices</td></tr>"
	}

	if len(devices) == 0 {
		return "<tr><td colspan=\"4\">No devices connected</td></tr>"
	}

	// Find max bytes for bar width calculation
	type devRow struct {
		Name   string
		MAC    string
		Bytes  uint64
		Domain string
	}
	var rows []devRow
	var maxBytes uint64
	for _, d := range devices {
		name := resolveDeviceName(d)
		snaps, _ := s.db.GetDeviceUsage(d.MAC, 1)
		var total uint64
		if len(snaps) > 0 {
			total = snaps[0].BytesTotal
		}
		if total > maxBytes {
			maxBytes = total
		}
		rows = append(rows, devRow{Name: name, MAC: d.MAC, Bytes: total})
	}

	var sb strings.Builder
	for _, row := range rows {
		pct := 0.0
		if maxBytes > 0 {
			pct = float64(row.Bytes) / float64(maxBytes) * 100
		}
		sb.WriteString(fmt.Sprintf(
			"<tr><td>%s</td><td>%s</td><td>%s</td><td><div class=\"usage-bar\" style=\"width:%.0f%%\"></div></td></tr>",
			row.Name, row.MAC, formatBytes(row.Bytes), pct,
		))
	}
	return sb.String()
}

// collectCapStatusHTML generates the plan cap progress bar HTML fragment.
func (s *Server) collectCapStatusHTML() string {
	settings, err := s.db.GetSettings()
	if err != nil {
		return renderCapStatusHTML(0, 20*1024*1024*1024)
	}

	capGB := 20.0
	if v, ok := settings["plan_cap_gb"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			capGB = parsed
		}
	}
	capBytes := uint64(capGB * 1024 * 1024 * 1024)

	// Sum total usage from all devices
	devices, _ := s.db.GetDevices()
	var totalUsed uint64
	for _, d := range devices {
		snaps, _ := s.db.GetDeviceUsage(d.MAC, 1)
		if len(snaps) > 0 {
			totalUsed += snaps[0].BytesTotal
		}
	}

	return renderCapStatusHTML(totalUsed, capBytes)
}

// collectSavingsHTML generates the savings display HTML fragment.
func (s *Server) collectSavingsHTML() string {
	summary, err := s.pihole.FetchBlockedCount()
	if err != nil {
		return renderSavingsHTML("$0.00")
	}

	cfg := DefaultSavingsConfig()
	settings, err := s.db.GetSettings()
	if err == nil {
		if rate, ok := settings["overage_rate_per_mb"]; ok {
			if r, err := strconv.ParseFloat(rate, 64); err == nil {
				cfg.OverageRatePerMB = r
			}
		}
	}

	result := CalcSavings(summary.QueriesBlocked, 0, cfg)
	return renderSavingsHTML(result.FormattedAmount)
}

// collectCategoryData gathers domain categories for the pie chart.
func (s *Server) collectCategoryData() CategoryData {
	domains, err := s.pihole.FetchTopDomains(20)
	if err != nil {
		return CategoryData{Labels: []string{}, Values: []int{}}
	}

	catCounts := make(map[string]int)
	for _, d := range domains {
		cat := "Other"
		if s.categories != nil {
			cat = s.categories.Categorize(d.Domain)
		}
		catCounts[cat] += d.Count
	}

	labels := make([]string, 0, len(catCounts))
	values := make([]int, 0, len(catCounts))
	for cat, count := range catCounts {
		labels = append(labels, cat)
		values = append(values, count)
	}

	return CategoryData{Labels: labels, Values: values}
}

// collectAlertHTML generates alert banners at usage thresholds.
func (s *Server) collectAlertHTML() string {
	settings, _ := s.db.GetSettings()
	capGB := 20.0
	if v, ok := settings["plan_cap_gb"]; ok {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			capGB = parsed
		}
	}
	capBytes := uint64(capGB * 1024 * 1024 * 1024)

	devices, _ := s.db.GetDevices()
	var totalUsed uint64
	for _, d := range devices {
		snaps, _ := s.db.GetDeviceUsage(d.MAC, 1)
		if len(snaps) > 0 {
			totalUsed += snaps[0].BytesTotal
		}
	}

	if capBytes == 0 {
		return ""
	}
	pct := float64(totalUsed) / float64(capBytes) * 100
	return renderAlertHTML(pct)
}

// StartPolling runs the data collection loop that persists nftables counters to SQLite.
func (s *Server) StartPolling(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.cfg.PollIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectAndPersist()
		}
	}
}

// collectAndPersist reads nftables counters, computes deltas, and writes to SQLite.
func (s *Server) collectAndPersist() {
	counters, err := ReadPerMACCounters()
	if err != nil {
		log.Printf("WARN: failed to read nft counters: %v", err)
		return
	}

	s.mu.Lock()
	deltas := ComputeDeltas(s.prevCounters, counters)
	s.prevCounters = counters
	s.mu.Unlock()

	for mac, delta := range deltas {
		total := counters[mac]
		if err := s.db.WriteUsageSnapshot(mac, total, delta); err != nil {
			log.Printf("WARN: failed to write usage: %v", err)
		}
		if err := s.db.WriteDevice(mac, "", ""); err != nil {
			log.Printf("WARN: failed to write device: %v", err)
		}
	}
}

// --- HTML fragment rendering helpers ---

// renderCapStatusHTML generates the plan cap progress bar HTML with color escalation.
// Color classes: green (<50%), yellow (50-75%), orange (75-90%), red (>90%).
func renderCapStatusHTML(usedBytes, capBytes uint64) string {
	if capBytes == 0 {
		return `<div class="cap-bar"><div class="cap-fill green" style="width:0%"></div></div><p class="cap-text">0 B / 0 B used</p>`
	}
	pct := float64(usedBytes) / float64(capBytes) * 100
	if pct > 100 {
		pct = 100
	}

	color := "green"
	switch {
	case pct >= 90:
		color = "red"
	case pct >= 75:
		color = "orange"
	case pct >= 50:
		color = "yellow"
	}

	return fmt.Sprintf(
		`<div class="cap-bar"><div class="cap-fill %s" style="width:%.0f%%"></div></div><p class="cap-text">%s / %s used</p>`,
		color, pct, formatBytes(usedBytes), formatBytes(capBytes),
	)
}

// renderDevicesHTML generates HTML table body rows for devices.
func renderDevicesHTML(devices []DeviceResponse) string {
	if len(devices) == 0 {
		return "<tr><td colspan=\"4\">No devices connected</td></tr>"
	}
	var sb strings.Builder
	for _, d := range devices {
		sb.WriteString(fmt.Sprintf(
			"<tr><td>%s</td><td>%s</td><td>%s</td></tr>",
			d.Name, formatBytes(d.BytesTotal), d.TopDomain,
		))
	}
	return sb.String()
}

// renderSavingsHTML generates the savings display paragraph.
func renderSavingsHTML(amount string) string {
	return fmt.Sprintf(`<p class="savings-amount">%s saved this session</p>`, amount)
}

// renderAlertHTML generates alert banners at usage thresholds per D-20.
// Thresholds: 50% (warning), 75% (warning), 90% (danger).
func renderAlertHTML(pct float64) string {
	switch {
	case pct >= 90:
		return fmt.Sprintf(`<div class="alert alert-danger">Data usage at %.0f%% of plan cap -- consider reducing usage</div>`, pct)
	case pct >= 75:
		return fmt.Sprintf(`<div class="alert alert-warning">Data usage at %.0f%% of plan cap</div>`, pct)
	case pct >= 50:
		return fmt.Sprintf(`<div class="alert alert-warning">Data usage at %.0f%% of plan cap</div>`, pct)
	default:
		return ""
	}
}
