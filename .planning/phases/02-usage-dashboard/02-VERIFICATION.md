---
phase: 02-usage-dashboard
verified: 2026-03-23T16:27:04Z
status: passed
score: 5/5 success criteria verified
re_verification: false
---

# Phase 2: Usage Dashboard Verification Report

**Phase Goal:** Pilots can see exactly what's eating their Starlink data cap in real time, with dollar-amount savings and plan cap tracking -- the viral screenshot moment
**Verified:** 2026-03-23T16:27:04Z
**Status:** PASSED
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A passenger connecting for the first time is intercepted by a captive portal with terms acceptance before internet access is granted | VERIFIED | `nftables-dashboard.conf.j2` has DNAT rule redirecting unauthenticated MACs to Caddy; `captive.go:HandleCaptiveAccept` adds MAC to `allowed_macs` nftables set; `captive.html` is iOS CNA-safe plain HTML form POST to `/api/captive/accept` |
| 2 | Dashboard shows per-device breakdown with top domains by bytes consumed and a category pie chart | VERIFIED | SSE `event: devices` sends HTML fragment of device table; `event: categories` sends JSON for Chart.js doughnut chart; `HandleGetDomains` returns domain+category data from Pi-hole + CategoryMap; `index.html` has `id="category-pie"` canvas |
| 3 | Real-time bandwidth graph updates live showing current throughput across all connected devices | VERIFIED | SSE `event: bandwidth` and `event: chart-data` emitted every 5s from `HandleSSE`; `index.html` has `id="bw-chart"` canvas initialized with `new Chart()` and updated via `htmx:sseMessage`; `htmx-ext-sse.js` (9KB), `htmx.min.js` (51KB), `chart.umd.min.js` (209KB) all present |
| 4 | Dashboard displays a dollar amount of bandwidth saved based on Starlink overage rates | VERIFIED | `savings.go:CalcSavings` produces `FormattedAmount: fmt.Sprintf("$%.2f", dollarAmount)` using 150KB/ad and 5KB/tracker constants; SSE `event: savings` sends HTML fragment `<p class="savings-amount">$X.XX saved this session</p>`; CSS `.savings-amount` styled in green |
| 5 | User can configure their Starlink plan cap and see usage-against-cap with alerts at 50%, 75%, and 90% thresholds | VERIFIED | `settings.html` has `plan_cap_gb`, `billing_cycle_start`, `overage_rate_per_mb` inputs with `hx-put="/api/settings"` and `fetch('/api/settings')` pre-populate; `renderCapStatusHTML` computes percentage with green/yellow/orange/red classes; `renderAlertHTML` fires at 50%/75%/90% via SSE `event: alerts`; `TestHandlePutSettings`, `TestCapStatusHTML_Green`, `TestCapStatusHTML_Red`, `TestRenderAlertHTML` all pass |

