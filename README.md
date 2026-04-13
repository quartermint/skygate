# SkyGate

> Open source bandwidth management appliance for general aviation aircraft on Starlink.

![Status: Alpha](https://img.shields.io/badge/status-alpha-orange)
![Go](https://img.shields.io/badge/go-1.22+-00ADD8)
![Platform: RPi](https://img.shields.io/badge/platform-raspberry--pi-C51A4A)

A Raspberry Pi-based device that sits between passengers and a Starlink Mini. It combines DNS-level ad/tracker blocking (Pi-hole), a content-compressing proxy (WireGuard tunnel to remote server), and dynamic QoS (CAKE) to cut data usage on metered aviation plans. A real-time usage dashboard shows pilots exactly what is eating their 20 GB cap.

**Who it's for:** GA pilots flying with Starlink Mini who need to stretch a metered data plan across a full flight, without configuring anything complex.

---

## Architecture

```
On-aircraft (Raspberry Pi)          Remote server (VPS)
+---------------------------------+  +-----------------------+
| hostapd  -- WiFi AP             |  | WireGuard server      |
| dnsmasq  -- DHCP + DNS + ipsets |  | Go proxy              |
| Pi-hole  -- DNS blocking        |--+  - Image -> WebP      |
| WireGuard client                |     - CSS/JS minify      |
| CAKE qdisc -- QoS / shaping     |     - SQLite usage log   |
| Go daemon -- usage stats + SSE  |  +-----------------------+
| Caddy    -- dashboard web UI    |
+---------------------------------+
```

### Two-layer approach

| Layer | Mechanism | Applies to |
|-------|-----------|------------|
| Layer 1 (DNS) | Pi-hole blocks ad/tracker/CDN domains | All devices, zero friction |
| Layer 2 (Proxy) | Go proxy via WireGuard strips images, minifies JS/CSS | Browsers with CA cert installed |

### Go daemons

| Binary | Purpose |
|--------|---------|
| `skygate-bypass` | Resolves aviation app domains (ForeFlight, Garmin, weather APIs) into nftables bypass sets so safety-critical traffic never hits the proxy |
| `skygate-dashboard` | Reads `/sys/class/net/` stats and nftables counters; streams per-device bandwidth via SSE |
| `skygate-tunnel-monitor` | Monitors WireGuard tunnel health and failover |
| `skygate-proxy` | Content-compressing proxy (CGO, requires libwebp) -- runs on remote VPS |

---

## Quickstart

### Build

```bash
# Build all Pi daemons (current platform)
make build

# Cross-compile for Raspberry Pi (linux/arm64)
make cross-build

# Build proxy server (requires libwebp-dev)
make build-proxy
```

### Deploy to Pi via Ansible

```bash
# Dry run
make deploy-check

# Apply
make deploy
```

Configure your Pi host in `pi/ansible/` inventory. Default host alias is `skygate`.

### Remote server (content proxy)

```bash
# Build and start WireGuard + proxy stack
cd server && docker compose up -d
```

---

## Configuration

Pi config lives at `/data/skygate/` after deployment:

```
/data/skygate/
├── bypass-domains.yaml   # Aviation domains to route direct (never proxied)
├── config.yaml           # Bandwidth cap, QoS settings, proxy endpoint
└── ca/                   # Intermediate CA for MITM proxy (auto-generated)
```

The bypass daemon re-resolves aviation domains every 60 seconds by default (`--interval` flag).

---

## Development

```bash
# Go tests
make test-go

# BATS integration tests (requires bats-core)
make test-bats

# Lint Ansible playbooks
make lint-ansible
```

---

## Hardware

- Raspberry Pi 5 (4GB recommended) or Pi 4 (2GB minimum)
- Raspberry Pi OS Lite (Bookworm, ARM64, headless)
- USB WiFi adapter for AP mode (onboard WiFi used for Starlink uplink)
- Remote VPS: any 1GB+ instance (Hetzner CX22 or equivalent)

---

## License

MIT License. See LICENSE for details.
