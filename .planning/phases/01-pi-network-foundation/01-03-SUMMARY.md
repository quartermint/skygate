---
phase: 01-pi-network-foundation
plan: 03
subsystem: infra
tags: [go, bypass-daemon, nftables, ansible, systemd, dns-resolution]

# Dependency graph
requires:
  - phase: 01-01
    provides: Go module, bypass daemon stub, Ansible role stubs, bypass-domains.yaml config
provides:
  - Modular Go bypass daemon (config, resolver, nftset split into separate files)
  - Ansible routing role deploying bypass daemon binary + config + systemd service
  - systemd service with nftables dependency and security hardening
  - IPv4-only DNS resolution with wildcard domain stripping and deduplication
affects: [01-04, 01-05]

# Tech tracking
tech-stack:
  added: []
  patterns: [exported-function-api, platform-build-tags, ansible-role-handlers, systemd-security-hardening]

key-files:
  created:
    - cmd/bypass-daemon/config.go
    - cmd/bypass-daemon/resolver.go
    - cmd/bypass-daemon/nftset.go
    - cmd/bypass-daemon/nftset_linux.go
    - cmd/bypass-daemon/nftset_stub.go
    - cmd/bypass-daemon/config_test.go
    - cmd/bypass-daemon/resolver_test.go
    - pi/ansible/roles/routing/tasks/main.yml
    - pi/ansible/roles/routing/handlers/main.yml
    - pi/ansible/roles/routing/templates/skygate-bypass.service.j2
    - pi/systemd/skygate-bypass.service
  modified:
    - cmd/bypass-daemon/main.go
    - cmd/bypass-daemon/main_test.go

key-decisions:
  - "Exported function names (LoadConfig, ResolveDomains, UpdateNftSet, FormatNftCommand) for testability and package-level API"
  - "IPv4-only filtering via ip.To4() -- Starlink Mini and Pi networking is IPv4 in GA context"
  - "Nft command formatting as separate testable function (FormatNftCommand) decoupled from execution"

patterns-established:
  - "Go daemon modules: config.go (loading), resolver.go (DNS), nftset.go (constants + formatting), nftset_{linux,stub}.go (platform execution)"
  - "Ansible role structure: tasks + handlers + templates for daemon deployment"
  - "systemd service pattern: After nftables, Restart=always, ProtectSystem=strict"

requirements-completed: [ROUTE-01]

# Metrics
duration: 7min
completed: 2026-03-23
---

# Phase 1 Plan 03: Bypass Daemon & Routing Role Summary

**Modular Go bypass daemon with DNS resolution, nftables set population, and Ansible routing role deploying binary + systemd service with security hardening**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-23T08:43:36Z
- **Completed:** 2026-03-23T08:51:16Z
- **Tasks:** 2
- **Files created/modified:** 13

## Accomplishments
- Refactored monolithic bypass daemon into 5 Go source files (config, resolver, nftset, nftset_linux, nftset_stub) with exported API
- Added IPv4-only filtering (ip.To4() check) and FormatNftCommand for testable nft CLI argument generation
- 8 unit tests across 3 test files covering config loading (valid/missing/empty), DNS resolution (standard/wildcard/invalid/dedup), and nft command formatting
- Ansible routing role deploys cross-compiled binary, config file, and systemd service with handler-based restarts
- systemd service starts after nftables.service with security hardening (ProtectSystem=strict, ProtectHome=true, PrivateTmp=true)

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement bypass daemon with config loading, DNS resolution, and nftset population** - `dce82c7` (feat)
2. **Task 2: Create Ansible routing role with bypass daemon deployment and systemd service** - `34c6e4a` (feat)

## Files Created/Modified
- `cmd/bypass-daemon/config.go` - Config struct and LoadConfig YAML parser
- `cmd/bypass-daemon/resolver.go` - ResolveDomains with wildcard stripping, IPv4 filtering, dedup
- `cmd/bypass-daemon/nftset.go` - Constants (inet/skygate/bypass_v4) and FormatNftCommand
- `cmd/bypass-daemon/nftset_linux.go` - Linux UpdateNftSet and FlushNftSet via nft CLI
- `cmd/bypass-daemon/nftset_stub.go` - macOS no-op stubs for UpdateNftSet and FlushNftSet
- `cmd/bypass-daemon/main.go` - Refactored entry point using exported functions
- `cmd/bypass-daemon/config_test.go` - Tests: LoadConfig valid, missing, empty
- `cmd/bypass-daemon/resolver_test.go` - Tests: ResolveDomains standard, wildcard, invalid, dedup
- `cmd/bypass-daemon/main_test.go` - Test: FormatNftCommand generates correct nft CLI args
- `pi/ansible/roles/routing/tasks/main.yml` - Deploy binary, config, systemd service with handlers
- `pi/ansible/roles/routing/handlers/main.yml` - reload systemd and restart skygate-bypass handlers
- `pi/ansible/roles/routing/templates/skygate-bypass.service.j2` - Ansible template with variable substitution
- `pi/systemd/skygate-bypass.service` - Reference systemd unit for standalone use

## Decisions Made
- Exported function names (LoadConfig, ResolveDomains, FormatNftCommand) for testability -- breaking change from plan 01's lowercase functions, but necessary for proper Go API
- IPv4-only filtering via ip.To4() -- Starlink Mini and Pi networking in GA context is IPv4-only
- FormatNftCommand separated from UpdateNftSet for testability on non-Linux platforms
- Removed old nft_linux.go/nft_stub.go (lowercase updateNftSet) and replaced with nftset_linux.go/nftset_stub.go (uppercase UpdateNftSet + FlushNftSet)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Go test `-v` verbose output not displayed in this environment (Go 1.26 behavior), but all tests confirmed passing via `ok` status and exit code 0
- Old nft_linux.go/nft_stub.go from plan 01 had to be deleted (not just superseded) to avoid duplicate symbol errors with the new nftset_*.go files

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Bypass daemon is production-ready with modular structure for future enhancements (additional set types, IPv6 support)
- Routing Ansible role ready for deployment to Pi hardware
- Cross-compilation verified: `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build` succeeds
- systemd service integrates with nftables from plan 02's networking role

## Self-Check: PASSED

All 14 created/modified files verified present. Both task commits (dce82c7, 34c6e4a) confirmed in git log. SUMMARY.md exists.

---
*Phase: 01-pi-network-foundation*
*Completed: 2026-03-23*