**Score:** 5/5 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/dashboard-daemon/config.go` | Config struct with YAML loading | VERIFIED | `type Config struct` with Port, PollIntervalSec, PiHoleAddress, PiHolePassword, DBPath, CategoriesFile, StaticDir; `func LoadConfig` present |
| `cmd/dashboard-daemon/db.go` | SQLite schema + WAL mode | VERIFIED | `PRAGMA journal_mode=WAL`; all 6 tables (device_usage, devices, domain_stats, settings, savings_log, portal_accepted) via `CREATE TABLE IF NOT EXISTS`; default settings INSERT OR IGNORE for plan_cap_gb, billing_cycle_start, overage_rate_per_mb |
| `cmd/dashboard-daemon/nftables.go` | nft JSON parser + delta computation | VERIFIED | `func ParseNftCounters(data []byte) (map[string]uint64, error)` and `func ComputeDeltas` both present; 6 nftables tests pass |
| `cmd/dashboard-daemon/categories.go` | Domain-to-category mapping | VERIFIED | `func LoadCategories(path string)` and `func (cm *CategoryMap) Categorize(domain string) string` present; subdomain walking tested |
| `cmd/dashboard-daemon/pihole.go` | Pi-hole FTL v6 API client | VERIFIED | `func NewPiHoleClient`, `Authenticate`, `FetchTopDomains`, `FetchBlockedCount` all present; `X-FTL-SID` header set; httptest-based tests pass |
| `cmd/dashboard-daemon/savings.go` | Savings calculator | VERIFIED | `CalcSavings` with AvgAdPayloadBytes=153600, AvgTrackerPayloadBytes=5120; `SavingsResult` with `FormattedAmount`; 4 table-driven tests pass |
| `cmd/dashboard-daemon/sse.go` | SSE streaming handler | VERIFIED | `func (s *Server) HandleSSE` emits 6 named events: bandwidth, chart-data, devices, cap-status, savings, categories; plus threshold alerts; `flusher.Flush()` present; disconnect cleanup tested |
| `cmd/dashboard-daemon/api.go` | REST API handlers | VERIFIED | `type Server struct`, `func NewServer`, `HandleGetDevices`, `HandleGetDomains`, `HandleGetSavings`, `HandleGetSettings`, `HandlePutSettings` all present; in-memory DB tests pass |
| `cmd/dashboard-daemon/captive.go` | Captive portal accept handler | VERIFIED | `func (s *Server) HandleCaptiveAccept`; calls `AcceptDevice`; records in `portal_accepted` table; POST returns 200, GET returns 405 |
| `cmd/dashboard-daemon/main.go` | Daemon entrypoint | VERIFIED | `LoadConfig`, `NewDB`, `LoadCategories`, `NewPiHoleClient`, `NewServer`; all routes registered including `/api/events`, `/api/stats/devices`, `/api/settings`, `/api/captive/accept`; `signal.Notify` for graceful shutdown; `StartPolling` goroutine |
| `cmd/dashboard-daemon/nftables_linux.go` | Linux nftables operations | VERIFIED | `//go:build linux`; `ReadPerMACCounters`, `AddAllowedMAC`, `RemoveAllowedMAC`, `IsAllowedMAC` |
| `cmd/dashboard-daemon/nftables_stub.go` | macOS dev stubs | VERIFIED | `//go:build !linux`; returns mock data for 3 devices |
| `pi/config/dashboard.yaml` | Production config defaults | VERIFIED | `port: 8081`, `poll_interval_sec: 5` present |
| `pi/config/domain-categories.yaml` | Domain-to-category mapping file | VERIFIED | 125 domains across 8 categories (Social Media, Streaming, Ads and Trackers, OS Updates, Cloud Sync, News, Aviation, Other) |
| `pi/static/index.html` | Dashboard with SSE + charts | VERIFIED | `hx-ext="sse"`; `sse-connect="/api/events"`; `sse-swap="cap-status"`, `sse-swap="savings"`, `sse-swap="devices"`; `id="bw-chart"`, `id="category-pie"`; `new Chart(` initializations; `chart.umd.min.js` included |
| `pi/static/captive.html` | Captive portal page | VERIFIED | `action="/api/captive/accept"` `method="POST"`; no JavaScript dependencies |
| `pi/static/settings.html` | Settings configuration page | VERIFIED | `plan_cap_gb`, `billing_cycle_start`, `overage_rate_per_mb` inputs; `hx-put="/api/settings"`; `fetch('/api/settings')` on `DOMContentLoaded` |
| `pi/static/style.css` | Mobile-first responsive CSS | VERIFIED | 332 lines; `.cap-fill.green/yellow/orange/red`; `.savings-amount`; `.device-table`; `.alert-warning`, `.alert-danger`; `.captive`, `.accept-btn`; `@media (min-width: 768px)` |
| `pi/static/htmx.min.js` | HTMX 2.0.8 runtime | VERIFIED | 51,250 bytes |
| `pi/static/htmx-ext-sse.js` | HTMX SSE extension 2.2.4 | VERIFIED | 8,921 bytes |
| `pi/static/chart.umd.min.js` | Chart.js 4.5.1 UMD build | VERIFIED | 208,522 bytes |
| `pi/ansible/roles/dashboard/tasks/main.yml` | Ansible deployment tasks | VERIFIED | 14 tasks deploying binary, static files, config, Caddy, systemd service, nftables additions |
| `pi/ansible/roles/dashboard/templates/Caddyfile.j2` | Caddy reverse proxy + CNA interception | VERIFIED | `@captive_check` matcher on `captive.apple.com`, `connectivitycheck.gstatic.com`, `www.msftconnecttest.com`, `msftconnecttest.com`; `reverse_proxy localhost:{{ dashboard_port }}`; `/api/*` proxied; `/static/*` served |
| `pi/ansible/roles/dashboard/templates/skygate-dashboard.service.j2` | systemd service template | VERIFIED | `ExecStart={{ skygate_opt_dir }}/skygate-dashboard`; `After=network-online.target nftables.service pihole-FTL.service`; `ProtectSystem=strict`; `ReadWritePaths={{ skygate_data_dir }}` |
| `pi/ansible/roles/dashboard/templates/nftables-dashboard.conf.j2` | nftables captive portal rules | VERIFIED | `allowed_macs` set (24h timeout); `device_counters` set (dynamic, counter); DNAT rule for unauthenticated MACs; per-device byte tracking rule |
| `pi/systemd/skygate-dashboard.service` | Reference systemd service | VERIFIED | `ExecStart=/opt/skygate/skygate-dashboard`; `Restart=always` |
| `Makefile` | Dashboard build targets | VERIFIED | `DASHBOARD_BINARY = skygate-dashboard`; `build-dashboard:`, `cross-build-dashboard:` with `CGO_ENABLED=0`; `build:` and `cross-build:` aggregate both daemons |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `cmd/dashboard-daemon/db.go` | `modernc.org/sqlite` | database/sql driver import | VERIFIED | `go.mod` line 7: `modernc.org/sqlite v1.47.0` |
| `cmd/dashboard-daemon/main.go` | `cmd/dashboard-daemon/db.go` | `NewDB` call | VERIFIED | `main.go:40: db, err := NewDB(cfg.DBPath)` |
| `cmd/dashboard-daemon/main.go` | `cmd/dashboard-daemon/sse.go` | HTTP handler registration | VERIFIED | `main.go:74: mux.HandleFunc("/api/events", srv.HandleSSE)` |
| `cmd/dashboard-daemon/sse.go` | `cmd/dashboard-daemon/nftables.go` | `ReadPerMACCounters` + `ComputeDeltas` in event loop | VERIFIED | `sse.go:115,122`: both called in `HandleSSE` and `collectAndPersist` |
| `cmd/dashboard-daemon/api.go` | `cmd/dashboard-daemon/db.go` | DB query calls in handlers | VERIFIED | `api.go:77,164,192`: `GetDeviceUsage`, `GetSettings`, `PutSetting` called in handlers |
| `cmd/dashboard-daemon/captive.go` | `cmd/dashboard-daemon/captive_stub.go` (non-Linux) | `AcceptDevice` on accept | VERIFIED | `captive.go:38: AcceptDevice(mac, ip)` |
| `cmd/dashboard-daemon/savings.go` | `cmd/dashboard-daemon/pihole.go` | `FetchBlockedCount` for savings input | VERIFIED | `api.go:HandleGetSavings` calls Pi-hole client + `CalcSavings` |
| `pi/static/index.html` | `/api/events` | HTMX SSE connection | VERIFIED | `index.html:19: sse-connect="/api/events"` |
| `pi/static/index.html` | `/static/chart.umd.min.js` | script tag | VERIFIED | `index.html: src="/static/chart.umd.min.js"` |
| `pi/static/captive.html` | `/api/captive/accept` | form POST | VERIFIED | `captive.html:23: action="/api/captive/accept" method="POST"` |
| `pi/static/settings.html` | `/api/settings` | HTMX PUT + fetch() pre-populate | VERIFIED | `settings.html:16: hx-put="/api/settings"`; `settings.html:44: fetch('/api/settings')` |
| `pi/ansible/roles/dashboard/templates/Caddyfile.j2` | `cmd/dashboard-daemon/main.go` | reverse_proxy to daemon port | VERIFIED | `Caddyfile.j2:25: reverse_proxy localhost:{{ dashboard_port }}` |
| `pi/ansible/roles/dashboard/templates/nftables-dashboard.conf.j2` | `cmd/dashboard-daemon/captive.go` | `allowed_macs` set shared | VERIFIED | nftables set name matches nftables_linux.go constant `nftAllowedSet = "allowed_macs"` |
| `Makefile` | `cmd/dashboard-daemon/` | go build target | VERIFIED | `Makefile:21: go build -o bin/$(DASHBOARD_BINARY) ./cmd/dashboard-daemon/` |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| `pi/static/index.html` | bandwidth, devices, cap-status, savings, categories | SSE `/api/events` -> `HandleSSE` -> `ReadPerMACCounters` (nftables stub on macOS / real on Linux) + `WriteUsageSnapshot` (SQLite) | Yes -- nftables counters flow to SQLite, then to SSE events as HTML fragments and JSON | FLOWING |
| `pi/static/settings.html` | plan_cap_gb, billing_cycle_start, overage_rate_per_mb | `fetch('/api/settings')` -> `HandleGetSettings` -> `db.GetSettings()` -> SQLite settings table | Yes -- DB returns real values including defaults seeded by `NewDB` | FLOWING |
| `cmd/dashboard-daemon/sse.go` | cap-status HTML | `db.GetSettings()` for plan cap + `ReadPerMACCounters` for current usage | Yes -- percentage computed from real settings and nftables counters | FLOWING |
| `cmd/dashboard-daemon/sse.go` | savings HTML | `pihole.FetchBlockedCount()` -> `CalcSavings()` | Yes on real Pi with Pi-hole; gracefully degrades to $0.00 if Pi-hole unreachable | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 47 unit tests pass | `go test ./cmd/dashboard-daemon/ -short` | `ok github.com/quartermint/skygate/cmd/dashboard-daemon` (cached) | PASS |
| Binary compiles for host platform | `go build ./cmd/dashboard-daemon/` | exits 0 | PASS |
| Cross-compiles for Pi (linux/arm64, CGO_ENABLED=0) | `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /dev/null ./cmd/dashboard-daemon/` | exits 0 | PASS |
| `go vet` reports no issues | `go vet ./cmd/dashboard-daemon/` | exits 0 (no output) | PASS |
| Static JS libraries are real files (not empty) | `wc -c pi/static/*.js` | htmx: 51KB, sse-ext: 9KB, chart: 209KB | PASS |
| Settings page pre-populates on load | `grep -n "DOMContentLoaded\|fetch.*api/settings" pi/static/settings.html` | found at lines 43-44 | PASS |

