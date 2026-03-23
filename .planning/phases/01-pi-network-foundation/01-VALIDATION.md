---
phase: 1
slug: pi-network-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (daemon) + BATS (bash scripts) + Ansible `--check` + integration shell scripts |
| **Config file** | None yet — Wave 0 creates go.mod, test files, and BATS setup |
| **Quick run command** | `go test ./... -short && ansible-lint pi/ansible/` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~15 seconds (unit/lint); integration tests require Pi hardware |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -short && ansible-lint pi/ansible/`
- **After every plan wave:** Run `make test` (full suite including integration if Pi available)
- **Before `/gsd:verify-work`:** Full suite must be green + all integration tests pass on physical Pi
- **Max feedback latency:** 15 seconds (unit/lint only)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | NET-01 | integration (Pi) | `ssh pi 'hostapd_cli status && iwconfig wlan1'` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | NET-02 | integration (Pi) | `ssh pi 'cat /var/lib/misc/dnsmasq.leases && dig @192.168.4.1 google.com'` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 1 | DNS-01 | integration (Pi) | `ssh pi 'dig @192.168.4.1 doubleclick.net \| grep NXDOMAIN'` | ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 1 | ROUTE-01 | unit (Go) | `go test ./cmd/bypass-daemon/ -run TestResolveAndPopulate -short` | ❌ W0 | ⬜ pending |
| 01-03-02 | 03 | 1 | ROUTE-01 | integration (Pi) | `ssh pi 'nft list set inet skygate bypass_v4 && ip rule show'` | ❌ W0 | ⬜ pending |
| 01-04-01 | 04 | 2 | QOS-01 | integration (Pi) | `ssh pi 'tc qdisc show dev eth0 \| grep cake'` | ❌ W0 | ⬜ pending |
| 01-04-02 | 04 | 2 | QOS-01 | unit (bash/BATS) | `bats pi/scripts/tests/test_autorate.bats` | ❌ W0 | ⬜ pending |
| 01-05-01 | 05 | 2 | -- | manual + integration | `ssh pi 'mount \| grep "on / " \| grep ro'` | ❌ W0 | ⬜ pending |
| 01-05-02 | 05 | 2 | -- | lint + check | `ansible-playbook pi/ansible/playbook.yml --check --diff` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `go.mod` — Go module initialization
- [ ] `cmd/bypass-daemon/main_test.go` — unit tests for DNS resolution and nftables set management
- [ ] `pi/scripts/tests/test_autorate.bats` — BATS tests for autorate script logic
- [ ] `pi/ansible/playbook.yml` — skeleton playbook for `ansible-lint`
- [ ] `Makefile` — top-level build/test/deploy targets
- [ ] Install BATS on dev machine: `brew install bats-core`
- [ ] Install Ansible: `pip3 install ansible ansible-lint`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WiFi AP accepts connections from real device | NET-01 | Requires physical Pi + client device | Connect phone to Pi WiFi, verify IP assignment, browse google.com |
| ForeFlight/Garmin Pilot bypass works | ROUTE-01 | Requires app + Pi + Starlink | Open ForeFlight on iPad connected to Pi WiFi, verify data sync works |
| Pi survives abrupt power loss | -- | Requires physical power cycling | Power off Pi via master switch, wait 10s, power on, verify AP comes back |
| CAKE reduces bufferbloat under load | QOS-01 | Requires multiple devices + bandwidth test | Connect 3+ devices, run iperf3, verify latency stays stable |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
