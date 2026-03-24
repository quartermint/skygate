# Phase 1: Pi Network Foundation - Context

**Gathered:** 2026-03-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Pi boots into a working WiFi access point that blocks ads/trackers at DNS level, routes aviation apps directly to Starlink, and shapes traffic to prevent bufferbloat — all on a corruption-resistant read-only root filesystem. No dashboard, no tunnel, no proxy — just the network foundation.

</domain>

<decisions>
## Implementation Decisions

### Development Approach
- **D-01:** Laptop-first development — core services developed and tested on macOS/Linux, deployed to Pi via Ansible for integration testing
- **D-02:** Ansible playbook for Pi configuration — declarative, reproducible deploys defining packages, configs, and services
- **D-03:** tc netem for local Starlink simulation (latency, jitter, bandwidth caps) during laptop development
- **D-04:** Real Starlink testing available via remote friend with ground Starlink — useful for validating tunnel/QoS behavior over actual satellite, but not aircraft-specific
- **D-05:** Test client devices are phones/laptops connecting to Pi WiFi and browsing via Safari/Chrome — no native test app needed

### WiFi AP Identity
- **D-06:** SSID set by pilot during first-boot setup. No hardcoded default — pilot customizes on first use
- **D-07:** 2.4 GHz only — better range in aircraft cabin, wider device compatibility, single USB adapter
- **D-08:** Default random password printed on device sticker. Pilot can change via web UI (Phase 2)
- **D-09:** Maximum 8 simultaneous devices — covers 1-4 passengers with 2 devices each

### DNS Blocking Scope
- **D-10:** Conservative out of the box — ads and trackers blocked by default only. Video CDN, update, and cloud sync blocking are opt-in categories (deferred to v2)
- **D-11:** Silent block (Pi-hole NXDOMAIN) — no custom block pages. Blank ad slots, failed connections. Standard Pi-hole behavior

### Aviation Bypass List
- **D-12:** Default bypass list ships with: ForeFlight (*.foreflight.com), Garmin Pilot (*.garmin.com, fly.garmin.com), Weather APIs (aviationweather.gov, NOAA/NWS), ADS-B services (FlightAware, Flightradar24)
- **D-13:** Bypass list managed via YAML/JSON config file on the Pi. Pilot edits via SSH or web UI (web UI in Phase 2)
- **D-14:** DNS responses for bypass domains populate ipset dynamically — traffic for these domains routes direct to Starlink

### Claude's Discretion
- Exact OverlayFS configuration and which directories are writable
- USB WiFi adapter recommendation (research should evaluate MediaTek MT7612U vs Realtek RTL8812BU)
- hostapd channel selection and power settings
- Pi-hole blocklist selection (which community lists to include)
- CAKE qdisc initial bandwidth parameters
- cake-autorate configuration values for Starlink profile
- First-boot setup implementation (minimal — just SSID and optional password change)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Full project vision, constraints, key decisions
- `.planning/REQUIREMENTS.md` — v1 requirements with REQ-IDs for this phase: NET-01, NET-02, DNS-01, ROUTE-01, QOS-01

### Research
- `.planning/research/STACK.md` — Recommended stack with specific versions (Pi 5, hostapd, Pi-hole v6.3+, WireGuard, CAKE)
- `.planning/research/ARCHITECTURE.md` — Split architecture, component boundaries, data flow
- `.planning/research/PITFALLS.md` — SD card corruption (OverlayFS), WiFi chip instability (USB adapter), captive portal cross-platform issues
- `.planning/research/FEATURES.md` — Feature dependencies, MVP critical path

### Design Document
- `~/.gstack/projects/skygate/ryanstern-unknown-design-20260322-161803.md` — Full approved design doc from /office-hours with adversarial review

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield project, no existing code

### Established Patterns
- None — patterns will be established in this phase (Ansible playbook structure, config file format, service management)

### Integration Points
- Phase 2 (Dashboard) will read from whatever data persistence this phase establishes (nftables counters, DNS query logs)
- Phase 3 (Tunnel) will extend the nftables rules and routing tables established here
- Config file format chosen here (YAML/JSON) becomes the standard for all phases

</code_context>

<specifics>
## Specific Ideas

- Development workflow: develop on laptop, deploy to Pi via Ansible, test with phone as client device
- Remote friend with ground Starlink available for real-world satellite latency testing (not aviation, but same constellation)
- First-boot experience should be minimal in Phase 1 — just SSID/password setup, full wizard deferred to Phase 2+

</specifics>

<deferred>
## Deferred Ideas

- iOS companion app / TestFlight for data testing — not in v1 scope, web dashboard is the client interface
- Video CDN, update, and cloud sync DNS blocking categories — deferred to v2 requirements
- Web UI for bypass list management — Phase 2 (dashboard)
- Custom block page when domains are blocked — deferred, silent block is sufficient

</deferred>

---

*Phase: 01-pi-network-foundation*
*Context gathered: 2026-03-23*
