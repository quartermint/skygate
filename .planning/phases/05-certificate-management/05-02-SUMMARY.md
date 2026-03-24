---
phase: 05-certificate-management
plan: 02
subsystem: api
tags: [mode-selection, cert-download, mobileconfig, nftables, sqlite, captive-portal]

# Dependency graph
requires:
  - phase: 02-dashboard-captive
    provides: "Server struct, DB, captive portal, nftables sets, portal_accepted table"
  - phase: 05-certificate-management
    provides: "CA keypair at /data/skygate/ca/ (Plan 01)"
provides:
  - "Mode selection API (GET/POST /api/mode) with per-device persistence"
  - "Max Savings IP list API (GET /api/mode/ips) for proxy mode awareness"
  - "Certificate download handlers (/ca.mobileconfig, /ca.crt)"
  - "Mode selection HTML page with Quick Connect and Max Savings options"
  - "iOS cert install guide with Certificate Trust Settings warning"
  - "Android cert install guide with browser-only caveat"
  - "Post-flight cert removal instructions for all platforms"
affects: [05-certificate-management, proxy-server]

# Tech tracking
tech-stack:
  added: [text/template, crypto/sha256, encoding/pem]
  patterns: [platform build tags for nftables, device_modes table, JOIN query for IP mapping]

key-files:
  created:
    - cmd/dashboard-daemon/mode.go
    - cmd/dashboard-daemon/mode_linux.go
    - cmd/dashboard-daemon/mode_stub.go
    - cmd/dashboard-daemon/mode_test.go
    - cmd/dashboard-daemon/certdownload.go
    - cmd/dashboard-daemon/certdownload_test.go
    - pi/static/mode-select.html
    - pi/static/cert-install-ios.html
    - pi/static/cert-install-android.html
    - pi/static/cert-remove.html
  modified:
    - cmd/dashboard-daemon/db.go
    - cmd/dashboard-daemon/nftables.go
    - cmd/dashboard-daemon/config.go
    - cmd/dashboard-daemon/main.go

key-decisions:
  - "device_modes table uses INSERT OR REPLACE for upsert, consistent with portal_accepted pattern"
  - "GetMaxSavingsIPs uses JOIN with portal_accepted for MAC-to-IP resolution (Research Pattern 3)"
  - "Deterministic UUIDs for .mobileconfig derived from cert SHA-256 -- consistent across requests"
  - "nftables maxsavings_macs operations log warnings on failure but do not fail request (eventual consistency)"

patterns-established:
  - "Mode selection pattern: POST /api/mode sets mode + nftables, GET /api/mode reads from DB"
  - "Cert download pattern: read PEM, extract DER, serve with correct MIME type and Content-Disposition"
  - "HTML guide pattern: numbered steps with dark aviation theme, self-contained CSS, no JS dependencies for guides"

requirements-completed: [CERT-01, CERT-02]

# Metrics
duration: 9min
completed: 2026-03-24
---

# Phase 5 Plan 2: Mode Selection + Cert Download Summary

**Per-device Quick Connect / Max Savings mode selection with SQLite persistence, nftables integration, IP mapping API for proxy awareness, and cert download handlers (.mobileconfig, .crt) with platform-specific install guides**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-24T00:31:28Z
- **Completed:** 2026-03-24T00:40:32Z
- **Tasks:** 2
- **Files modified:** 14

## Accomplishments
- Mode selection API (POST/GET /api/mode) persists per-device mode in SQLite device_modes table with "quickconnect" as default
- GET /api/mode/ips endpoint returns source IPs of Max Savings devices by joining device_modes with portal_accepted -- enables remote proxy to distinguish MITM vs passthrough per-device
- Certificate download: .mobileconfig with application/x-apple-aspen-config MIME type, .crt with application/x-x509-ca-cert as raw DER bytes
- Four HTML pages: mode selection card UI, iOS 3-step cert guide (with Certificate Trust Settings), Android 2-step guide (with browser-only caveat), post-flight removal for iOS/Android/macOS/Windows
- nftables maxsavings_macs set operations follow existing allowed_macs pattern with platform build tags

## Task Commits

Each task was committed atomically:

1. **Task 1: Mode selection API with nftables integration, DB persistence, and mode/ips endpoint** - `f1e21c2` (feat)
2. **Task 2: Cert download handlers, HTML pages, and route registration** - `1ee3821` (feat)

_Both tasks used TDD: tests written first (RED), implementation (GREEN), verified._

## Files Created/Modified
- `cmd/dashboard-daemon/mode.go` - HandleSetMode, HandleGetMode, HandleGetMaxSavingsIPs HTTP handlers
- `cmd/dashboard-daemon/mode_linux.go` - AddMaxSavingsMAC/RemoveMaxSavingsMAC nftables commands
- `cmd/dashboard-daemon/mode_stub.go` - macOS dev stubs for nftables mode operations
- `cmd/dashboard-daemon/mode_test.go` - 11 tests for mode API, DB persistence, IP join
- `cmd/dashboard-daemon/certdownload.go` - HandleMobileConfig and HandleCertDownloadDER handlers
- `cmd/dashboard-daemon/certdownload_test.go` - 4 tests for cert download with real x509 cert fixture
- `cmd/dashboard-daemon/db.go` - device_modes table, SetDeviceMode, GetDeviceMode, GetMaxSavingsMACs, GetMaxSavingsIPs
- `cmd/dashboard-daemon/nftables.go` - nftMaxSavingsSet constant
- `cmd/dashboard-daemon/config.go` - CACertPath field with default
- `cmd/dashboard-daemon/main.go` - Route registration for /api/mode, /api/mode/ips, /ca.mobileconfig, /ca.crt
- `pi/static/mode-select.html` - Two-option mode selection with Quick Connect default
- `pi/static/cert-install-ios.html` - iOS cert install steps including Certificate Trust Settings
- `pi/static/cert-install-android.html` - Android cert install with browser-only caveat
- `pi/static/cert-remove.html` - Post-flight cert removal for all platforms

## Decisions Made
- device_modes table uses INSERT OR REPLACE for upsert, consistent with existing portal_accepted pattern
- GetMaxSavingsIPs uses JOIN with portal_accepted for MAC-to-IP resolution -- this is the data source for the remote proxy's per-device MITM decision (Research Pattern 3)
- Deterministic UUIDs for .mobileconfig derived from cert SHA-256 hash -- ensures consistent profile across repeated downloads
- nftables maxsavings_macs operations log warnings on error but do not fail the HTTP request -- eventual consistency per Pitfall 6
- Mode selection page uses platform detection (navigator.userAgent) to redirect to correct cert install guide after choosing Max Savings

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all code is production-ready. Stubs exist only for macOS dev (mode_stub.go) which is the established cross-platform development pattern.

## Next Phase Readiness
- Mode selection API and cert download handlers are operational
- /api/mode/ips endpoint is ready for remote proxy polling (Plan 03 Task 3)
- All 4 HTML pages ready for captive portal flow integration
- All existing dashboard-daemon tests continue passing (no regressions)

## Self-Check: PASSED

All 10 created files verified on disk. Both commit hashes (f1e21c2, 1ee3821) verified in git log.

---
*Phase: 05-certificate-management*
*Completed: 2026-03-24*
