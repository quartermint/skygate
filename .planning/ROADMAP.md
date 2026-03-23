# Roadmap: SkyGate

## Overview

SkyGate delivers a Raspberry Pi-based bandwidth management appliance for GA aircraft in five phases. Phase 1 builds the network foundation (WiFi AP, DNS blocking, QoS, aviation app bypass) on a read-only root filesystem. Phase 2 delivers the dashboard -- the "whoa" moment that shows pilots what's eating their 20 GB cap. Phases 3-5 layer on the novel split-proxy architecture: WireGuard tunnel, content compression on a remote server, and CA cert distribution with two-tier UX. Each phase delivers a coherent, independently verifiable capability.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Pi Network Foundation** - WiFi AP with DNS blocking, aviation bypass, QoS, and read-only root filesystem
- [ ] **Phase 2: Usage Dashboard** - Per-device usage tracking, captive portal, bandwidth savings display -- the "whoa" moment
- [ ] **Phase 3: Tunnel Infrastructure** - WireGuard tunnel to remote server with policy-based routing
- [ ] **Phase 4: Content Compression Proxy** - Go-based MITM proxy on remote server for image transcoding and JS/CSS minification
- [ ] **Phase 5: Certificate Management** - CA cert distribution, two-tier UX ("Quick Connect" vs "Max Savings"), cert-pinning bypass

## Phase Details

### Phase 1: Pi Network Foundation
**Goal**: Pi boots into a working WiFi access point that blocks ads/trackers at DNS level, routes aviation apps directly to Starlink, and shapes traffic to prevent bufferbloat -- all on a corruption-resistant read-only filesystem
**Depends on**: Nothing (first phase)
**Requirements**: NET-01, NET-02, DNS-01, ROUTE-01, QOS-01
**Success Criteria** (what must be TRUE):
  1. A passenger device can connect to the Pi's WiFi network and browse the internet through Starlink
  2. Ad/tracker domains are blocked at DNS level -- loading a page with ads shows blank ad slots instead of content
  3. ForeFlight and Garmin Pilot connect and sync without interference from DNS blocking or proxy
  4. The Pi survives abrupt power loss (master switch off) without filesystem corruption and boots back to working state
  5. Latency remains stable under load -- no bufferbloat spikes when multiple devices are active
**Plans**: 5 plans

Plans:
- [x] 01-01-PLAN.md -- Project scaffolding: Go module, Ansible skeleton, config files, Makefile, test infrastructure
- [x] 01-02-PLAN.md -- WiFi AP + Pi-hole: hostapd, DHCP/DNS, DNS ad/tracker blocking, aviation whitelisting
- [x] 01-03-PLAN.md -- Aviation bypass: Go bypass daemon, nftables sets, policy routing, Ansible routing role
- [ ] 01-04-PLAN.md -- QoS: CAKE autorate script, BATS tests, Ansible QoS role
- [ ] 01-05-PLAN.md -- Read-only filesystem: OverlayFS, data partition, first-boot setup, final review

### Phase 2: Usage Dashboard
**Goal**: Pilots can see exactly what's eating their Starlink data cap in real time, with dollar-amount savings and plan cap tracking -- the viral screenshot moment
**Depends on**: Phase 1
**Requirements**: DASH-01, DASH-02, DASH-03, DASH-04, DASH-05, DASH-06
**Success Criteria** (what must be TRUE):
  1. A passenger connecting for the first time is intercepted by a captive portal with terms acceptance before internet access is granted
  2. The dashboard shows per-device breakdown of data usage with top domains by bytes consumed and a category pie chart
  3. A real-time bandwidth graph updates live showing current throughput across all connected devices
  4. The dashboard displays a dollar amount of bandwidth saved (e.g., "$12.50 saved this flight") based on Starlink overage rates
  5. User can configure their Starlink plan cap and see usage-against-cap with alerts at 50%, 75%, and 90% thresholds
**Plans**: 4 plans

Plans:
- [ ] 02-01-PLAN.md -- Go daemon foundation: config, SQLite DB layer, nftables counter parser, domain categories
- [ ] 02-02-PLAN.md -- Dashboard frontend: HTMX/Chart.js static assets, HTML pages, mobile-first CSS
- [ ] 02-03-PLAN.md -- API + business logic: Pi-hole client, savings calculator, SSE streaming, REST endpoints, captive portal handler
- [ ] 02-04-PLAN.md -- Deployment: Ansible dashboard role, Caddy config, nftables captive portal rules, systemd service, Makefile updates

### Phase 3: Tunnel Infrastructure
**Goal**: Non-aviation traffic flows through an encrypted WireGuard tunnel to a remote server while aviation apps continue routing directly to Starlink
**Depends on**: Phase 1
**Requirements**: TUN-01, ROUTE-02
**Success Criteria** (what must be TRUE):
  1. WireGuard tunnel establishes automatically on boot and maintains connection through Starlink satellite handoffs
  2. Non-bypass web traffic routes through the tunnel to the remote server while aviation app traffic goes direct to Starlink
  3. If the tunnel drops, traffic falls back to direct routing and the tunnel auto-reconnects without manual intervention
**Plans**: TBD

### Phase 4: Content Compression Proxy
**Goal**: A remote proxy server compresses web content (images, JS, CSS) before it traverses the expensive satellite link, delivering 80-90% additional savings beyond DNS blocking
**Depends on**: Phase 3
**Requirements**: PROXY-01, PROXY-02
**Success Criteria** (what must be TRUE):
  1. Images load visibly smaller (WebP transcoding, quality reduction) when browsing through the proxy compared to direct
  2. JS and CSS files are minified by the proxy, reducing transfer size on the satellite link
  3. The remote proxy server deploys with a single `docker compose up` command including WireGuard server endpoint
**Plans**: TBD

### Phase 5: Certificate Management
**Goal**: Passengers choose their savings level -- "Quick Connect" for zero-friction DNS blocking or "Max Savings" with CA cert install for full proxy compression -- and cert-pinned apps never break
**Depends on**: Phase 2, Phase 4
**Requirements**: CERT-01, CERT-02, CERT-03
**Success Criteria** (what must be TRUE):
  1. Captive portal presents two clear options: "Quick Connect" (DNS only, no setup) and "Max Savings" (install CA cert for proxy compression)
  2. An iOS user can download and install a .mobileconfig profile to trust the proxy's CA certificate; an Android user can follow a guided cert install flow
  3. Banking apps, authentication services, and other cert-pinned apps work normally even with "Max Savings" mode enabled
**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Pi Network Foundation | 3/5 | In Progress | - |
| 2. Usage Dashboard | 0/4 | Planned    |  |
| 3. Tunnel Infrastructure | 0/TBD | Not started | - |
| 4. Content Compression Proxy | 0/TBD | Not started | - |
| 5. Certificate Management | 0/TBD | Not started | - |
