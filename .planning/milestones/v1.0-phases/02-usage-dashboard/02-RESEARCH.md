# Phase 2: Usage Dashboard - Research

**Researched:** 2026-03-23
**Domain:** Go daemon + HTMX real-time dashboard, nftables per-device tracking, captive portal, Pi-hole integration
**Confidence:** HIGH

## Summary

Phase 2 builds the "whoa moment" dashboard that shows pilots exactly what is eating their Starlink 20 GB data cap. The technical domain spans five distinct subsystems: (1) a Go monitoring daemon that reads nftables per-MAC counters and Pi-hole DNS stats, (2) an HTMX+SSE real-time dashboard served by Caddy, (3) a captive portal that intercepts new device connections, (4) a SQLite persistence layer for usage history and settings, and (5) a domain-to-category mapping system for the "viral screenshot" pie chart.

The existing Phase 1 codebase establishes strong patterns to follow: Go daemons with platform-specific build tags (`_linux.go` / `_stub.go`), YAML config files, Ansible roles with Jinja2 templates, systemd service units, and cross-compilation via `GOOS=linux GOARCH=arm64 CGO_ENABLED=0`. Phase 2 introduces a critical decision point around CGo: mattn/go-sqlite3 requires CGo while modernc.org/sqlite does not. Given the existing `CGO_ENABLED=0` cross-compilation pattern, modernc.org/sqlite is the recommended choice to maintain build simplicity, despite a ~2x INSERT performance penalty that is irrelevant at this data volume (~1 write per 5 seconds).

The dashboard front-end uses HTMX 2.0.8 with the SSE extension 2.2.4 and Chart.js 4.5.1 UMD build. All three are single-file CDN includes with no build toolchain. The Go daemon exposes SSE endpoints that push real-time per-device bandwidth events, and REST endpoints for historical data. Caddy serves static HTML files and reverse-proxies `/api/*` to the Go daemon.

**Primary recommendation:** Build a single `cmd/dashboard-daemon/` Go binary following the bypass-daemon pattern, using modernc.org/sqlite for persistence (no CGo), standard library `net/http` for SSE (no r3labs/sse dependency needed), and nftables dynamic sets with per-MAC counters for device tracking. Serve the dashboard via Caddy with HTMX + Chart.js UMD as static files.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Single-page overview with all sections visible -- non-technical pilots shouldn't navigate between pages to understand their data usage
- **D-02:** Per-device breakdown as a table: device name/MAC, total bytes consumed, top domain, horizontal bar showing relative usage
- **D-03:** Real-time bandwidth graph as a streaming line chart (last 5 minutes) via HTMX SSE extension -- minimal custom JS
- **D-04:** Category pie chart for domain breakdown -- this is the "whoa" moment per PROJECT.md ("the viral screenshot moment")
- **D-05:** Information hierarchy top-to-bottom: (1) plan cap usage bar with dollar savings, (2) real-time bandwidth graph, (3) per-device table, (4) category pie chart
- **D-06:** First device connect triggers captive portal intercept -- HTTP request to known captive portal check URLs gets redirected to terms page
- **D-07:** Captive portal detection via HTTP intercept on standard OS check URLs (iOS captive.apple.com, Android connectivitycheck.gstatic.com, Windows msftconnecttest.com, macOS captive.apple.com)
- **D-08:** Flow: connect to WiFi -> captive portal auto-opens -> terms acceptance -> redirect to dashboard main page
- **D-09:** Terms page is minimal -- brief usage policy, accept button, no data collection beyond MAC address for per-device tracking
- **D-10:** After terms acceptance, device MAC is added to an allowed set -- subsequent connections skip terms until set is cleared
- **D-11:** Per-device byte tracking via nftables per-MAC counters read by Go daemon at ~5s intervals (matches DASH-01 requirement)
- **D-12:** Per-domain breakdown from Pi-hole FTL query log -- DNS queries already logged, no additional packet capture needed
- **D-13:** SQLite database in /data/skygate/ with WAL mode for concurrent read/write -- persists across reboots on data partition
- **D-14:** Go daemon exposes SSE endpoint for real-time dashboard updates and REST endpoints for historical data
- **D-15:** Phase 2 savings = DNS blocking savings only -- estimate bytes saved from blocked domains using average payload size heuristics (e.g., avg ad payload ~150KB, avg tracker ~5KB)
- **D-16:** Dollar conversion uses user-configurable overage rate with sensible default (Starlink overage pricing, approximately $0.01/MB as baseline)
- **D-17:** Savings display format: "$X.XX saved this session" prominently at top of dashboard, with breakdown available
- **D-18:** Settings page accessible from dashboard navigation -- pilot configures Starlink plan cap (GB), billing cycle start date, and overage rate
- **D-19:** Usage-against-cap displayed as a progress bar at top of dashboard with color escalation: green (<50%), yellow (50-75%), orange (75-90%), red (>90%)
- **D-20:** Alert banners appear at 50%, 75%, and 90% thresholds -- non-intrusive dashboard banners, not popups

### Claude's Discretion
- Exact nftables per-MAC counter implementation (named counters vs dynamic rules)
- Pi-hole FTL log parsing approach (SQLite gravity DB vs log file vs API)
- Domain-to-category mapping database design (how domains map to "Social Media", "Streaming", etc.)
- Caddy reverse proxy configuration for dashboard + API
- HTMX component structure and SSE event naming
- Device name resolution strategy (DHCP hostname, mDNS, or user-assigned names)
- SQLite schema design (tables, indexes, retention policy)
- Captive portal HTTP intercept implementation (nftables DNAT vs Caddy redirect)
- Dashboard responsive layout (mobile-first since passengers use phones)
- Chart library selection for pie chart and line graph (server-rendered SVG vs lightweight JS library)

