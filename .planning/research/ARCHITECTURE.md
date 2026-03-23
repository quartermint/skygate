# Architecture Research

**Domain:** Aviation bandwidth management appliance (Pi-based split proxy)
**Researched:** 2026-03-22
**Confidence:** HIGH (core network patterns well-documented; compy maintenance status verified)

## Standard Architecture

### System Overview

```
                          AIRCRAFT                                          GROUND
 ┌──────────────────────────────────────────────────────────┐     ┌──────────────────────────────────┐
 │  Raspberry Pi 4/5                                         │     │  Remote Server (VPS / Mac Mini)  │
 │                                                           │     │                                  │
 │  ┌──────────┐   ┌──────────┐   ┌────────────────────┐    │     │  ┌──────────┐   ┌────────────┐  │
 │  │ hostapd  │──>│ dnsmasq  │──>│ nftables/iptables  │    │     │  │ WireGuard│──>│ compy/Go   │  │
 │  │ WiFi AP  │   │ DHCP+DNS │   │ Policy Routing     │    │     │  │ Server   │   │ Proxy      │  │
 │  │ wlan0    │   │          │   │ fwmark + ipset     │    │     │  │          │   │ (MITM +    │  │
 │  └──────────┘   └─────┬────┘   └────────┬───────────┘    │     │  └──────────┘   │ compress)  │  │
 │                        │                 │                 │     │                 └──────┬─────┘  │
 │  ┌──────────┐   ┌─────┴────┐   ┌────────┴───────────┐    │     │  ┌──────────┐          │        │
 │  │ Captive  │   │ Pi-hole  │   │ WireGuard Client   │────┼─────┼─>│ WireGuard│          │        │
 │  │ Portal   │   │ (FTL)    │   │ wg0                │    │     │  │ Server   │──────────┘        │
 │  │ (Go/     │   │ DNS sink │   └────────────────────┘    │     │  └──────────┘                   │
 │  │  Caddy)  │   └──────────┘                              │     │                                  │
 │  └──────────┘                                             │     │  ┌──────────┐                    │
 │  ┌──────────┐   ┌──────────┐                              │     │  │ Mgmt API │ (Phase C only)    │
 │  │ Usage    │   │ CAKE     │                              │     │  │ Config   │                    │
 │  │ Monitor  │   │ QoS      │                              │     │  └──────────┘                    │
 │  │ Daemon   │   │ (tc)     │                              │     │                                  │
 │  └──────────┘   └──────────┘                              │     │  ┌──────────┐                    │
 │                                                           │     │  │ Usage DB │                    │
 │  ┌──────────────────────────────────────────────────┐     │     │  │ (SQLite) │                    │
 │  │  eth0 ─────────────── Starlink Mini (WAN)        │     │     │  └──────────┘                    │
 │  └──────────────────────────────────────────────────┘     │     │                                  │
 └──────────────────────────────────────────────────────────┘     └──────────────────────────────────┘
                              |
                              | Starlink satellite link
                              | (20 GB cap, $250/mo)
                              |
                         [ Internet ]
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **hostapd** | WiFi access point — creates "SkyGate" SSID, WPA2 auth, client association | hostapd on wlan0, bridge or routed mode. Config: `/etc/hostapd/hostapd.conf` |
| **dnsmasq** | DHCP server for wlan0 clients + DNS forwarding to Pi-hole | dnsmasq bound to wlan0 subnet (e.g., 10.0.0.0/24). Assigns IPs, sets Pi-hole as DNS |
| **Pi-hole (FTL)** | DNS sinkhole — blocks ad/tracker/video CDN domains. Layer 1 savings | Pi-hole's FTL engine handles DNS. Uses its own embedded dnsmasq fork (FTLDNS). Blocklists updated via `pihole -g` cron |
| **nftables/iptables** | Policy routing — marks traffic for bypass (aviation) vs tunnel (everything else). Also: captive portal redirect for unauthenticated devices | fwmark 0x1 = bypass (direct to Starlink), fwmark 0x2 = tunnel (via WireGuard). ipset populated dynamically from DNS responses |
| **WireGuard (client)** | Encrypted tunnel from Pi to remote server. All non-bypass traffic transits this tunnel | wg0 interface. Keepalive for Starlink NAT traversal. Auto-reconnect built-in |
| **Captive Portal** | Terms acceptance, CA cert download, dashboard link. Intercepts HTTP from unauthenticated clients | Lightweight Go server or Caddy with nftables redirect rules. Releases client MAC after acceptance |
| **Usage Monitor** | Per-device bandwidth tracking — polls nftables/iptables counters, stores time-series, serves WebSocket to dashboard | Go daemon polling every 5s. Reads per-IP byte counters from nftables. Stores in SQLite. Exposes SSE/WebSocket |
| **CAKE QoS** | Prevents Starlink bufferbloat by dynamically adjusting bandwidth ceiling based on latency | `tc qdisc add dev eth0 root cake bandwidth Xmbit`. Custom autorate script (not cake-autorate, which is OpenWrt-only) using fping latency probes |
| **WireGuard (server)** | VPN endpoint on remote server. Accepts tunnel from Pi, routes traffic to proxy | wg0 on remote server. Single peer (Phase B) or multi-peer (Phase C) |
| **compy / Go proxy** | Content-compressing forward proxy with MITM — image transcoding, JS/CSS minification | compy binary listening on localhost. WireGuard traffic forwarded here. CA cert signs on-the-fly certs for HTTPS interception |
| **Usage DB** | Persistent storage for bandwidth stats, per-device usage, content rule effectiveness | SQLite with WAL mode. Lightweight, no separate DB server needed |

## Recommended Project Structure

```
skygate/
├── pi/                         # Everything that runs on the Raspberry Pi
│   ├── setup.sh                # One-command Pi provisioning script
│   ├── config/
│   │   ├── hostapd.conf        # WiFi AP configuration
│   │   ├── dnsmasq.conf        # DHCP + DNS forwarding
│   │   ├── pihole/             # Pi-hole configuration overlays
│   │   ├── wireguard/
│   │   │   └── wg0.conf        # WireGuard client config (templated)
│   │   ├── nftables.conf       # Policy routing rules + captive portal redirect
│   │   ├── bypass-domains.txt  # Aviation app domains (ForeFlight, Garmin, wx)
│   │   ├── pinned-apps.txt     # SNI bypass list (cert-pinned apps)
│   │   └── cake-qos.sh         # CAKE qdisc setup + autorate parameters
│   ├── portal/                 # Captive portal + usage dashboard
│   │   ├── main.go             # Go HTTP server (portal + dashboard + API)
│   │   ├── handlers/           # HTTP handlers (terms, cert download, dashboard)
│   │   ├── monitor/            # Usage monitoring daemon
│   │   │   ├── collector.go    # nftables counter polling
│   │   │   ├── store.go        # SQLite time-series storage
│   │   │   └── ws.go           # WebSocket/SSE for real-time dashboard
│   │   ├── templates/          # HTML templates (dashboard, terms, cert install)
│   │   ├── static/             # CSS, JS, favicon
│   │   └── ca/                 # CA cert generation + distribution
│   ├── autorate/               # Custom CAKE autorate for Raspberry Pi OS
│   │   └── autorate.sh         # Bash: fping + tc qdisc change loop
│   └── systemd/                # Service unit files
│       ├── skygate-portal.service
│       ├── skygate-monitor.service
│       └── skygate-autorate.service
├── server/                     # Everything that runs on the remote server
│   ├── docker-compose.yml      # One-command server deployment
│   ├── Dockerfile              # compy/Go proxy + WireGuard
│   ├── wireguard/
│   │   └── wg0.conf            # WireGuard server config (templated)
│   ├── proxy/                  # Content compression proxy
│   │   ├── main.go             # Custom Go proxy (or compy wrapper)
│   │   ├── rules/              # Content stripping rules
│   │   │   ├── rules.json      # Video block, image compress, social text-only
│   │   │   └── loader.go       # Rule parser + hot-reload
│   │   ├── transcoder/         # Image transcoding (JPEG quality, WebP, resize)
│   │   │   └── image.go        # Uses Go image libs (disintegration/imaging)
│   │   ├── minifier/           # HTML/CSS/JS minification
│   │   │   └── minify.go       # Uses tdewolff/minify
│   │   └── mitm/               # MITM certificate management
│   │       └── ca.go           # On-the-fly cert generation
│   └── usage/                  # Usage logging
│       └── logger.go           # Logs domain, bytes, device to SQLite
├── hardware/                   # Physical hardware files
│   ├── case/                   # 3D printable case (STL, STEP)
│   ├── bom.md                  # Bill of materials
│   └── wiring.md               # Power + LED wiring guide
├── image/                      # SD card image builder
│   ├── build-image.sh          # Creates pre-configured Pi image
│   └── firstboot.sh            # First-boot setup wizard
├── rules/                      # Shared content rules repository
│   ├── blocklists/             # Custom Pi-hole blocklists (video CDNs, updates)
│   ├── bypass-domains.txt      # Aviation app bypass domains (canonical)
│   ├── pinned-apps.txt         # Certificate-pinned app SNI list
│   └── content-rules.json      # Proxy content stripping rules
├── docs/                       # Documentation
│   ├── architecture.md         # This document, adapted
│   ├── setup-guide.md          # End-user setup instructions
│   └── privacy.md              # Privacy disclosure + trust model
├── Makefile                    # Top-level build/deploy commands
└── README.md                   # Project overview, screenshots, quick start
```

### Structure Rationale

- **`pi/` vs `server/`:** Clean separation mirrors the physical deployment — what runs on-aircraft vs what runs on the ground. A developer can work on either without the other.
- **`pi/portal/` as Go:** A single Go binary handles captive portal, usage dashboard, and monitoring daemon. Go compiles to a static binary, uses minimal RAM (~10-20MB), and avoids Python/Node.js dependency bloat on the Pi. Caddy is an alternative but adds another binary.
- **`rules/` at top level:** Content rules are shared between Pi (DNS blocklists) and server (proxy stripping rules). Keeping them top-level makes them easy to update independently and pull via git.
- **`image/` separate from `pi/`:** The SD card image builder is a build tool, not runtime software. It consumes `pi/` artifacts to produce a flashable image.

## Architectural Patterns

### Pattern 1: Policy-Based Routing with DNS-Driven ipset

**What:** Instead of static IP-based routing rules, use DNS query responses to dynamically populate nftables/ipset sets. When Pi-hole resolves `foreflight.com`, the resolved IPs are added to a `bypass` ipset. nftables marks packets destined for bypass IPs with fwmark 0x1 (direct to Starlink), everything else gets fwmark 0x2 (WireGuard tunnel).

**When to use:** Whenever you need domain-based (not IP-based) routing decisions on a Linux gateway. IPs change; domain names are stable.

**Trade-offs:**
- Pro: Automatically adapts as CDN IPs change. No manual IP list maintenance.
- Pro: dnsmasq/Pi-hole already resolves every query -- piggyback on existing DNS path.
- Con: First connection to a new IP requires a DNS lookup to have occurred. Race condition is rare but possible.
- Con: ipset entries need TTL management (expire stale entries).

**Implementation:**

```bash
# Create ipset for aviation bypass
ipset create aviation_bypass hash:ip timeout 3600