---

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| DASH-01 | 02-01, 02-03, 02-04 | Per-device data usage tracked in near-real-time (~5s intervals, persisted to SQLite) | SATISFIED | `ReadPerMACCounters` + `ComputeDeltas` in SSE loop; `WriteUsageSnapshot` to SQLite device_usage table; 5s poll interval configured; nftables device_counters set tracks per-MAC bytes |
| DASH-02 | 02-01, 02-02, 02-03 | Web dashboard displays top domains by bytes consumed, per-device breakdown, and category pie chart | SATISFIED | `HandleGetDomains` returns Pi-hole top domains + category mapping; SSE `devices` event sends device table HTML; SSE `categories` event sends JSON for Chart.js doughnut; 125-domain YAML covers 8 categories |
| DASH-03 | 02-02, 02-03 | Dashboard shows real-time bandwidth graph of current throughput | SATISFIED | SSE `chart-data` event sends `{labels, values}` JSON for Chart.js line chart; 60-point ring buffer for 5-minute history; `index.html` initializes Chart.js line chart and updates via `htmx:sseMessage` |
| DASH-04 | 02-02, 02-03, 02-04 | Captive portal intercepts first HTTP request, shows terms acceptance, links to dashboard | SATISFIED | nftables DNAT redirects unauthenticated MACs to Caddy; Caddyfile serves `captive.html` for CNA check URLs (iOS/Android/Windows); `HandleCaptiveAccept` adds to `allowed_macs` nftables set; `portal_accepted` table records acceptance |
| DASH-05 | 02-02, 02-03 | Dashboard displays bandwidth savings as dollar amount based on Starlink overage rates | SATISFIED | `CalcSavings` computes dollar amount using 150KB/ad + 5KB/tracker; `$X.XX` formatted via `fmt.Sprintf`; SSE `savings` event sends HTML with `.savings-amount` class styled green; overage rate configurable in settings |
| DASH-06 | 02-02, 02-03 | User can configure Starlink plan cap and billing cycle; usage against cap with alerts at 50%, 75%, 90% | SATISFIED | `settings.html` has all three configurable fields; `HandlePutSettings` validates and persists; `renderCapStatusHTML` colors bar green/yellow/orange/red by threshold; `renderAlertHTML` emits warning/danger banners at 50%/75%/90% via SSE `alerts` event |