### Deferred Ideas (OUT OF SCOPE)
- WiFi password change via web UI (mentioned in Phase 1 D-08) -- could be settings page addition
- Bypass list management via web UI (Phase 1 D-13) -- settings page feature
- Video CDN / update / sync blocking toggle UI -- requires v2 DNS categories (DNS-02, DNS-03, DNS-04)
- Per-device bandwidth throttling -- v2 requirement (MON-03)
- Flight-aware session tracking -- v2 requirement (MON-01)
- "Panic button" to disable all filtering -- v2 requirement (RES-02)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DASH-01 | Per-device data usage tracked in near-real-time (bytes per MAC address, ~5s intervals, persisted to SQLite) | nftables dynamic set with `typeof ether saddr` and counter flag; Go daemon polls via `nft -j list set` every 5s; modernc.org/sqlite for CGo-free persistence |
| DASH-02 | Web dashboard displays top domains by bytes consumed, per-device breakdown, and category pie chart | Pi-hole FTL API (`/api/stats/top_domains`) for domain stats; Chart.js 4.5.1 UMD for pie chart; HTMX for server-rendered device table |
| DASH-03 | Dashboard shows real-time bandwidth graph of current throughput | HTMX SSE extension 2.2.4 with `sse-connect` and `sse-swap`; Chart.js streaming line chart; Go daemon SSE endpoint via standard library |
| DASH-04 | Captive portal intercepts first HTTP request from new devices, shows terms acceptance, and links to dashboard | nftables DNAT redirect for unauthenticated MACs to Caddy; captive portal detection URL intercept; allowed_macs nftables set |
| DASH-05 | Dashboard displays bandwidth savings as dollar amount based on Starlink overage rate | Pi-hole blocked query count x average payload heuristics; configurable overage rate stored in SQLite settings table |
| DASH-06 | User can configure Starlink plan cap and billing cycle; dashboard shows usage against cap with alerts at 50%, 75%, 90% | SQLite settings table; REST PUT endpoint for config; CSS progress bar with color escalation; SSE-pushed alert banners |
</phase_requirements>

## Standard Stack

### Core (Phase 2 specific)

| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| Go (standard library `net/http`) | 1.22+ (project uses 1.26.1) | SSE server, REST API, HTTP server | Zero dependency SSE is trivial: set `Content-Type: text/event-stream`, `Cache-Control: no-cache`, flush after each write. No need for r3labs/sse library overhead. |
| modernc.org/sqlite | v1.47.0 | SQLite driver for Go (pure Go, no CGo) | Enables `CGO_ENABLED=0` cross-compilation matching bypass-daemon pattern. 2x slower INSERTs irrelevant at ~0.2 writes/sec. WAL mode supported. |
| HTMX | 2.0.8 | Dashboard interactivity, SSE integration | 14KB gzipped, single `<script>` tag, SSE extension for real-time updates. Decision locked in CLAUDE.md. |
| htmx-ext-sse | 2.2.4 | HTMX SSE extension | Separate extension since HTMX 2.0. Handles connection, reconnection with exponential backoff, named event routing. |
| Chart.js | 4.5.1 (UMD build) | Pie chart + line chart | ~70KB minified UMD build. Single `<script>` tag, no npm/build tools. Supports pie, doughnut, line chart types. CDN or self-hosted. |
| Caddy | 2.9+ | Static file server + reverse proxy | ARM64 binary available. Serves dashboard HTML + proxies `/api/*` to Go daemon. Already in project stack. |
| nftables | 1.0.6+ (Bookworm) | Per-MAC byte counting + captive portal DNAT | Dynamic sets with `typeof ether saddr` + counter flag for per-device tracking. JSON output via `nft -j` for programmatic parsing. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `gopkg.in/yaml.v3` | v3.0.1 | Config file parsing | Already in go.mod. Dashboard daemon config (port, intervals, Pi-hole address). |
| Pi-hole FTL API | v6.3+ REST API | DNS query statistics | `/api/stats/top_domains`, `/api/stats/top_clients`, `/api/stats/summary` endpoints. Requires session auth. |
| Pi-hole FTL DB | `/etc/pihole/pihole-FTL.db` | Direct database queries (fallback) | Read-only access to `queries` VIEW for per-domain stats if API proves insufficient. |
| dnsmasq DHCP leases | `/var/lib/misc/dnsmasq.leases` | Device hostname resolution | Parse lease file for MAC-to-hostname mapping. Format: `epoch MAC IP hostname client-id`. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| modernc.org/sqlite | mattn/go-sqlite3 | Requires CGo, breaks `CGO_ENABLED=0` cross-compile. 2x faster INSERTs but irrelevant at this volume. Use mattn only if modernc proves buggy on ARM64. |
| Standard library SSE | r3labs/sse v2.10.0 | r3labs adds stream management, client subscriptions, auto-cleanup. Overkill for single-stream dashboard. Use if multi-stream becomes needed. |
| Chart.js 4.5.1 | Server-rendered SVG | No JS dependency but much harder to animate and update in real-time. Use for static reports only. |
| Chart.js 4.5.1 | Chartist | Smaller but less maintained, fewer chart types. Chart.js community is 10x larger. |
| nftables dynamic set | /proc/net/arp + iptables counters | Legacy approach. nftables is default on Bookworm, dynamic sets are purpose-built for this. |

### Installation

**Go module additions:**
```bash
cd /Users/ryanstern/skygate
go get modernc.org/sqlite
go get gopkg.in/yaml.v3  # already present
```

**Static assets (Pi deployment):**
```bash
# Download to pi/static/ for Ansible deployment
curl -o pi/static/htmx.min.js https://cdn.jsdelivr.net/npm/htmx.org@2.0.8/dist/htmx.min.js
curl -o pi/static/htmx-ext-sse.js https://cdn.jsdelivr.net/npm/htmx-ext-sse@2.2.4/sse.js
curl -o pi/static/chart.umd.min.js https://cdn.jsdelivr.net/npm/chart.js@4.5.1/dist/chart.umd.min.js
```

**Caddy ARM64 binary:** Download from https://caddyserver.com/download?package=github.com/caddyserver/caddy/v2&os=linux&arch=arm64

## Architecture Patterns

