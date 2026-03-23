# Project Research Summary

**Project:** SkyGate
**Domain:** Embedded Linux network appliance for aviation bandwidth management
**Researched:** 2026-03-22
**Confidence:** MEDIUM-HIGH

## Executive Summary

SkyGate is a Raspberry Pi-based WiFi bridge that sits between passengers and a Starlink Mini on GA aircraft, combining DNS-level blocking, content-compressing proxy, dynamic QoS, and a usage dashboard to reduce metered satellite data usage by 60-90%. The market opportunity is real and urgent: Starlink restructured GA pricing in March 2026 (from $50/100GB to $250/20GB), the entire community response has been political (petitions, FCC complaints), and zero technical solutions exist. SkyGate would be the first.

The recommended approach is a split architecture: the Pi handles WiFi AP, DNS filtering, traffic routing, QoS, and the usage dashboard. A remote server handles content compression via a Go-based MITM proxy, with traffic tunneled through WireGuard. This split is architecturally necessary -- the proxy must fetch full content on cheap terrestrial bandwidth, then send only compressed content over the expensive satellite link. The core stack is entirely open-source Linux tooling: hostapd, Pi-hole, WireGuard, CAKE qdisc, and custom Go binaries for the dashboard and proxy.

The biggest risk is not technical but operational: SD card corruption from power loss (Pi has no graceful shutdown in aircraft), WiFi chip instability as an access point (requires external USB adapter), and captive portal compatibility across iOS/Android devices. The compy proxy (the PROJECT.md's chosen compression engine) is unmaintained since ~2021 with 35 open issues. The recommendation is to build a custom Go proxy on elazarl/goproxy with tdewolff/minify and imaging libraries -- proven, maintained components assembled for SkyGate's specific needs.

## Key Findings

### Recommended Stack

The stack splits cleanly into Pi-side (on-aircraft) and server-side (remote) components. All Pi-side components are standard Linux tools and Go binaries, keeping RAM usage under 400MB on a 4GB Pi 5. The server side runs in Docker Compose for one-command deployment.

**Core technologies:**
- **Raspberry Pi 5 (4GB) + Pi OS Lite Bookworm ARM64:** Standard embedded Linux platform, kernel 6.12 with native WireGuard and CAKE support
- **hostapd + dnsmasq:** WiFi AP with DHCP and DNS-based routing via ipset integration
- **Pi-hole v6.3+:** DNS filtering with 45.1k-star community blocklists, Layer 1 savings (50-60%)
- **WireGuard (kernel module):** In-kernel VPN tunnel, ARM-optimized crypto, 4000 LOC
- **Custom Go proxy (on goproxy + minify + imaging):** Content compression with MITM, replacing unmaintained compy
- **CAKE qdisc:** In-kernel traffic shaping preventing Starlink bufferbloat
- **Go + HTMX + Caddy:** Dashboard and captive portal with zero Node.js/npm dependencies

### Expected Features

**Must have (table stakes):**
- WiFi access point with WPA2 -- the product surface
- DNS-level blocking (ads, trackers, video CDNs, updates) -- immediate 50-60% savings
- Per-device usage dashboard with category breakdown -- the "whoa" moment, the viral screenshot
- Captive portal with terms acceptance -- legal cover and trust establishment
- Aviation app bypass routing -- ForeFlight/Garmin direct to Starlink (safety-critical)

**Should have (competitive):**
- Content compression proxy via WireGuard tunnel -- 80-90% savings (the novel architecture)
- "Quick Connect" vs "Max Savings" two-tier UX -- graduated engagement
- Dynamic QoS via CAKE -- prevents Starlink bufferbloat
- Bandwidth savings display with dollar conversion -- "$12.50 saved this flight"

**Defer (v2+):**
- Social media text-only mode -- fragile HTML rewriting, ongoing maintenance
- Pre-configured SD card image -- requires stable full stack first
- Hosted multi-tenant proxy service -- Phase C business concern
- 3D printed aviation case -- parallel workstream, not blocking

### Architecture Approach

Split architecture with two deployment targets: Pi (on-aircraft) and remote server (ground). Pi handles Layer 1 (DNS blocking, all devices, zero friction) and traffic routing. Server handles Layer 2 (content compression, browser-only, CA cert required). WireGuard tunnel connects them. Policy-based routing via nftables marks sends aviation app traffic directly to Starlink while everything else goes through the tunnel.

**Major components:**
1. **WiFi AP + DHCP + DNS (hostapd/dnsmasq/Pi-hole)** -- network foundation and Layer 1 filtering
2. **Policy Router (nftables + ipset)** -- DNS-driven traffic routing decisions (bypass vs tunnel)
3. **Usage Monitor + Dashboard (Go daemon + HTMX)** -- per-device tracking, real-time UI, captive portal
4. **WireGuard Tunnel** -- encrypted pipe to remote proxy
5. **Content Proxy (Go on remote server)** -- MITM, image transcoding, JS/CSS minification
6. **QoS Engine (CAKE)** -- prevents bufferbloat on variable Starlink link

### Critical Pitfalls

1. **SD card corruption from power loss** -- Use read-only root with OverlayFS from day one. Aircraft power is binary (master switch off = instant power loss). Must design for this in Phase 1.
2. **Onboard WiFi chip instability as AP** -- Use external USB WiFi adapter for AP, Ethernet for Starlink uplink. Onboard chip drops clients, has poor range, and doesn't support reliable AP mode.
3. **Captive portal cross-platform compatibility** -- iOS, Android, and Windows all detect captive portals differently. Must intercept platform-specific probe URLs. Requires real-device testing.
4. **DNS blocking breaks aviation safety apps** -- Aviation app allowlist must ship in v0.1. ForeFlight/Garmin use same CDNs as ad networks. Include "panic button" to disable all filtering.
5. **compy proxy is unmaintained** -- 35 open issues, last commit ~2021, no releases. Build custom Go proxy on maintained libraries instead.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Pi Foundation (WiFi AP + DNS + Dashboard)
**Rationale:** The dashboard is the primary value proposition per field observations. DNS blocking is immediate value. No remote server needed. This is the shippable v0.1.
**Delivers:** Working Pi that passengers connect to, with DNS blocking and real-time usage dashboard.
**Addresses:** WiFi AP, Pi-hole, usage dashboard, captive portal, aviation app bypass, static CAKE QoS.
**Avoids:** SD card corruption (read-only root from day one), WiFi instability (external USB adapter), DNS blocking aviation apps (allowlist ships here).

### Phase 2: WireGuard Tunnel + Policy Routing
**Rationale:** Tunnel infrastructure must be solid before building the proxy. Policy routing (aviation bypass vs tunnel) is the critical traffic decision layer. Tunnel resilience over satellite is the hardest networking problem.
**Delivers:** Working WireGuard tunnel to remote server with DNS-driven bypass routing.
**Uses:** WireGuard (kernel), nftables, ipset, dnsmasq ipset directives.
**Avoids:** WireGuard instability (MTU tuning, keepalive, fallback routing), Starlink speed cap confusion (dashboard messaging).

### Phase 3: Content Compression Proxy
**Rationale:** The proxy is the novel core but depends on tunnel (Phase 2) and dashboard (Phase 1). Build on goproxy, not compy. Image transcoding + JS/CSS minification first, social media text-only later.
**Delivers:** Go-based MITM proxy on remote server with image compression and minification.
**Uses:** elazarl/goproxy, kolesa-team/go-webp, tdewolff/minify, Docker Compose.
**Avoids:** compy unmaintained dependency, mitmproxy memory issues on ARM.

### Phase 4: CA Cert Distribution + Two-Tier UX
**Rationale:** MITM proxy (Phase 3) requires CA cert on devices. Captive portal (Phase 1) provides the distribution mechanism. Two-tier UX ("Quick Connect" vs "Max Savings") is the user-facing integration of all prior phases.
**Delivers:** Per-device CA cert generation, iOS .mobileconfig and Android cert download, mode selection in captive portal.
**Avoids:** MITM breaking cert-pinned apps (bypass list), shared CA key vulnerability (per-device generation).

### Phase 5: Product Polish + Distribution
**Rationale:** SD card image, 3D case, and setup wizard are product polish that only makes sense after the technical stack is proven.
**Delivers:** Pre-configured SD card image (rpi-image-gen), 3D printed case (STL files), first-boot wizard, documentation.
**Uses:** rpi-image-gen YAML config, Foundry for 3D printing.

### Phase Ordering Rationale

- **Dashboard before proxy:** Field observations confirm pilots need visibility before control. The dashboard is the hook, the proxy is the payoff. Shipping dashboard-only validates market demand before investing in proxy complexity.
- **DNS before WireGuard:** Pi-hole provides 50-60% savings with zero tunnel infrastructure. Validates the approach independently.
- **Tunnel before proxy:** WireGuard resilience over satellite is the hardest networking problem and must be proven before building a proxy that depends on it.
- **CA cert after proxy:** The two-tier UX only makes sense once the proxy exists. Cert distribution is a captive portal extension (Phase 1 + Phase 3 integration).
- **SD image last:** Can only build a distributable image after all components are integrated and tested in the field.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1:** Captive portal cross-platform detection (iOS CNA behavior is poorly documented, needs physical device testing matrix)
- **Phase 2:** WireGuard MTU tuning for Starlink (empirical testing required, varies by satellite constellation)
- **Phase 2:** cake-autorate porting from OpenWrt to Raspberry Pi OS (may need custom autorate script instead)
- **Phase 3:** goproxy MITM reliability for image transcoding pipeline (content-length mismatches are a known issue)
- **Phase 4:** iOS .mobileconfig CA cert distribution (Apple's MDM/profile requirements may have changed)

Phases with standard patterns (skip research-phase):
- **Phase 1 (hostapd/Pi-hole):** Well-documented, thousands of guides, standard setup
- **Phase 5 (rpi-image-gen):** Official Raspberry Pi tool with clear documentation

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | MEDIUM-HIGH | Core Linux tools well-established. Custom Go proxy approach is sound but unproven for this specific use case. compy maintenance status verified (unmaintained). |
| Features | HIGH | Validated against pilot forum observations. Dashboard-first strategy strongly supported by field data. Feature dependencies are clear. |
| Architecture | HIGH | Split proxy architecture is sound (enterprise PEPs use similar patterns). DNS-driven policy routing is well-documented. |
| Pitfalls | HIGH | Hardware pitfalls (SD corruption, WiFi instability) well-documented in Pi community. Starlink-specific behavior is MEDIUM (less documented for aviation use). |

**Overall confidence:** MEDIUM-HIGH

### Gaps to Address

- **Starlink in-flight latency profile:** Need empirical measurement of WireGuard tunnel performance over actual Starlink aviation connection. Forum reports suggest 40-60ms base + satellite handoff spikes, but aviation-specific data is sparse.
- **compy benchmarking:** While recommending custom Go proxy, compy should be benchmarked first as proof-of-concept. It may work well enough for v0.2 before building custom.
- **cake-autorate on Raspberry Pi OS:** Official support is OpenWrt/Asus Merlin only. Porting feasibility needs validation during Phase 2 planning. Fallback is a simpler custom autorate bash script.
- **USB WiFi adapter selection:** Need specific adapter recommendation with confirmed AP mode + range testing in aircraft cabin dimensions. MediaTek MT7612U and Realtek RTL8812BU are candidates but need physical testing.
- **Starlink ToS compliance:** Does proxying traffic violate Starlink's Terms of Service? Likely fine (you own the network, passengers consent), but needs legal review.

## Sources

### Primary (HIGH confidence)
- [Pi-hole documentation](https://docs.pi-hole.net/) -- v6.3/6.4, prerequisites, FTLDNS architecture
- [WireGuard official](https://www.wireguard.com/) -- kernel integration, ARM performance, NAT traversal
- [Raspberry Pi OS downloads](https://www.raspberrypi.com/software/operating-systems/) -- Bookworm Lite ARM64, kernel 6.12
- [rpi-image-gen announcement](https://www.raspberrypi.com/news/introducing-rpi-image-gen-build-highly-customised-raspberry-pi-software-images/) -- 2025 official image builder
- [nftables wiki](https://wiki.nftables.org/) -- native sets, counters, policy routing
- [Caddy official](https://caddyserver.com/) -- ARM64 support confirmed
- [HTMX official](https://htmx.org/) -- 14KB, SSE extension, no build step

### Secondary (MEDIUM confidence)
- [compy GitHub](https://github.com/barnacs/compy) -- 209 stars, unmaintained since ~2021, 35 open issues
- [goproxy GitHub](https://github.com/elazarl/goproxy) -- v1.8.2, MITM support, active maintenance
- [cake-autorate GitHub](https://github.com/lynxthecat/cake-autorate) -- OpenWrt/Merlin only, Starlink support
- [openNDS docs](https://opennds.readthedocs.io/) -- v10.3.0, captive portal framework
- [dnsmasq ipset integration](https://man.archlinux.org/man/dnsmasq.8.en) -- DNS-based routing
- [Raspberry Pi forum: WiFi AP instability](https://forums.raspberrypi.com/) -- onboard chip limitations
- [mitmproxy memory issues](https://github.com/mitmproxy/mitmproxy/issues/6371) -- ARM memory growth confirmed

### Tertiary (LOW confidence)
- Pilots of America forum threads -- market validation (anecdotal, no quantitative data)
- Starlink aviation pricing changes -- confirmed by multiple sources but specifics may shift
- cake-autorate standalone Linux compatibility -- inferred from codebase, not officially tested

---
*Research completed: 2026-03-22*
*Ready for roadmap: yes*
