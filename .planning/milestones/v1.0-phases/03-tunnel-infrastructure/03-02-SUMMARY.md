---
phase: 03-tunnel-infrastructure
plan: 02
subsystem: infra
tags: [wireguard, tunnel-monitor, health-check, fallback, state-machine, go]

# Dependency graph
requires:
  - phase: 01-network-foundation
    provides: "bypass-daemon pattern (build tags, YAML config, signal handling, ticker loop)"
provides:
  - "Tunnel health monitor daemon with WireGuard handshake-based health checks"
  - "State machine with hysteresis for tunnel up/down detection (prevents flapping)"
  - "Routing fallback via ip rule add/del for fwmark 0x2 table 200"
  - "Platform stubs for macOS development of Linux-specific networking code"
affects: [03-tunnel-infrastructure, 04-content-proxy]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Monitor state machine with hysteresis counters", "ip rule manipulation for routing fallback"]

key-files:
  created:
    - cmd/tunnel-monitor/main.go
    - cmd/tunnel-monitor/config.go
    - cmd/tunnel-monitor/config_test.go
    - cmd/tunnel-monitor/health.go
    - cmd/tunnel-monitor/health_test.go
    - cmd/tunnel-monitor/fallback.go
    - cmd/tunnel-monitor/fallback_test.go
    - cmd/tunnel-monitor/health_linux.go
    - cmd/tunnel-monitor/health_stub.go
    - cmd/tunnel-monitor/fallback_linux.go
    - cmd/tunnel-monitor/fallback_stub.go
  modified: []

key-decisions:
  - "Standalone Go binary for tunnel monitor (separation of concerns from bypass daemon)"
  - "Hysteresis via consecutive-count thresholds (3 fail / 3 recover) prevents flapping on transient Starlink handoffs"
  - "Fallback removes ip rule (fwmark 0x2 table 200) so traffic falls through to main table -- atomic and reversible"

patterns-established:
  - "Monitor state machine: healthy/degraded with configurable fail_count/recover_count hysteresis"
  - "Platform stubs for ip rule and wg show: same build tag pattern as bypass daemon nftset"

requirements-completed: [TUN-01, ROUTE-02]

# Metrics
duration: 5min
completed: 2026-03-23
---

# Phase 3 Plan 2: Tunnel Monitor Daemon Summary

**Go tunnel-monitor daemon with WireGuard handshake health checks, hysteresis state machine, and ip rule routing fallback**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-23T19:33:58Z
- **Completed:** 2026-03-23T19:39:19Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Config, health check, and fallback modules with 15 passing tests (TDD)
- State machine with 3-check hysteresis in both directions prevents flapping on Starlink satellite handoffs
- Main daemon follows bypass-daemon pattern: signal handling, context cancellation, ticker loop
- Platform stubs enable full macOS development and testing without Linux-only tools (wg, ip)

## Task Commits

Each task was committed atomically:

1. **Task 1: Config, health check, and fallback logic with tests** - `d995319` (feat)
2. **Task 2: Main daemon, platform stubs, and integration** - `887b18e` (feat)

_Note: Task 1 was TDD -- tests written first, then implementation._

## Files Created/Modified
- `cmd/tunnel-monitor/config.go` - YAML config loading with interface, thresholds, routing params
- `cmd/tunnel-monitor/config_test.go` - Config loading tests (valid, defaults, missing file, invalid YAML)
- `cmd/tunnel-monitor/health.go` - CheckHandshake parser + Monitor state machine with hysteresis
- `cmd/tunnel-monitor/health_test.go` - Handshake parsing + state transition tests (healthy/unhealthy/no-handshake/flapping)
- `cmd/tunnel-monitor/fallback.go` - FormatAddRule/FormatDelRule for ip rule argument generation
- `cmd/tunnel-monitor/fallback_test.go` - Rule formatting tests
- `cmd/tunnel-monitor/main.go` - Daemon entry point with signal handling and ticker-based health checks
- `cmd/tunnel-monitor/health_linux.go` - Linux: exec `wg show` for real handshake data
- `cmd/tunnel-monitor/health_stub.go` - macOS: simulated healthy handshake
- `cmd/tunnel-monitor/fallback_linux.go` - Linux: exec `ip rule` for routing manipulation
- `cmd/tunnel-monitor/fallback_stub.go` - macOS: no-op logging stub

## Decisions Made
- Standalone Go binary at cmd/tunnel-monitor/ (separation of concerns from bypass daemon -- different failure domains, independent restart)
- Hysteresis with configurable thresholds (3 consecutive checks before transition) prevents flapping on transient Starlink satellite handoffs
- Fallback removes ip rule for fwmark 0x2 table 200 so traffic falls through to main table -- atomic, instant, reversible single-command recovery
- CheckHandshake treats timestamp=0 as unhealthy (never connected) without error

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Tunnel monitor daemon ready for Ansible deployment (systemd service template in next plan)
- Config structure ready for tunnel-monitor.yaml template
- Follows same patterns as bypass-daemon for Makefile cross-build targets
- .gitignore updated with /tunnel-monitor binary exclusion

---
*Phase: 03-tunnel-infrastructure*
*Completed: 2026-03-23*