All 6 required requirements fully satisfied. No orphaned requirements found.

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | - | No TODO/FIXME/placeholder/return null stubs found in any Go source or static files | - | - |

No anti-patterns detected. All data paths flow to real implementations (SQLite reads/writes, nftables counter parsing, Pi-hole API calls). Platform stubs exist only in `*_stub.go` files for macOS development, matching the established Phase 1 pattern.

---

### Human Verification Required

#### 1. Captive Portal CNA Sheet Auto-Open

**Test:** Connect an iPhone to the SkyGate WiFi AP (without being in `allowed_macs`). Open Settings -> WiFi and tap on the SkyGate network.
**Expected:** iOS automatically opens the CNA mini-browser sheet showing `captive.html` with the terms acceptance form. Tapping "Accept & Connect" submits the form and grants internet access.
**Why human:** Requires real iOS device, real Raspberry Pi, active nftables DNAT, and Caddy Host-header matching against CNA check domains. Cannot be verified with grep or unit tests.

#### 2. Real-Time Dashboard SSE Updates

**Test:** Load `http://192.168.4.1` on a device connected to SkyGate WiFi. Have a second device actively browsing.
**Expected:** The bandwidth graph updates every 5 seconds showing live throughput. The device table shows the second device with increasing byte counts. The category pie chart reflects domains visited.
**Why human:** Requires live Pi with active nftables counters, real Pi-hole domain data, and running dashboard daemon. SSE streaming visuals cannot be verified programmatically.

