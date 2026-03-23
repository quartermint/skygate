# Phase 1: Pi Network Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-23
**Phase:** 01-pi-network-foundation
**Areas discussed:** Development approach, WiFi AP identity, DNS blocking scope, Aviation bypass list

---

## Development Approach

### Dev Target

| Option | Description | Selected |
|--------|-------------|----------|
| Laptop-first (Recommended) | Develop with Docker/systemd-nspawn on macOS/Linux, deploy to Pi for integration testing | ✓ |
| Pi-first | Develop directly on Raspberry Pi hardware from day one | |
| Hybrid | Core services in Docker on laptop, hardware-specific bits on Pi only | |

**User's choice:** Laptop-first
**Notes:** None

### Dev Stack

| Option | Description | Selected |
|--------|-------------|----------|
| Ansible playbook (Recommended) | Declarative config management — playbook defines packages, configs, services | ✓ |
| Shell scripts | Simple bash scripts for setup and config | |
| Docker on Pi | Containerized services on the Pi itself | |

**User's choice:** Ansible playbook
**Notes:** None

### Starlink Simulation

| Option | Description | Selected |
|--------|-------------|----------|
| tc netem simulation (Recommended) | Linux traffic control to simulate Starlink latency, jitter, and bandwidth caps | |
| Real Starlink only | Only test with actual Starlink hardware | |
| You decide | Claude picks the best simulation approach | |

**User's choice:** Other — "I have a friend who is stationary with Starlink that would be willing to test for me, maybe iOS easiest for data tests? TestFlight?"
**Notes:** Friend is remote (hence Starlink, not aviation). Could test page loads over real Starlink to validate latency/bandwidth behavior. tc netem also used for local development. iOS TestFlight mention noted as deferred idea — no native app in v1.

---

## WiFi AP Identity

### SSID

| Option | Description | Selected |
|--------|-------------|----------|
| SkyGate | Clean, matches product name | |
| SkyGate-XXXX | Product name + last 4 of MAC | |
| Custom on setup | Pilot sets SSID during first-boot wizard | ✓ |

**User's choice:** Custom on setup
**Notes:** None

### WiFi Band

| Option | Description | Selected |
|--------|-------------|----------|
| 2.4 GHz only (Recommended) | Better range, wider compatibility, single adapter | ✓ |
| 5 GHz only | Faster speeds, shorter range | |
| Dual-band | Both bands, requires two radios | |

**User's choice:** 2.4 GHz only
**Notes:** None

### Password Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Default on sticker (Recommended) | Random password printed on device sticker | ✓ |
| Open + captive portal | No WiFi password, captive portal handles access | |
| Pilot sets on first boot | First-boot wizard requires password creation | |

**User's choice:** Default on sticker
**Notes:** None

### Max Clients

| Option | Description | Selected |
|--------|-------------|----------|
| 8 devices (Recommended) | Covers 1-4 passengers with 2 devices each | ✓ |
| 16 devices | Generous limit for larger cabins | |
| You decide | Claude picks a sensible default | |

**User's choice:** 8 devices
**Notes:** None

---

## DNS Blocking Scope

### Blocklist Aggressiveness

| Option | Description | Selected |
|--------|-------------|----------|
| Conservative (Recommended) | Ads + trackers only. Pilots enable more themselves | ✓ |
| Aggressive | Block ads, trackers, video CDNs, updates, cloud sync by default | |
| Category toggles | All categories available but pilot enables each one | |

**User's choice:** Conservative
**Notes:** None

### Block UX

| Option | Description | Selected |
|--------|-------------|----------|
| Silent block (Recommended) | Pi-hole returns NXDOMAIN — blank ad slots, failed connections | ✓ |
| Block page | Custom "Blocked by SkyGate" page explaining why | |
| You decide | Claude picks best approach | |

**User's choice:** Silent block
**Notes:** None

---

## Aviation Bypass List

### Default Bypass Apps

| Option | Description | Selected |
|--------|-------------|----------|
| ForeFlight (Recommended) | Most popular GA EFB | ✓ |
| Garmin Pilot | Second most popular GA EFB | ✓ |
| Weather APIs | aviationweather.gov, NOAA/NWS | ✓ |
| ADS-B services | FlightAware, Flightradar24 | ✓ |

**User's choice:** All four selected
**Notes:** None

### Bypass Management

| Option | Description | Selected |
|--------|-------------|----------|
| Config file (Recommended) | YAML/JSON file on Pi, pilot edits via SSH or web UI | ✓ |
| Dashboard UI only | Add/remove via web dashboard (Phase 2 dependency) | |
| You decide | Claude picks best management approach | |

**User's choice:** Config file
**Notes:** None

---

## Claude's Discretion

- OverlayFS configuration
- USB WiFi adapter recommendation
- hostapd channel/power settings
- Pi-hole blocklist selection
- CAKE qdisc parameters
- cake-autorate Starlink profile
- First-boot setup implementation

## Deferred Ideas

- iOS companion app / TestFlight for data testing
- Video CDN, update, cloud sync DNS blocking categories (v2)
- Web UI for bypass list management (Phase 2)
- Custom block page for blocked domains
