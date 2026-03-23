# Feature Landscape

**Domain:** Bandwidth management appliance for GA aviation (metered Starlink satellite internet)
**Researched:** 2026-03-22
**Confidence:** HIGH (cross-referenced Pi-hole, Firewalla, OpenWRT, Peplink, airline IFC systems, GateSentry, Eero, RaspAP, cake-autorate, compy, GlassWire, Phoenix DPI, and GA pilot forum discussions)

## Table Stakes

Features users expect. Missing = product feels incomplete or untrustworthy.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **WiFi access point** | Passengers must connect to SkyGate, not directly to Starlink. This IS the product. Every router, hotspot, and network appliance provides this. | Low | hostapd + dnsmasq. Well-trodden Pi territory (RaspAP has 5K+ stars doing exactly this). WPA2 minimum. |
| **DNS-level ad/tracker blocking** | Pi-hole has 45K stars and is the de facto standard. Users who know enough to want bandwidth management already know Pi-hole exists. If SkyGate can't do what Pi-hole does, it's DOA. | Low | Pi-hole or AdGuard Home. Blocks video CDNs, ad networks, trackers, update servers at domain level. Works for ALL devices including native apps, zero client config. |
| **Per-device data usage tracking** | The entire product thesis is "show pilots what's eating their data." Without per-device breakdown, it's just a Pi-hole in a box. Every consumer router (Eero, Firewalla, Asus, Netgear) shows per-device usage. OpenWRT has multiple packages for this (nlbwmon, wrtbwmon, YAMon). | Medium | Read iptables counters or nftables per MAC address. Persist to SQLite. Update in near-real-time (~5s intervals). Must survive reboots. |
| **Usage dashboard (web UI)** | GlassWire, Firewalla, Eero app, and every enterprise monitoring tool (Grafana, Datadog) all provide visual usage data. Pilots need a "whoa" moment looking at a pie chart of what ate their 20 GB. The dashboard IS the hook. | Medium | Served via Caddy on the Pi. Mobile-responsive (passengers will view on phones). Key views: top domains by bytes, per-device breakdown, category pie chart, real-time bandwidth graph. |
| **Captive portal** | Standard pattern for guest WiFi everywhere (hotels, airports, airlines). Passengers expect to see a portal when connecting to a new network. Portal establishes trust, provides terms acceptance, and surfaces the dashboard. CoovaChilli, nodogsplash, OpenNDS all provide this. | Medium | Redirect HTTP requests to portal page on first connect. Terms acceptance, link to dashboard, network name/branding. Must work reliably on iOS (Apple's CNA) and Android captive portal detection. |
| **Aviation app bypass** | Safety-critical. ForeFlight, Garmin Pilot, and weather APIs (aviationweather.gov) MUST route directly to Starlink without proxy interference. Pilots will reject any device that degrades their primary flight tools. Every IFC system (Gogo, Viasat) prioritizes operational traffic. | Medium | Policy-based routing via iptables marks + ipset. Maintain bypass domain list in config file. DNS responses for bypass domains populate ipset dynamically. Must include ADS-B services, weather APIs. |
| **Video streaming block** | Netflix, YouTube, TikTok, Disney+ are the single biggest bandwidth hogs (3-7 GB/hr per device). Blocking video CDNs via DNS is the highest-impact single action. Airlines universally throttle or block video on bandwidth-constrained connections. Phoenix DPI explicitly lists this as a core IFC policy. | Low | DNS blocklist for known video CDN domains (googlevideo.com, fbcdn.net/video, tiktokcdn.com, etc.). Friendly "blocked" message via captive portal: "Video streaming paused to save satellite data." |
| **OS/app update blocking** | A single iOS update can burn 5+ GB. Apple's swscan/swdownload domains, Google Play, Windows Update are well-known. Every metered connection guide recommends disabling auto-updates. | Low | DNS blocklist for update domains. Specific, well-documented domain lists available. |
| **Background refresh/cloud sync blocking** | iCloud backup, Google Photos sync, Dropbox, OneDrive all run silently in the background. iPad Pilot News recommends enabling "Low Data Mode" per device manually -- SkyGate should handle this at the network level. | Low | DNS blocklist for cloud sync domains. This is the silent bandwidth killer passengers don't even know about. |
| **Offline/degraded mode** | Starlink drops out. WireGuard tunnel drops. The Pi must keep working. Every enterprise appliance (Peplink, Mushroom Networks) handles failover gracefully. A device that bricks when connectivity hiccups in flight is unacceptable. | Medium | Pi-hole continues DNS blocking if tunnel drops. Dashboard shows "DEGRADED" status. Auto-reconnect via WireGuard keepalive. Direct routing fallback if tunnel is unreachable. |
| **Zero-config basic operation** | Eero's entire brand is "setup in minutes." The GA pilot audience is explicitly non-technical. If basic operation requires SSH, config files, or terminal commands, the market shrinks to hobbyists only. Pre-configured SD card image must boot into working state. | Medium | Pre-flashed SD card with hostapd + Pi-hole + dashboard. Default SSID "SkyGate", default password on a sticker. Basic operation (DNS blocking + dashboard) works out of the box. Advanced features (proxy, custom rules) available via web UI. |

## Differentiators

Features that set SkyGate apart. Not expected by users but deliver outsized value. These are what make SkyGate more than "Pi-hole on a plane."

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Content compression proxy (split architecture)** | The novel core: Pi tunnels traffic to remote server via WireGuard; server fetches content on cheap terrestrial bandwidth, strips/compresses, sends only stripped version over Starlink. A 5 MB Instagram page becomes 200 KB of text and thumbnails. No consumer product does this for aviation. Enterprise PEPs cost $50K+. | High | compy (Go binary, ~20MB RAM) handles image transcoding (JPEG quality reduction, PNG/JPEG to WebP), JS/CSS minification. Runs on remote server. WireGuard tunnel from Pi. Two modes: "Quick Connect" (DNS only) and "Max Savings" (full proxy with CA cert). |
| **Two-tier TLS strategy ("Quick Connect" vs "Max Savings")** | Elegantly solves the HTTPS problem. Quick Connect = zero friction, DNS blocking only. Max Savings = install CA cert for full content stripping in browsers. No other consumer product offers this graduated approach. Airlines handle it invisibly; SkyGate makes it transparent and user-controlled. | Medium | Captive portal presents both options. CA cert download/install flow for iOS (profile install) and Android (certificate install). Certificate pinning bypass list for banking/auth apps. Privacy disclosure in captive portal terms. |
| **Social media text-only mode** | Strips Instagram, Twitter/X, Facebook to text + compressed thumbnails in browser. Original content is 3-5 GB/hr; text-only is 50-100 MB/hr. 95%+ savings. No consumer product offers this. Airlines just throttle; SkyGate surgically removes the bloat while keeping the content readable. | High | BeautifulSoup/HTML rewriting on the proxy server. Removes video/iframe/heavy script tags, recompresses images per compression rules. Only works for browser traffic with CA cert installed. Fragile -- HTML structures change frequently. Needs rule update mechanism. |
| **Real-time bandwidth savings counter** | "SkyGate saved you 4.2 GB this flight ($42 in overages)." Dollar-amount savings visualization directly tied to Starlink's pricing. Firewalla shows usage but doesn't calculate cost savings. This is the viral screenshot -- pilots post this to PoA forums. | Medium | Compare actual bytes transferred vs estimated uncompressed bytes. Multiply savings by $/GB overage rate ($100/GB on Business, estimated on Aviation plans). Display prominently on dashboard. |
| **Dynamic QoS (cake-autorate)** | Starlink bandwidth is variable (weather, obstructions, satellite handoffs). cake-autorate dynamically adjusts CAKE bandwidth ceiling based on real-time latency measurements, preventing bufferbloat. 498 stars, first-class Starlink support. No consumer product integrates this for aviation. | Medium | cake-autorate is a bash script that runs on any Linux. Needs CAKE qdisc support in kernel. Has Starlink-specific satellite switch compensation. OpenWRT-focused but runs on Pi. |
| **Pre-flight data caching** | Cache popular content (weather briefings, charts, commonly visited sites) while on ground WiFi. Serve from cache in flight. Airlines use edge caching (Thales FlytEDGE, Netskrt Edge CDN) to preload popular content. A lightweight version for GA: pre-cache destination weather, NOTAM pages, popular news sites before takeoff. | High | Squid or custom cache layer on the Pi. "Pre-flight mode" triggers cache warming of configured URLs. Deferred -- nice to have but complex cache invalidation logic. |
| **Category-based bandwidth visualization** | Not just "device X used 500 MB" but "video: 40%, social: 25%, web: 20%, updates: 10%, aviation: 5%." Firewalla does category breakdown. GlassWire does per-app categorization. For SkyGate, category mapping via DNS domains to known categories gives pilots actionable intelligence about where data goes. | Medium | Map DNS queries to categories using domain-to-category database (similar to Pi-hole's group management or Firewalla's app identification). Display as pie chart or stacked bar on dashboard. |
| **Per-device bandwidth throttling** | Limit individual devices to X Mbps or X MB total. Firewalla, Peplink, and OpenWRT all offer this. Useful when one passenger is a bandwidth hog. "Passenger in seat 3 is burning through data -- throttle them to 2 Mbps." | Medium | tc (traffic control) + iptables per MAC address. Web UI for pilot to set per-device limits. Could auto-throttle devices exceeding a threshold. |
| **Data budget / cap tracking** | "You've used 8.2 of 20 GB this billing cycle. At current rate, you'll hit your cap in 3 days." Projected usage based on historical consumption. ISPs (Xfinity, AT&T) provide this via their portals. SkyGate should show this against the Starlink plan's actual cap. | Low | User configures plan cap (20 GB) and billing cycle start date in setup wizard. Dashboard tracks cumulative usage across flights. Alerts at 50%, 75%, 90% thresholds. |
| **Flight-aware session tracking** | Group usage by flight (takeoff to landing), not just by day. "Flight KPAO-KLAX: 1.8 GB used, 1.2 GB saved." Each flight becomes a data event with a savings report. No consumer product does flight-aware sessions. | Medium | Detect flight sessions via Starlink connectivity (connected = in flight, or GPS if available). Log per-session usage. Dashboard shows flight history with per-flight savings breakdown. |
| **Hardware kit with 3D printed aviation case** | Physical product differentiator. STL files in repo for self-print. Pre-built kits for non-technical pilots. PETG material, aviation-appropriate mounting (velcro, rubber feet). Status LEDs (power, link, data). The "Eero moment" -- an object you can hold and show people. | Medium | BOM ~$92, sell assembled for $149-179. Pi 5 + pre-flashed SD card + USB-C power adapter (12V aircraft to 5V Pi) + case + LEDs. Designed in Foundry (existing print management infrastructure). |
| **Community-maintained rule updates** | DNS blocklists update weekly via Pi-hole's built-in mechanism. Content stripping rules (BeautifulSoup selectors) update via git-based rule repository. Stale rules (>90 days untested) flagged in dashboard. This is the "living product" -- it gets better as the community contributes. | Medium | Git-based rule repo on GitHub accepting PRs. CI smoke tests for rule validity. Auto-pull on schedule. Self-hosted users: git pull or cron. Hosted service: pushed automatically. |

## Anti-Features

Features to explicitly NOT build. These are tempting but would either scope-creep the project, violate trust, or serve the wrong audience.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Native app content modification** | Certificate-pinned iOS/Android apps (Instagram, banking, etc.) reject MITM proxies and break. Attempting this creates a terrible UX where apps randomly fail. Airlines don't even try this. | DNS blocking only for native apps. Content stripping for browser traffic only (with CA cert). Be honest about the two-layer approach in documentation. |
| **Deep Packet Inspection (DPI) engine** | Phoenix DPI and enterprise solutions use Layer 7 DPI for traffic classification. Massively complex, resource-intensive, and raises serious privacy/legal concerns. A Pi cannot run a real DPI engine. | DNS-level domain categorization provides 80% of the classification value at 1% of the complexity. Use domain-to-category mapping instead of packet inspection. |
| **User tracking / marketing analytics** | Captive portal vendors (Purple, Aislelabs, SplashAccess) collect emails, social logins, and behavioral data for marketing. This violates the trust model. Pilots and passengers are not leads to be harvested. | Captive portal collects NOTHING beyond terms acceptance. No email, no social login, no analytics. Privacy-first. The trust model is the product's armor. |
| **Multi-tenant hosted proxy in v1** | Multi-tenancy requires per-device billing, WireGuard peer management, account isolation, and compliance infrastructure. This is a Phase C concern. Building it early adds massive complexity for zero v1 value. | v1 is single-tenant: one Pi, one server. Clean abstractions (config files, not hardcoded keys) but no multi-tenant architecture. Hosted service is a separate product milestone. |
| **Commercial airline integration** | Different market (enterprise procurement, DO-160 certification, STC requirements), different scale (200+ passengers), different regulatory environment (TSO compliance). Airlines use Phoenix DPI, Hughes, Viasat enterprise stacks. | GA only. 1-8 passengers. PED under AC 91.21-1D. No certification needed. If airlines want SkyGate, they'll reach out -- don't design for them. |
| **Non-aviation verticals in v1** | Boats, RVs, remote workers, expeditions all have metered satellite. The tech transfers. But GA is the wedge market with the clearest pain point and most vocal community. Generalizing the UI, bypass rules, and marketing dilutes the GA focus. | v1 copy, config, and defaults are 100% GA aviation. YAGNI guard. When aviation is validated, fork the rules/config for marine, RV, remote worker verticals. Architecture stays generic; product positioning stays specific. |
| **Parental controls / content filtering** | Firewalla, GateSentry, and home routers all offer parental controls (porn blocking, time limits, social media scheduling). SkyGate is not a parental control device. Adding content categories beyond bandwidth optimization muddies the value proposition. | Block content categories ONLY for bandwidth reasons (video, updates, cloud sync). Not for content modesty reasons. If a domain uses 0 bytes (text-only news site), it passes regardless of content. |
| **VPN for privacy** | Travel routers (RaspAP, GL.iNet) offer VPN tunneling for privacy/security on untrusted WiFi. SkyGate's WireGuard tunnel is for proxy routing, not passenger privacy. Passengers already have their own VPNs if they want them. | WireGuard tunnel serves the proxy architecture only. Do not market as a privacy/security VPN. If passengers run their own VPNs, their traffic passes through the tunnel as encrypted blobs (DNS blocking still applies). |
| **Speed testing / ISP monitoring** | Some router products (Eero, Firewalla) include built-in speed tests and ISP performance monitoring. This is noise for SkyGate -- pilots know Starlink is variable. | Show real-time throughput on the dashboard as an indicator of link health, but do not build a speed test feature. cake-autorate already measures and adapts to link quality. |
| **App Store / distribution platform** | Enterprise IFC systems (Gogo, Viasat) offer curated content portals with entertainment, games, and media. This is a massive content licensing and curation effort. | SkyGate is infrastructure, not a content platform. The captive portal shows the dashboard and savings, not curated entertainment. Passengers use their own apps. |
| **Mesh networking / range extension** | Eero's core value is mesh coverage across a home. Aircraft cabins are small (most GA aircraft: 15-25 feet of cabin). A single Pi WiFi radio covers the entire space with margin. Mesh adds complexity for zero benefit. | Single access point. If signal is weak in a specific aircraft layout, recommend antenna positioning in docs. No mesh. |

## Feature Dependencies

```
WiFi AP (hostapd) ─────────────────────────────────────────────────────┐
    │                                                                   │
    ├── Captive Portal ──── Terms Acceptance                            │
    │       │                                                           │
    │       ├── Usage Dashboard (web UI) ──── Per-device tracking       │
    │       │       │                                                   │
    │       │       ├── Category breakdown (requires domain mapping)    │
    │       │       ├── Savings counter (requires proxy bytes tracking) │
    │       │       └── Data budget tracking (requires plan config)     │
    │       │                                                           │
    │       └── CA Cert Download (for "Max Savings" mode)               │
    │               │                                                   │
    │               └── Content compression proxy ──┐                   │
    │                       │                       │                   │
    │                       ├── Image transcoding   │                   │
    │                       ├── JS/CSS minification │                   │
    │                       └── Social text-only    │                   │
    │                                               │                   │
    ├── DNS Blocking (Pi-hole) ─────────────────────┤                   │
    │       │                                       │                   │
    │       ├── Video CDN blocking                  │                   │
    │       ├── Update server blocking              │                   │
    │       ├── Cloud sync blocking                 │                   │
    │       └── Ad/tracker blocking                 │                   │
    │                                               │                   │
    ├── Aviation App Bypass (policy routing) ────────┤                  │
    │                                               │                   │
    ├── WireGuard Tunnel ───────────────────────────┘                   │
    │       │                                                           │
    │       └── Remote Server (Docker Compose) ──┐                      │
    │               │                            │                      │
    │               ├── compy proxy              │                      │
    │               └── WireGuard server         │                      │
    │                                                                   │
    ├── Dynamic QoS (cake-autorate) ─── independent of tunnel           │
    │                                                                   │
    └── Offline/Degraded Mode ──── fallback routing if tunnel drops     │
                                                                        │
Hardware Kit (case, LEDs, power) ──── independent of software ──────────┘
```

**Critical path for MVP (Phase B.1: Awareness):**
1. WiFi AP (hostapd + dnsmasq)
2. DNS blocking (Pi-hole)
3. Per-device usage tracking (iptables counters + SQLite)
4. Usage dashboard (Caddy + static web UI)
5. Captive portal (redirect + terms)

**Critical path for Phase B.2 (Control):**
6. Aviation app bypass (policy routing)
7. WireGuard tunnel
8. Remote server with compy proxy
9. CA cert distribution via captive portal
10. Content compression rules

**Independent features (can ship at any phase):**
- cake-autorate (no dependencies beyond the Pi and network interface)
- Data budget tracking (dashboard feature, just needs plan config)
- Hardware kit (parallel effort, physical design)

## MVP Recommendation

**Prioritize (Phase B.1 -- shippable v0.1):**
1. WiFi AP + Pi-hole DNS blocking -- immediate 50-60% bandwidth savings, zero complexity for passengers
2. Per-device usage dashboard -- the "whoa" moment, the viral screenshot, the hook
3. Captive portal with terms acceptance -- trust establishment, dashboard access point
4. Video/update/cloud sync DNS blocking -- highest-impact blocklists, passive savings
5. cake-autorate QoS -- prevents Starlink bufferbloat from day one, independent component

**Prioritize (Phase B.2 -- full architecture):**
6. Aviation app bypass routing -- safety-critical, must ship before any proxy work
7. WireGuard tunnel + remote compy proxy -- the novel core architecture
8. "Quick Connect" vs "Max Savings" captive portal modes -- graduated engagement
9. Bandwidth savings counter with dollar amounts -- the viral moment

**Defer:**
- **Social media text-only mode**: High complexity, fragile (HTML selectors break), requires constant rule maintenance. Ship basic image compression first; text-only mode is a v2 feature when the rule community is established.
- **Pre-flight data caching**: Complex cache invalidation, unclear what to cache, low ROI compared to DNS blocking + proxy compression. Defer until post-launch user feedback reveals what pilots actually want cached.
- **Per-device bandwidth throttling**: Nice to have but adds UI complexity. Most GA flights are 1-4 passengers; blanket QoS via cake-autorate handles the common case. Throttling is a power-user feature for later.
- **Flight-aware session tracking**: Requires flight detection logic (GPS or connectivity patterns). Useful for the savings report viral moment but not MVP-critical. Add when basic usage tracking is solid.
- **Hardware kit**: Parallel effort but not blocking. Software ships first (GitHub + SD card image). Physical kit ships when software is validated with real flights.

## Sources

- [Pi-hole documentation](https://docs.pi-hole.net/) - DNS blocking features, blocklist management (HIGH confidence)
- [Firewalla Smart Queue](https://help.firewalla.com/hc/en-us/articles/360056976594-Firewalla-Feature-Smart-Queue) - QoS, bandwidth monitoring, parental controls (HIGH confidence)
- [Firewalla Bandwidth Monitoring](https://help.firewalla.com/hc/en-us/articles/23902791597587) - Per-device usage tracking patterns (HIGH confidence)
- [Eero experience](https://eero.com/experience) - Zero-config setup UX benchmark (HIGH confidence)
- [cake-autorate GitHub](https://github.com/lynxthecat/cake-autorate) - Dynamic Starlink QoS, 498 stars (HIGH confidence)
- [compy GitHub](https://github.com/barnacs/compy) - HTTP/HTTPS compression proxy, image transcoding, MITM (HIGH confidence)
- [RaspAP](https://raspap.com/) - Raspberry Pi AP + captive portal reference implementation (HIGH confidence)
- [Peplink QoS](https://www.rvmobileinternet.com/guides/peplink-qos-user-groups-bandwidth-limits/) - Enterprise bandwidth management patterns (MEDIUM confidence)
- [Phoenix DPI IFC optimization](https://phoenixdpi.com/blog/in-flight-bandwidth-optimization/) - Airline bandwidth management policies (MEDIUM confidence)
- [GateSentry GitHub](https://github.com/fifthsegment/Gatesentry) - Proxy + DNS combo with per-user stats (MEDIUM confidence)
- [OpenWRT bandwidth monitoring (nlbwmon)](https://forum.openwrt.org/t/monitoring-bandwidth-used-per-user-because-metered-internet/2167) - Per-device metered internet monitoring patterns (HIGH confidence)
- [Starlink GA pricing (Flying Magazine)](https://www.flyingmag.com/starlinks-pricing-shift-a-bait-and-switch-for-general-aviation/) - Market context, pricing tiers (HIGH confidence)
- [iPad Pilot News Starlink tips](https://ipadpilotnews.com/2025/03/flying-with-starlink-satellite-internet-tips-for-pilots/) - Pilot data management advice (LOW confidence -- article thin on specifics)
- [GlassWire features](https://www.glasswire.com/features/) - Per-app bandwidth visualization patterns (MEDIUM confidence)
- [Viasat IFC](https://www.viasat.com/aviation/commercial-aviation/in-flight-connectivity/) - Airline bandwidth allocation approach (MEDIUM confidence)
- [Gogo 5G ATG](https://www.gogoair.com/) - Business aviation connectivity features (MEDIUM confidence)
- [AdGuard Home vs Pi-hole comparison](https://github.com/adguardTeam/adGuardHome/wiki/Comparison) - DNS filtering feature comparison (HIGH confidence)
- [BusinessCom Sentinel PEP](https://www.bcsatellite.net/products-services/sentinel-bandwidth-management-and-optimization/) - Satellite PEP compression benchmarks (MEDIUM confidence)
- [Edge caching in aviation (PXCom/Yocova)](https://public.yocova.com/news-insights/edge-caching-architecture-arrives-on-the-connected-aircraft/) - Pre-flight caching patterns (LOW confidence -- airline-scale, not GA)
