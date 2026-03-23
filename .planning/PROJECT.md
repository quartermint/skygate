# SkyGate

## What This Is

An open source bandwidth management appliance for general aviation aircraft using Starlink satellite internet. A Raspberry Pi-based device that sits between passengers and the Starlink Mini, combining DNS-level blocking, content-compressing proxy, and dynamic QoS to reduce data usage by 60-90% on metered aviation plans. Three distribution layers: fully open source (GitHub), hosted proxy service, and pre-built hardware kits with 3D printed case. The Eero of GA internet — "unbox and fly."

## Core Value

**Show pilots what's eating their data, then give them the controls to stop it.** The usage dashboard is the hook — pilots have zero visibility into their 20 GB cap. Awareness creates demand for controls.

## Requirements

### Validated

- WireGuard tunnel to remote proxy server for content stripping — Phase 3
- Policy-based routing: aviation domains → direct, everything else → tunnel — Phase 3
- Offline/degraded mode — Pi-hole continues if tunnel drops, auto-reconnect — Phase 3
- One-command Docker Compose deployment for remote server — Phase 3

### Active

- [ ] Pi acts as WiFi access point passengers connect to (hostapd + dnsmasq)
- [ ] DNS-level blocking of video CDNs, ad networks, trackers, update servers (Pi-hole)
- [ ] Real-time per-device usage dashboard accessible via captive portal (top domains by bytes, category breakdown, bandwidth graph)
- [ ] Captive portal with terms acceptance and usage dashboard link
- [ ] Aviation app bypass — ForeFlight, Garmin Pilot, weather APIs route directly to Starlink, not through proxy
- [ ] Dynamic QoS preventing Starlink bufferbloat (cake-autorate)

- [ ] Content compression proxy on remote server — image transcoding (JPEG→WebP, quality reduction), JS/CSS minification (compy)
- [ ] MITM support with CA cert distribution via captive portal ("Quick Connect" vs "Max Savings" modes)

- [ ] Certificate pinning bypass list for banking/auth apps

- [ ] Content stripping rules: video block, image compress, social media text-only, update block
- [ ] 3D printed aviation case (PETG, STL files in repo)
- [ ] Pre-configured SD card image for zero-config setup

### Out of Scope

- **Commercial airline integration** — different market, different scale, different certification
- **Non-aviation verticals in v1** — boats, RVs, remote workers are future markets but v1 is GA-only (YAGNI guard)
- **Multi-tenant hosted proxy in v1** — Phase B is single-tenant, multi-tenancy deferred to hosted service buildout
- **FAA STC certification** — device operates as PED under AC 91.21-1D, no certification needed
- **Native app content modification** — cert-pinned iOS/Android apps can't be proxied; DNS blocking only for native apps

## Context

### Market Catalyst (March 2026)
Starlink restructured GA aviation pricing: $50/mo (100 GB, any speed) → $250-$1,000/mo (20 GB, speed-tiered). 100 mph speed cap on standard plans makes them useless in flight. AOPA, IAOPA formal complaints. 4,000+ petition signatures. FCC "bait and switch" complaints. Flying Magazine editorial. Meanwhile airlines get FREE Starlink. No technical counter-response exists — only political/legal action.

### Field Observations (Pilots of America forums)
1. **Pilots will pay** the $250/mo despite complaints. Value of in-flight connectivity too high.
2. **Pilots don't know what eats their data** — zero visibility into their 20 GB cap. Need to be told.
3. **Entire response is political** — petitions, FCC complaints, lobbying. No one thinking about technical solutions. Non-technical GA community ("boomer-mindset") can't conceive of a technical counter-move.

### Competitive Landscape (GitHub, 2026-03-22)
Zero direct competitors. Exhaustive search confirmed no repos for aviation bandwidth management. Adjacent building blocks exist but nobody has assembled them:
- **compy** (209 stars, Go) — HTTP/HTTPS compression proxy, built-in MITM + image transcoding
- **cake-autorate** (498 stars, bash) — dynamic Starlink QoS, prevents bufferbloat
- **bandwidth-hero** (284 stars, JS) — browser extension + proxy for image compression
- **Pi-hole** (45.1k stars) — DNS ad/tracker blocker
- **coovachilli** (642 stars, C) — captive portal with RADIUS + bandwidth stats

### Design Document
Full approved design doc: `~/.gstack/projects/skygate/ryanstern-unknown-design-20260322-161803.md`
Generated via gstack /office-hours with 2 rounds of adversarial review (21 issues found/fixed).

## Constraints

- **HTTPS encryption**: ~95% of web traffic encrypted. DNS blocking works for all traffic; content stripping requires MITM proxy with CA cert (browser-only). Two-layer approach: Layer 1 (DNS, all devices) + Layer 2 (proxy, browsers with CA cert).
- **Starlink hardware**: Mini is a PED (11.5x10", 2.5 lbs, 20-40W, 12V/24V). Pi must coexist.
- **FAA compliance**: PED under AC 91.21-1D. No interference with avionics. Low power, no external antennas.
- **Aircraft environment**: Vibration, temp variation, limited power, weight/balance sensitivity.
- **Latency budget**: Starlink (~40-60ms) + WireGuard (~5-10ms) + proxy processing (~10-200ms) = ~100-300ms total. Image recompression: inline with 500ms timeout, cache for subsequent loads.
- **UX bar**: "Just works like an Eero." Zero-config basic operation. Non-technical GA pilots are the primary audience.
- **Pi resources**: Min 2GB RAM. compy (Go, ~20MB) preferred over mitmproxy (Python, ~200-500MB) for proxy engine.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Dashboard ships before content stripping | Pilots don't know what eats their data — awareness creates demand for controls. "Whoa" moment is the pie chart, not the silent proxy. | — Pending |
| compy over mitmproxy for proxy engine | Go binary (~20MB RAM) vs Python (~200-500MB). Built-in image transcoding + JS/CSS minification without addons. Could run on Pi directly. | — Pending |
| Custom Go proxy on goproxy, not compy | compy unmaintained (last commit ~2021). Build fresh on elazarl/goproxy with minify + imaging libs. | Phase 3 decision |
| Dual-fwmark policy routing | 0x1 bypass (table 100, direct Starlink), 0x2 tunnel (table 200, WireGuard). Clean separation. | Phase 3 validated |
| Static CAKE on wg0, autorate on eth0 only | Single autorate instance on physical link. wg0 gets static bandwidth ceiling. Simpler, no measurement interference. | Phase 3 validated |
| Tunnel monitor with 3-check hysteresis | Prevents flapping during brief satellite handoffs. 3 consecutive failures to degrade, 3 successes to recover. | Phase 3 validated |
| cake-autorate for Starlink QoS | 498 stars, first-class Starlink support, prevents bufferbloat via dynamic CAKE bandwidth adjustment. Runs on any Linux including Pi. | — Pending |
| Two-layer TLS strategy | Layer 1 (DNS, all devices, zero friction) + Layer 2 (MITM proxy, browser-only, requires CA cert). "Quick Connect" vs "Max Savings" modes. | — Pending |
| Split Phase B into B.1 (Awareness) + B.2 (Control) | B.1 = dashboard + DNS blocking (shippable v0.1). B.2 = WireGuard + compy proxy (full architecture). | — Pending |
| Open source with three distribution layers | GitHub (hackers) + hosted proxy (convenience) + hardware kits (plug-and-play). Covers full market. | — Pending |
| quartermint GitHub org | Aviation context alongside OpenEFB and SFR. | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-03-23 after Phase 3*
