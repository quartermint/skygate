---
phase: 02-usage-dashboard
plan: 03
subsystem: api, daemon
tags: [go, sse, rest-api, pihole, savings, captive-portal, htmx]

requires:
  - phase: 02-usage-dashboard
    provides: Go dashboard daemon data layer (DB, nftables parser, categories, config)
provides:
  - Pi-hole FTL v6 API client with session authentication
  - Bandwidth savings calculator with configurable payload heuristics
  - SSE event streaming handler (6 named events at 5s intervals)
  - REST API handlers for device stats, domain stats, savings, settings CRUD
  - Captive portal acceptance handler with nftables MAC set integration
  - Dashboard daemon main.go entrypoint wiring all components
affects: [02-04]

tech-stack:
  added: []
  patterns: [Server struct for shared handler state, SSE via stdlib net/http with Flusher, HTML fragment rendering for HTMX sse-swap, ring buffer chart history]

key-files:
  created:
    - cmd/dashboard-daemon/pihole.go
    - cmd/dashboard-daemon/pihole_test.go
    - cmd/dashboard-daemon/savings.go
    - cmd/dashboard-daemon/savings_test.go
    - cmd/dashboard-daemon/sse.go
    - cmd/dashboard-daemon/sse_test.go
    - cmd/dashboard-daemon/api.go
    - cmd/dashboard-daemon/api_test.go
    - cmd/dashboard-daemon/captive.go
    - cmd/dashboard-daemon/captive_test.go
  modified:
    - cmd/dashboard-daemon/main.go
    - cmd/dashboard-daemon/captive_linux.go
    - cmd/dashboard-daemon/captive_stub.go

key-decisions:
  - "Server struct holds shared state (config, db, categories, pihole, prevCounters, chartHistory) across all handlers"
  - "SSE uses stdlib net/http Flusher interface -- no r3labs/sse dependency needed"
  - "Savings calculator uses conservative estimates: 150KB/ad, 5KB/tracker (per D-15, RESEARCH Open Question 3)"
  - "Captive portal returns simple HTML redirect page for iOS CNA compatibility (no JS)"
  - "ARP table lookup via /proc/net/arp on Linux with stub fallback on macOS"

patterns-established:
  - "Server method receivers for all HTTP handlers enable shared state without globals"
  - "Ring buffer (maxChartHistory=60) for 5-min bandwidth chart at 5s intervals"
  - "HTML fragment rendering functions (renderCapStatusHTML, renderSavingsHTML, renderAlertHTML) for HTMX SSE swap"
  - "Pi-hole v6 session auth via X-FTL-SID header on all API requests"
  - "Graceful shutdown with 10s timeout context for HTTP server drain"

requirements-completed: [DASH-01, DASH-02, DASH-03, DASH-04, DASH-05, DASH-06]

duration: 8min
completed: 2026-03-23
---

# Phase 2 Plan 3: API Layer and Daemon Engine Summary

**Pi-hole API client, savings calculator, SSE streaming (6 events), REST API endpoints, captive portal accept, and daemon main.go -- 47 unit tests passing, cross-compiles for linux/arm64**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-23T16:02:39Z
- **Completed:** 2026-03-23T16:10:45Z
- **Tasks:** 3
- **Files modified:** 13

## Accomplishments
- Pi-hole FTL v6 API client with session-based authentication (POST /api/auth), top domain fetching, and blocked query count retrieval
- Bandwidth savings calculator converting blocked DNS queries to dollar amounts using configurable payload heuristics ($X.XX format per D-17)
- SSE endpoint streaming 6 named events every 5 seconds: bandwidth, chart-data, devices, cap-status, savings, categories -- plus threshold alerts at 50%/75%/90%
- REST API serving device stats, domain stats with category mapping, savings summary, and settings CRUD with key validation
- Captive portal acceptance handler: MAC from form/ARP lookup, nftables allowed_macs set addition, SQLite portal_accepted recording, iOS CNA-friendly HTML response
- Dashboard daemon main.go: config loading, DB init, Pi-hole auth, HTTP route registration, background polling loop, graceful signal-based shutdown

## Task Commits

Each task was committed atomically:

