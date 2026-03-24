---
phase: 3
slug: tunnel-infrastructure
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + BATS (bash automated testing) |
| **Config file** | pi/tests/ and server/tests/ |
| **Quick run command** | `go test ./cmd/tunnel-monitor/... ./internal/...` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/tunnel-monitor/... ./internal/...`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | TUN-01 | unit+integration | `go test ./...` | TBD | ⬜ pending |
| TBD | TBD | TBD | ROUTE-02 | unit+integration | `go test ./...` | TBD | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test stubs for WireGuard tunnel monitor daemon
- [ ] Test stubs for policy routing nftables rules
- [ ] BATS tests for tunnel fallback behavior

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WireGuard tunnel establishes through Starlink | TUN-01 | Requires physical Pi + Starlink hardware | Deploy to Pi, verify `wg show wg0` shows handshake |
| Satellite handoff resilience | TUN-01 | Requires sustained Starlink connection | Monitor handshake recency during flight simulation |
| Fallback routing during tunnel outage | TUN-01 | Requires stopping WireGuard server and verifying traffic reroutes | Stop Docker container, verify browsing continues via direct route |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
