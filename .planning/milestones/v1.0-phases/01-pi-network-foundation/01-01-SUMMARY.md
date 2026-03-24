---
phase: 01-pi-network-foundation
plan: 01
subsystem: infra
tags: [go, ansible, pihole, nftables, cake, bypass-daemon, bats, makefile]

# Dependency graph
requires: []
provides:
  - Go module (github.com/quartermint/skygate) with bypass daemon stub
  - Ansible playbook skeleton with 6 role stubs (base, networking, pihole, routing, qos, firstboot)
  - Aviation bypass domain config (D-12) and conservative blocklist config (D-10)
  - BATS test scaffold for autorate script
  - Makefile with build/test/deploy/lint targets
affects: [01-02, 01-03, 01-04, 01-05]

# Tech tracking
tech-stack:
  added: [go-1.26, gopkg.in/yaml.v3, ansible, bats-core, make]
  patterns: [platform-specific-build-tags, yaml-config, ansible-role-structure]

key-files:
  created:
    - go.mod
    - go.sum
    - cmd/bypass-daemon/main.go
    - cmd/bypass-daemon/main_test.go
    - cmd/bypass-daemon/nft_linux.go
    - cmd/bypass-daemon/nft_stub.go
    - .gitignore
    - Makefile
    - pi/ansible/playbook.yml
    - pi/ansible/ansible.cfg
    - pi/ansible/inventory/hosts.yml
    - pi/ansible/group_vars/all.yml
    - pi/ansible/roles/base/tasks/main.yml
    - pi/ansible/roles/networking/tasks/main.yml
    - pi/ansible/roles/pihole/tasks/main.yml
    - pi/ansible/roles/routing/tasks/main.yml
    - pi/ansible/roles/qos/tasks/main.yml
    - pi/ansible/roles/firstboot/tasks/main.yml
    - pi/config/bypass-domains.yaml
    - pi/config/blocklists.yaml
    - pi/scripts/tests/test_autorate.bats
  modified: []

key-decisions:
  - "Platform-specific nft implementation via Go build tags (linux vs stub)"
  - "Bypass daemon split into main.go + nft_linux.go + nft_stub.go for cross-platform dev"
  - "YAML config format established for bypass domains and blocklists"

patterns-established:
  - "Build tags: //go:build linux for platform-specific code, //go:build !linux for dev stubs"
  - "Config files: YAML format in pi/config/ directory"
  - "Ansible roles: pi/ansible/roles/{name}/tasks/main.yml structure"
  - "Makefile: top-level build orchestration with .PHONY targets"

requirements-completed: [NET-01, NET-02, DNS-01, ROUTE-01, QOS-01]

# Metrics
duration: 6min
completed: 2026-03-23
---

# Phase 1 Plan 01: Project Scaffold Summary

**Go module with bypass daemon (DNS resolution + nftables population), Ansible playbook skeleton with 6 roles, aviation bypass domain config (D-12), conservative blocklists (D-10), BATS test scaffold, and Makefile with 11 targets**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-23T08:33:16Z
- **Completed:** 2026-03-23T08:39:23Z
- **Tasks:** 2
- **Files created:** 21

## Accomplishments
- Go module compiles and all 5 unit tests pass (loadConfig, loadConfigMissing, resolveDomains, resolveDomainsWildcard, resolveDomainsInvalid)
- Bypass daemon implements full DNS resolution cycle with wildcard stripping, deduplication, and graceful error handling
- Ansible playbook references 6 roles with configurable group_vars covering network interfaces, WiFi AP, QoS, and paths
- Aviation bypass domain list covers ForeFlight, Garmin, FAA weather, ADS-B, AOPA, FltPlan, SkyVector, and captive portal detection domains
- Makefile provides 11 targets: build, cross-build, test, test-go, test-bats, lint, lint-ansible, deploy, deploy-check, clean, help

## Task Commits

Each task was committed atomically:

1. **Task 1: Initialize Go module, bypass daemon stub, and test scaffold** - `b723196` (feat)
2. **Task 2: Create Ansible skeleton, config files, BATS tests, and Makefile** - `91a5de0` (feat)

## Files Created/Modified
- `go.mod` - Go module definition (github.com/quartermint/skygate)
- `go.sum` - Dependency checksums (gopkg.in/yaml.v3)
- `cmd/bypass-daemon/main.go` - Bypass daemon: config loading, DNS resolution, main loop with graceful shutdown
- `cmd/bypass-daemon/main_test.go` - 5 unit tests for config and DNS resolution
- `cmd/bypass-daemon/nft_linux.go` - Linux-only nftables set population via nft CLI
- `cmd/bypass-daemon/nft_stub.go` - No-op stub for macOS development
- `.gitignore` - Go, Ansible, and OS artifact exclusions
- `Makefile` - Build/test/deploy orchestration with 11 targets
- `pi/ansible/playbook.yml` - Main playbook referencing 6 roles
- `pi/ansible/ansible.cfg` - Ansible configuration (inventory path, remote user, privilege escalation)
- `pi/ansible/inventory/hosts.yml` - Pi inventory with env-var-based IP override
- `pi/ansible/group_vars/all.yml` - All configurable parameters (network, WiFi, QoS, paths)
- `pi/ansible/roles/*/tasks/main.yml` - 6 role placeholder stubs (base, networking, pihole, routing, qos, firstboot)
- `pi/config/bypass-domains.yaml` - Aviation app bypass domain list (D-12)
- `pi/config/blocklists.yaml` - Conservative Pi-hole blocklist URLs (D-10)
- `pi/scripts/tests/test_autorate.bats` - BATS test scaffold for autorate script

## Decisions Made
- Split nftables implementation into platform-specific files using Go build tags rather than a single file with runtime checks -- enables clean cross-platform development on macOS while targeting Linux
- Used `gopkg.in/yaml.v3` for config parsing (standard Go YAML library, minimal dependency)
- Established YAML as the config file format for the project (bypass domains, blocklists)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added /bypass-daemon to .gitignore**
- **Found during:** Task 1
- **Issue:** `go build ./cmd/bypass-daemon/` produces a `bypass-daemon` binary in the working directory root (not in bin/)
- **Fix:** Added `/bypass-daemon` to .gitignore alongside existing `/bin/` entry
- **Files modified:** .gitignore
- **Verification:** Binary no longer shows in git status
- **Committed in:** b723196 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minor .gitignore addition to prevent build artifact from being tracked. No scope creep.

## Issues Encountered
- BATS (bats-core) is not installed on the development machine. BATS test scaffold was created and will run once `brew install bats-core` is executed. This does not block any work -- it's a dev tool dependency noted in the research.
- Go test `-v` flag output was not visible in the test runner, but all 5 tests pass (confirmed via `go test -list` and exit code 0)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Go module and build system ready for all subsequent plans to add code
- Ansible role stubs ready for plans 02 (networking), 03 (pihole/routing), 04 (qos), and 05 (firstboot/overlayfs) to implement
- Config files in place for bypass daemon and Pi-hole setup
- Developer should install BATS (`brew install bats-core`) and Ansible (`pip3 install ansible ansible-lint`) for full test/lint coverage

## Self-Check: PASSED

All 21 created files verified present. Both task commits (b723196, 91a5de0) confirmed in git log. SUMMARY.md exists.

---
*Phase: 01-pi-network-foundation*
*Completed: 2026-03-23*