1. **Task 1: Pi-hole API client and savings calculator**
   - `51d3f4c` (test) - Failing tests for Pi-hole client and savings calculator
   - `e6ea199` (feat) - Implementation passing all 9 tests
2. **Task 2: SSE handler, REST API endpoints, and captive portal accept**
   - `3629868` (test) - Failing tests for SSE, REST API, and captive portal
   - `8913a0a` (feat) - Implementation passing all 14 handler tests
3. **Task 3: Dashboard daemon main.go entrypoint**
   - `d54eb50` (feat) - Full entrypoint with all routes, polling, and graceful shutdown

## Files Created/Modified
- `cmd/dashboard-daemon/pihole.go` - Pi-hole FTL v6 API client with session auth, top domains, blocked count
- `cmd/dashboard-daemon/pihole_test.go` - httptest-based tests for Pi-hole client (auth, domains, errors)
- `cmd/dashboard-daemon/savings.go` - Bandwidth savings calculator with configurable payload heuristics
- `cmd/dashboard-daemon/savings_test.go` - Table-driven tests for savings (ads-only, mixed, zero, custom rate)
- `cmd/dashboard-daemon/sse.go` - SSE event streaming handler, polling loop, HTML rendering helpers
- `cmd/dashboard-daemon/sse_test.go` - SSE content-type, event format, disconnect cleanup, cap status HTML tests
- `cmd/dashboard-daemon/api.go` - Server struct, REST API handlers, device name resolution, byte formatting
- `cmd/dashboard-daemon/api_test.go` - REST endpoint tests with in-memory DB (devices, domains, savings, settings)
- `cmd/dashboard-daemon/captive.go` - Captive portal acceptance, ARP lookup, portal_accepted DB recording
- `cmd/dashboard-daemon/captive_test.go` - POST acceptance and GET rejection tests
- `cmd/dashboard-daemon/main.go` - Daemon entrypoint with config, DB, routes, polling, signal handling
- `cmd/dashboard-daemon/captive_linux.go` - Added ARP table lookup via /proc/net/arp
- `cmd/dashboard-daemon/captive_stub.go` - Added lookupARPTable stub for macOS dev

## Decisions Made
- **Server struct over globals:** All HTTP handlers are methods on Server, enabling clean dependency injection and test isolation via newTestServer() helper
- **stdlib SSE over r3labs/sse:** SSE protocol is trivial (event/data/flush). No need for library overhead on a single-stream dashboard.
- **Conservative savings estimates:** 150KB/ad, 5KB/tracker per RESEARCH.md guidance. Better to underestimate for pilot trust.
- **iOS CNA-friendly captive portal response:** Simple HTML with no JS, dark aviation theme, direct link to dashboard URL
- **ARP table lookup for MAC resolution:** Reads /proc/net/arp on Linux as fallback when MAC not provided in form data

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added ARP table lookup for captive portal**
- **Found during:** Task 2 (captive portal implementation)
- **Issue:** Plan referenced lookupARPTable for MAC resolution from IP, but platform-specific implementations were needed
- **Fix:** Added lookupARPTable to captive_linux.go (reads /proc/net/arp) and captive_stub.go (returns empty string)
- **Files modified:** cmd/dashboard-daemon/captive_linux.go, cmd/dashboard-daemon/captive_stub.go
- **Verification:** Tests pass, captive portal falls back gracefully when MAC not found
- **Committed in:** 8913a0a (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Necessary for complete captive portal MAC resolution. No scope creep.

## Issues Encountered
None -- all tasks executed cleanly with TDD flow.

## Known Stubs
None. All data paths are wired to real implementations (Pi-hole API, nftables counters, SQLite). Platform stubs exist only for macOS development (nftables, ARP) which is the established Phase 1 pattern.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Complete API layer ready for frontend integration (Plan 02-02 dashboard HTML connects via SSE to /api/events)
- Plan 02-04 (Ansible deployment) can proceed: all Go binary components complete, systemd service pattern established
- Full binary cross-compiles for linux/arm64 with CGO_ENABLED=0

## Self-Check: PASSED

All 13 key files verified present. All 5 commit hashes verified in git log.

---
*Phase: 02-usage-dashboard*
*Completed: 2026-03-23*