# In dnsmasq/Pi-hole custom config, add ipset directives:
# ipset=/foreflight.com/garminpilot.com/aviationweather.gov/aviation_bypass

# nftables rules (or iptables equivalent)
nft add table inet skygate
nft add chain inet skygate prerouting { type filter hook prerouting priority mangle \; }
nft add set inet skygate bypass_ips { type ipv4_addr \; flags timeout \; timeout 1h \; }
nft add rule inet skygate prerouting ip daddr @bypass_ips meta mark set 0x1
nft add rule inet skygate prerouting meta mark != 0x1 meta mark set 0x2

# Policy routing
ip rule add fwmark 0x1 table 100  # Table 100: default via eth0 (Starlink direct)
ip rule add fwmark 0x2 table 200  # Table 200: default via wg0 (WireGuard tunnel)
ip route add default dev eth0 table 100
ip route add default dev wg0 table 200
```

### Pattern 2: Captive Portal with MAC-Based Authentication Release

**What:** Unauthenticated devices get HTTP traffic redirected to the captive portal page. After terms acceptance, the device's MAC address is added to a whitelist. Whitelisted MACs bypass portal redirect and get normal routing (through Pi-hole + policy routing).

**When to use:** Any WiFi hotspot that requires terms acceptance before granting internet access. Standard pattern for hotel/airport/coffee shop WiFi.

**Trade-offs:**
- Pro: Works with all devices (iOS, Android, laptops) -- they all detect captive portals via probe URLs.
- Pro: MAC-based release survives TCP connection drops. Device doesn't re-auth until MAC is removed.
- Con: MAC addresses can be spoofed. Not security-critical here (it's a terms page, not a paywall).
- Con: iOS 14+ randomizes MAC addresses per network. The randomized MAC is consistent for a given SSID, so it works -- but the same physical device appears as different MACs on different SSIDs.

**Implementation approach:**

```
[New client connects to wlan0]
       |
       v
