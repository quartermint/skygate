---
phase: 02-usage-dashboard
plan: 01
subsystem: database, daemon
tags: [go, sqlite, nftables, yaml, modernc-sqlite, categories]

requires:
  - phase: 01-pi-network-foundation
    provides: Go module, build tags pattern, nftables sets, YAML config pattern
provides:
  - Go dashboard daemon config loader with YAML and defaults
  - SQLite database layer with 6-table schema (WAL mode, modernc.org/sqlite)
  - nftables per-MAC counter JSON parser with delta computation
  - Domain-to-category mapping system with subdomain matching
  - Platform stubs for macOS development (nftables, captive portal)
  - 125-domain YAML mapping across 8 categories
affects: [02-02, 02-03, 02-04]

tech-stack:
  added: [modernc.org/sqlite v1.47.0]
  patterns: [pure Go SQLite (CGO_ENABLED=0), nanosecond timestamps for usage snapshots, subdomain-walking category matcher]

key-files:
  created:
    - cmd/dashboard-daemon/config.go
    - cmd/dashboard-daemon/config_test.go
    - cmd/dashboard-daemon/db.go
    - cmd/dashboard-daemon/db_test.go
    - cmd/dashboard-daemon/nftables.go
    - cmd/dashboard-daemon/nftables_linux.go
    - cmd/dashboard-daemon/nftables_stub.go
    - cmd/dashboard-daemon/nftables_test.go
    - cmd/dashboard-daemon/categories.go
    - cmd/dashboard-daemon/categories_test.go
    - cmd/dashboard-daemon/captive_linux.go
    - cmd/dashboard-daemon/captive_stub.go
    - cmd/dashboard-daemon/main.go
    - pi/config/dashboard.yaml
    - pi/config/domain-categories.yaml
  modified:
    - go.mod
    - go.sum
    - .gitignore

key-decisions:
  - "modernc.org/sqlite chosen over mattn/go-sqlite3 for CGO_ENABLED=0 cross-compilation compatibility"
  - "Nanosecond timestamps for device_usage to avoid UNIQUE constraint collisions at high write rates"
  - "Subdomain matching walks up domain labels (m.facebook.com matches facebook.com) without suffix-map complexity"

patterns-established:
  - "Dashboard daemon follows bypass-daemon patterns: YAML config, platform build tags, exported functions"
  - "SQLite WAL mode + PRAGMA synchronous=NORMAL + busy_timeout=5000 for concurrent access"
  - "INSERT OR IGNORE for default settings preserves user modifications"
  - "ON CONFLICT DO UPDATE for upsert patterns in settings and devices tables"

requirements-completed: [DASH-01, DASH-02]

duration: 9min
completed: 2026-03-23
---

# Phase 2 Plan 1: Go Daemon Foundation Summary

**Go dashboard daemon data layer with SQLite WAL persistence, nftables per-MAC counter parser, domain-to-category mapper, and YAML config -- 23 unit tests passing**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-23T15:47:05Z
- **Completed:** 2026-03-23T15:56:06Z
- **Tasks:** 2
- **Files modified:** 18

## Accomplishments
- Complete SQLite database layer with 6 tables (device_usage, devices, domain_stats, settings, savings_log, portal_accepted) in WAL mode using pure Go modernc.org/sqlite
- nftables per-MAC JSON counter parser with delta computation handling normal, new device, and counter reset scenarios
- Domain-to-category mapping with 125 domains across 8 categories (Social Media, Streaming, Ads and Trackers, OS Updates, Cloud Sync, News, Aviation, Other) with automatic subdomain matching
- YAML config loader with production defaults matching deployment paths (/data/skygate/, /opt/skygate/)
- Platform-specific build tags for Linux nftables operations and macOS dev stubs

## Task Commits

Each task was committed atomically:

1. **Task 1: Dashboard daemon config, SQLite DB, and domain categories**
   - `9039d6f` (test) - Failing tests for config, DB, and categories
   - `19229a9` (feat) - Implementation passing all 14 tests
