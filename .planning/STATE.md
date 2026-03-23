---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: Ready to plan
stopped_at: Completed 02-04-PLAN.md
last_updated: "2026-03-23T16:29:41.536Z"
progress:
  total_phases: 5
  completed_phases: 1
  total_plans: 9
  completed_plans: 8
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-22)

**Core value:** Show pilots what's eating their data, then give them the controls to stop it.
**Current focus:** Phase 02 — usage-dashboard

## Current Position

Phase: 03
Plan: Not started

## Performance Metrics

**Velocity:**

- Total plans completed: 4
- Average duration: 5.75 min
- Total execution time: 0.38 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 4 | 23min | 5.75min |

**Recent Trend:**

- Last 4 plans: 6, 4, 7, 6 min
- Trend: stable

*Updated after each plan completion*
| Phase 01 P01 | 6min | 2 tasks | 21 files |
| Phase 01 P02 | 4min | 2 tasks | 13 files |
| Phase 01 P03 | 7min | 2 tasks | 13 files |
| Phase 01 P04 | 6min | 2 tasks | 7 files |
| Phase 01 P05 | 3 | 2 tasks | 7 files |
| Phase 02 P02 | 3 | 2 tasks | 7 files |
| Phase 02 P01 | 9 | 2 tasks | 18 files |
| Phase 02 P03 | 8 | 3 tasks | 13 files |
| Phase 02 P04 | 3 | 2 tasks | 9 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Dashboard-first strategy -- Phase 1 builds network foundation, Phase 2 delivers the "whoa" moment dashboard
- [Roadmap]: Custom Go proxy on goproxy + minify + imaging libraries, not unmaintained compy
- [Roadmap]: Read-only root with OverlayFS from day one (Phase 1) to prevent SD card corruption
- [Phase 01]: Platform-specific nft via Go build tags (linux vs stub) for cross-platform dev
- [Phase 01]: YAML established as config file format for bypass domains and blocklists
- [Phase 01]: Pi-hole v6 TOML config with NULL blocking mode for silent NXDOMAIN responses (D-11)
- [Phase 01]: Pi-hole web interface disabled -- SkyGate has its own dashboard
- [Phase 01]: nftables bypass_v4 set with 1h timeout for aviation IP caching
- [Phase 01]: Exported Go function names (LoadConfig, ResolveDomains, FormatNftCommand) for testability and package-level API
- [Phase 01]: IPv4-only filtering in bypass daemon resolver -- GA Starlink networking is IPv4
- [Phase 01]: systemd service pattern: After nftables.service, Restart=always, ProtectSystem=strict for daemon deployment
- [Phase 01]: DRY_RUN made environment-overridable for BATS test compatibility
- [Phase 01]: Autorate script uses BASH_SOURCE guard for sourceable testing without running main loop
- [Phase 01]: OverlayFS enabled manually after Ansible deploy (not automated) to prevent bricking during setup
- [Phase 01]: Data partition uses ext4 with data=journal for crash-safe writes on /data
- [Phase 01]: First-boot uses serial console TTY input -- simpler than web UI, works before AP is configured
- [Phase 05]: "Quick Connect" is the default mode -- zero friction DNS-only for passengers who just tap Continue
- [Phase 05]: Per-device mode selection tracked by MAC address in nftables sets
- [Phase 05]: Per-appliance unique CA keypair generated on first boot, stored at /data/skygate/ca/ (0600, root only)
- [Phase 05]: iOS cert install via .mobileconfig profile; Android via direct .crt download
- [Phase 05]: Hardcoded never-MITM categories: banking, auth, gov, health, payments -- cannot be removed by user
- [Phase 05]: Intermediate CA delegated to remote proxy for leaf cert signing (root CA key never leaves Pi)
- [Phase 05]: Cert-pinning bypass uses nftables set + proxy TCP passthrough (traffic still routes through WireGuard)
- [Phase 05]: YAML config + dashboard UI for bypass list management (consistent with Phase 1 pattern)
- [Phase 05]: Post-flight cert removal instructions via dashboard + physical QR card
- [Phase 02]: Chart.js singleton pattern: create once, update via SSE to avoid memory leaks
- [Phase 02]: Captive portal has zero JS dependencies for iOS CNA compatibility
- [Phase 02]: Dark aviation theme (#0f172a base) for cockpit readability and viral screenshot aesthetics
- [Phase 02]: modernc.org/sqlite chosen over mattn/go-sqlite3 for CGO_ENABLED=0 cross-compilation compatibility
- [Phase 02]: Nanosecond timestamps for device_usage to avoid UNIQUE constraint collisions at high write rates
- [Phase 02]: Subdomain matching via domain-label walking for category lookup (m.facebook.com matches facebook.com)
- [Phase 02]: Server struct holds shared state across handlers (config, db, categories, pihole, counters)
- [Phase 02]: stdlib SSE via net/http Flusher -- no r3labs/sse dependency needed
- [Phase 02]: Conservative savings: 150KB/ad, 5KB/tracker payload heuristics for pilot trust
- [Phase 02]: Caddy host-header matching for CNA check URL interception -- DNAT preserves Host, @captive_check matcher triggers on known CNA domains
- [Phase 02]: nftables allowed_macs set with 24h timeout for captive portal session management
- [Phase 02]: Multi-daemon Makefile pattern: per-daemon build/cross-build targets with aggregate targets

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 1]: USB WiFi adapter selection needs physical testing -- onboard chip unreliable as AP
- [Phase 1]: Captive portal cross-platform detection (iOS CNA behavior) needs device testing matrix
- [Phase 3]: cake-autorate porting from OpenWrt to Pi OS may need custom autorate script

## Session Continuity

Last session: 2026-03-23T16:19:54.362Z
Stopped at: Completed 02-04-PLAN.md
Resume file: None