[dnsmasq assigns IP via DHCP]
       |
       v
[Client tries HTTP request]
       |
       v
[nftables checks: is src MAC in whitelist?]
       |
    NO ──────────────────────> [REDIRECT to portal (port 8080)]
       |                              |
       |                       [Terms page + optional CA cert]
       |                              |
       |                       [User accepts terms]
       |                              |
       |                       [Portal adds MAC to nftables set]
       |                              |
    YES <─────────────────────────────┘
       |
       v
[Normal routing: Pi-hole DNS + policy routing]
```

### Pattern 3: Two-Layer TLS Strategy (DNS Block + Optional MITM Proxy)

**What:** Layer 1 (DNS blocking) operates on ALL traffic from ALL devices with zero friction -- no cert install, no configuration. Layer 2 (MITM proxy) operates only on traffic from devices whose users chose to install the CA certificate. This creates two user experiences: "Quick Connect" (Layer 1 only, ~50-60% savings) and "Max Savings" (Layer 1 + 2, ~80-90% savings).

**When to use:** Any bandwidth optimization system dealing with HTTPS traffic. You cannot modify encrypted content without MITM. The two-layer approach acknowledges that not all users will install a CA cert.

**Trade-offs:**
- Pro: Layer 1 alone provides massive value (blocks video CDNs, ads, updates, trackers). Zero friction.
- Pro: Layer 2 is opt-in. Users who don't install the cert still get DNS blocking benefits.
- Pro: Cert-pinned apps (banking, Apple services) are unaffected -- they're on the SNI bypass list.
- Con: Layer 2 only works for browser traffic. Native apps with cert pinning reject the MITM cert.
- Con: Privacy perception risk. "Installing a CA cert" sounds scary. Must be communicated carefully.

### Pattern 4: Graceful Degradation Chain

**What:** The system is designed to lose capabilities progressively, never catastrophically. If the remote server goes down, the Pi still provides DNS blocking + QoS. If Pi-hole goes down, hostapd still provides WiFi + QoS. If CAKE goes down, WiFi still works.

**When to use:** Any appliance where reliability matters more than feature completeness. Aircraft connectivity has no tech support hotline.

**Degradation chain:**

```
Full System         = WiFi + DNS Block + QoS + WireGuard Tunnel + MITM Proxy
                      (80-90% savings)
                           |
                     [Remote server unreachable]
                           v
