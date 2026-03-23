---
phase: 2
slug: usage-dashboard
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-23
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + BATS (bash) |
| **Config file** | go.mod (Go), pi/scripts/tests/ (BATS) |
| **Quick run command** | `go test ./... -short` |
| **Full suite command** | `make test` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -short`
- **After every plan wave:** Run `make test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | DASH-01 | unit | `go test ./cmd/dashboard-daemon/ -short` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | DASH-01 | unit | `go test ./cmd/dashboard-daemon/ -run TestSQLite` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | DASH-04 | integration | `curl -s http://192.168.4.1/ \| grep -q "terms"` | ❌ W0 | ⬜ pending |
| 02-03-01 | 03 | 2 | DASH-02, DASH-03 | unit | `go test ./cmd/dashboard-daemon/ -run TestSSE` | ❌ W0 | ⬜ pending |
| 02-04-01 | 04 | 2 | DASH-05 | unit | `go test ./cmd/dashboard-daemon/ -run TestSavings` | ❌ W0 | ⬜ pending |
| 02-05-01 | 05 | 3 | DASH-06 | unit | `go test ./cmd/dashboard-daemon/ -run TestPlanCap` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/dashboard-daemon/monitor_test.go` — stubs for DASH-01 (per-device tracking)
- [ ] `cmd/dashboard-daemon/sse_test.go` — stubs for DASH-03 (SSE endpoint)
- [ ] `cmd/dashboard-daemon/savings_test.go` — stubs for DASH-05 (savings calculation)
- [ ] `cmd/dashboard-daemon/config_test.go` — stubs for DASH-06 (plan cap config)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Captive portal triggers on iOS/Android/macOS/Windows | DASH-04 | OS-specific CNA behavior requires real devices | Connect device to Pi WiFi, verify captive portal sheet appears |
| Real-time graph updates visually | DASH-03 | Visual correctness requires human inspection | Open dashboard in browser, verify line chart animates with SSE data |
| Category pie chart renders correctly | DASH-02 | Chart rendering is visual | Load dashboard, verify pie chart shows domain categories |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