### Recommended Project Structure
```
cmd/
  bypass-daemon/           # Existing Phase 1
  dashboard-daemon/        # NEW: Phase 2 main binary
    main.go                # Entrypoint, signal handling, config loading
    config.go              # YAML config struct + loader
    config_test.go         # Config tests
    db.go                  # SQLite schema, migrations, queries
    db_test.go             # DB tests with in-memory SQLite
    nftables.go            # nft JSON parsing, per-MAC counter reading
    nftables_linux.go      # Linux-specific nft command execution
    nftables_stub.go       # macOS dev stub (returns mock data)
    pihole.go              # Pi-hole FTL API client
    pihole_test.go         # Pi-hole API tests with httptest
    sse.go                 # SSE endpoint handler
    api.go                 # REST API handlers (GET stats, PUT settings)
    api_test.go            # API handler tests
    captive.go             # Captive portal redirect logic
    captive_linux.go       # nftables MAC set management (Linux)
    captive_stub.go        # Captive portal stub (macOS dev)
    categories.go          # Domain-to-category mapping
    savings.go             # Bandwidth savings calculation
    savings_test.go        # Savings calculation tests
pi/
  static/                  # NEW: Dashboard static files
    index.html             # Single-page dashboard
    settings.html          # Settings page
    captive.html           # Captive portal terms page
    htmx.min.js            # HTMX 2.0.8
    htmx-ext-sse.js        # SSE extension 2.2.4
    chart.umd.min.js       # Chart.js 4.5.1
    style.css              # Dashboard styles (mobile-first)
  ansible/
    roles/
      dashboard/           # NEW: Ansible role
        tasks/main.yml     # Deploy binary, static files, Caddy, systemd
        templates/
          Caddyfile.j2     # Caddy reverse proxy config
          skygate-dashboard.service.j2  # systemd service
          nftables-dashboard.conf.j2    # nftables additions for captive portal
        handlers/main.yml  # Service restart handlers
  config/
    dashboard.yaml         # NEW: Dashboard daemon config
  systemd/
    skygate-dashboard.service  # NEW: systemd service template
```

### Pattern 1: Go Daemon with Platform Build Tags
**What:** Follow bypass-daemon's pattern of `_linux.go` and `_stub.go` files for platform-specific code.
**When to use:** All nftables interactions, any Linux-only system calls.
**Example:**
```go
// nftables_linux.go
//go:build linux

package main

import (
    "encoding/json"
    "os/exec"
)

// ReadPerMACCounters executes nft -j and parses per-MAC byte counters.
func ReadPerMACCounters() (map[string]uint64, error) {
    cmd := exec.Command("nft", "-j", "list", "set", "inet", "skygate", "device_counters")
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    // Parse JSON output for set elements with counters
    var result nftJSON
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, err
    }
    counters := make(map[string]uint64)
    // Extract MAC -> bytes from nft JSON structure
    for _, elem := range result.Elements {
        counters[elem.MAC] = elem.Counter.Bytes
    }
    return counters, nil
}
```

```go
// nftables_stub.go
//go:build !linux

package main

import "log"

// ReadPerMACCounters returns mock data on non-Linux platforms.
func ReadPerMACCounters() (map[string]uint64, error) {
    log.Println("INFO: nftables stub -- returning mock per-MAC counters")
    return map[string]uint64{
        "aa:bb:cc:dd:ee:01": 1048576,  // 1 MB
        "aa:bb:cc:dd:ee:02": 524288,   // 512 KB
    }, nil
}
```

### Pattern 2: Standard Library SSE
**What:** Implement SSE using Go's `net/http` directly instead of r3labs/sse.
**When to use:** Single-stream SSE endpoints where library overhead is unnecessary.
**Example:**
```go
// sse.go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type BandwidthEvent struct {
    Timestamp int64             `json:"ts"`
    Devices   map[string]uint64 `json:"devices"` // MAC -> bytes/sec
    TotalBps  uint64            `json:"total_bps"`
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-r.Context().Done():
            return
        case <-ticker.C:
            event := s.collectBandwidthEvent()
            data, _ := json.Marshal(event)
            fmt.Fprintf(w, "event: bandwidth\ndata: %s\n\n", data)
            flusher.Flush()
        }
    }
}
```

### Pattern 3: HTMX SSE Dashboard Integration
**What:** HTMX SSE extension connects to Go daemon SSE endpoint and swaps HTML fragments.
**When to use:** Real-time dashboard updates without custom JavaScript.
**Example:**
```html
<!-- Dashboard page with SSE connection -->
<body hx-ext="sse">
  <div sse-connect="/api/events" sse-swap="bandwidth">
    <!-- This div is swapped with server-rendered HTML on each SSE event -->
    <div id="bandwidth-graph">Loading...</div>
  </div>

  <div sse-connect="/api/events" sse-swap="devices">
    <table id="device-table">Loading...</table>
  </div>

  <!-- For Chart.js updates, use hx-trigger on SSE events -->
  <div hx-trigger="sse:bandwidth" id="chart-container">
    <canvas id="bandwidth-chart"></canvas>
  </div>
</body>
```

### Pattern 4: nftables Captive Portal with MAC Allow Set
**What:** Use nftables to redirect HTTP traffic from unauthenticated devices to captive portal.
**When to use:** Captive portal implementation (DASH-04).
**Example nftables rules:**
```
table inet skygate {
    # Devices that have accepted terms (populated by Go daemon)
    set allowed_macs {
        type ether_addr
        flags timeout
        timeout 24h
    }

    # Per-device byte counters (populated dynamically from packet flow)
    set device_counters {
        typeof ether saddr
        flags dynamic
        counter
    }

    chain forward {
        # ... existing rules ...

        # Track per-device bytes (dynamic counter per MAC)
        ether saddr != @allowed_macs counter drop  # Block unauthenticated
        ether saddr @allowed_macs update @device_counters { ether saddr } accept
    }

    chain prerouting {
        # ... existing rules ...

        # Captive portal: redirect HTTP from unauthenticated devices
        iifname "wlan1" ether saddr != @allowed_macs tcp dport 80 \
            dnat to 192.168.4.1:80
    }
}
```

### Pattern 5: Caddy Reverse Proxy Configuration
**What:** Caddy serves static dashboard files and proxies API requests to Go daemon.
**When to use:** All HTTP serving on the Pi.
**Example Caddyfile:**
```
:80 {
    # API requests -> Go daemon
    handle /api/* {
        reverse_proxy localhost:8081
    }

    # Captive portal terms page
    handle /captive* {
        root * /opt/skygate/static
        file_server
    }

    # Dashboard static files (default)
    handle {
        root * /opt/skygate/static
        file_server
    }
}
```