Degraded (L1 only)  = WiFi + DNS Block + QoS + Direct routing
                      (50-60% savings, dashboard shows DEGRADED)
                           |
                     [Pi-hole crashes]
                           v
Minimal             = WiFi + QoS + Direct routing
                      (bufferbloat prevention only)
                           |
                     [CAKE/autorate crashes]
                           v
Passthrough         = WiFi only (Pi as dumb AP)
                      (no savings, but internet still works)
```

## Data Flow

### Primary Request Flow (Full System)

```
[Passenger Device]
    |
    | WiFi (wlan0, WPA2)
    v
[hostapd] ──> [dnsmasq DHCP]
    |
    | DNS query
    v
[Pi-hole (FTLDNS)]
    |
    ├── Domain in blocklist? ──> [Block: return 0.0.0.0] (Layer 1 savings)
    |
    ├── Domain in bypass list? ──> [Resolve + add IP to bypass ipset]
    |                                    |
    |                                    v
    |                              [fwmark 0x1 ──> eth0 ──> Starlink direct]
    |
    └── All other domains ──> [Resolve normally]
                                    |
                                    v
                              [fwmark 0x2 ──> wg0 ──> WireGuard tunnel]
                                    |
                                    v
                              [Remote Server: WireGuard endpoint]
                                    |
                                    v
                              [compy / Go proxy]
                                    |
                                    ├── Device has CA cert?
                                    |       YES ──> [MITM: intercept HTTPS]
                                    |                    |
                                    |                    ├── Image? ──> [Recompress: JPEG q30, max 800px, WebP]
                                    |                    ├── JS/CSS? ──> [Minify]
                                    |                    ├── HTML? ──> [Strip tags, remove video embeds]
                                    |                    └── Video? ──> [Block with placeholder]
                                    |
                                    |       NO ──> [Forward as-is through tunnel]
                                    |
                                    v
                              [Response back through WireGuard]
                                    |
                                    v
                              [Passenger receives optimized page]
