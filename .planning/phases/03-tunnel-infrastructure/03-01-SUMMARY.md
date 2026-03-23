---
phase: 03-tunnel-infrastructure
plan: 01
subsystem: infra
tags: [wireguard, docker-compose, nftables, ansible, policy-routing, tunnel]

# Dependency graph
requires:
  - phase: 01-network-foundation
    provides: "nftables.conf.j2 with bypass_v4 set and fwmark 0x1, group_vars/all.yml, routing role pattern"
provides:
  - "Server Docker Compose stack with linuxserver/wireguard endpoint"
  - "Pi WireGuard Ansible role with wg0.conf template (Table=off, MTU 1420)"
  - "Extended nftables with dual-fwmark architecture (0x1 bypass, 0x2 tunnel)"
  - "Policy routing table 200 for fwmark 0x2 via wg0"
  - "Tunnel monitor systemd service and config template"
  - "WireGuard and tunnel monitor group_vars"
affects: [03-tunnel-infrastructure, 04-content-proxy]

# Tech tracking
tech-stack:
  added: [linuxserver/wireguard, wireguard-tools]
  patterns: [dual-fwmark policy routing, conditional nftables via wg_enabled, Table=off custom routing]

key-files:
  created:
    - server/docker-compose.yml
    - server/.env.example
    - pi/ansible/roles/wireguard/tasks/main.yml
    - pi/ansible/roles/wireguard/templates/wg0.conf.j2
    - pi/ansible/roles/wireguard/templates/skygate-tunnel-monitor.service.j2
    - pi/ansible/roles/wireguard/templates/tunnel-monitor.yaml.j2
    - pi/ansible/roles/wireguard/handlers/main.yml
    - pi/config/tunnel-monitor.yaml
  modified:
    - pi/ansible/group_vars/all.yml
    - pi/ansible/roles/networking/templates/nftables.conf.j2

key-decisions:
  - "Table=off in wg0.conf delegates routing to nftables policy rules, preventing wg-quick conflicts"
  - "ct mark restore updated from 'ct mark 0x1' to 'ct mark != 0x0' to support dual-fwmark architecture"
  - "MSS clamping to 1380 on wg0 as defense-in-depth against MTU-related silent packet loss"
  - "All tunnel nftables rules gated by wg_enabled variable for clean Phase 1 fallback"
  - "Policy routing via Ansible shell task (not PostUp/PostDown) for reliability per Pitfall 5"

patterns-established:
  - "Conditional nftables blocks: wg_enabled | default(false) for feature-gated firewall rules"
  - "WireGuard Ansible role follows routing role pattern: template config, deploy binary, systemd service"
  - "Dual-fwmark architecture: 0x1 for bypass (table 100), 0x2 for tunnel (table 200)"

requirements-completed: [TUN-01, ROUTE-02]

# Metrics
duration: 3min
completed: 2026-03-23
---

# Phase 03 Plan 01: WireGuard Tunnel Plumbing Summary

**WireGuard tunnel infrastructure with server Docker Compose endpoint, Pi Ansible role (Table=off, MTU 1420), dual-fwmark nftables (0x1 bypass, 0x2 tunnel), and policy routing table 200**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-23T19:33:58Z
- **Completed:** 2026-03-23T19:37:48Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Server Docker Compose stack with linuxserver/wireguard for one-command WireGuard server deployment
- Pi WireGuard Ansible role with complete config template, tunnel monitor service, and policy routing
- Extended nftables template with dual-fwmark architecture: bypass (0x1) routes direct, non-bypass AP traffic (0x2) routes through WireGuard tunnel
- All tunnel features conditionally gated by wg_enabled (defaults to false for safe Phase 1 operation)

## Task Commits

Each task was committed atomically:

1. **Task 1: Server Docker Compose + WireGuard Ansible role + group_vars** - `c993a13` (feat)
2. **Task 2: Extend nftables template with tunnel marking and wg0 forwarding** - `fb4c6d8` (feat)

## Files Created/Modified
- `server/docker-compose.yml` - WireGuard server endpoint via linuxserver/wireguard Docker image
- `server/.env.example` - Server configuration template (SERVERURL, PEERS, INTERNAL_SUBNET)
- `pi/ansible/roles/wireguard/tasks/main.yml` - WireGuard Ansible deployment role (install, config, policy routing, services)
- `pi/ansible/roles/wireguard/templates/wg0.conf.j2` - WireGuard client config with Table=off, MTU 1420, PersistentKeepalive 25
- `pi/ansible/roles/wireguard/templates/skygate-tunnel-monitor.service.j2` - Tunnel monitor systemd service with CAP_NET_ADMIN
- `pi/ansible/roles/wireguard/templates/tunnel-monitor.yaml.j2` - Tunnel monitor Ansible config template
- `pi/ansible/roles/wireguard/handlers/main.yml` - Handlers for systemd reload, wg-quick restart, tunnel monitor restart
- `pi/config/tunnel-monitor.yaml` - Default tunnel monitor config for development/testing
- `pi/ansible/group_vars/all.yml` - Extended with WireGuard, tunnel monitor, and CAKE-on-wg0 variables
- `pi/ansible/roles/networking/templates/nftables.conf.j2` - Extended with tunnel marking, wg0 forwarding, MSS clamping

## Decisions Made
- **Table=off in wg0.conf:** Prevents wg-quick from installing its own routing rules that would conflict with SkyGate's custom policy routing. All routing handled by nftables fwmark + ip rule.
- **ct mark restore to != 0x0:** Updated from Phase 1's `ct mark 0x1` to `ct mark != 0x0` to restore marks for both bypass (0x1) and tunnel (0x2) connection tracking entries. Without this, subsequent packets in tunneled connections lose their fwmark per Pitfall 2.
- **MSS clamping to 1380:** Defense-in-depth on wg0 outbound TCP SYN to prevent MTU-related silent packet loss (1420 tunnel MTU minus TCP/IP headers).
- **Policy routing via Ansible shell task:** Uses idempotent `ip rule add` / `ip route add` in the wireguard role tasks rather than PostUp/PostDown in wg0.conf, per Pitfall 5 (PostUp can fail silently under systemd).
- **wg_enabled gate:** All tunnel nftables rules wrapped in `wg_enabled | default(false)` conditionals. When false, the Pi operates identically to Phase 1 (all traffic direct via Starlink).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required. WireGuard keys are PLACEHOLDERs; actual key exchange happens during deployment (D-10 workflow).

## Next Phase Readiness
- WireGuard tunnel plumbing complete; Phase 3 Plan 02 (tunnel monitor Go daemon) can build on this foundation
- Phase 4 content proxy can add its Docker service alongside the wireguard service in docker-compose.yml
- nftables dual-fwmark architecture ready for tunnel traffic routing once WireGuard keys are provisioned

## Self-Check: PASSED

All 10 created/modified files verified on disk. Both task commits (c993a13, fb4c6d8) verified in git log.

---
*Phase: 03-tunnel-infrastructure*
*Completed: 2026-03-23*
