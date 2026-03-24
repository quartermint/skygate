---
phase: 01-pi-network-foundation
plan: 02
subsystem: infra
tags: [ansible, hostapd, pihole, nftables, dhcp, dns-blocking, wifi-ap, nat]

# Dependency graph
requires:
  - phase: 01-01
    provides: "Ansible playbook skeleton, group_vars/all.yml, role stubs, config files"
provides:
  - Base OS role with IP forwarding, IPv6 disabled, and data directories
  - Networking role with hostapd WiFi AP (2.4GHz WPA2-PSK, 8 clients)
  - NetworkManager exclusion for AP interface (wlan1)
  - nftables firewall with input/forward policy drop, bypass_v4 set, NAT masquerade
  - Policy routing for aviation bypass via fwmark 0x1
  - Pi-hole role with DHCP server on AP subnet (192.168.4.100-200)
  - DNS blocking with StevenBlack + OISD Light (conservative per D-10)
  - Aviation domain whitelist (ForeFlight, Garmin, FAA, ADS-B, AOPA, etc.)
affects: [01-03, 01-04, 01-05]

# Tech tracking
tech-stack:
  added: [hostapd, pihole-v6, nftables, dnsmasq]
  patterns: [ansible-role-with-handlers, jinja2-templates-from-group-vars, nftables-named-sets]

key-files:
  created:
    - pi/ansible/roles/base/tasks/main.yml
    - pi/ansible/roles/base/handlers/main.yml
    - pi/ansible/roles/networking/tasks/main.yml
    - pi/ansible/roles/networking/handlers/main.yml
    - pi/ansible/roles/networking/templates/hostapd.conf.j2
    - pi/ansible/roles/networking/templates/unmanaged.conf.j2
    - pi/ansible/roles/networking/templates/nftables.conf.j2
    - pi/ansible/roles/pihole/tasks/main.yml
    - pi/ansible/roles/pihole/handlers/main.yml
    - pi/ansible/roles/pihole/templates/pihole-setupVars.conf.j2
    - pi/ansible/roles/pihole/templates/pihole.toml.j2
    - pi/ansible/roles/pihole/templates/01-skygate-whitelist.conf.j2
    - pi/ansible/roles/pihole/files/install-pihole.sh
  modified: []

key-decisions:
  - "Pi-hole v6 TOML config with NULL blocking mode for silent NXDOMAIN responses (D-11)"
  - "Pi-hole web interface disabled (INSTALL_WEB_SERVER=false) -- SkyGate has its own dashboard"
  - "nftables bypass_v4 set with 1h timeout for aviation IP caching"
  - "DHCP bound to AP interface via custom dnsmasq config in /etc/dnsmasq.d/"

patterns-established:
  - "Ansible handlers: notify pattern for service restarts on config changes"
  - "Template variables: all values sourced from group_vars/all.yml, no hardcoded IPs"
  - "nftables structure: single inet table 'skygate' with named sets and chains"
  - "Pi-hole integration: setupVars.conf for install, pihole.toml for v6 runtime, dnsmasq.d for custom DNS config"

requirements-completed: [NET-01, NET-02, DNS-01]

# Metrics
duration: 4min
completed: 2026-03-23
---

# Phase 1 Plan 02: Network Stack Summary

**Ansible roles for hostapd WiFi AP (2.4GHz/WPA2/8 clients), Pi-hole v6 DHCP+DNS blocking with aviation domain whitelisting, and nftables firewall with bypass_v4 set for policy routing**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-23T08:43:17Z
- **Completed:** 2026-03-23T08:47:07Z
- **Tasks:** 2
- **Files created:** 13

## Accomplishments
- Base role installs core packages (iproute2, nftables, fping, conntrack), enables IPv4 forwarding, disables IPv6, creates data directories
- Networking role deploys hostapd for 2.4GHz WPA2-PSK AP with max 8 clients, excludes wlan1 from NetworkManager, deploys nftables firewall with NAT and aviation bypass set
- Pi-hole role installs unattended, configures DHCP on AP subnet (192.168.4.100-200), enables conservative DNS blocking (StevenBlack + OISD Light), whitelists 13 aviation/connectivity domains via regex
- nftables configuration implements complete Phase 1 firewall: input/forward policy drop, bypass_v4 named set for aviation IPs (fwmark 0x1), masquerade NAT from AP to Starlink

## Task Commits

Each task was committed atomically:

1. **Task 1: Create base and networking Ansible roles** - `46cafef` (feat)
2. **Task 2: Create Pi-hole Ansible role with DHCP and DNS blocking** - `564f717` (feat)

## Files Created/Modified
- `pi/ansible/roles/base/tasks/main.yml` - Base OS config: packages, IP forwarding, IPv6 disable, data dirs
- `pi/ansible/roles/base/handlers/main.yml` - Handler: sysctl reload
- `pi/ansible/roles/networking/tasks/main.yml` - Networking: hostapd install, NM exclusion, static IP, nftables, policy routing
- `pi/ansible/roles/networking/handlers/main.yml` - Handlers: restart NetworkManager, hostapd, nftables
- `pi/ansible/roles/networking/templates/hostapd.conf.j2` - WiFi AP config: 2.4GHz, WPA2-PSK, max 8 clients
- `pi/ansible/roles/networking/templates/unmanaged.conf.j2` - NetworkManager exclusion for wlan1
- `pi/ansible/roles/networking/templates/nftables.conf.j2` - Firewall: bypass_v4 set, NAT, input/forward policy drop
- `pi/ansible/roles/pihole/tasks/main.yml` - Pi-hole install, DHCP config, whitelist, blocklists, gravity
- `pi/ansible/roles/pihole/handlers/main.yml` - Handler: restart pihole-FTL
- `pi/ansible/roles/pihole/templates/pihole-setupVars.conf.j2` - Pi-hole unattended install pre-seed
- `pi/ansible/roles/pihole/templates/pihole.toml.j2` - Pi-hole v6 config: DHCP, NULL blocking, DNS upstream
- `pi/ansible/roles/pihole/templates/01-skygate-whitelist.conf.j2` - Custom dnsmasq: bind to AP interface, DNS option for DHCP
- `pi/ansible/roles/pihole/files/install-pihole.sh` - Idempotent Pi-hole install script

## Decisions Made
- Pi-hole v6 uses TOML config (`pihole.toml`) with NULL blocking mode for silent NXDOMAIN responses per D-11
- Pi-hole web server/interface disabled (`INSTALL_WEB_SERVER=false`) since SkyGate provides its own dashboard via Caddy
- nftables bypass_v4 set uses 1-hour timeout for aviation IP entries (populated by Go bypass daemon)
- DHCP bound to AP interface via custom dnsmasq config in `/etc/dnsmasq.d/01-skygate.conf`

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Base, networking, and pihole roles fully implemented with templates, handlers, and idempotent tasks
- nftables bypass_v4 set ready for plan 03 (routing/bypass daemon) to populate with resolved aviation IPs
- Pi-hole DHCP/DNS stack ready for testing once deployed to a physical Pi with USB WiFi adapter
- All configuration driven by group_vars/all.yml -- no hardcoded values in any templates

## Self-Check: PASSED

All 13 created files verified present. Both task commits (46cafef, 564f717) confirmed in git log. SUMMARY.md exists.

---
*Phase: 01-pi-network-foundation*
*Completed: 2026-03-23*