#### 3. Dollar Savings Screenshot Moment

**Test:** After 10+ minutes of browsing with Pi-hole ad blocking active, open the dashboard.
**Expected:** A green dollar amount (e.g., "$1.47 saved this session") is prominently displayed. The number is plausible for the browsing activity.
**Why human:** Requires real Pi-hole blocked query count. Savings calculation accuracy and "screenshot worthy" visual impact cannot be verified without a live deployment.

#### 4. Settings Persistence Across Daemon Restarts

**Test:** Configure a 50 GB plan cap in Settings, restart the `skygate-dashboard` service, reload the settings page.
**Expected:** The plan cap field shows 50 GB. The usage bar percentage reflects the new cap.
**Why human:** Requires live Pi with systemd service management and real SQLite persistence.

---

### Gaps Summary

None. All phase 2 goals achieved.

The Go daemon foundation (Plan 01), frontend static files (Plan 02), API layer (Plan 03), and deployment infrastructure (Plan 04) are all complete. 47 unit tests pass. The binary compiles for both the host platform and linux/arm64. All 6 DASH requirements are satisfied by concrete, substantive, wired implementations.

The only items requiring human verification are behavioral end-to-end flows that depend on a live Raspberry Pi, real iOS/Android devices, and active Pi-hole -- none of which represent code gaps.

---

_Verified: 2026-03-23T16:27:04Z_
_Verifier: Claude (gsd-verifier)_
