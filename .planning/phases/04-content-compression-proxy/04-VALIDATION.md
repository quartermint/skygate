---
phase: 04
slug: content-compression-proxy
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 04 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — Go test infrastructure established in Phase 1 |
| **Quick run command** | `go test ./server/proxy/... -short -count=1` |
| **Full suite command** | `go test ./... -short -count=1 && bats pi/scripts/tests/` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./server/proxy/... -short -count=1`
- **After every plan wave:** Run `go test ./... -short -count=1 && bats pi/scripts/tests/`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | PROXY-01 | unit | `go test ./server/proxy/... -run TestTranscode -count=1` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | PROXY-01 | unit | `go test ./server/proxy/... -run TestMinify -count=1` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | PROXY-02 | unit | `go test ./server/proxy/... -run TestConfig -count=1` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | PROXY-02 | integration | `docker compose -f server/docker-compose.yml config` | ✅ | ⬜ pending |
| 04-03-01 | 03 | 2 | PROXY-01, PROXY-02 | integration | `go test ./... -short -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `server/proxy/*_test.go` — test stubs for image transcoding and minification
- [ ] Go module dependencies added (go-webp, tdewolff/minify, goproxy)

*Existing Go test infrastructure from Phase 1/2 covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Images visibly smaller via proxy | PROXY-01 | Requires real browser + real websites | Browse test URLs via proxy, compare screenshot file sizes |
| Docker Compose deploys successfully | PROXY-02 | Requires Docker daemon on server | Run `docker compose up -d` on remote server, verify health |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
