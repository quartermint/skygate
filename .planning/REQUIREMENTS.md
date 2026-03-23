# Requirements: SkyGate

**Defined:** 2026-03-23
**Core Value:** Show pilots what's eating their data, then give them the controls to stop it.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Networking

- [ ] **NET-01**: Pi serves as WiFi access point with WPA2 that passengers connect to (hostapd + dnsmasq)
- [ ] **NET-02**: Connected devices receive IP addresses via DHCP with DNS routed through Pi-hole

### DNS Filtering

- [ ] **DNS-01**: Pi-hole blocks ads, trackers, and known malicious domains at DNS level with community blocklists

### Dashboard & Monitoring

- [ ] **DASH-01**: Per-device data usage tracked in near-real-time (bytes per MAC address, ~5s intervals, persisted to SQLite)
- [ ] **DASH-02**: Web dashboard displays top domains by bytes consumed, per-device breakdown, and category pie chart
- [ ] **DASH-03**: Dashboard shows real-time bandwidth graph of current throughput
- [ ] **DASH-04**: Captive portal intercepts first HTTP request from new devices, shows terms acceptance, and links to dashboard
- [ ] **DASH-05**: Dashboard displays bandwidth savings as dollar amount based on Starlink overage rate
- [ ] **DASH-06**: User can configure Starlink plan cap and billing cycle; dashboard shows usage against cap with alerts at 50%, 75%, 90%

### Traffic Routing

- [ ] **ROUTE-01**: Aviation apps (ForeFlight, Garmin Pilot, aviationweather.gov, ADS-B services) bypass proxy and route directly to Starlink via DNS-driven ipset
- [ ] **ROUTE-02**: Policy-based routing via nftables sends non-bypass traffic through WireGuard tunnel to remote proxy

### Quality of Service

- [ ] **QOS-01**: CAKE qdisc with cake-autorate dynamically adjusts bandwidth ceiling based on real-time latency, preventing Starlink bufferbloat

### Tunnel Infrastructure

- [ ] **TUN-01**: WireGuard kernel-mode tunnel connects Pi to remote proxy server with keepalive and auto-reconnect on connectivity loss

### Content Proxy

- [ ] **PROXY-01**: Go-based MITM proxy on remote server (built on goproxy) transcodes images (JPEG quality reduction, PNG/JPEG to WebP) and minifies JS/CSS
- [ ] **PROXY-02**: Remote proxy server deployable via one-command Docker Compose with WireGuard server endpoint

### Certificate Management

- [ ] **CERT-01**: Captive portal presents "Quick Connect" (DNS blocking only, zero friction) and "Max Savings" (proxy + CA cert) mode selection
- [ ] **CERT-02**: Per-device CA certificate generated and downloadable via captive portal — iOS .mobileconfig profile and Android cert install flow
- [ ] **CERT-03**: Certificate pinning bypass list prevents proxy from breaking banking, auth, and cert-pinned apps

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### DNS Filtering (Extended)

- **DNS-02**: Video CDN domain blocking (YouTube, Netflix, TikTok, Disney+) with friendly captive portal message
- **DNS-03**: OS/app update domain blocking (iOS swscan, Windows Update, Google Play)
- **DNS-04**: Background cloud sync domain blocking (iCloud backup, Google Photos, Dropbox, OneDrive)

### Resilience

- **RES-01**: Offline/degraded mode — Pi-hole continues DNS blocking if tunnel drops, dashboard shows degraded status, direct routing fallback
- **RES-02**: "Panic button" in dashboard to disable all filtering instantly

### Product Polish

- **POL-01**: Zero-config operation — pre-configured SD card boots into working state with default SSID "SkyGate"
- **POL-02**: Pre-configured SD card image via rpi-image-gen
- **POL-03**: First-boot setup wizard via web UI

### Advanced Monitoring

- **MON-01**: Flight-aware session tracking — group usage by flight with per-flight savings report
- **MON-02**: Category-based bandwidth visualization with domain-to-category mapping database
- **MON-03**: Per-device bandwidth throttling via tc per MAC address

### Hardware

- **HW-01**: 3D printed aviation case (PETG, STL files, status LEDs)
- **HW-02**: Hardware kit BOM and assembly documentation

### Content

- **CONT-01**: Social media text-only mode — strip Instagram/Twitter/Facebook to text + compressed thumbnails

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Native app content modification | Cert-pinned iOS/Android apps reject MITM — DNS blocking only for native apps |
| Deep Packet Inspection engine | Massively complex, resource-intensive on Pi, privacy/legal concerns — use DNS categorization instead |
| User tracking / marketing analytics | Violates trust model — captive portal collects nothing beyond terms acceptance |
| Multi-tenant hosted proxy | Phase C business concern — v1 is single-tenant (one Pi, one server) |
| Commercial airline integration | Different market (enterprise procurement, DO-160 cert), different scale (200+ pax) |
| Non-aviation verticals in v1 | GA-only wedge market — boats, RVs, remote workers are future forks |
| Parental controls / content filtering | Block for bandwidth only, not content modesty — if a domain uses 0 bytes, it passes |
| VPN for privacy | WireGuard tunnel serves proxy routing, not passenger privacy |
| App Store / content platform | SkyGate is infrastructure, not a content portal |
| Mesh networking | Single AP covers GA cabin (15-25 ft) — no mesh needed |
| Pre-flight data caching | Complex cache invalidation, unclear what to cache, low ROI vs proxy compression |
| Speed testing | cake-autorate already measures link quality — no separate speed test feature |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| NET-01 | — | Pending |
| NET-02 | — | Pending |
| DNS-01 | — | Pending |
| DASH-01 | — | Pending |
| DASH-02 | — | Pending |
| DASH-03 | — | Pending |
| DASH-04 | — | Pending |
| DASH-05 | — | Pending |
| DASH-06 | — | Pending |
| ROUTE-01 | — | Pending |
| ROUTE-02 | — | Pending |
| QOS-01 | — | Pending |
| TUN-01 | — | Pending |
| PROXY-01 | — | Pending |
| PROXY-02 | — | Pending |
| CERT-01 | — | Pending |
| CERT-02 | — | Pending |
| CERT-03 | — | Pending |

**Coverage:**
- v1 requirements: 18 total
- Mapped to phases: 0
- Unmapped: 18 ⚠️

---
*Requirements defined: 2026-03-23*
*Last updated: 2026-03-23 after initial definition*