2. **Task 2: nftables per-MAC counter parser with platform stubs**
   - `70213a0` (test) - Failing tests for nftables parser and stubs
   - `d19e8fe` (feat) - Implementation passing all 9 nftables/captive tests
3. **Housekeeping:** `2162b9d` (chore) - Add dashboard-daemon binary to .gitignore

## Files Created/Modified
- `cmd/dashboard-daemon/config.go` - Config struct with YAML loading and defaults (Port, PollIntervalSec, PiHoleAddress, etc.)
- `cmd/dashboard-daemon/db.go` - SQLite schema init, WAL mode, CRUD operations (WriteUsageSnapshot, GetDeviceUsage, GetSettings, PutSetting, WriteDevice, GetDevices)
- `cmd/dashboard-daemon/categories.go` - Domain-to-category mapper with subdomain walking
- `cmd/dashboard-daemon/nftables.go` - ParseNftCounters (JSON parser) and ComputeDeltas (counter delta calculation)
- `cmd/dashboard-daemon/nftables_linux.go` - Linux: ReadPerMACCounters, AddAllowedMAC, RemoveAllowedMAC, IsAllowedMAC
- `cmd/dashboard-daemon/nftables_stub.go` - macOS stubs returning mock data (3 devices)
- `cmd/dashboard-daemon/captive_linux.go` - Linux: AcceptDevice wraps AddAllowedMAC
- `cmd/dashboard-daemon/captive_stub.go` - macOS stub for AcceptDevice
- `cmd/dashboard-daemon/main.go` - Placeholder entrypoint for build verification
- `pi/config/dashboard.yaml` - Production config with defaults
- `pi/config/domain-categories.yaml` - 125 domains across 8 categories
- `go.mod` / `go.sum` - Added modernc.org/sqlite and transitive dependencies
- `.gitignore` - Added /dashboard-daemon binary

## Decisions Made
- **modernc.org/sqlite over mattn/go-sqlite3:** Maintains existing CGO_ENABLED=0 cross-compilation pattern. 2x slower INSERTs irrelevant at ~0.2 writes/sec.
- **Nanosecond timestamps for device_usage:** Avoids UNIQUE constraint collisions when writes happen faster than 1ms (found during testing).
- **Subdomain matching via domain-label walking:** Simpler than suffix-map approach. `m.facebook.com` -> try `facebook.com` -> try `com`. O(depth) where depth is typically 2-3.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed UNIQUE constraint collision in WriteUsageSnapshot**
- **Found during:** Task 1 (DB implementation)
- **Issue:** Second-precision timestamps caused UNIQUE(timestamp, mac_addr) collisions in rapid test writes. Would also affect production if daemon polling is faster than 1 second.
- **Fix:** Changed WriteUsageSnapshot to use nanosecond-precision timestamps (UnixNano) instead of second-precision (Unix)
- **Files modified:** cmd/dashboard-daemon/db.go
- **Verification:** TestGetDeviceUsage passes with 3 rapid sequential writes
- **Committed in:** 19229a9 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Necessary correctness fix. No scope creep.

## Issues Encountered
None beyond the auto-fixed timestamp issue.

## Known Stubs
None. All data paths are wired to real implementations (SQLite, nftables parser). The main.go is a placeholder entrypoint but this is intentional -- the full daemon loop is built in Plan 3 (API + business logic).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Config, DB, nftables parsing, and category mapping are the complete data layer foundation
- Plan 02-02 (dashboard frontend) can proceed: HTMX/Chart.js static assets, HTML pages
- Plan 02-03 (API + business logic) can proceed: all data types and DB functions exported
- Plan 02-04 (deployment) can proceed: YAML configs ready for Ansible templating

## Self-Check: PASSED

All 15 key files verified present. All 5 commit hashes verified in git log.

---
*Phase: 02-usage-dashboard*
*Completed: 2026-03-23*
