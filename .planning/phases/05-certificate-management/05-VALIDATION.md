---
phase: 05
slug: certificate-management
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 05 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — tests are in cmd/proxy-server/ and cmd/dashboard-daemon/ |
| **Quick run command** | `go test ./cmd/proxy-server/... ./cmd/dashboard-daemon/... -short` |
| **Full suite command** | `go test ./... -short` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/proxy-server/... ./cmd/dashboard-daemon/... -short`
- **After every plan wave:** Run `go test ./... -short`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | CERT-01 | unit | `go test ./cmd/proxy-server/... -run TestIntermediate` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 1 | CERT-02 | unit | `go test ./cmd/dashboard-daemon/... -run TestMobileconfig` | ❌ W0 | ⬜ pending |
| 05-02-01 | 02 | 2 | CERT-01 | unit | `go test ./cmd/dashboard-daemon/... -run TestModeSelection` | ❌ W0 | ⬜ pending |
| 05-02-02 | 02 | 2 | CERT-03 | unit | `go test ./cmd/proxy-server/... -run TestBypass` | ✅ exists | ⬜ pending |
| 05-03-01 | 03 | 3 | CERT-01 | integration | `go test ./... -short` | ✅ exists | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/proxy-server/intermediate_test.go` — intermediate CA generation and chain validation
- [ ] `cmd/dashboard-daemon/mobileconfig_test.go` — .mobileconfig profile generation
- [ ] `cmd/dashboard-daemon/mode_test.go` — per-device mode selection API

*Existing infrastructure covers proxy bypass testing (cmd/proxy-server/proxy_test.go).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| iOS .mobileconfig install + trust | CERT-02 | Requires physical iOS device | Download profile, install in Settings, enable in Certificate Trust Settings |
| Android cert install guide | CERT-02 | Requires physical Android device | Download .crt, install via Security settings |
| Banking app works with Max Savings | CERT-03 | Requires real banking app on device with CA cert | Open banking app, verify normal operation |
| Captive portal mode selection UX | CERT-01 | Requires WiFi-connected device hitting captive portal | Connect to AP, verify two-option captive portal |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