### Anti-Patterns to Avoid
- **Building SSE with WebSockets:** SSE is unidirectional (server-to-client), which is exactly what a dashboard needs. WebSockets add unnecessary bidirectional complexity.
- **Polling from HTMX instead of SSE:** HTMX `hx-trigger="every 5s"` works but creates N requests vs 1 persistent connection. SSE is the correct pattern for real-time dashboards.
- **Using CGo for cross-compilation:** mattn/go-sqlite3 requires CGo which breaks the existing `CGO_ENABLED=0 GOOS=linux GOARCH=arm64` cross-compilation pattern. Use modernc.org/sqlite.
- **Shelling out to `nft` for every counter read:** Parse `nft -j list set` once per cycle, not per-device. One exec per 5s cycle, not one per device per 5s.
- **Bundling Chart.js via npm:** The project has zero Node.js toolchain on the Pi. Use the UMD build as a static file.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SSE event stream | Custom TCP socket handler | Go `net/http` with `Flusher` interface | SSE is literally `fmt.Fprintf(w, "event: name\ndata: json\n\n")` + `Flush()`. The protocol is trivial. |
| Real-time charts | Custom SVG/Canvas drawing | Chart.js 4.5.1 UMD build | Handles animation, responsive resize, tooltips, legend. 70KB for production-quality charts. |
| Dashboard interactivity | Custom JavaScript SPA framework | HTMX 2.0.8 + SSE extension | Declarative HTML attributes replace hundreds of lines of JS. `sse-swap="event"` does automatic DOM replacement. |
| Captive portal detection | Custom HTTP sniffer | nftables DNAT + OS captive portal detection URLs | All modern OSes probe known URLs on WiFi connect. Redirect those HTTP requests to the terms page. |
| SQLite WAL mode | Custom file locking | modernc.org/sqlite with `PRAGMA journal_mode=WAL` | WAL handles concurrent Go daemon writes + Caddy-proxied reads. Battle-tested concurrency model. |
| Domain categorization | Custom ML classifier | Static YAML mapping with curated domain lists | For v1, a ~500-domain YAML file mapping top domains to categories (Social, Streaming, Ads, OS Updates, etc.) is sufficient. ML is massive overkill. |

**Key insight:** The "whoa moment" is in the data presentation, not the data collection. Invest effort in the dashboard UX (layout, colors, savings framing), not custom infrastructure. Every piece of data collection uses existing OS/network primitives (nftables counters, Pi-hole DNS logs, DHCP leases).

## Common Pitfalls

### Pitfall 1: Chart.js with SSE -- Destroying/Recreating Charts
**What goes wrong:** Each SSE event creates a new Chart instance on the same canvas, causing memory leaks and visual flicker.
**Why it happens:** Chart.js stores state on the canvas element. Creating `new Chart(ctx, config)` without destroying the old one leaks memory.
**How to avoid:** Create Chart instances once on page load. On SSE events, call `chart.data.datasets[0].data = newData; chart.update()` instead of re-creating.
**Warning signs:** Dashboard gets slower over time, memory usage grows, charts flicker on update.

### Pitfall 2: nftables Counter Overflow vs Delta Calculation
**What goes wrong:** Raw nftables counters are monotonically increasing totals since last flush. Reading raw counters gives total-since-boot, not per-interval bandwidth.
**Why it happens:** nftables counters accumulate. You need to compute deltas between reads.
**How to avoid:** Store previous counter values in the Go daemon. Compute `current - previous` for each 5s interval. Handle counter reset (system reboot) by treating negative delta as a fresh start.
**Warning signs:** Bandwidth graph shows ever-increasing line instead of per-interval throughput.

### Pitfall 3: Pi-hole FTL Database Locking
**What goes wrong:** Direct SQLite reads of pihole-FTL.db can conflict with FTL's own writes, causing "database is locked" errors.
**Why it happens:** Pi-hole FTL holds its own database connection with its own locking strategy.
**How to avoid:** Prefer the Pi-hole REST API (`/api/stats/top_domains`) over direct database access. If direct DB access is needed, open in read-only mode with `?mode=ro` and set a busy timeout. Alternative: use the `queries` VIEW which is optimized.
**Warning signs:** Intermittent "database is locked" errors in Go daemon logs.

### Pitfall 4: Captive Portal -- iOS CNA (Captive Network Assistant) Quirks
**What goes wrong:** iOS opens captive portal pages in a special "CNA" mini-browser with limited capabilities. JavaScript may be restricted, cookies don't persist to Safari, and the CNA may dismiss unexpectedly.
**Why it happens:** iOS CNA is not Safari. It's a stripped-down WebKit view with security restrictions.
**How to avoid:** Keep the captive portal terms page extremely simple: plain HTML, minimal CSS, a single form POST for acceptance. After acceptance, redirect to a "success" page that iOS CNA recognizes as internet access (HTTP 200 with expected content). Then direct user to open Safari/Chrome and navigate to the dashboard URL (192.168.4.1).
**Warning signs:** iOS devices stuck in captive portal loop, terms page not rendering correctly, acceptance not persisting.

### Pitfall 5: DHCP Hostname Not Always Available
**What goes wrong:** Per-device table shows only MAC addresses because DHCP hostname is missing or generic.
**Why it happens:** Many devices (especially phones) don't send DHCP hostnames, or send generic names like "android-abc123" or "iPhone".
**How to avoid:** Multi-strategy device naming: (1) DHCP hostname from lease file, (2) mDNS/Bonjour name, (3) MAC OUI lookup for manufacturer, (4) user-assigned name stored in SQLite. Display priority: user-assigned > DHCP hostname > OUI manufacturer > truncated MAC.
**Warning signs:** Dashboard shows a wall of MAC addresses instead of friendly names.

### Pitfall 6: SSE Connection Limits
**What goes wrong:** Multiple browser tabs/devices each hold an SSE connection. On a Pi with limited resources, too many connections can exhaust file descriptors or memory.
**Why it happens:** SSE connections are persistent HTTP connections. Each dashboard viewer = 1 long-lived connection.
**How to avoid:** Set a reasonable connection limit (e.g., 16 concurrent SSE connections -- matches max 8 devices x 2 tabs). Use Go's `http.Server.MaxHeaderBytes` and connection timeouts. The SSE handler should clean up on client disconnect via `r.Context().Done()`.
**Warning signs:** New SSE connections refused, Go daemon logs connection errors.

### Pitfall 7: modernc.org/sqlite ARM64 Compilation
**What goes wrong:** modernc.org/sqlite is pure Go but uses code generation that may produce platform-specific code. First build on a new platform can be slow.
**Why it happens:** The library transpiles C SQLite to Go, and the generated code is large.
**How to avoid:** Test cross-compilation to `GOOS=linux GOARCH=arm64` early. The binary will be larger than bypass-daemon (~15-20MB vs ~5MB) due to SQLite code inclusion. This is acceptable for the Pi.
**Warning signs:** Build failures on cross-compilation, unexpectedly large binary size.