```

### Usage Monitoring Data Flow

```
[nftables per-IP byte counters]
    |
    | Poll every 5 seconds
    v
[Usage Monitor Daemon (Go)]
    |
    ├── Calculate delta bytes per device since last poll
    ├── Resolve IP ──> MAC ──> hostname (from dnsmasq leases)
    ├── Categorize by domain (from Pi-hole query log)
    |
    ├── Store in SQLite ──> [time-series: device, bytes_in, bytes_out, timestamp]
    |
    └── Push via WebSocket/SSE ──> [Dashboard in browser]
                                       |
                                       ├── Per-device pie chart (who's using data)
                                       ├── Top domains by bytes
                                       ├── Category breakdown (video/social/web/updates)
                                       ├── Real-time bandwidth graph
                                       └── Total data used vs 20 GB cap
```

### Key Data Flows

1. **DNS query flow:** Device -> dnsmasq -> Pi-hole FTL -> (blocked | bypass ipset | normal resolve). This is the critical filtering layer. Every single network request starts here. Pi-hole's query log is also the source of domain-level usage attribution.

2. **Traffic routing flow:** After DNS resolution, nftables inspects the destination IP. Bypass ipset match -> fwmark 0x1 -> route table 100 -> eth0 -> Starlink direct. No match -> fwmark 0x2 -> route table 200 -> wg0 -> WireGuard tunnel -> remote proxy. This is the core split architecture decision point.

3. **Usage monitoring flow:** nftables maintains per-IP byte counters on the FORWARD chain. The Go daemon polls these counters, computes deltas, joins with dnsmasq lease data (IP -> MAC -> hostname) and Pi-hole query log (IP -> domains queried), then writes to SQLite and pushes real-time updates to the dashboard via WebSocket/SSE.

4. **Captive portal flow:** New device connects -> dnsmasq assigns IP -> device tries captive portal probe -> nftables redirects HTTP to portal -> user accepts terms -> portal adds MAC to whitelist set -> subsequent traffic routes normally.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1 Pi, 1 server (Phase B) | Single WireGuard peer. compy runs as single process. SQLite for everything. This handles 4-8 simultaneous devices comfortably. |
| 1 Pi, hosted service (Phase C) | Pi registers via API. Server manages multiple WireGuard peers (one per Pi/customer). Need per-peer traffic accounting. Proxy is shared (stateless -- each request is independent). SQLite -> PostgreSQL for multi-tenant usage data. |
| Many Pis, hosted service (Phase C+) | WireGuard peer management becomes the bottleneck. Consider WireGuard interface per customer or userspace WireGuard (wireguard-go). Proxy scales horizontally (stateless). Need billing integration (Stripe). |

### Scaling Priorities

1. **First bottleneck: WireGuard peer management.** A single WireGuard interface supports ~thousands of peers, but config reloads disrupt existing connections. For hosted service, use `wg set` for dynamic peer add/remove without restart. Or run one WireGuard interface per customer pod.

2. **Second bottleneck: Proxy CPU on image transcoding.** Image recompression is CPU-intensive. For hosted service with many concurrent users, run multiple proxy instances behind a load balancer. Each request is stateless -- horizontal scaling is straightforward.

3. **Not a bottleneck: Pi resources.** The Pi handles DNS + routing + monitoring + dashboard for 4-8 devices. A Pi 4 (4 cores, 2-4GB RAM) has massive headroom for this workload. Pi-hole uses ~50MB RAM. WireGuard is kernel-mode. The Go portal binary uses ~10-20MB. Total: <200MB of 2GB+ available.

## Anti-Patterns

### Anti-Pattern 1: Running the Proxy on the Pi

**What people do:** Put the compression proxy (compy/mitmproxy) on the Pi itself to avoid needing a remote server.

**Why it's wrong:** The entire point of the split architecture is that the proxy fetches the FULL content on cheap terrestrial bandwidth (remote server -> internet), then sends only the COMPRESSED version over the expensive satellite link (remote server -> WireGuard -> Pi -> device). If the proxy runs on the Pi, the full uncompressed content traverses the Starlink link first, defeating the purpose. You save nothing on satellite bandwidth -- you only save on the last hop (Pi -> device WiFi), which is free.

**Do this instead:** Always run the content proxy on the remote server. The Pi's job is DNS blocking, routing, QoS, and the dashboard. The server's job is content compression.

### Anti-Pattern 2: Using mitmproxy in Python on the Pi

**What people do:** Install mitmproxy (Python, ~200-500MB RAM) on the Pi for local HTTPS inspection.

**Why it's wrong:** Beyond the bandwidth argument above, mitmproxy's Python runtime consumes 200-500MB RAM on a device with 2-4GB total. It also has significant startup time and CPU overhead for TLS operations. The Pi should run lean services (Go binaries, shell scripts, C daemons).

**Do this instead:** Use compy (Go, ~20MB RAM) on the remote server. If compy's content modification isn't flexible enough, build a custom Go proxy using elazarl/goproxy (MITM library) + tdewolff/minify (HTML/CSS/JS) + disintegration/imaging (image transcoding). Keep it in Go for Pi-friendly resource usage if you ever want to support a local-only mode.

### Anti-Pattern 3: Relying on cake-autorate Directly

**What people do:** Try to install cake-autorate (the OpenWrt/Asus Merlin script) on Raspberry Pi OS.

**Why it's wrong:** cake-autorate is designed for OpenWrt's busybox environment with specific dependencies (entware, jsonfilter). It does not support standard Linux distributions. Attempting to run it on Raspberry Pi OS will fail or require extensive patching.

**Do this instead:** Use the CAKE qdisc directly (`tc qdisc add dev eth0 root cake bandwidth Xmbit`). CAKE is in the Linux kernel since 4.19 and works on any distro including Raspberry Pi OS. Write a minimal custom autorate script (~50 lines of bash) that: (1) pings a reflector with fping, (2) measures RTT, (3) adjusts CAKE bandwidth ceiling with `tc qdisc change` when latency spikes. The logic is simple -- cake-autorate's complexity comes from OpenWrt portability, not the algorithm.

### Anti-Pattern 4: Bridged Mode Instead of Routed Mode

**What people do:** Bridge wlan0 and eth0 so all devices are on the same L2 network as Starlink.

**Why it's wrong:** Bridging prevents the Pi from performing any per-device traffic control, policy routing, or captive portal redirect. The Pi becomes invisible at the network layer. You need the Pi to be a router (L3) to inspect and mark packets.

**Do this instead:** Use routed mode. wlan0 has its own subnet (e.g., 10.0.0.0/24). eth0 connects to Starlink. Pi performs NAT (masquerade) from wlan0 to eth0. This gives full control over traffic flowing between the two interfaces.

### Anti-Pattern 5: Transparent Proxy via iptables REDIRECT

**What people do:** Use `iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-port 8080` to transparently redirect all HTTPS traffic to a local MITM proxy.

**Why it's wrong:** Transparent HTTPS proxying via REDIRECT modifies packets, breaks connection tracking, and causes captive portal detection issues on iOS/Android (devices think they're offline). TPROXY is the correct kernel mechanism for transparent proxying, but it adds complexity. More importantly, for SkyGate the proxy is REMOTE (on the server), not local -- you can't REDIRECT to a remote host.

**Do this instead:** Route traffic through the WireGuard tunnel to the remote server. On the remote server, compy runs as a forward proxy. The WireGuard tunnel effectively IS the transparent proxy mechanism -- all tunneled traffic hits the proxy. No iptables redirect needed on the Pi side.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| **Starlink Mini** | Pi's eth0 connects to Starlink's LAN port via Ethernet. Pi gets DHCP address from Starlink router. | Starlink doesn't support bridge mode -- it always NATs. Pi is double-NATted (Starlink NAT + Pi NAT). WireGuard's UDP handles NAT traversal fine. |
| **Pi-hole blocklists** | Pi-hole pulls community blocklists via `pihole -g` (cron, weekly). Additional SkyGate-specific lists (video CDNs, update servers) added as custom lists. | Steven Black's unified list + OISD list cover ads/trackers. SkyGate adds aviation-specific video/update blocking lists. |
| **Content rules repo** | Pi pulls `rules.json` from GitHub on schedule (or manual `git pull`). Server hot-reloads rules without restart. | Rules include selectors, domain patterns, compression targets. Each rule has `tested_date`. Stale rules (>90 days) flagged in dashboard. |
| **WireGuard NAT traversal** | WireGuard persistent keepalive (every 25s) maintains NAT mapping through Starlink's CGNAT. | Critical for Starlink -- without keepalive, the tunnel drops after ~2 minutes of inactivity due to NAT timeout. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| **Portal <-> Monitor** | In-process (same Go binary) or Unix socket | The usage monitor feeds real-time data to the dashboard. Keeping them in the same process avoids IPC overhead. |
| **Pi <-> Remote Server** | WireGuard tunnel (encrypted UDP) | All data plane traffic. Control plane (health checks) also through tunnel. Single point of connectivity. |
| **Pi-hole <-> nftables** | DNS response -> ipset population (dnsmasq `ipset=` directive) | Pi-hole's underlying dnsmasq adds resolved IPs to ipsets based on domain matching. This populates the bypass routing set. |
| **Monitor <-> nftables** | Monitor reads nftables counters via `nft list counters` (JSON output) | One-directional: monitor reads, never writes to nftables rules. Polling, not event-driven. |
| **Portal <-> nftables** | Portal writes (add/remove MAC from whitelist set) on auth events | Portal calls `nft add element inet skygate whitelist { XX:XX:XX:XX:XX:XX }` when user accepts terms. |
| **Autorate <-> CAKE** | Autorate script calls `tc qdisc change dev eth0 root cake bandwidth Xmbit` | Bash script adjusts bandwidth ceiling. Runs as separate systemd service. Independent of all other components. |

## Critical Architecture Decisions for Build Order

### Decision 1: Pi-hole's dnsmasq vs standalone dnsmasq

Pi-hole ships its own fork of dnsmasq called FTLDNS. When you install Pi-hole, it replaces the system dnsmasq. This means you CANNOT run a separate dnsmasq for DHCP alongside Pi-hole's FTLDNS -- they'll conflict on port 53.

**Resolution:** Use Pi-hole's built-in DHCP server (it has one). Or configure Pi-hole's FTLDNS to handle DHCP for wlan0. This eliminates the need for a separate dnsmasq process. hostapd still handles WiFi AP independently.

### Decision 2: compy Maintenance Risk

compy (barnacs/compy) has not been actively maintained. Open issues dating back years without resolution. The Go dependencies are likely outdated. The core functionality (MITM + image transcoding + minification) works but expect to fork and maintain.

**Resolution:** Start with compy as-is for Phase B.2 proof-of-concept. Plan to build a custom Go proxy using well-maintained libraries:
- **elazarl/goproxy** (6.2k stars, actively maintained) for MITM proxy foundation
- **tdewolff/minify** (3.6k stars, actively maintained) for HTML/CSS/JS minification
- **disintegration/imaging** (5.1k stars, actively maintained) for image resizing/recompression
- **chai2010/webp** for WebP encoding

This gives SkyGate a maintained, modular proxy without depending on a stale upstream.

### Decision 3: nftables vs iptables

Raspberry Pi OS (Bookworm, based on Debian 12) ships with nftables as the default firewall backend. iptables commands still work via the `iptables-nft` compatibility layer, but new projects should use nftables natively.

**Resolution:** Use nftables for all firewall rules. nftables has better performance, native set/map support (for ipsets), and JSON output (for the monitoring daemon to parse). The `ipset` utility is replaced by nftables native sets.

### Suggested Build Order (Dependencies)

```
Phase B.1 (Awareness) -- no remote server needed
──────────────────────────────────────────────────
1. hostapd + dnsmasq (WiFi AP)
   └── No dependencies. Foundation layer.

2. Pi-hole installation
   └── Depends on: hostapd (needs working network)
   └── Replaces standalone dnsmasq for DNS

3. nftables policy routing skeleton
   └── Depends on: Pi-hole (ipset population from DNS)
   └── Initially just: bypass ipset + default mark

4. Usage monitoring daemon (Go)
   └── Depends on: nftables (reads counters)
   └── Depends on: Pi-hole query log (domain attribution)

5. Captive portal + dashboard (Go)
   └── Depends on: Usage monitor (real-time data feed)
   └── Depends on: nftables (MAC whitelist management)

6. CAKE QoS + autorate script
   └── No dependencies on other components
   └── Can be developed/tested in parallel with 1-5

Phase B.2 (Control) -- requires remote server
──────────────────────────────────────────────────
7. WireGuard tunnel (Pi client + server)
   └── Depends on: nftables routing (tunnel vs bypass marks)

8. compy / Go proxy on remote server
   └── Depends on: WireGuard (traffic reaches proxy via tunnel)

9. CA cert generation + distribution via portal
   └── Depends on: Captive portal (cert download page)
   └── Depends on: Proxy (cert must match proxy's CA)

10. Content stripping rules
    └── Depends on: Proxy (rules loaded by proxy)
    └── Iterative: add rules, test, measure savings

Phase C (Product Polish)
──────────────────────────────────────────────────
11. SD card image builder
12. 3D printed case design
13. Setup wizard (web-based)
14. Hosted proxy service (multi-tenant)
15. Status LEDs + hardware integration
```

## Sources

- [Raspberry Pi WiFi AP Guide](https://raspberrypi-guide.github.io/networking/create-wireless-access-point) -- hostapd + dnsmasq routed AP setup
- [Domain-Based Split Tunneling with WireGuard](https://starsandmanifolds.xyz/blog/domain-based-split-tunneling-using-wireguard) -- nftables + dnsmasq ipset + fwmark + policy routing (HIGH confidence, detailed implementation)
- [WireGuard Split Tunneling on Ubuntu](https://oneuptime.com/blog/post/2026-03-02-configure-split-tunneling-wireguard-ubuntu/view) -- policy routing with fwmark
- [WireGuard Routing & Network Namespaces](https://www.wireguard.com/netns/) -- official WireGuard routing documentation
- [compy GitHub (barnacs/compy)](https://github.com/barnacs/compy) -- HTTP/HTTPS compression proxy with MITM, image transcoding, minification (LOW confidence on maintenance status)
- [cake-autorate GitHub](https://github.com/lynxthecat/cake-autorate) -- OpenWrt/Asus Merlin only, NOT compatible with Raspberry Pi OS
- [CAKE qdisc man page](https://man7.org/linux/man-pages/man8/tc-cake.8.html) -- CAKE available in kernel 4.19+, works on any Linux (HIGH confidence)
- [CAKE qdisc IPv4 bandwidth management](https://oneuptime.com/blog/post/2026-03-20-cake-qdisc-ipv4-bandwidth-management/view) -- tc commands for CAKE on Debian
- [Pi-hole + hostapd coexistence](https://discourse.pi-hole.net/t/raspberry-pi-as-access-point-along-with-pihole/1435) -- Pi-hole's FTLDNS replaces system dnsmasq
- [hostapd and Pi-hole integration guide](https://amedeos.github.io/hostapd/2020/05/21/hostapd-and-pihole-a-perfect-union.html) -- practical setup
- [nftables counters wiki](https://wiki.nftables.org/wiki-nftables/index.php/Counters) -- per-element counters in nftables sets
- [Bandwidth monitoring with iptables](https://www.linux.com/training-tutorials/bandwidth-monitoring-iptables/) -- per-IP byte counters polling pattern
- [iptables traffic accounting](https://catonmat.net/traffic-accounting-with-iptables) -- FORWARD chain byte counters for gateway monitoring
- [RaspAP Captive Portal (Nodogsplash)](https://docs.raspap.com/features-insiders/captive/) -- lightweight captive portal for embedded Linux
- [openNDS documentation](https://opennds.readthedocs.io/en/stable/faq.html) -- dynamic captive portal with FAS
- [Transparent proxying with mitmproxy](https://docs.mitmproxy.org/stable/howto/transparent/) -- iptables REDIRECT + TPROXY patterns
- [elazarl/goproxy](https://github.com/elazarl/goproxy) -- actively maintained Go MITM proxy library
- [Google nftables Go library](https://github.com/google/nftables) -- Go bindings for nftables (useful for monitor daemon)
- [Captive portal iptables template](https://github.com/stemid/captiveportal/blob/master/docs/examples/iptables/iptables.j2) -- MAC-based authentication release pattern

---
*Architecture research for: SkyGate aviation bandwidth management appliance*
*Researched: 2026-03-22*
