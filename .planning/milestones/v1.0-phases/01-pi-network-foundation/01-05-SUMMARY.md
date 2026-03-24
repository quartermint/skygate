---
phase: 01-pi-network-foundation
plan: 05
subsystem: infra
tags: [overlayfs, systemd, ansible, firstboot, sd-card, raspberry-pi]

# Dependency graph
requires:
  - phase: 01-02
    provides: "Base Ansible role with package install, sysctl, directory creation"
  - phase: 01-03
    provides: "Pi-hole role that writes to /etc/pihole (needs /data/pihole symlink)"
  - phase: 01-04
    provides: "QoS autorate script at /opt/skygate/autorate.sh"
provides:
  - "OverlayFS read-only root with /data writable partition (ext4, data=journal)"
  - "/data/pihole symlink for Pi-hole persistence across OverlayFS"
  - "tmpfs mounts for /tmp, /var/tmp, /var/log (volatile directories)"
  - "First-boot systemd oneshot service for SSID/password setup (D-06, D-08)"
  - "OverlayFS enable/disable documentation at /data/skygate/overlayfs-commands.txt"
affects: [02-dashboard-captive-portal, phase-image-builder]

# Tech tracking
tech-stack:
  added: [overlayfs, ext4-journal]
  patterns: [systemd-oneshot-firstboot, writable-data-partition, raspi-config-nonint]

key-files:
  created:
    - pi/ansible/roles/base/templates/fstab-data.j2
    - pi/ansible/roles/firstboot/tasks/main.yml
    - pi/ansible/roles/firstboot/templates/firstboot.sh.j2
    - pi/ansible/roles/firstboot/templates/skygate-firstboot.service.j2
    - pi/scripts/firstboot.sh
    - pi/systemd/skygate-firstboot.service
  modified:
    - pi/ansible/roles/base/tasks/main.yml

key-decisions:
  - "OverlayFS enabled manually after Ansible deploy (not automated) to prevent bricking during setup"
  - "Data partition uses ext4 with data=journal for crash-safe writes on /data"
  - "First-boot uses serial console TTY input -- simpler than web UI, works before AP is configured"

patterns-established:
  - "Writable /data partition: all persistent state lives on /data, root is read-only"
  - "systemd oneshot with ConditionPathExists: run-once services guard on flag file"
  - "raspi-config nonint: scriptable enable/disable for OverlayFS"

requirements-completed: [NET-01, NET-02]

# Metrics
duration: 2min
completed: 2026-03-23
---

# Phase 1 Plan 5: OverlayFS Read-Only Root and First-Boot Setup Summary

**OverlayFS read-only root with ext4 /data partition for crash-safe persistence, plus systemd oneshot first-boot wizard for pilot WiFi SSID/password configuration**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-23T22:48:47Z
- **Completed:** 2026-03-23T22:50:51Z
- **Tasks:** 2 (1 auto + 1 checkpoint auto-approved)
- **Files modified:** 7

## Accomplishments
- Extended base Ansible role with writable /data partition (mmcblk0p3, ext4, data=journal), Pi-hole symlink to /data/pihole, tmpfs for volatile dirs, and OverlayFS enable/disable documentation
- Created first-boot Ansible role with systemd oneshot service that prompts pilot for SSID/password on first boot via serial console (per D-06, D-08)
- Reference copies of firstboot.sh and skygate-firstboot.service for development use

## Task Commits

Each task was committed atomically:

1. **Task 1: Add OverlayFS and data partition setup to base role, create first-boot role** - `c191343` (feat)
2. **Task 2: Review complete Phase 1 network foundation** - Auto-approved (checkpoint:human-verify)

## Files Created/Modified
- `pi/ansible/roles/base/tasks/main.yml` - Extended with /data partition mount, persistent dirs, Pi-hole symlink, tmpfs mounts, OverlayFS documentation
- `pi/ansible/roles/base/templates/fstab-data.j2` - fstab entry for /dev/mmcblk0p3 -> /data
- `pi/ansible/roles/firstboot/tasks/main.yml` - Deploys firstboot script, systemd service, enables service
- `pi/ansible/roles/firstboot/templates/firstboot.sh.j2` - Ansible-templated first-boot script with Jinja2 variables
- `pi/ansible/roles/firstboot/templates/skygate-firstboot.service.j2` - Ansible-templated systemd oneshot service
- `pi/scripts/firstboot.sh` - Reference/development copy of first-boot script with hardcoded paths
- `pi/systemd/skygate-firstboot.service` - Reference/development copy of systemd service unit

## Decisions Made
- OverlayFS enabled manually via `raspi-config nonint do_overlayfs 0` after full Ansible deploy -- automated enable would brick the Pi if any config step fails
- ext4 with `data=journal` mount option for /data partition -- ensures metadata + data journaling for crash safety on abrupt power loss
- First-boot uses serial console TTY (StandardInput=tty, TTYPath=/dev/tty1) -- simpler than web UI, works before WiFi AP is configured, aligns with initial setup workflow
- WPA2 password validation (8-63 chars) in firstboot script prevents invalid hostapd config
- OverlayFS remount logic in firstboot script handles case where root is already read-only when first-boot runs

## Deviations from Plan

None - plan executed exactly as written. Task 1 files were already committed by a prior execution at `c191343`.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Complete Phase 1 network foundation with 8 Ansible roles: base, networking, pihole, routing, qos, firstboot, dashboard, wireguard
- All Go daemon tests pass (bypass-daemon, dashboard-daemon, tunnel-monitor)
- All BATS tests pass (autorate)
- Ready for Phase 2 (dashboard/captive portal) or hardware testing on actual Raspberry Pi

## Self-Check: PASSED

All 7 created/modified files verified present. Commit `c191343` verified in history.

---
*Phase: 01-pi-network-foundation*
*Completed: 2026-03-23*