## Code Examples

### nftables Dynamic Set with Per-MAC Counters
```nft
# Source: nftables wiki (Sets + Counters documentation)
# Add to existing inet skygate table

set device_counters {
    typeof ether saddr
    flags dynamic
    counter
    timeout 24h
}

set allowed_macs {
    type ether_addr
    flags timeout
    timeout 24h
}

# In forward chain, after existing rules:
# Count bytes per authenticated device
iifname "wlan1" ether saddr @allowed_macs \
    update @device_counters { ether saddr } accept
```

### Reading nftables JSON Counters from Go
```go
// Source: nftables wiki (Output text modifiers) + libnftables-json(5)
// nft -j list set inet skygate device_counters

type nftResult struct {
    Nftables []json.RawMessage `json:"nftables"`
}

type nftSetElem struct {
    Elem struct {
        Val     string `json:"val"`     // MAC address
        Counter struct {
            Packets uint64 `json:"packets"`
            Bytes   uint64 `json:"bytes"`
        } `json:"counter"`
    } `json:"elem"`
}

func parseNftJSON(data []byte) (map[string]uint64, error) {
    var result nftResult
    if err := json.Unmarshal(data, &result); err != nil {
        return nil, err
    }
    counters := make(map[string]uint64)
    for _, raw := range result.Nftables {
        var elem nftSetElem
        if json.Unmarshal(raw, &elem) == nil && elem.Elem.Val != "" {
            counters[elem.Elem.Val] = elem.Elem.Counter.Bytes
        }
    }
    return counters, nil
}
```

### HTMX SSE Dashboard with Chart.js
```html
<!-- Source: htmx.org/extensions/sse/ + chartjs.org/docs -->
<!DOCTYPE html>
<html>
<head>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="/static/htmx.min.js"></script>
    <script src="/static/htmx-ext-sse.js"></script>
    <script src="/static/chart.umd.min.js"></script>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body hx-ext="sse">
    <!-- Plan cap usage bar (SSE-updated) -->
    <div sse-connect="/api/events" sse-swap="cap-status">
        <div id="cap-bar">Loading...</div>
    </div>

    <!-- Savings display (SSE-updated) -->
    <div sse-connect="/api/events" sse-swap="savings">
        <div id="savings">$0.00 saved</div>
    </div>

    <!-- Bandwidth chart (JS updates on SSE event) -->
    <div id="chart-section">
        <canvas id="bw-chart"></canvas>
    </div>

    <!-- Device table (SSE HTML swap) -->
    <div sse-connect="/api/events" sse-swap="devices">
        <table id="device-table">
            <tr><th>Device</th><th>Usage</th><th>Top Domain</th></tr>
        </table>
    </div>

    <!-- Category pie chart -->
    <div id="pie-section">
        <canvas id="category-pie"></canvas>
    </div>

    <script>
    // Initialize charts once, update via SSE events
    const bwChart = new Chart(document.getElementById('bw-chart'), {
        type: 'line',
        data: { labels: [], datasets: [{ label: 'Mbps', data: [] }] },
        options: { animation: false, scales: { x: { display: false } } }
    });

    // Listen for SSE bandwidth events to update chart data
    document.body.addEventListener('htmx:sseMessage', function(e) {
        if (e.detail.type === 'chart-data') {
            const data = JSON.parse(e.detail.data);
            bwChart.data.labels = data.labels;
            bwChart.data.datasets[0].data = data.values;
            bwChart.update('none'); // No animation for real-time
        }
    });
    </script>
</body>
</html>
```

### SQLite Schema
```sql
-- Source: modernc.org/sqlite documentation + SQLite WAL mode docs

-- Enable WAL mode for concurrent reads/writes
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;

-- Per-device usage snapshots (written every 5s by daemon)
CREATE TABLE IF NOT EXISTS device_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    mac_addr TEXT NOT NULL,
    bytes_total INTEGER NOT NULL,
    bytes_delta INTEGER NOT NULL,  -- delta since last snapshot
    UNIQUE(timestamp, mac_addr)
);
CREATE INDEX IF NOT EXISTS idx_device_usage_ts ON device_usage(timestamp);
CREATE INDEX IF NOT EXISTS idx_device_usage_mac ON device_usage(mac_addr);

-- Device metadata (friendly names, OUI info)
CREATE TABLE IF NOT EXISTS devices (
    mac_addr TEXT PRIMARY KEY,
    hostname TEXT,          -- from DHCP lease
    user_name TEXT,         -- user-assigned friendly name
    oui_vendor TEXT,        -- from MAC OUI lookup
    first_seen INTEGER NOT NULL,
    last_seen INTEGER NOT NULL
);

-- Domain statistics (aggregated from Pi-hole, periodic snapshots)
CREATE TABLE IF NOT EXISTS domain_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    domain TEXT NOT NULL,
    query_count INTEGER NOT NULL,
    blocked INTEGER NOT NULL DEFAULT 0,  -- 1 if blocked by Pi-hole
    category TEXT           -- mapped category (Social, Streaming, etc.)
);
CREATE INDEX IF NOT EXISTS idx_domain_stats_ts ON domain_stats(timestamp);

-- Settings (key-value store for user configuration)
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Default settings inserted on first run
INSERT OR IGNORE INTO settings VALUES ('plan_cap_gb', '20', strftime('%s', 'now'));
INSERT OR IGNORE INTO settings VALUES ('billing_cycle_start', '1', strftime('%s', 'now'));
INSERT OR IGNORE INTO settings VALUES ('overage_rate_per_mb', '0.01', strftime('%s', 'now'));

-- Savings log (DNS blocking savings estimates)
CREATE TABLE IF NOT EXISTS savings_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    blocked_queries INTEGER NOT NULL,
    estimated_bytes_saved INTEGER NOT NULL,
    dollar_amount REAL NOT NULL
);

-- Captive portal accepted devices
CREATE TABLE IF NOT EXISTS portal_accepted (
    mac_addr TEXT PRIMARY KEY,
    accepted_at INTEGER NOT NULL,
    ip_addr TEXT
);

-- Retention: delete records older than 30 days
-- Run periodically from daemon
-- DELETE FROM device_usage WHERE timestamp < strftime('%s', 'now', '-30 days');
-- DELETE FROM domain_stats WHERE timestamp < strftime('%s', 'now', '-30 days');
-- DELETE FROM savings_log WHERE timestamp < strftime('%s', 'now', '-30 days');
```

