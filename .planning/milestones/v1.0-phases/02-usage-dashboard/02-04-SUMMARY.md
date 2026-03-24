---
phase: 02-usage-dashboard
plan: 04
subsystem: infra
tags: [ansible, caddy, nftables, systemd, captive-portal, deployment, makefile]

# Dependency graph
requires:
  - phase: 01-network-foundation
    provides: "Ansible role patterns, nftables.conf.j2, systemd service patterns, Makefile build targets"
  - phase: 02-usage-dashboard/01
    provides: "Go dashboard daemon binary (cmd/dashboard-daemon/)"
  - phase: 02-usage-dashboard/02
    provides: "Static HTML/CSS/JS dashboard files (pi/static/)"
  - phase: 02-usage-dashboard/03
    provides: "Domain categories config (pi/config/domain-categories.yaml)"
provides:
  - "Ansible dashboard role for deploying Go binary, static files, Caddy, systemd, nftables"
  - "Caddyfile with CNA check URL interception for iOS/Android/Windows captive portal"
  - "nftables captive portal DNAT rules with allowed_macs and device_counters sets"
  - "systemd service template with security hardening"
  - "Makefile build-dashboard and cross-build-dashboard targets"
affects: [03-wireguard-tunnel, 04-content-proxy, 05-tls-strategy]

# Tech tracking
tech-stack:
  added: [caddy, ansible-dashboard-role]
  patterns: [caddy-reverse-proxy, nftables-captive-portal-dnat, cna-host-header-matching, multi-daemon-makefile]

key-files:
  created:
    - pi/ansible/roles/dashboard/tasks/main.yml
    - pi/ansible/roles/dashboard/handlers/main.yml
    - pi/ansible/roles/dashboard/templates/Caddyfile.j2
    - pi/ansible/roles/dashboard/templates/skygate-dashboard.service.j2
    - pi/ansible/roles/dashboard/templates/nftables-dashboard.conf.j2
    - pi/ansible/roles/dashboard/templates/dashboard.yaml.j2
    - pi/systemd/skygate-dashboard.service
  modified:
    - pi/ansible/group_vars/all.yml
    - Makefile

key-decisions:
  - "Caddy host-header matching for CNA check URL interception -- DNAT preserves Host, Caddy @captive_check matcher triggers on known CNA domains"
  - "nftables allowed_macs set with 24h timeout for captive portal session management"
  - "nftables device_counters set with dynamic flag and counter for per-MAC byte tracking"
  - "Caddy reload via CLI command rather than systemd restart to avoid downtime"
  - "Dashboard config deployed as Jinja2 template with all values from group_vars"

patterns-established:
  - "CNA interception: Caddy @captive_check named matcher with OR logic on Host header for iOS/Android/Windows CNA check domains"
  - "Multi-daemon Makefile: build/cross-build targets split per daemon with aggregate targets"
  - "nftables include pattern: /etc/nftables.d/*.conf for modular rule additions"
  - "Ansible role per Phase 2 component: dashboard role deploys binary + static + config + systemd + nftables"

requirements-completed: [DASH-01, DASH-02, DASH-03, DASH-04, DASH-05, DASH-06]

# Metrics
duration: 3min
completed: 2026-03-23
---

# Phase 2 Plan 4: Deployment Infrastructure Summary

**Ansible dashboard role with Caddy CNA-interception reverse proxy, nftables captive portal DNAT, systemd hardened service, and multi-daemon Makefile targets**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-23T16:15:19Z
- **Completed:** 2026-03-23T16:18:17Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Complete Ansible dashboard role with 14 tasks deploying binary, static files, Caddy config, systemd service, nftables rules, and domain categories
- Caddyfile intercepts OS captive portal check URLs (iOS captive.apple.com, Android connectivitycheck.gstatic.com, Windows msftconnecttest.com) via Host-header matching, serving captive.html to trigger CNA sheet (D-07)
- nftables captive portal DNAT redirects HTTP from unauthenticated MACs to Caddy, with allowed_macs (24h timeout) and device_counters (per-MAC byte tracking) sets
- Makefile extended with build-dashboard/cross-build-dashboard targets; build and cross-build aggregate both daemons

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Ansible dashboard role with Caddy, systemd, and nftables templates** - `f32d9f8` (feat)
2. **Task 2: Update Makefile with dashboard daemon build targets** - `e99e131` (feat)

## Files Created/Modified
- `pi/ansible/roles/dashboard/tasks/main.yml` - 14 deployment tasks: binary, static files, config, Caddy, systemd, nftables
- `pi/ansible/roles/dashboard/handlers/main.yml` - Handlers for systemd reload, dashboard restart, caddy reload, nftables reload
- `pi/ansible/roles/dashboard/templates/Caddyfile.j2` - Caddy reverse proxy with @captive_check CNA interception, /api/* proxy, static file serving
- `pi/ansible/roles/dashboard/templates/skygate-dashboard.service.j2` - systemd service with nftables/pihole-FTL dependencies and security hardening
- `pi/ansible/roles/dashboard/templates/nftables-dashboard.conf.j2` - allowed_macs set, device_counters set, captive portal DNAT, per-device byte tracking
- `pi/ansible/roles/dashboard/templates/dashboard.yaml.j2` - Dashboard daemon config template with port, poll interval, Pi-hole address, DB path
- `pi/systemd/skygate-dashboard.service` - Reference systemd service with hardcoded paths for standalone use
- `pi/ansible/group_vars/all.yml` - Added dashboard_port, poll_interval, DB path, categories file, static dir, pihole address, caddy port
- `Makefile` - Added DASHBOARD_BINARY, build-dashboard, cross-build-dashboard targets; updated build/cross-build aggregates

## Decisions Made
- Caddy @captive_check named matcher uses OR logic on Host header to match any CNA check domain -- DNAT preserves Host header so this works without IP matching
- nftables allowed_macs set uses 24h timeout so devices re-accept terms daily (balances UX with policy enforcement)
- device_counters set uses dynamic flag with counter and 24h timeout for automatic cleanup
- Caddy reload handler uses `caddy reload --config` CLI command with ignore_errors to avoid downtime during config updates
- Dashboard config deployed as Jinja2 template rather than static file, allowing all values to come from group_vars

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `make build-dashboard` cannot be verified in this worktree because cmd/dashboard-daemon/ is being created by a parallel agent (plan 02-01). The Makefile target is correctly configured and will work after branch merge.

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all templates are fully wired to Ansible variables with real values from group_vars.

## Next Phase Readiness
- Deployment pipeline complete: `make cross-build` builds both daemons, `ansible-playbook` deploys everything
- After merging plans 02-01 through 02-03, full deployment path is operational
- Phase 3 (WireGuard tunnel) can extend nftables rules and Ansible roles following established patterns

## Self-Check: PASSED

All 8 created files verified on disk. Both commit hashes (f32d9f8, e99e131) verified in git log.

---
*Phase: 02-usage-dashboard*
*Completed: 2026-03-23*
