# SkyGate

## What This Is

An open source bandwidth management appliance for general aviation aircraft using Starlink satellite internet. A Raspberry Pi 5-based device that sits between passengers and the Starlink Mini, combining DNS-level blocking (Pi-hole), a content-compressing MITM proxy (Go on goproxy), WireGuard tunnel infrastructure, dynamic QoS (CAKE autorate), and a real-time usage dashboard — reducing data usage by 60-90% on metered aviation plans. Two-tier UX: "Quick Connect" (DNS only, zero friction) and "Max Savings" (proxy compression with CA cert). Deployed via Ansible to Pi, Docker Compose to remote server. The Eero of GA internet — "unbox and fly."

## Core Value

**Show pilots what's eating their data, then give them the controls to stop it.** The usage dashboard is the hook — pilots have zero visibility into their 20 GB cap. Awareness creates demand for controls.

## Current State

**v1.0 shipped 2026-03-24.** All 5 phases complete, 18 plans executed, 18/18 requirements satisfied.

**Codebase:**
- Go: ~7,700 LOC across 4 daemons (bypass-daemon, dashboard-daemon, proxy-server, tunnel-monitor)
- HTML: ~825 LOC (dashboard, captive portal, cert install guides)
- Ansible: ~7,600 LOC YAML (9 roles: base, networking, pihole, dashboard, routing, wireguard, certificate, qos, firstboot)
- 115 git commits over 2 days

**What's deployed:**
- Pi: WiFi AP (hostapd), DNS blocking (Pi-hole v6), aviation bypass routing (Go daemon + nftables), CAKE QoS (autorate), captive portal, usage dashboard (HTMX/SSE/Chart.js), cert management, OverlayFS read-only root
- Remote server: WireGuard endpoint, Go MITM proxy (WebP transcoding, JS/CSS minification), Docker Compose deployment

**What's NOT yet tested on hardware:**
- Physical Pi WiFi AP connectivity
- Pi-hole DNS blocking on live traffic
- OverlayFS power-loss resilience
- CAKE autorate on Starlink link
- iOS/Android cert install flows

## Requirements

### Validated

- ✓ NET-01: WiFi access point with WPA2 — v1.0
- ✓ NET-02: DHCP + DNS routing through Pi-hole — v1.0
- ✓ DNS-01: Ad/tracker DNS blocking with community blocklists — v1.0
- ✓ DASH-01: Per-device usage tracking (bytes per MAC, SQLite) — v1.0
- ✓ DASH-02: Dashboard with top domains, category pie chart — v1.0
- ✓ DASH-03: Real-time bandwidth graph (SSE + Chart.js) — v1.0
- ✓ DASH-04: Captive portal with terms acceptance — v1.0
- ✓ DASH-05: Dollar savings display based on Starlink overage rate — v1.0
- ✓ DASH-06: Plan cap tracking with threshold alerts — v1.0
- ✓ ROUTE-01: Aviation app bypass routing (Go daemon + nftables ipset) — v1.0
- ✓ ROUTE-02: Policy routing via WireGuard tunnel — v1.0
- ✓ QOS-01: CAKE autorate with dynamic bandwidth adjustment — v1.0
- ✓ TUN-01: WireGuard tunnel with auto-reconnect — v1.0
- ✓ PROXY-01: Go MITM proxy with WebP transcoding + JS/CSS minification — v1.0
- ✓ PROXY-02: Docker Compose one-command deployment — v1.0
- ✓ CERT-01: Quick Connect / Max Savings two-tier mode selection — v1.0
- ✓ CERT-02: Per-device CA cert download (iOS .mobileconfig, Android .crt) — v1.0
- ✓ CERT-03: Cert-pinning bypass for banking/auth/gov/health apps — v1.0

### Active (v2.0 candidates)

- [ ] DNS-02: Video CDN domain blocking (YouTube, Netflix, TikTok)
- [ ] DNS-03: OS/app update domain blocking (iOS, Windows Update, Google Play)
- [ ] DNS-04: Background cloud sync blocking (iCloud, Google Photos, Dropbox)
- [ ] RES-02: "Panic button" to disable all filtering instantly
- [ ] POL-02: Pre-configured SD card image via rpi-image-gen
- [ ] MON-01: Flight-aware session tracking with per-flight savings report
- [ ] MON-03: Per-device bandwidth throttling via tc per MAC
- [ ] Brotli decompression support for Content-Encoding: br responses
- [ ] Cert rotation and renewal automation
- [ ] Content stripping rules: video block, social media text-only

### Out of Scope

- **Commercial airline integration** — different market, scale, certification
- **Non-aviation verticals in v1** — boats, RVs, remote workers are future markets
- **Multi-tenant hosted proxy** — single-tenant first, multi-tenancy for hosted service
- **FAA STC certification** — device operates as PED under AC 91.21-1D
- **Native app content modification** — cert-pinned apps can't be proxied; DNS only
- **OCSP responder** — unnecessary complexity for appliance use case
- **Per-passenger cert generation** — per-appliance CA is simpler and sufficient
- **MDM enterprise deployment** — GA pilots manage their own devices

## Constraints

- **HTTPS encryption**: ~95% of traffic encrypted. Two-layer approach: Layer 1 (DNS, all devices) + Layer 2 (MITM proxy, browsers with CA cert).
- **Starlink hardware**: Mini is a PED (11.5x10", 2.5 lbs, 20-40W, 12V/24V). Pi coexists.
- **FAA compliance**: PED under AC 91.21-1D. No interference with avionics.
- **Aircraft environment**: Vibration, temp variation, limited power, weight/balance sensitivity.
- **Latency budget**: Starlink (~40-60ms) + WireGuard (~5-10ms) + proxy (~10-200ms) = ~100-300ms total.
- **UX bar**: "Just works like an Eero." Zero-config basic operation.
- **Pi resources**: 4GB RAM. Go daemons ~20-50MB total.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Dashboard ships before content stripping | Awareness creates demand for controls | ✓ Good — Phase 2 before Phase 4 |
| Custom Go proxy on goproxy, not compy | compy unmaintained (2021). Fresh build on elazarl/goproxy | ✓ Good — clean API, active maintenance |
| Dual-fwmark policy routing (0x1 bypass, 0x2 tunnel) | Clean separation of bypass vs tunnel traffic | ✓ Good — validated Phase 3 |
| Static CAKE on wg0, autorate on eth0 only | Single autorate on physical link, no measurement interference | ✓ Good — validated Phase 3 |
| Tunnel monitor with 3-check hysteresis | Prevents flapping during satellite handoffs | ✓ Good — validated Phase 3 |
| Two-layer TLS: "Quick Connect" vs "Max Savings" | Zero friction for most users, full compression opt-in | ✓ Good — validated Phase 5 |
| Per-appliance unique CA, intermediate delegation | Root key stays on Pi, intermediate goes to proxy | ✓ Good — standard PKI pattern |
| Hardcoded never-MITM domains (28 entries) | Banking/auth/gov/health always bypass MITM | ✓ Good — trust over savings |
| modernc.org/sqlite over mattn/go-sqlite3 | CGO_ENABLED=0 cross-compilation | ✓ Good — simpler builds |
| OverlayFS manual enable after Ansible deploy | Prevents bricking during setup | ✓ Good — safe approach |
| Open source with three distribution layers | GitHub + hosted proxy + hardware kits | — Pending business validation |
| MaxSavingsIPSet polling (10s interval) | Proxy can't see MACs, polls Pi for mode-aware IPs | ✓ Good — validated Phase 5 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

---
*Last updated: 2026-03-24 after v1.0 milestone*