### Captive Portal nftables DNAT
```nft
# Source: Arch Linux wiki (captive portal), nftables wiki (DNAT)
# Add to prerouting chain in inet skygate table

chain prerouting {
    type nat hook prerouting priority dstnat;

    # Captive portal: redirect HTTP from unauthenticated devices to dashboard
    iifname "wlan1" ether saddr != @allowed_macs tcp dport 80 \
        dnat to 192.168.4.1:80

    # Also redirect HTTPS (for captive portal detection on newer OS versions)
    # Note: This will cause cert error, but triggers captive portal UI
    iifname "wlan1" ether saddr != @allowed_macs tcp dport 443 \
        dnat to 192.168.4.1:80

    # DNS must work for captive portal detection (already allowed in input chain)
}
```

### Pi-hole FTL API Access from Go
```go
// Source: Pi-hole API documentation (docs.pi-hole.net/api/)
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type PiHoleTopDomains struct {
    TopDomains []struct {
        Domain string `json:"domain"`
        Count  int    `json:"count"`
    } `json:"top_domains"`
}

func (s *Server) fetchTopDomains(limit int) (*PiHoleTopDomains, error) {
    url := fmt.Sprintf("http://localhost:%s/api/stats/top_domains?count=%d",
        s.cfg.PiHolePort, limit)

    req, _ := http.NewRequest("GET", url, nil)
    // Pi-hole v6 session auth
    req.Header.Set("X-FTL-SID", s.piholeSessionID)

    resp, err := s.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result PiHoleTopDomains
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}
```

### Domain-to-Category Mapping
```yaml
# pi/config/domain-categories.yaml
# Static mapping of top domains to display categories
# Source: Curated from StevenBlack hosts, Pi-hole common blocklists

categories:
  Social Media:
    - facebook.com
    - instagram.com
    - twitter.com
    - x.com
    - tiktok.com
    - snapchat.com
    - reddit.com
    - linkedin.com
    - pinterest.com
    - threads.net

  Streaming:
    - youtube.com
    - netflix.com
    - hulu.com
    - disneyplus.com
    - max.com
    - peacocktv.com
    - spotify.com
    - music.apple.com
    - twitch.tv
    - primevideo.com

  OS Updates:
    - swscan.apple.com
    - updates.cdn-apple.com
    - windowsupdate.com
    - play.googleapis.com
    - dl.google.com

  Cloud Sync:
    - icloud.com
    - photos.googleapis.com
    - dropbox.com
    - onedrive.live.com
    - drive.google.com

  Ads & Trackers:
    - doubleclick.net
    - googlesyndication.com
    - googleadservices.com
    - facebook.com/tr
    - analytics.google.com

  News:
    - cnn.com
    - foxnews.com
    - nytimes.com
    - bbc.com
    - reuters.com

  Aviation:
    - foreflight.com
    - garmin.com
    - aviationweather.gov
    - flightaware.com

  Other: []  # Catch-all for unmapped domains
```

## Discretion Recommendations

For areas marked as Claude's Discretion in CONTEXT.md, here are research-backed recommendations:

### nftables Per-MAC Counter Implementation
**Recommendation:** Dynamic set with `typeof ether saddr` + `counter` flag + `flags dynamic`.
**Rationale:** Named counters require pre-declaring each MAC. Dynamic sets auto-create entries from packet flow. The `counter` flag on a dynamic set tracks packets and bytes per element automatically. Read via `nft -j list set inet skygate device_counters`.

### Pi-hole FTL Log Parsing Approach
**Recommendation:** Use the Pi-hole v6 REST API (`/api/stats/top_domains`, `/api/stats/top_clients`) as primary, with direct FTL database reads as fallback.
**Rationale:** The REST API is stable, documented, and handles database locking internally. Direct DB access risks lock contention with FTL. The API provides aggregated stats efficiently. Session authentication via `/api/auth` with the Pi-hole password.

### Domain-to-Category Mapping
**Recommendation:** Static YAML file with ~200-500 domain-to-category mappings, loaded at daemon startup. Unknown domains classified as "Other".
**Rationale:** ML/API classification is overkill for v1. The top 200 domains cover 80%+ of typical web traffic. Curate from StevenBlack hosts extensions (social, streaming) and Pi-hole common blocklists. User can extend via YAML.

### Caddy Reverse Proxy Configuration
**Recommendation:** Single Caddyfile with `handle /api/*` for reverse proxy and `handle` for static files. Bind to `:80` only (no TLS needed on LAN).
**Rationale:** Caddy's `handle` directive provides clean path-based routing. No TLS complexity for a LAN-only dashboard.

### HTMX Component Structure and SSE Event Naming
**Recommendation:** Single SSE connection to `/api/events`. Multiple named events: `bandwidth` (5s throughput data), `devices` (device table HTML), `cap-status` (plan cap progress bar HTML), `savings` (dollar savings HTML). Chart data sent as JSON via `chart-data` event for JS processing.
**Rationale:** Single SSE connection minimizes overhead. Named events allow HTMX to swap specific DOM sections independently. Chart.js requires JS data updates, so those events carry JSON rather than HTML.

