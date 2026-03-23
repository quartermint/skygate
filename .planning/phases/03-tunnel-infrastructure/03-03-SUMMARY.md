---
phase: 03-tunnel-infrastructure
plan: 03
subsystem: infra
tags: [makefile, ansible, cake-qos, wireguard, nftables, bats]

# Dependency graph
requires:
  - phase: 03-tunnel-infrastructure plan 01
    provides: WireGuard Ansible role, nftables tunnel rules, group_vars wg_* variables
  - phase: 03-tunnel-infrastructure plan 02
    provides: tunnel-monitor Go daemon at cmd/tunnel-monitor/
  - phase: 01-pi-network-foundation
    provides: Makefile pattern, QoS role, autorate script, nftables template
provides:
  - Makefile build/cross-build targets for tunnel-monitor daemon
  - Ansible playbook wireguard role inclusion in correct dependency order
  - CAKE qdisc on wg0 with static bandwidth ceiling when wg_enabled is true
  - Autorate script WireGuard CAKE initialization at startup
  - 14 BATS tests validating nftables tunnel template rules
affects: [04-content-proxy, 05-tls-ca]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Conditional Ansible task with wg_enabled guard for tunnel-specific resources"
    - "Static CAKE on wg0 (no dynamic autorate) per research recommendation"
    - "BATS template validation: grep-based assertion against Jinja2 templates"

key-files:
  created:
    - pi/scripts/tests/test_nftables_tunnel.bats
  modified:
    - Makefile
    - pi/ansible/playbook.yml
    - pi/ansible/roles/qos/tasks/main.yml
    - pi/ansible/roles/qos/templates/autorate.sh.j2
    - pi/scripts/autorate.sh

key-decisions:
  - "Static CAKE bandwidth on wg0 (not dynamic autorate) -- single autorate instance for eth0 only, per research recommendation"
  - "Wireguard role placed after routing and before qos in playbook dependency order"

patterns-established:
  - "Conditional wg_enabled tasks: use when: wg_enabled | default(false) for all tunnel-related Ansible tasks"
  - "Multi-daemon Makefile: TUNNEL_BINARY follows BINARY_NAME/DASHBOARD_BINARY naming pattern"

requirements-completed: [TUN-01, ROUTE-02]

# Metrics
duration: 6min
completed: 2026-03-23
---

# Phase 3 Plan 3: Integration Wiring Summary

**Makefile tunnel-monitor targets, Ansible playbook wireguard role, CAKE QoS on wg0, autorate WG support, and 14 BATS nftables tunnel validation tests**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-23T19:48:22Z
- **Completed:** 2026-03-23T19:53:54Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Makefile builds all three daemons (bypass, dashboard, tunnel-monitor) for local and ARM64 cross-compilation
- Ansible playbook deploys wireguard role in correct dependency order (after routing, before qos)
- QoS role conditionally initializes CAKE on wg0 with static bandwidth when wg_enabled is true
- Autorate script initializes wg0 CAKE at startup without dynamic adjustment (static ceiling per research)
- 14 BATS tests validate nftables template contains all Phase 3 tunnel rules while preserving Phase 1 rules

## Task Commits

Each task was committed atomically:

1. **Task 1: Makefile build targets and playbook wiring** - `dd4eccb` (feat)
2. **Task 2: QoS CAKE on wg0 + autorate extension + BATS tests** - `23fa42a` (feat)

## Files Created/Modified
- `Makefile` - Added TUNNEL_BINARY, build-tunnel, cross-build-tunnel targets; updated aggregate targets
- `pi/ansible/playbook.yml` - Added wireguard role between routing and qos
- `pi/ansible/roles/qos/tasks/main.yml` - Added conditional CAKE qdisc on wg0 when wg_enabled
- `pi/ansible/roles/qos/templates/autorate.sh.j2` - Added WG_ENABLED/WG_INTERFACE/WG_CAKE_RATE_KBPS vars, apply_wg_cake() function, startup init
- `pi/scripts/autorate.sh` - Rendered copy updated with matching WG extensions
- `pi/scripts/tests/test_nftables_tunnel.bats` - 14 BATS tests validating nftables tunnel rules

## Decisions Made
- Static CAKE bandwidth on wg0 (not dynamic autorate) -- research recommends single autorate instance on eth0 only, with wg0 using a fixed ceiling to avoid measurement interference between interfaces
- Wireguard role placed after routing and before qos in playbook -- needs nftables from networking, policy routes from routing, and qos needs wg0 interface to exist for CAKE initialization

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All Phase 3 components (plans 01, 02, 03) are now wired together
- Makefile builds all three daemons; Ansible deploys the full tunnel stack
- CAKE QoS prevents bufferbloat on both eth0 and wg0
- Ready for Phase 4 (content proxy) which will add the proxy container to the remote server Docker Compose

## Self-Check: PASSED

All 6 files verified present. Both task commits (dd4eccb, 23fa42a) verified in git log.

---
*Phase: 03-tunnel-infrastructure*
*Completed: 2026-03-23*
