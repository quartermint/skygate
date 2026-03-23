# Phase 2: Usage Dashboard - Context

**Gathered:** 2026-03-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Per-device usage tracking, captive portal with terms acceptance, and a bandwidth savings dashboard — the "whoa" moment that shows pilots what's eating their 20 GB Starlink cap. No tunnel, no proxy, no certificate management — just awareness and plan tracking on the Pi.

</domain>

<decisions>
## Implementation Decisions

### Dashboard Layout & Information Hierarchy
- **D-01:** Single-page overview with all sections visible — non-technical pilots shouldn't navigate between pages to understand their data usage
- **D-02:** Per-device breakdown as a table: device name/MAC, total bytes consumed, top domain, horizontal bar showing relative usage
- **D-03:** Real-time bandwidth graph as a streaming line chart (last 5 minutes) via HTMX SSE extension — minimal custom JS
- **D-04:** Category pie chart for domain breakdown — this is the "whoa" moment per PROJECT.md ("the viral screenshot moment")
- **D-05:** Information hierarchy top-to-bottom: (1) plan cap usage bar with dollar savings, (2) real-time bandwidth graph, (3) per-device table, (4) category pie chart

### Captive Portal Flow
- **D-06:** First device connect triggers captive portal intercept — HTTP request to known captive portal check URLs gets redirected to terms page
- **D-07:** Captive portal detection via HTTP intercept on standard OS check URLs (iOS captive.apple.com, Android connectivitycheck.gstatic.com, Windows msftconnecttest.com, macOS captive.apple.com)
- **D-08:** Flow: connect to WiFi → captive portal auto-opens → terms acceptance → redirect to dashboard main page
- **D-09:** Terms page is minimal — brief usage policy, accept button, no data collection beyond MAC address for per-device tracking
- **D-10:** After terms acceptance, device MAC is added to an allowed set — subsequent connections skip terms until set is cleared

### Data Collection & Persistence
- **D-11:** Per-device byte tracking via nftables per-MAC counters read by Go daemon at ~5s intervals (matches DASH-01 requirement)
- **D-12:** Per-domain breakdown from Pi-hole FTL query log — DNS queries already logged, no additional packet capture needed
- **D-13:** SQLite database in /data/skygate/ with WAL mode for concurrent read/write — persists across reboots on data partition
- **D-14:** Go daemon exposes SSE endpoint for real-time dashboard updates and REST endpoints for historical data

### Savings Calculation Model
- **D-15:** Phase 2 savings = DNS blocking savings only — estimate bytes saved from blocked domains using average payload size heuristics (e.g., avg ad payload ~150KB, avg tracker ~5KB)
- **D-16:** Dollar conversion uses user-configurable overage rate with sensible default (Starlink overage pricing, approximately $0.01/MB as baseline)
- **D-17:** Savings display format: "$X.XX saved this session" prominently at top of dashboard, with breakdown available

### Plan Cap Configuration
- **D-18:** Settings page accessible from dashboard navigation — pilot configures Starlink plan cap (GB), billing cycle start date, and overage rate
- **D-19:** Usage-against-cap displayed as a progress bar at top of dashboard with color escalation: green (<50%), yellow (50-75%), orange (75-90%), red (>90%)
- **D-20:** Alert banners appear at 50%, 75%, and 90% thresholds — non-intrusive dashboard banners, not popups

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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Full project vision, constraints, key decisions (dashboard-first strategy)
- `.planning/REQUIREMENTS.md` — v1 requirements: DASH-01, DASH-02, DASH-03, DASH-04, DASH-05, DASH-06
- `CLAUDE.md` — Technology stack decisions, recommended libraries, alternatives considered

### Phase 1 Foundation (what this phase builds on)
- `.planning/phases/01-pi-network-foundation/01-CONTEXT.md` — Network foundation decisions, config patterns, integration points
- `pi/ansible/roles/networking/templates/nftables.conf.j2` — Existing nftables rules with per-interface counters (forward chain)
- `pi/ansible/group_vars/all.yml` — Network config: AP interface, subnet, data/opt directories

### Existing Codebase
- `cmd/bypass-daemon/` — Go daemon pattern: YAML config, platform build tags, systemd service
- `pi/config/bypass-domains.yaml` — YAML config format established in Phase 1
- `pi/systemd/` — systemd service templates for Go daemons
- `Makefile` — Build/test/deploy targets pattern

### Technology Stack
- `CLAUDE.md` §Technology Stack — HTMX 2.0, Caddy 2.9, Go daemon, SQLite, r3labs/sse

### Design Document
- `~/.gstack/projects/skygate/ryanstern-unknown-design-20260322-161803.md` — Full approved design doc

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Go module** (`github.com/quartermint/skygate`): YAML config loader pattern from bypass-daemon
- **Ansible roles structure**: new `dashboard` role will follow existing pattern (tasks/templates/handlers)
- **nftables template**: already has per-interface counters in forward chain — extend with per-MAC counters
- **systemd service templates**: follow skygate-bypass.service pattern for dashboard daemon
- **Makefile**: extend with dashboard build/deploy targets

### Established Patterns
- Go daemons with platform-specific build tags (linux vs stub for macOS dev)
- YAML for config files, Jinja2 for Ansible templates
- Cross-compile: GOOS=linux GOARCH=arm64 CGO_ENABLED=0
- Data persistence in /data/skygate (survives read-only root)
- Binaries in /opt/skygate

### Integration Points
- nftables forward chain counters → Go daemon reads for per-device stats
- Pi-hole FTL → Go daemon reads for per-domain DNS stats
- Go daemon → SSE endpoint → HTMX dashboard real-time updates
- Caddy → serves static HTMX pages + reverse proxy to Go daemon API
- Captive portal → nftables DNAT or Caddy redirect for new devices
- Dashboard served from gateway IP (192.168.4.1) on port 80

</code_context>

<specifics>
## Specific Ideas

- Dashboard is the "viral screenshot moment" — the pie chart showing where data goes is the hook that makes pilots demand controls
- Dollar savings display is critical for perceived value — "$12.50 saved this flight" is more impactful than "blocked 847 requests"
- Passengers use phones primarily — dashboard must be mobile-responsive
- HTMX + SSE means zero npm/build toolchain on the Pi — just static HTML files served by Caddy
- Follow bypass-daemon Go patterns for the new monitoring daemon (cmd/dashboard-daemon or cmd/monitor-daemon)

</specifics>

<deferred>
## Deferred Ideas

- WiFi password change via web UI (mentioned in Phase 1 D-08) — could be settings page addition
- Bypass list management via web UI (Phase 1 D-13) — settings page feature
- Video CDN / update / sync blocking toggle UI — requires v2 DNS categories (DNS-02, DNS-03, DNS-04)
- Per-device bandwidth throttling — v2 requirement (MON-03)
- Flight-aware session tracking — v2 requirement (MON-01)
- "Panic button" to disable all filtering — v2 requirement (RES-02)

</deferred>

---

*Phase: 02-usage-dashboard*
*Context gathered: 2026-03-23*
