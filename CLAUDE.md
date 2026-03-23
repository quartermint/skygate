<!-- GSD:project-start source:PROJECT.md -->
## Project

**SkyGate**

An open source bandwidth management appliance for general aviation aircraft using Starlink satellite internet. A Raspberry Pi-based device that sits between passengers and the Starlink Mini, combining DNS-level blocking, content-compressing proxy, and dynamic QoS to reduce data usage by 60-90% on metered aviation plans. Three distribution layers: fully open source (GitHub), hosted proxy service, and pre-built hardware kits with 3D printed case. The Eero of GA internet — "unbox and fly."

**Core Value:** **Show pilots what's eating their data, then give them the controls to stop it.** The usage dashboard is the hook — pilots have zero visibility into their 20 GB cap. Awareness creates demand for controls.

### Constraints

- **HTTPS encryption**: ~95% of web traffic encrypted. DNS blocking works for all traffic; content stripping requires MITM proxy with CA cert (browser-only). Two-layer approach: Layer 1 (DNS, all devices) + Layer 2 (proxy, browsers with CA cert).
- **Starlink hardware**: Mini is a PED (11.5x10", 2.5 lbs, 20-40W, 12V/24V). Pi must coexist.
- **FAA compliance**: PED under AC 91.21-1D. No interference with avionics. Low power, no external antennas.
- **Aircraft environment**: Vibration, temp variation, limited power, weight/balance sensitivity.
- **Latency budget**: Starlink (~40-60ms) + WireGuard (~5-10ms) + proxy processing (~10-200ms) = ~100-300ms total. Image recompression: inline with 500ms timeout, cache for subsequent loads.
- **UX bar**: "Just works like an Eero." Zero-config basic operation. Non-technical GA pilots are the primary audience.
- **Pi resources**: Min 2GB RAM. compy (Go, ~20MB) preferred over mitmproxy (Python, ~200-500MB) for proxy engine.
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->
## Technology Stack

## Recommended Stack
### On-Aircraft (Raspberry Pi)
| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Raspberry Pi 5 (4GB) | Pi 5 | Hardware platform | 2x CPU vs Pi 4, same CYW43455 WiFi chip, DDR50 SDIO for 3x faster WiFi near-range. 4GB sufficient for all services. $60. |
| Raspberry Pi OS Lite (Bookworm) | Debian 12, Kernel 6.12, ARM64 | Base OS | Official headless image. ARM64 for Go binary compatibility. NetworkManager is now default (replaces dhcpcd). Trixie (Debian 13) available but Bookworm is battle-tested. |
| hostapd | 2.10+ (Bookworm repo) | WiFi access point daemon | Industry standard for Linux AP mode. Supports WPA2-PSK, 802.11ac, channel selection. Use with external USB WiFi adapter for AP; onboard WiFi for uplink to Starlink. |
| dnsmasq | 2.89+ (Bookworm repo) | DHCP + DNS forwarder | Lightweight, single-binary. Built-in ipset integration for DNS-based routing (`ipset=/domain.com/BYPASS_SET`). Pi-hole uses dnsmasq internally but we need direct control for ipset rules. |
| Pi-hole | v6.3+ (Core v6.3, FTL v6.4, Web v6.4) | DNS-level ad/tracker/CDN blocking | 45.1k stars, mature, community-maintained blocklists. Operates as Layer 1 (all devices, zero friction). Installed via `curl -sSL https://install.pi-hole.net \| bash`. |
| WireGuard | Kernel module (built into Linux 5.6+) | VPN tunnel to remote proxy | In-kernel since Linux 5.6, zero userspace overhead. ~4000 LOC, ARM-optimized crypto. Pi 5 kernel 6.12 has native WireGuard. `wg-quick` for config management. |
| CAKE qdisc | `tc-cake` (iproute2, Bookworm repo) | Traffic shaping / QoS | Built into modern Linux kernels. Prevents Starlink bufferbloat via `tc qdisc add dev wlan0 root cake bandwidth Xmbit`. Better than fq_codel for variable-rate links. |
| nftables | 1.0.6+ (Bookworm default) | Firewall + packet marking | Default in Debian 12+, replaces iptables. Use for policy-based routing marks: aviation bypass (mark 0x1 -> direct) vs tunnel (mark 0x2 -> WireGuard). iptables-nft compatibility layer available. |
| ipset | 7.x (Bookworm repo) | Dynamic IP sets for routing decisions | Used with dnsmasq to dynamically populate bypass sets from DNS responses. `ipset create aviation_bypass hash:ip` populated by dnsmasq ipset directives. |
| Go (custom daemon) | 1.22+ | Usage monitoring daemon | Single binary, ~5MB compiled. Reads `/sys/class/net/*/statistics/`, parses nftables counters per-device, exposes SSE endpoint for real-time dashboard. Go chosen over Python for memory footprint (20MB vs 100MB+). |
| Caddy | 2.9+ | Captive portal + usage dashboard | Single Go binary, automatic TLS (not needed here but useful for HTTPS dashboard), built-in reverse proxy. Serves static dashboard files and proxies API to Go daemon. ARM64 binary available. |
| HTMX | 2.0+ | Dashboard interactivity | 14KB gzipped, zero build step. SSE extension for real-time bandwidth graphs. No Node.js, no bundler, no npm on the Pi. Perfect for embedded dashboards. |
### Remote Server (Content Stripping)
| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Docker Compose | v2.x | One-command deployment | `docker compose up -d` deploys WireGuard + proxy + SQLite. Standard for self-hosted appliance servers. |
| WireGuard (server) | Kernel module | VPN endpoint | Same WireGuard, server side. Use `linuxserver/wireguard` Docker image or `wg-easy` for web UI peer management. |
| Custom Go proxy | Built on goproxy | HTTP/HTTPS compression proxy with MITM | Fork/extend compy's approach using `elazarl/goproxy` (v1.8+) as the proxy foundation. Built-in MITM via dynamic TLS cert generation. Add image transcoding, JS/CSS minification as response handlers. See "Why Not compy Directly" below. |
| libwebp / go-webp | kolesa-team/go-webp | Image transcoding JPEG/PNG -> WebP | CGo bindings to libwebp. Encode JPEG at quality 30, resize to max 800px width. Inline with 500ms timeout. |
| tdewolff/minify | v2.x | HTML/CSS/JS minification | Pure Go, no CGo. Minifies HTML, CSS, JS, JSON, SVG in-stream. Used as response body transformer in proxy pipeline. |
| SQLite | 3.x | Usage logging + per-device stats | WAL mode for concurrent reads. Stores: timestamp, domain, bytes_original, bytes_compressed, device_id. No need for Postgres at this scale. |
| Caddy | 2.9+ | Management API (Phase C) | Optional management interface for hosted service. Reverse proxy to Go management API. |
### Supporting Libraries (Go Daemon)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `elazarl/goproxy` | v1.8+ | MITM proxy framework | Core of remote proxy server. Handles CONNECT tunneling, TLS interception, request/response hooks. |
| `kolesa-team/go-webp` | latest | WebP encoding | Image compression pipeline. Encode intercepted JPEG/PNG responses to WebP at configurable quality. |
| `tdewolff/minify` | v2.21+ | CSS/JS/HTML minification | Applied to all text/html, text/css, application/javascript responses flowing through proxy. |
| `nfnt/resize` or `disintegration/imaging` | latest | Image resizing | Resize images to max 800px width before WebP encoding. `imaging` preferred (more maintained). |
| `r3labs/sse` | v2.x | Server-Sent Events | Dashboard real-time updates. Pi daemon pushes per-device bandwidth events to connected browsers. |
| `mattn/go-sqlite3` | latest | SQLite driver | Usage logging on both Pi and remote server. CGo required but well-supported on ARM64. |
| `vishvananda/netlink` | latest | Linux netlink interface | Programmatic access to network interfaces, routes, qdiscs from Go. Avoids shelling out to `ip` and `tc`. |
### Development Tools
| Tool | Purpose | Notes |
|------|---------|-------|
| `rpi-image-gen` | Custom SD card image builder | Official Raspberry Pi tool (2025). YAML-based config, builds from Debian/RPi OS packages. Produces flashable `.img` for Raspberry Pi Imager. Replaces older `pi-gen`. |
| `sdm` | SD card image manager | Alternative to rpi-image-gen. Script-based customization of Raspberry Pi OS images. Good for iterative development. |
| QEMU | Cross-compilation testing | Test ARM64 images on x86 dev machine. `qemu-user-static` for running ARM binaries in Docker. |
| Wireshark / tcpdump | Network debugging | Essential for verifying proxy behavior, DNS blocking, WireGuard tunnel traffic. |
| `iperf3` | Bandwidth testing | Measure actual throughput through proxy pipeline. Before/after comparison. |
| `fping` | Latency measurement | Required by cake-autorate. Also useful for monitoring Starlink link quality. |
## Installation
### Pi Setup (from fresh Raspberry Pi OS Lite)
# System update
# WiFi AP
# DNS filtering
# WireGuard (kernel module already present)
# Traffic shaping (CAKE is in-kernel, just need iproute2)
# Firewall / routing
# Captive portal web server
# Download Caddy ARM64 binary from https://caddyserver.com/download
# Go daemon (pre-compiled binary, not built on Pi)
# Cross-compile: GOOS=linux GOARCH=arm64 go build -o skygate-daemon ./cmd/daemon
# HTMX (just a JS file, no npm)
### Remote Server Setup
# Clone repo
# One-command deploy
# Generates: WireGuard server, Go proxy, SQLite DB
# Output: server public key, endpoint, client config snippet
### Go Module Dependencies (development)
# Proxy server
# Pi daemon
## Alternatives Considered
| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Custom Go proxy (on goproxy) | compy (barnacs/compy) | Never as-is. compy is unmaintained (last commit ~2021, 35 open issues, no releases). However, study its architecture -- it proves the concept. Extract patterns, don't depend on the binary. |
| Custom Go proxy (on goproxy) | mitmproxy (Python) | Only if Go proxy proves too rigid for complex content stripping (social media HTML rewriting). mitmproxy's Python addon system is more flexible. BUT: 200-500MB RAM, known memory leaks on ARM (grows from 169MB to 1.5GB+ over hours). Would require periodic restart cron. Not viable for 2GB Pi. |
| Caddy (captive portal) | Nginx | If you need more control over HTTP routing. Caddy is simpler for this use case (static files + reverse proxy). Nginx has no ARM64 build issues either. |
| hostapd + dnsmasq | NetworkManager `nmcli` | On Bookworm, NetworkManager is default and can create hotspots with `nmcli device wifi hotspot`. Simpler but less control over DHCP ranges, DNS, and ipset integration. Use hostapd when you need dnsmasq ipset features. |
| hostapd + dnsmasq | RaspAP | RaspAP bundles hostapd + dnsmasq + web UI. Overkill for SkyGate (we have our own dashboard). Adds unnecessary attack surface and dependencies. |
| nftables | iptables (legacy) | Never on Bookworm. iptables-legacy is deprecated. Use nftables directly or via iptables-nft compatibility layer. |
| CAKE qdisc (manual tc) | cake-autorate (lynxthecat) | cake-autorate is powerful but officially supports OpenWrt and Asus Merlin only. Running on Raspberry Pi OS requires manual porting of bash scripts, systemd service creation, and dependency installation (fping, iputils-ping). Worth attempting in Phase 2 but have manual CAKE config as fallback. |
| Pi-hole | AdGuard Home | AdGuard Home is a viable alternative with built-in HTTPS filtering and a modern UI. But Pi-hole has 10x the community, more blocklists, and is the recognized standard. Pilots will recognize "Pi-hole" by name. |
| rpi-image-gen | pi-gen | pi-gen is the older tool (still used for official images). rpi-image-gen is the official replacement (2025), YAML-based, faster builds. Use pi-gen only if rpi-image-gen has issues with specific customizations. |
| Go for daemon | Python | Python is slower, uses 5-10x more RAM, and requires a runtime on the Pi. Go compiles to a single static binary. Only advantage of Python is faster prototyping, but the daemon is simple enough that Go is faster overall. |
| HTMX for dashboard | React/Vue SPA | SPAs require Node.js toolchain, produce 500KB+ bundles, and add complexity. HTMX is 14KB, works with server-rendered HTML fragments, and needs zero build infrastructure. Perfect for embedded. |
| SQLite | PostgreSQL | PostgreSQL is overkill for per-device usage logs. SQLite WAL mode handles concurrent reads from dashboard + writes from daemon. No external process to manage. |
## What NOT to Use
| Avoid | Why | Use Instead |
|-------|-----|-------------|
| compy (barnacs/compy) as a dependency | Unmaintained since ~2021. 35 open issues. No releases. Uses deprecated Travis CI. Go module path may not resolve cleanly. Last meaningful activity 4+ years ago. | Build custom Go proxy on `elazarl/goproxy`. Study compy's image transcoding approach but implement fresh. |
| mitmproxy on the Pi | Python. 200-500MB RAM baseline. Known memory leak pattern on ARM (grows to 1.5GB+). Requires Python runtime + pip dependencies on the Pi. Violates the "20MB Go binary" resource constraint. | Custom Go proxy on remote server. If mitmproxy is needed, run it on the remote server only (Docker, more RAM). |
| OpenVPN | 70,000+ LOC vs WireGuard's 4,000. Slower connection establishment. Higher CPU usage on ARM. No kernel-level integration. | WireGuard (in-kernel since Linux 5.6). |
| coovachilli | Outdated, poor documentation, "works but not smoothly" per community feedback. RADIUS-based auth is overkill for a captive portal with terms acceptance. | Custom captive portal with Caddy + Go. SkyGate needs a usage dashboard, not RADIUS auth. openNDS (v10.3+) if you want a framework. |
| nodogsplash | Unmaintained since spring 2020. Forked into openNDS which is actively developed. | openNDS (v10.3+) if framework needed, or custom Caddy-based portal (recommended). |
| Squid proxy | Enterprise HTTP proxy. Massive configuration surface. Not designed for content modification. Memory-heavy. | Custom Go proxy. Squid solves caching, not content stripping. |
| Node.js on the Pi | V8 runtime, 50MB+ baseline memory. npm dependency hell on ARM. No benefit over Go for a network daemon. | Go for all Pi-side code. |
| Raspberry Pi OS Desktop | Includes X11, Wayland, desktop apps. Wastes 500MB+ RAM on a headless appliance. | Raspberry Pi OS Lite (headless). |
| Pi Zero / Pi Zero 2W | Pi Zero: single-core, 512MB RAM -- cannot run Pi-hole + WireGuard + daemon simultaneously. Pi Zero 2W: quad-core but still 512MB. Insufficient. | Pi 4 (2GB minimum) or Pi 5 (4GB recommended). |
## Stack Patterns by Variant
- Skip WireGuard, remote proxy, Docker
- Pi runs: hostapd + dnsmasq + Pi-hole + CAKE + Go daemon (dashboard)
- Total RAM usage: ~200-300MB
- Can run on Pi 4 2GB comfortably
- Pi adds: WireGuard client, nftables policy routing, ipset bypass rules
- Remote server adds: WireGuard server, Go proxy, SQLite
- Total Pi RAM usage: ~300-400MB
- Remote server: any VPS with 1GB+ RAM (Hetzner CX22: 2 vCPU, 4GB, ~$4.50/mo)
- Use `rpi-image-gen` with YAML config specifying all packages + configs
- Output: `.img` file flashable via Raspberry Pi Imager
- Include first-boot setup wizard (Go binary serving web UI on AP)
- Requires: bash, fping, iputils-ping, iproute2
- Port cake-autorate scripts from OpenWrt to systemd service
- Configure CAKE on WireGuard interface (wg0) and WiFi interface (wlan0/wlan1)
- Fallback: static CAKE bandwidth ceiling if autorate porting is problematic
## Version Compatibility
| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Pi-hole v6.x | dnsmasq 2.89+ | Pi-hole ships its own FTL (dnsmasq fork). When running alongside custom dnsmasq for ipset, use Pi-hole's FTL as the DNS server and configure ipset rules in Pi-hole's custom config (`/etc/pihole/dnsmasq.d/`). |
| WireGuard (kernel) | Linux kernel 6.12 (Pi 5) | Native kernel module. No DKMS needed. `wireguard-tools` provides `wg` and `wg-quick` userspace utilities. |
| nftables 1.0.6 | Bookworm kernel 6.12 | Full nftables support. iptables-nft compatibility layer translates iptables syntax to nftables backend. Pi-hole uses iptables-nft internally. |
| Go 1.22+ | ARM64 (aarch64) | Cross-compile with `GOOS=linux GOARCH=arm64`. CGo needed for go-webp (libwebp) and go-sqlite3. Install `aarch64-linux-gnu-gcc` for cross-CGo. |
| CAKE qdisc | iproute2 6.1+ | `tc-cake` man page available. CAKE compiled into Bookworm kernel by default. Verify: `tc qdisc add dev lo root cake help` should list options. |
| Caddy 2.9 | ARM64 Linux | Download pre-built binary. No runtime dependencies. Can also build from source with `xcaddy`. |
| HTMX 2.0 | Any browser | No server-side version constraints. Include as static JS file. SSE extension bundled. |
## Sources
- [Pi-hole official docs](https://docs.pi-hole.net/) -- v6.3/6.4 version confirmed, prerequisites checked (HIGH confidence)
- [WireGuard official site](https://www.wireguard.com/) -- kernel integration, performance claims verified (HIGH confidence)
- [cake-autorate GitHub](https://github.com/lynxthecat/cake-autorate) -- OpenWrt/Merlin support confirmed, standalone Linux requires manual porting (MEDIUM confidence)
- [Raspberry Pi OS downloads](https://www.raspberrypi.com/software/operating-systems/) -- Bookworm Lite ARM64, kernel 6.12 confirmed (HIGH confidence)
- [rpi-image-gen announcement](https://www.raspberrypi.com/news/introducing-rpi-image-gen-build-highly-customised-raspberry-pi-software-images/) -- official 2025 tool, YAML config confirmed (HIGH confidence)
- [compy GitHub](https://github.com/barnacs/compy) -- 209 stars, 35 open issues, last commit ~2021, unmaintained (HIGH confidence on status)
- [mitmproxy memory issues](https://github.com/mitmproxy/mitmproxy/issues/6371) -- memory growth on ARM confirmed via multiple GitHub issues (HIGH confidence)
- [goproxy GitHub](https://github.com/elazarl/goproxy) -- v1.8.2, active maintenance, MITM support confirmed (MEDIUM confidence)
- [Caddy official](https://caddyserver.com/) -- ARM64 support, automatic HTTPS, static file serving confirmed (HIGH confidence)
- [HTMX official](https://htmx.org/) -- 14KB size, SSE extension, no build step confirmed (HIGH confidence)
- [dnsmasq ipset integration](https://man.archlinux.org/man/dnsmasq.8.en) -- `ipset=` directive for DNS-based routing confirmed (HIGH confidence)
- [openNDS docs](https://opennds.readthedocs.io/) -- v10.3.0, actively maintained fork of nodogsplash (MEDIUM confidence)
- [Raspberry Pi 5 WiFi](https://www.tomshardware.com/news/raspberry-pi-5-wi-fi-faster) -- CYW43455 chip, DDR50, 5GHz hotspot issues noted (HIGH confidence)
- [linuxserver/wireguard Docker](https://hub.docker.com/r/linuxserver/wireguard) -- Docker Compose setup for server-side WireGuard (HIGH confidence)
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd:quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd:debug` for investigation and bug fixing
- `/gsd:execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd:profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
