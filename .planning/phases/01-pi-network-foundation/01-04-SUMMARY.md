---
phase: 01-pi-network-foundation
plan: 04
subsystem: infra
tags: [bash, cake-qdisc, autorate, bats, ansible, systemd, traffic-shaping, starlink]

# Dependency graph
requires:
  - phase: 01-02
    provides: "Networking role with eth0 uplink interface, nftables, base packages"
provides:
  - CAKE autorate bash script with RTT-based dynamic bandwidth adjustment (5-100 Mbps)
  - 9 BATS tests covering all autorate algorithm edge cases
  - Ansible QoS role with CAKE initialization, autorate deployment, and systemd service
  - Templated autorate script with all 11 parameters driven by group_vars
affects: [01-05]

# Tech tracking
tech-stack:
  added: [cake-qdisc, fping, bc, bats-core]
  patterns: [env-overridable-defaults, bash-source-guard-for-testing, ansible-template-from-reference-script]

key-files:
  created:
    - pi/scripts/autorate.sh
    - pi/ansible/roles/qos/templates/autorate.sh.j2
    - pi/ansible/roles/qos/templates/skygate-autorate.service.j2
    - pi/ansible/roles/qos/handlers/main.yml
    - pi/systemd/skygate-autorate.service
  modified:
    - pi/scripts/tests/test_autorate.bats
    - pi/ansible/roles/qos/tasks/main.yml

key-decisions:
  - "DRY_RUN made environment-overridable for BATS test sourcing compatibility"
  - "BATS test path uses relative ../autorate.sh (not absolute /opt/skygate/) for dev-machine testing"
  - "autorate.sh uses BASH_SOURCE guard so it can be sourced for unit testing without running main loop"

patterns-established:
  - "Bash scripts: environment-overridable defaults with ${VAR:-default} pattern"
  - "BATS testing: source script in setup(), export -f functions, test pure functions without system dependencies"
  - "Ansible QoS role: tasks install deps, init qdisc, deploy template, enable systemd service"

requirements-completed: [QOS-01]

# Metrics
duration: 6min
completed: 2026-03-23
---

# Phase 1 Plan 04: QoS Traffic Shaping Summary

**CAKE autorate bash script with fping RTT measurement, dynamic 5-100 Mbps bandwidth adjustment, 9 BATS tests, and Ansible QoS role with systemd service**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-23T08:54:40Z
- **Completed:** 2026-03-23T09:00:40Z
- **Tasks:** 2
- **Files created/modified:** 7

## Accomplishments
- Autorate script implements complete RTT-based bandwidth control: measures latency via fping every 2s, decreases rate by 30% on bufferbloat, increases by 5 Mbps after 5 stable cycles, with floor at 5 Mbps and ceiling at 100 Mbps
- All 9 BATS tests pass covering decrease on bufferbloat, floor enforcement, stable-period increase, ceiling enforcement, steady-state hold, stable-count reset, boundary condition (exact threshold), dry-run output verification, and convergence behavior
- Ansible QoS role installs bc dependency, initializes CAKE qdisc on eth0, deploys templated autorate script with all 11 configurable parameters from group_vars, and enables systemd service with auto-restart

## Task Commits

Each task was committed atomically:

1. **Task 1: Create autorate bash script and BATS tests** - `2144e94` (feat)
2. **Task 2: Create Ansible QoS role with CAKE and autorate systemd service** - `5a7e5df` (feat)

## Files Created/Modified
- `pi/scripts/autorate.sh` - Reference autorate script: RTT measurement, rate calculation, CAKE application with env-overridable defaults
- `pi/scripts/tests/test_autorate.bats` - 9 BATS tests for autorate algorithm (replaced stub from plan 01)
- `pi/ansible/roles/qos/tasks/main.yml` - QoS role: install bc, init CAKE, deploy autorate, enable systemd service
- `pi/ansible/roles/qos/handlers/main.yml` - Handlers: restart skygate-autorate, reload systemd
- `pi/ansible/roles/qos/templates/autorate.sh.j2` - Ansible-templated autorate with all parameters from group_vars
- `pi/ansible/roles/qos/templates/skygate-autorate.service.j2` - Templated systemd unit using skygate_opt_dir
- `pi/systemd/skygate-autorate.service` - Reference systemd unit file (ExecStart=/opt/skygate/autorate.sh)

## Decisions Made
- Made DRY_RUN environment-overridable (`${DRY_RUN:-false}`) so BATS tests can set DRY_RUN=true before sourcing the script, avoiding tc command dependency during testing
- BATS tests source the script and call functions directly rather than running the script as a subprocess, enabling pure function testing without fping/tc
- Autorate template uses all 11 Ansible variables from group_vars/all.yml -- operators can tune every parameter without editing scripts

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed DRY_RUN not being overridable via environment**
- **Found during:** Task 1 (BATS test execution)
- **Issue:** Script set `DRY_RUN=false` unconditionally, overwriting the `DRY_RUN=true` exported by BATS setup. The `apply_cake` test failed because it tried to run `tc` on macOS.
- **Fix:** Changed `DRY_RUN=false` to `DRY_RUN="${DRY_RUN:-false}"` so environment can override the default
- **Files modified:** pi/scripts/autorate.sh
- **Verification:** `bats pi/scripts/tests/test_autorate.bats` passes all 9 tests including apply_cake dry-run test
- **Committed in:** 2144e94 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed BATS apply_cake test using `run` for sourced function**
- **Found during:** Task 1 (BATS test execution)
- **Issue:** BATS `run` command creates a subshell where sourced functions are unavailable (exit code 127). Test used `run apply_cake 20000` which failed.
- **Fix:** Changed test to call `apply_cake` directly and capture output via command substitution instead of `run`
- **Files modified:** pi/scripts/tests/test_autorate.bats
- **Verification:** Test 8 passes with status 0
- **Committed in:** 2144e94 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes were necessary for BATS tests to pass on macOS development machine. The DRY_RUN fix also improves the script's flexibility. No scope creep.

## Issues Encountered
- BATS was not installed on PATH; required `brew install bats-core` during execution. This was noted as a known issue in plan 01 SUMMARY.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- QoS role fully implements the final Ansible role stub from plan 01 (alongside base, networking, pihole, routing from plans 02 and 03)
- Plan 05 (firstboot/overlayfs) is the remaining plan in phase 01
- Autorate parameters can be tuned with real Starlink data once deployed to physical Pi (cake_baseline_rtt_ms, cake_threshold_ms, cake_decrease_factor are the key tuning knobs)

## Self-Check: PASSED

All 7 created/modified files verified present. Both task commits (2144e94, 5a7e5df) confirmed in git log. SUMMARY.md exists.

---
*Phase: 01-pi-network-foundation*
*Completed: 2026-03-23*