### Device Name Resolution Strategy
**Recommendation:** Multi-tier: (1) user-assigned name (stored in SQLite `devices.user_name`), (2) DHCP hostname from `/var/lib/misc/dnsmasq.leases`, (3) MAC OUI vendor lookup from embedded table, (4) truncated MAC as last resort.
**Rationale:** DHCP hostnames are unreliable (many devices don't send them). OUI vendor lookup (Apple, Samsung, etc.) provides useful context. User can override via settings page.

### SQLite Schema Design
**Recommendation:** Five tables as documented in Code Examples above. 30-day retention with periodic cleanup. WAL mode with `PRAGMA synchronous=NORMAL` for write performance.
**Rationale:** Separating `device_usage` (high-frequency time-series) from `devices` (low-frequency metadata) keeps the hot table small. Settings as key-value for flexibility. 30-day retention prevents SD card fill.

### Captive Portal HTTP Intercept
**Recommendation:** nftables DNAT in the prerouting chain, redirecting ports 80 and 443 from unauthenticated MACs to 192.168.4.1:80. Caddy serves the captive portal page. On terms acceptance, Go daemon adds MAC to `allowed_macs` nftables set via `nft add element`.
**Rationale:** nftables DNAT is the standard Linux captive portal approach. It intercepts at the network layer before Caddy sees the request, ensuring all HTTP traffic from new devices hits the terms page. Port 443 redirect triggers cert error but is needed for captive portal detection on newer OS versions.

### Dashboard Responsive Layout
**Recommendation:** Mobile-first CSS with a single breakpoint at 768px. Stack all sections vertically on mobile (phone). Side-by-side device table + pie chart on tablet/desktop. CSS Grid for layout.
**Rationale:** Passengers primarily use phones. The dashboard must look good on iPhone/Android screens. Pilot may use iPad in cockpit. Simple CSS Grid with `@media (min-width: 768px)` is sufficient.

### Chart Library Selection
**Recommendation:** Chart.js 4.5.1 UMD build (~70KB minified).
**Rationale:** Supports both pie chart (D-04) and streaming line chart (D-03). No build tools needed. CDN-downloadable single file. Active community (60k+ stars). Animation can be disabled for real-time updates.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| iptables per-MAC accounting | nftables dynamic sets with counters | nftables 0.9.5+ (2020) | Native per-element counters, JSON output, no iptables-legacy dependency |
| Pi-hole v5 PHP API | Pi-hole v6 REST API (FTL-hosted) | Pi-hole v6.0 (2025) | No PHP/lighttpd needed. FTL serves REST API directly. Session-based auth. |
| HTMX 1.x SSE built-in | HTMX 2.0 SSE as extension | HTMX 2.0 (2024) | Must load `htmx-ext-sse.js` separately. Extension version 2.2.4 adds reconnection improvements. |
| Chart.js 3.x | Chart.js 4.x | Chart.js 4.0 (2023) | Tree-shakeable ESM build + UMD for CDN use. Breaking: register components explicitly in ESM mode, but UMD auto-registers all. |
| mattn/go-sqlite3 (CGo) | modernc.org/sqlite (pure Go) | modernc.org/sqlite matured ~2023 | Enables `CGO_ENABLED=0` cross-compilation. Production-ready for moderate workloads. |

**Deprecated/outdated:**
- Pi-hole v5 PHP API (`/admin/api.php`): Replaced by v6 REST API. Do not use.
- HTMX 1.x `hx-sse` attribute: Removed in HTMX 2.0. Use `hx-ext="sse"` with `sse-connect`/`sse-swap`.
- iptables-legacy on Debian 12+: Replaced by nftables. iptables-nft compatibility layer exists but use native nftables.

## Open Questions

1. **Pi-hole v6 FTL API Authentication from Go Daemon**
   - What we know: v6 API requires session authentication. POST to `/api/auth` with password returns session ID.
   - What's unclear: How to securely store/retrieve the Pi-hole web password in the Go daemon config. Whether the daemon should use the FTL database directly instead.
   - Recommendation: Store Pi-hole API password in `/data/skygate/dashboard.yaml` (same security as Pi-hole itself on the same device). Authenticate once on daemon startup, refresh session as needed. Fallback to direct database read-only access if API auth proves problematic.

2. **nftables JSON Output Schema for Dynamic Set Elements**
   - What we know: `nft -j list set inet skygate device_counters` outputs JSON with set elements including counters.
   - What's unclear: Exact JSON structure for `ether_addr` typed elements with counters. libnftables-json(5) documents the schema but specific examples for MAC+counter are sparse.
   - Recommendation: Write a targeted integration test on Pi hardware early. If JSON parsing proves unreliable, fallback to text parsing of `nft list set` output.

3. **Savings Estimate Accuracy for DNS Blocking**
   - What we know: Can count blocked DNS queries from Pi-hole. Need to estimate bytes saved per blocked query.
   - What's unclear: How accurate are average payload heuristics? An ad blocked at DNS level saves the full page weight of the ad, but we don't know exactly how much that is.
   - Recommendation: Use conservative estimates: ads ~150KB avg, trackers ~5KB avg, CDN/updates ~1MB avg. Document these as configurable multipliers in the YAML config. Erring on the side of underestimation is better than overestimation for trust.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Dashboard daemon build | Yes | 1.26.1 | -- |
| nftables | Per-MAC counters, captive portal | Yes (on Pi, Bookworm) | 1.0.6+ | Verify on target Pi hardware |
| Pi-hole FTL | DNS stats API | Yes (on Pi) | v6.3+ | Direct FTL database read |
| Caddy | HTTP serving | Download required | 2.9+ ARM64 | Ansible deploys binary |
| SQLite | Data persistence | Via Go driver | modernc.org/sqlite v1.47.0 | Compiled into Go binary |
| HTMX | Dashboard UI | Download required | 2.0.8 | Bundle as static file |
| Chart.js | Charts | Download required | 4.5.1 UMD | Bundle as static file |

**Missing dependencies with no fallback:** None. All dependencies are either available or can be bundled/downloaded during deployment.

**Missing dependencies with fallback:** None applicable.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + BATS (Bash) |
| Config file | None needed -- Go `testing` is built-in, BATS already configured in Makefile |
| Quick run command | `go test ./cmd/dashboard-daemon/ -v -short` |
| Full suite command | `make test` (runs Go tests + BATS) |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DASH-01 | Per-MAC byte tracking at 5s intervals | unit | `go test ./cmd/dashboard-daemon/ -run TestReadPerMACCounters -v` | Wave 0 |
| DASH-01 | SQLite persistence of usage snapshots | unit | `go test ./cmd/dashboard-daemon/ -run TestDBWriteUsage -v` | Wave 0 |
| DASH-02 | Pi-hole API client returns top domains | unit | `go test ./cmd/dashboard-daemon/ -run TestFetchTopDomains -v` | Wave 0 |
| DASH-02 | Domain-to-category mapping | unit | `go test ./cmd/dashboard-daemon/ -run TestCategoryMapping -v` | Wave 0 |
| DASH-03 | SSE endpoint sends bandwidth events | unit | `go test ./cmd/dashboard-daemon/ -run TestSSEBandwidth -v` | Wave 0 |
| DASH-04 | Captive portal MAC set management | unit | `go test ./cmd/dashboard-daemon/ -run TestCaptivePortal -v` | Wave 0 |
| DASH-05 | Savings calculation from blocked queries | unit | `go test ./cmd/dashboard-daemon/ -run TestSavingsCalc -v` | Wave 0 |
| DASH-06 | Settings CRUD via REST API | unit | `go test ./cmd/dashboard-daemon/ -run TestSettingsAPI -v` | Wave 0 |
| DASH-06 | Cap alert threshold calculation | unit | `go test ./cmd/dashboard-daemon/ -run TestCapAlerts -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/dashboard-daemon/ -v -short`
- **Per wave merge:** `make test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/dashboard-daemon/db_test.go` -- covers DASH-01, DASH-06 (SQLite operations)
- [ ] `cmd/dashboard-daemon/nftables_test.go` -- covers DASH-01 (nft JSON parsing)
- [ ] `cmd/dashboard-daemon/pihole_test.go` -- covers DASH-02 (API client with httptest)
- [ ] `cmd/dashboard-daemon/sse_test.go` -- covers DASH-03 (SSE event format)
- [ ] `cmd/dashboard-daemon/captive_test.go` -- covers DASH-04 (MAC set logic)
- [ ] `cmd/dashboard-daemon/savings_test.go` -- covers DASH-05 (calculation logic)
- [ ] `cmd/dashboard-daemon/api_test.go` -- covers DASH-06 (REST endpoint tests)
- [ ] `cmd/dashboard-daemon/categories_test.go` -- covers DASH-02 (domain categorization)

## Project Constraints (from CLAUDE.md)

- **GSD Workflow:** All file changes must go through GSD commands (`/gsd:execute-phase`, `/gsd:quick`, etc.)
- **Go for Pi-side code:** No Python, no Node.js on the Pi. Single Go binary.
- **HTMX for dashboard:** No React/Vue SPA. HTMX + server-rendered HTML. Zero npm on Pi.
- **SQLite, not PostgreSQL:** WAL mode for concurrent access.
- **YAML for config files:** Established pattern from Phase 1 bypass-daemon.
- **Cross-compilation:** `GOOS=linux GOARCH=arm64` for Pi deployment. CGO_ENABLED=0 preferred.
- **Ansible for deployment:** Follow existing role structure (tasks/templates/handlers).
- **systemd for service management:** Follow skygate-bypass.service pattern.
- **Data persistence in /data/skygate:** Survives read-only root filesystem.
- **Binaries in /opt/skygate:** Deployment target for compiled Go binaries.
- **Platform build tags:** `_linux.go` / `_stub.go` for platform-specific code.
- **Exported Go function names:** For testability and package-level API.
- **No deprecated models:** Do not use Qwen3 or Gemini 2.0 variants anywhere.

## Sources

### Primary (HIGH confidence)
- [HTMX SSE Extension docs](https://htmx.org/extensions/sse/) -- Full SSE extension API, attributes, event handling, reconnection behavior
- [nftables wiki: Sets](https://wiki.nftables.org/wiki-nftables/index.php/Sets) -- Dynamic sets with `typeof` and `counter` flag
- [nftables wiki: Counters](https://wiki.nftables.org/wiki-nftables/index.php/Counters) -- Named and anonymous counters, per-element byte counting
- [nftables wiki: Meters](https://wiki.nftables.org/wiki-nftables/index.php/Meters) -- Dynamic element creation from packet flow
- [Pi-hole query database docs](https://docs.pi-hole.net/database/query-database/) -- FTL database schema (query_storage, domain_by_id, client_by_id, queries VIEW)
- [Pi-hole FTL API source (api.c)](https://github.com/pi-hole/FTL/blob/master/src/api/api.c) -- Complete endpoint registration: /api/stats/summary, /api/stats/top_domains, /api/stats/top_clients, /api/queries
- [Chart.js installation docs](https://www.chartjs.org/docs/latest/getting-started/installation.html) -- CDN/UMD build availability
- [modernc.org/sqlite Go package](https://pkg.go.dev/modernc.org/sqlite) -- Pure Go SQLite, v1.47.0
- [r3labs/sse Go package](https://pkg.go.dev/github.com/r3labs/sse/v2) -- SSE library v2.10.0

### Secondary (MEDIUM confidence)
- [Rayanfam: Captive portal detection](https://rayanfam.com/topics/captive-portal-detection-sample/) -- Cross-platform captive portal detection URLs (iOS, Android, Windows, macOS)
- [Fortinet: Captive portal auto-detection](https://community.fortinet.com/t5/FortiGate/Technical-Tip-Understanding-Captive-Portal-Auto-Detection/ta-p/400071) -- OS-specific probe URLs and expected responses
- [Jake Gold: Go + SQLite Best Practices](https://jacob.gold/posts/go-sqlite-best-practices/) -- WAL mode, PRAGMA settings, concurrent access patterns
- [ThreeDots: Go SSE + HTMX](https://threedots.tech/post/live-website-updates-go-sse-htmx/) -- Production Go SSE implementation patterns
- [StevenBlack/hosts](https://github.com/StevenBlack/hosts) -- Domain categorization source (social, fakenews, gambling, porn extensions)
- [Caddy common patterns](https://caddyserver.com/docs/caddyfile/patterns) -- Static files + reverse proxy Caddyfile patterns
- [SQLite CGo vs no-CGo benchmarks](https://github.com/multiprocessio/sqlite-cgo-no-cgo) -- mattn/go-sqlite3 vs modernc.org/sqlite performance comparison

### Tertiary (LOW confidence)
- [OpenWrt forum: nftables captive portal](https://forum.openwrt.org/t/nftables-help-needed-on-captive-portal-solution/204317) -- Community discussion, not official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- All libraries verified via npm/Go module registries. Versions confirmed current. Patterns well-documented.
- Architecture: HIGH -- Follows established Phase 1 patterns (Go daemon, Ansible, systemd). nftables dynamic sets documented in official wiki.
- Pitfalls: HIGH -- iOS CNA quirks, Chart.js memory leaks, SQLite locking are well-documented failure modes with known mitigations.
- Captive portal: MEDIUM -- nftables DNAT approach is standard but cross-platform captive portal detection has edge cases that need device testing.
- nftables JSON parsing: MEDIUM -- `nft -j` is documented but specific JSON schema for `ether_addr` typed sets with counters needs empirical verification on Pi hardware.
- Pi-hole v6 API: MEDIUM -- Endpoints confirmed from FTL source code, but authentication flow and exact response shapes need testing against running Pi-hole instance.

**Research date:** 2026-03-23
**Valid until:** 2026-04-22 (30 days -- stable domain, all tools mature)
