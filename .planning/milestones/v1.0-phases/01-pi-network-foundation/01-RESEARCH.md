# Phase 1: Pi Network Foundation - Research

**Researched:** 2026-03-23
**Domain:** Embedded Linux networking -- WiFi AP, DNS filtering, policy-based routing, traffic shaping, read-only filesystem
**Confidence:** HIGH

## Summary

Phase 1 builds the complete network foundation for SkyGate: a Raspberry Pi 5 running as a WiFi access point with DNS-level ad/tracker blocking via Pi-hole, policy-based routing that sends aviation app traffic directly to Starlink while marking all other traffic for future tunnel routing, CAKE-based QoS to prevent Starlink bufferbloat, and a read-only root filesystem that survives abrupt power loss. All development happens laptop-first with Ansible playbooks deploying to the Pi for integration testing.

The single most critical research finding is that **Pi-hole's FTL does NOT compile with `HAVE_NFTSET` support** (closed as "not planned" by maintainers). This means the architecture cannot use Pi-hole's embedded dnsmasq for nftset population. The solution is to run a lightweight standalone dnsmasq instance on a non-standard port (e.g., 5380) solely for nftset population of aviation bypass domains, with Pi-hole's FTL handling all DNS resolution and blocking on port 53. This is the DNS forwarding chain pattern: clients -> Pi-hole FTL (port 53, DNS filtering) -> for bypass domains, Pi-hole forwards to standalone dnsmasq (port 5380) which populates nftsets -> upstream DNS. Alternatively, a simpler approach: use a small Go helper daemon or cron script that resolves bypass domains and adds their IPs to nftables sets directly, bypassing the dnsmasq nftset mechanism entirely.

**Primary recommendation:** Use the Go helper daemon approach for aviation bypass routing (simpler, no second dnsmasq instance), hostapd with MT7612U USB adapter for AP, Pi-hole v6 for DNS filtering, nftables native sets for policy routing, CAKE with a custom autorate script for QoS, and raspi-config OverlayFS for read-only root with a separate writable data partition.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Laptop-first development -- core services developed and tested on macOS/Linux, deployed to Pi via Ansible for integration testing
- **D-02:** Ansible playbook for Pi configuration -- declarative, reproducible deploys defining packages, configs, and services
- **D-03:** tc netem for local Starlink simulation (latency, jitter, bandwidth caps) during laptop development
- **D-04:** Real Starlink testing available via remote friend with ground Starlink -- useful for validating tunnel/QoS behavior over actual satellite, but not aircraft-specific
- **D-05:** Test client devices are phones/laptops connecting to Pi WiFi and browsing via Safari/Chrome -- no native test app needed
- **D-06:** SSID set by pilot during first-boot setup. No hardcoded default -- pilot customizes on first use
- **D-07:** 2.4 GHz only -- better range in aircraft cabin, wider device compatibility, single USB adapter
- **D-08:** Default random password printed on device sticker. Pilot can change via web UI (Phase 2)
- **D-09:** Maximum 8 simultaneous devices -- covers 1-4 passengers with 2 devices each
- **D-10:** Conservative out of the box -- ads and trackers blocked by default only. Video CDN, update, and cloud sync blocking are opt-in categories (deferred to v2)
- **D-11:** Silent block (Pi-hole NXDOMAIN) -- no custom block pages. Blank ad slots, failed connections. Standard Pi-hole behavior
- **D-12:** Default bypass list ships with: ForeFlight (*.foreflight.com), Garmin Pilot (*.garmin.com, fly.garmin.com), Weather APIs (aviationweather.gov, NOAA/NWS), ADS-B services (FlightAware, Flightradar24)
- **D-13:** Bypass list managed via YAML/JSON config file on the Pi. Pilot edits via SSH or web UI (web UI in Phase 2)
- **D-14:** DNS responses for bypass domains populate ipset dynamically -- traffic for these domains routes direct to Starlink

### Claude's Discretion
- Exact OverlayFS configuration and which directories are writable
- USB WiFi adapter recommendation (research should evaluate MediaTek MT7612U vs Realtek RTL8812BU)
- hostapd channel selection and power settings
- Pi-hole blocklist selection (which community lists to include)
- CAKE qdisc initial bandwidth parameters
- cake-autorate configuration values for Starlink profile
- First-boot setup implementation (minimal -- just SSID and optional password change)

### Deferred Ideas (OUT OF SCOPE)
- iOS companion app / TestFlight for data testing -- not in v1 scope, web dashboard is the client interface
- Video CDN, update, and cloud sync DNS blocking categories -- deferred to v2 requirements
- Web UI for bypass list management -- Phase 2 (dashboard)
- Custom block page when domains are blocked -- deferred, silent block is sufficient
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NET-01 | Pi serves as WiFi access point with WPA2 that passengers connect to (hostapd + dnsmasq) | hostapd with MT7612U USB adapter for AP, NetworkManager disabled for AP interface, Pi-hole's built-in DHCP or standalone dnsmasq for DHCP |
| NET-02 | Connected devices receive IP addresses via DHCP with DNS routed through Pi-hole | Pi-hole v6 DHCP server on wlan1 (AP interface) subnet, DNS traffic directed to Pi-hole FTL on localhost |
| DNS-01 | Pi-hole blocks ads, trackers, and known malicious domains at DNS level with community blocklists | Pi-hole v6.3+ with StevenBlack unified list + OISD light, conservative defaults per D-10 |
| ROUTE-01 | Aviation apps bypass proxy and route directly to Starlink via DNS-driven ipset | Go helper daemon resolves bypass domains periodically, populates nftables native sets, fwmark 0x1 routes direct via eth0 |
| QOS-01 | CAKE qdisc with cake-autorate dynamically adjusts bandwidth ceiling based on real-time latency | CAKE on eth0 (Starlink uplink), custom autorate bash script with fping latency probes, 3-parameter control (min/base/max rates) |
</phase_requirements>

## Standard Stack

### Core (Phase 1 Only)

| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| Raspberry Pi 5 (4GB) | Pi 5 | Hardware platform | 2x CPU vs Pi 4, DDR50 SDIO, 4GB sufficient. $60. |
| Raspberry Pi OS Lite (Bookworm) | Debian 12, Kernel 6.12, ARM64 | Base OS | Official headless image. NetworkManager default. ARM64 for Go binaries. |
| hostapd | 2.10+ (Bookworm repo) | WiFi access point daemon | Industry standard for Linux AP mode. WPA2-PSK, 802.11n, channel selection. |
| Pi-hole | v6.3+ (FTL v6.4) | DNS-level ad/tracker blocking | 45.1k stars, mature blocklists. Layer 1 filtering. Built-in DHCP server. |
| nftables | 1.0.6+ (Bookworm default) | Firewall + packet marking + native sets | Replaces iptables. Native set/map support replaces ipset. JSON output for monitoring. |
| CAKE qdisc | tc-cake (iproute2, in-kernel) | Traffic shaping / QoS | In-kernel since 4.19. Prevents Starlink bufferbloat. Better than fq_codel for variable-rate links. |
| Go | 1.22+ (cross-compile from dev machine) | Bypass helper daemon + future dashboard | Single binary, ~5MB compiled. Cross-compile GOOS=linux GOARCH=arm64. |
| Ansible | 2.16+ (dev machine only) | Pi provisioning and deployment | Declarative, idempotent configuration. Locked decision D-02. |

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| fping | 5.x (Bookworm repo) | Latency probing for autorate | Used by custom CAKE autorate script to measure RTT |
| iproute2 | 6.1+ (Bookworm repo) | tc commands, ip rule/route | Policy routing tables, CAKE qdisc management |
| conntrack-tools | 1.4.7+ (Bookworm repo) | Connection tracking flush | Required when nftables sets are flushed to clear stale ct marks |
| jq | 1.6+ (Bookworm repo) | JSON parsing | Parse nftables JSON output in autorate/monitoring scripts |

### USB WiFi Adapter Recommendation

**Use MediaTek MT7612U-based adapter.** Do NOT use Realtek RTL8812BU.

| Criteria | MT7612U | RTL8812BU |
|----------|---------|-----------|
| Kernel driver | In-kernel since 4.19 (mt76 driver) | Requires out-of-tree driver (88x2bu) |
| AP mode support | Excellent, confirmed by morrownr/USB-WiFi project | Problematic, "wave people off if they mention AP mode" |
| Raspberry Pi compatibility | First-class, recommended by The Pi Hut and Core Electronics | USB2 mode only on Pi 4/5, TX power issues |
| Driver maintenance | Maintained in mainline kernel | Community-maintained, breaks on kernel updates |
| 2.4 GHz support | Yes (dual-band) | Yes (dual-band) |

Specific adapter: **ALFA AWUS036ACM** (MT7612U) or equivalent. Available at The Pi Hut, Amazon. ~$25-35.

Confidence: HIGH -- based on morrownr/USB-WiFi project recommendations (the canonical Linux WiFi adapter reference) and multiple community confirmations.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| hostapd + standalone dnsmasq | NetworkManager `nmcli device wifi hotspot` | Simpler but less control over DHCP ranges and no ipset/nftset integration |
| Pi-hole | AdGuard Home | AdGuard has built-in HTTPS filtering but Pi-hole has 10x community, pilots recognize the name |
| Custom Go bypass daemon | Standalone dnsmasq with nftset on port 5380 | More standard but adds a second dnsmasq process; Go daemon is simpler to manage |
| CAKE manual config | Full cake-autorate port | cake-autorate is OpenWrt-only; custom script extracts the core algorithm in ~50 lines of bash |

## Architecture Patterns

### Recommended Project Structure (Phase 1)
```
skygate/
+-- pi/                           # Everything deployed to the Pi
|   +-- ansible/                  # Ansible playbook and roles
|   |   +-- playbook.yml          # Main playbook
|   |   +-- inventory/
|   |   |   +-- hosts.yml         # Pi inventory (IP, SSH key)
|   |   +-- roles/
|   |   |   +-- base/             # OS config, OverlayFS, sysctl
|   |   |   +-- networking/       # hostapd, DHCP, IP forwarding, NAT
|   |   |   +-- pihole/           # Pi-hole install + config overlays
|   |   |   +-- routing/          # nftables rules, policy routing, bypass daemon
|   |   |   +-- qos/              # CAKE qdisc, autorate script
|   |   |   +-- firstboot/        # First-boot SSID/password setup
|   |   +-- files/                # Static config files
|   |   +-- templates/            # Jinja2 templates (hostapd.conf.j2, etc.)
|   +-- cmd/
|   |   +-- bypass-daemon/        # Go: resolves bypass domains, populates nftsets
|   |       +-- main.go
|   +-- config/
|   |   +-- bypass-domains.yaml   # Aviation app bypass list (D-12)
|   |   +-- blocklists.yaml       # Pi-hole blocklist URLs
|   +-- scripts/
|   |   +-- autorate.sh           # Custom CAKE autorate (bash + fping)
|   |   +-- firstboot.sh          # First-boot setup wizard
|   +-- systemd/                  # Service unit files
|       +-- skygate-bypass.service
|       +-- skygate-autorate.service
+-- Makefile                      # Top-level build/deploy commands
+-- go.mod                        # Go module definition
+-- go.sum
+-- .planning/                    # GSD planning artifacts
+-- CLAUDE.md
```

### Pattern 1: Aviation Bypass via Go Helper Daemon + nftables Native Sets

**What:** A small Go daemon reads `bypass-domains.yaml`, periodically resolves each domain (every 60s), and adds the resolved IPs to nftables native sets using the `google/nftables` Go library or by shelling out to `nft`. nftables marks packets destined for bypass IPs with fwmark 0x1, routing them direct to Starlink via eth0.

**Why not dnsmasq nftset?** Pi-hole's FTL does NOT compile with `HAVE_NFTSET`. Running a second dnsmasq instance adds complexity. A Go daemon is simpler, testable, and aligns with the project's Go stack.

**When to use:** Whenever you need domain-based routing decisions on a Linux gateway where the DNS server doesn't support nftset natively.

**Implementation:**

```go
// bypass-daemon/main.go (simplified)
package main

import (
    "net"
    "os/exec"
    "time"
    "gopkg.in/yaml.v3"
)

type Config struct {
    Domains []string `yaml:"domains"`
}

func main() {
    // Load bypass-domains.yaml
    // Create nftables set if not exists:
    //   nft add table inet skygate
    //   nft add set inet skygate bypass_v4 { type ipv4_addr; flags timeout; timeout 1h; }

    for {
        for _, domain := range config.Domains {
            ips, _ := net.LookupHost(domain)
            for _, ip := range ips {
                // nft add element inet skygate bypass_v4 { <ip> timeout 1h }
                exec.Command("nft", "add", "element", "inet", "skygate",
                    "bypass_v4", "{", ip, "timeout", "1h", "}").Run()
            }
        }
        time.Sleep(60 * time.Second)
    }
}
```

```bash
# nftables rules (deployed via Ansible)
table inet skygate {
    set bypass_v4 {
        type ipv4_addr
        flags timeout
        timeout 1h
    }

    chain prerouting {
        type filter hook prerouting priority mangle;
        # Aviation bypass: mark for direct routing
        ip daddr @bypass_v4 meta mark set 0x1 ct mark set meta mark
        # Restore connection marks for existing connections
        ct mark 0x1 meta mark set ct mark
    }

    chain postrouting {
        type nat hook postrouting priority srcnat;
        # NAT all outbound traffic via Starlink
        oifname "eth0" masquerade
    }
}
```

```bash
# Policy routing (deployed via Ansible, runs at boot)
# Table 100: direct to Starlink (aviation bypass)
ip rule add fwmark 0x1 table 100 priority 100
ip route add default dev eth0 table 100

# Default route: also via eth0 for Phase 1 (no tunnel yet)
# Phase 3 will add: ip rule add fwmark 0x2 table 200
# Phase 3 will add: ip route add default dev wg0 table 200
```

### Pattern 2: Read-Only Root with OverlayFS + Writable Data Partition

**What:** Root filesystem is read-only via raspi-config OverlayFS. A small separate partition (or tmpfs + periodic sync) handles mutable data.

**Recommended OverlayFS configuration:**
- Enable via `raspi-config nonint do_overlayfs 0` (enables overlay) + `raspi-config nonint do_boot_ro 0` (read-only boot)
- All writes go to RAM overlay; SD card root is never written to
- Writable areas via tmpfs in fstab:

```
tmpfs  /tmp               tmpfs  defaults,noatime,nosuid,nodev            0  0
tmpfs  /var/tmp           tmpfs  defaults,noatime,nosuid,nodev            0  0
tmpfs  /var/log           tmpfs  defaults,noatime,nosuid,nodev,size=50m   0  0
tmpfs  /run               tmpfs  defaults,noatime,nosuid,nodev,mode=0755  0  0
```

- Separate writable data partition (partition 3) for persistent config changes:

```
# /etc/fstab addition
/dev/mmcblk0p3  /data  ext4  defaults,noatime,data=journal,sync  0  2
```

- Symlinks from service directories to /data:
  - `/etc/pihole` -> `/data/pihole/` (Pi-hole gravity DB, config)
  - `/data/skygate/` (bypass-domains.yaml, autorate config, usage logs in Phase 2)

**Gotcha (Bookworm-specific):** When raspi-config enables read-only filesystem, it changes ALL mounted partitions to read-only (including /boot). The separate data partition must be mounted AFTER the overlay is applied, or explicitly in fstab with rw.

### Pattern 3: NetworkManager + hostapd Coexistence on Bookworm

**What:** Bookworm defaults to NetworkManager. hostapd needs exclusive control of the AP interface. NetworkManager must be told to ignore the USB WiFi adapter.

**Implementation:**

```ini
# /etc/NetworkManager/conf.d/unmanaged.conf
[keyfile]
unmanaged-devices=interface-name:wlan1
```

```bash
# Disable NetworkManager for the AP interface
sudo nmcli device set wlan1 managed no
```

Where:
- `wlan0` = onboard WiFi (unused, or for emergency recovery)
- `wlan1` = USB MT7612U adapter (AP mode via hostapd)
- `eth0` = Ethernet to Starlink Mini

**Key hostapd.conf settings (2.4 GHz, WPA2, 8 clients):**

```ini
interface=wlan1
driver=nl80211
# SSID configured during first-boot
ssid=SkyGate
hw_mode=g
channel=6
ieee80211n=1
wmm_enabled=1
ht_capab=[HT40+][SHORT-GI-20][SHORT-GI-40][DSSS_CCK-40]
auth_algs=1
wpa=2
wpa_key_mgmt=WPA-PSK
wpa_pairwise=CCMP
rsn_pairwise=CCMP
# Password configured during first-boot
wpa_passphrase=PLACEHOLDER
max_num_sta=8
country_code=US
ieee80211d=1
```

**Channel selection recommendation:** Channel 6 is the safest default for 2.4 GHz (one of three non-overlapping channels: 1, 6, 11). Aircraft environment has minimal external WiFi interference, so channel selection is less critical than ground installations. Set `country_code=US` for proper regulatory compliance.

### Pattern 4: Pi-hole v6 DHCP + DNS Integration

**What:** Pi-hole's FTL handles both DNS (port 53) and DHCP for the AP subnet. No separate dnsmasq instance for DHCP.

**Configuration via `/etc/pihole/pihole.toml`:**

```toml
[dhcp]
active = true
start = "192.168.4.100"
end = "192.168.4.200"
router = "192.168.4.1"
netmask = "255.255.255.0"
domain = "skygate.local"
leaseTime = "24h"

[dns]
upstreamDNS = ["1.1.1.1", "8.8.8.8"]

[misc]
etc_dnsmasq_d = true  # Allow custom dnsmasq config files
```

Custom dnsmasq config at `/etc/dnsmasq.d/01-skygate.conf`:

```
# Bind only to AP interface for DHCP
interface=wlan1
bind-interfaces

# DNS cache size (default 150 is too small)
cache-size=10000

# Set Pi-hole as DNS server for DHCP clients
dhcp-option=6,192.168.4.1
```

### Anti-Patterns to Avoid

- **Bridged mode (br0):** Bridging wlan1 and eth0 makes Pi invisible at L3. Cannot inspect/mark/route packets. Use routed mode with NAT.
- **Running two dnsmasq instances:** Pi-hole's FTL IS dnsmasq. Running a second one creates port 53 conflicts. Use Pi-hole's built-in DHCP.
- **iptables on Bookworm:** Use nftables natively. iptables-nft compatibility layer works but adds confusion. Pi-hole uses iptables-nft internally -- be aware of potential rule conflicts.
- **AP+STA on same radio:** Never try to use onboard WiFi for both AP and client. Use USB adapter for AP, Ethernet for Starlink uplink.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WiFi access point | Custom networking scripts | hostapd + dnsmasq/Pi-hole DHCP | hostapd handles 802.11 association, authentication, channel management. Decades of battle-testing. |
| DNS ad/tracker blocking | Custom blocklist parser | Pi-hole v6 | 45K stars, maintained blocklists, web admin UI, FTL performance. Community maintains the lists. |
| Traffic shaping | Custom queuing discipline | CAKE qdisc (in-kernel) | CAKE handles fair queuing, rate limiting, AQM in one qdisc. Kernel-level performance. |
| Latency-based rate adjustment | Complex autorate daemon | ~50-line bash script with fping + tc | The core autorate algorithm is simple: ping, measure RTT, adjust bandwidth. cake-autorate's complexity is OpenWrt portability, not the algorithm. |
| Firewall rules | Shell scripts with iptables | nftables config files (declarative) | nftables has native set support, JSON output, atomic rule replacement. Single config file. |
| Pi provisioning | Manual setup scripts | Ansible roles | Idempotent, declarative, testable with `--check` mode. Locked decision D-02. |
| Read-only root filesystem | Custom initramfs scripts | raspi-config OverlayFS | Built into Raspberry Pi OS. One command to enable. Maintained by RPi Foundation. |

**Key insight:** Phase 1 is almost entirely configuration, not code. The only custom code is the Go bypass daemon (~100 lines) and the autorate bash script (~50 lines). Everything else is Ansible deploying well-tested system packages with correct configuration files.

## Common Pitfalls

### Pitfall 1: Pi-hole FTL Lacks nftset Support
**What goes wrong:** You configure `nftset=` directives in Pi-hole's custom dnsmasq config, expecting DNS responses to populate nftables sets. Nothing happens -- FTL silently ignores the directive because it's compiled without `HAVE_NFTSET`.
**Why it happens:** Pi-hole maintainers closed this as "not planned" in Jan 2023 to avoid libnfttables dependency across all supported platforms.
**How to avoid:** Use the Go bypass daemon pattern (Pattern 1 above). Resolve bypass domains independently and populate nftables sets via `nft` CLI.
**Warning signs:** nftables bypass set stays empty despite DNS queries for bypass domains.

### Pitfall 2: NetworkManager Fights hostapd for Interface Control
**What goes wrong:** After enabling the AP with hostapd, NetworkManager reclaims the wireless interface, breaking the AP. Random disconnections, interface flapping.
**Why it happens:** Bookworm's NetworkManager manages all interfaces by default. hostapd and NetworkManager cannot both manage the same interface.
**How to avoid:** Create `/etc/NetworkManager/conf.d/unmanaged.conf` with `unmanaged-devices=interface-name:wlan1`. Verify with `nmcli device status` showing wlan1 as "unmanaged."
**Warning signs:** `journalctl -u hostapd` showing interface busy/unavailable errors.

### Pitfall 3: OverlayFS Makes ALL Partitions Read-Only
**What goes wrong:** Enabling OverlayFS via raspi-config also makes the boot partition and any additional partitions read-only. Pi-hole can't update its gravity database. Config changes don't persist.
**Why it happens:** Bookworm's OverlayFS implementation is broad -- it protects the entire SD card, not just the root partition. This is documented as a known issue on the Bookworm feedback tracker.
**How to avoid:** Mount the writable data partition AFTER overlay initialization, or use a USB drive for persistent data. Test by checking `mount | grep mmcblk0p3` shows "rw" after boot.
**Warning signs:** Pi-hole gravity update fails with "read-only filesystem" errors.

### Pitfall 4: CAKE Bandwidth Ceiling Too Static for Starlink
**What goes wrong:** You set CAKE to 50 Mbps. Starlink actual throughput varies 10-200 Mbps depending on satellite position, weather, congestion. When Starlink delivers 20 Mbps, CAKE tries to shape to 50 and causes packet loss. When delivering 150 Mbps, you waste 100 Mbps of available capacity.
**Why it happens:** Starlink LEO bandwidth is inherently variable. Static QoS settings are always wrong for satellite.
**How to avoid:** Custom autorate script that measures RTT via fping every 1-2 seconds and adjusts CAKE bandwidth with `tc qdisc change`. Start conservative (min: 5 Mbps, base: 20 Mbps, max: 100 Mbps). The algorithm: if RTT > baseline + threshold -> decrease bandwidth; if RTT stable for N seconds -> increase bandwidth toward max.
**Warning signs:** Dashboard showing consistent high latency (bufferbloat) or consistently low throughput (over-shaped).

### Pitfall 5: DNS Blocking Breaks Aviation Safety Apps
**What goes wrong:** Pi-hole's blocklists include CDN domains that ForeFlight and Garmin Pilot depend on. Weather data, chart tiles, and TFR updates fail silently.
**Why it happens:** Aviation apps share CDN infrastructure (Akamai, CloudFront) with ad networks. Blocklists have no concept of "this CDN serves safety-critical aviation data."
**How to avoid:** The bypass daemon must resolve bypass domains BEFORE Pi-hole can block them. Aviation domains must be whitelisted in Pi-hole AND routed directly. Use Pi-hole's whitelist for `*.foreflight.com`, `*.garmin.com`, `aviationweather.gov`, `*.flightaware.com`, `*.flightradar24.com`, `*.weather.gov`.
**Warning signs:** ForeFlight showing "no data" or stale weather. Any `.gov` domain in Pi-hole's blocked list.

### Pitfall 6: SD Card Corruption from Power Loss
**What goes wrong:** Without read-only root, the SD card corrupts after 3-5 abrupt power cuts. Pilot pulls master switch, Pi bricks itself.
**Why it happens:** ext4 with write-back caching accumulates dirty pages in RAM. Sudden power loss = incomplete writes = filesystem corruption.
**How to avoid:** Enable OverlayFS from day one (Phase 1 requirement). Test by pulling power 50+ times during active operation. Read-only root means zero writes to SD card root partition.
**Warning signs:** fsck errors in boot logs, failure to boot after a flight, "read-only filesystem" kernel emergency messages.

### Pitfall 7: Laptop Development Asymmetry (macOS vs Linux)
**What goes wrong:** Core networking tools (tc, nftables, hostapd, CAKE) don't exist on macOS. Development on laptop is limited to Go code and Ansible playbook writing.
**Why it happens:** tc/nftables/hostapd are Linux kernel features. macOS has `pfctl`/`dnctl` for traffic shaping but completely different syntax and capabilities.
**How to avoid:** Develop Go code and Ansible playbooks on macOS. Use Docker with a Linux container for nftables/tc testing if needed. Integration testing always on the actual Pi via Ansible deploy. Starlink simulation with tc netem runs ON the Pi (Linux), not the Mac.
**Warning signs:** Trying to test CAKE or nftables rules on macOS.

## Code Examples

### Custom CAKE Autorate Script

```bash
#!/bin/bash
# /opt/skygate/autorate.sh
# Custom CAKE autorate for Starlink (simplified from cake-autorate algorithm)
# Dependencies: fping, iproute2 (tc), bc

INTERFACE="eth0"              # Starlink uplink
REFLECTOR="1.1.1.1"          # Ping target
MIN_RATE_KBPS=5000            # 5 Mbps floor
BASE_RATE_KBPS=20000          # 20 Mbps baseline
MAX_RATE_KBPS=100000          # 100 Mbps ceiling
CURRENT_RATE=$BASE_RATE_KBPS
BASELINE_RTT_MS=40            # Starlink baseline RTT
THRESHOLD_MS=15               # Latency increase threshold
INCREASE_STEP_KBPS=5000       # 5 Mbps increase per cycle
DECREASE_FACTOR="0.7"         # Drop to 70% on bufferbloat detection
STABLE_COUNT=0
STABLE_THRESHOLD=5            # Cycles of stability before increasing

# Initialize CAKE
tc qdisc replace dev $INTERFACE root cake bandwidth ${CURRENT_RATE}kbit

while true; do
    # Measure RTT (3 pings, 200ms interval)
    RTT=$(fping -c 3 -p 200 -q $REFLECTOR 2>&1 | grep -oP 'avg = \K[0-9.]+' || echo "999")
    RTT_INT=${RTT%.*}

    EXCESS=$((RTT_INT - BASELINE_RTT_MS))

    if [ "$EXCESS" -gt "$THRESHOLD_MS" ]; then
        # Bufferbloat detected: decrease bandwidth
        CURRENT_RATE=$(echo "$CURRENT_RATE * $DECREASE_FACTOR" | bc | cut -d. -f1)
        [ "$CURRENT_RATE" -lt "$MIN_RATE_KBPS" ] && CURRENT_RATE=$MIN_RATE_KBPS
        STABLE_COUNT=0
        tc qdisc change dev $INTERFACE root cake bandwidth ${CURRENT_RATE}kbit
    else
        # Stable: count consecutive stable cycles
        STABLE_COUNT=$((STABLE_COUNT + 1))
        if [ "$STABLE_COUNT" -ge "$STABLE_THRESHOLD" ]; then
            # Ramp up toward max
            NEW_RATE=$((CURRENT_RATE + INCREASE_STEP_KBPS))
            [ "$NEW_RATE" -gt "$MAX_RATE_KBPS" ] && NEW_RATE=$MAX_RATE_KBPS
            if [ "$NEW_RATE" -ne "$CURRENT_RATE" ]; then
                CURRENT_RATE=$NEW_RATE
                tc qdisc change dev $INTERFACE root cake bandwidth ${CURRENT_RATE}kbit
            fi
            STABLE_COUNT=0
        fi
    fi

    sleep 2
done
```

### nftables Configuration (Full Phase 1)

```
#!/usr/sbin/nft -f
# /etc/nftables.conf -- SkyGate Phase 1

flush ruleset

table inet skygate {
    # Aviation bypass IP set (populated by Go daemon)
    set bypass_v4 {
        type ipv4_addr
        flags timeout
        timeout 1h
    }

    # Authenticated devices (MAC addresses) -- used by captive portal in Phase 2
    # Prepopulated: all devices authenticated in Phase 1 (no portal yet)
    set auth_devices {
        type ether_addr
    }

    chain input {
        type filter hook input priority filter; policy drop;

        # Allow established/related
        ct state established,related accept

        # Allow loopback
        iif "lo" accept

        # Allow DHCP on AP interface
        iifname "wlan1" udp dport 67 accept

        # Allow DNS on AP interface (to Pi-hole)
        iifname "wlan1" udp dport 53 accept
        iifname "wlan1" tcp dport 53 accept

        # Allow SSH from AP (development only -- remove in production)
        iifname "wlan1" tcp dport 22 accept

        # Allow ICMP
        ip protocol icmp accept

        # Drop everything else from passengers
        iifname "wlan1" drop
    }

    chain forward {
        type filter hook forward priority filter; policy accept;

        # Allow forwarding from AP to Starlink
        iifname "wlan1" oifname "eth0" accept

        # Allow established/related return traffic
        ct state established,related accept

        # Per-IP byte counters (for Phase 2 monitoring)
        iifname "wlan1" counter
        oifname "wlan1" counter
    }

    chain prerouting {
        type filter hook prerouting priority mangle;

        # Aviation bypass: direct routing via Starlink
        ip daddr @bypass_v4 meta mark set 0x1
        ip daddr @bypass_v4 ct mark set meta mark

        # Restore marks for existing connections
        ct mark 0x1 meta mark set ct mark

        # All other traffic: default route (via eth0 in Phase 1)
        # Phase 3 will add: mark 0x2 for WireGuard tunnel
    }

    chain postrouting {
        type nat hook postrouting priority srcnat;

        # NAT from AP to Starlink
        oifname "eth0" masquerade
    }
}
```

### Ansible Role Structure Example

```yaml
# pi/ansible/roles/networking/tasks/main.yml
---
- name: Install networking packages
  ansible.builtin.apt:
    name:
      - hostapd
      - iproute2
      - nftables
      - fping
      - conntrack
    state: present
    update_cache: true

- name: Disable NetworkManager for AP interface
  ansible.builtin.copy:
    dest: /etc/NetworkManager/conf.d/unmanaged.conf
    content: |
      [keyfile]
      unmanaged-devices=interface-name:{{ ap_interface }}
    mode: '0644'
  notify: restart networkmanager

- name: Configure static IP for AP interface
  ansible.builtin.template:
    src: ap-interface.nmconnection.j2
    dest: /etc/NetworkManager/system-connections/ap-static.nmconnection
    mode: '0600'

- name: Enable IP forwarding
  ansible.posix.sysctl:
    name: net.ipv4.ip_forward
    value: '1'
    sysctl_set: true
    reload: true

- name: Deploy hostapd configuration
  ansible.builtin.template:
    src: hostapd.conf.j2
    dest: /etc/hostapd/hostapd.conf
    mode: '0600'
  notify: restart hostapd

- name: Deploy nftables configuration
  ansible.builtin.template:
    src: nftables.conf.j2
    dest: /etc/nftables.conf
    mode: '0644'
  notify: restart nftables

- name: Configure policy routing
  ansible.builtin.template:
    src: skygate-routes.sh.j2
    dest: /opt/skygate/setup-routes.sh
    mode: '0755'

- name: Enable and start services
  ansible.builtin.systemd:
    name: "{{ item }}"
    enabled: true
    state: started
  loop:
    - hostapd
    - nftables
```

### Pi-hole Blocklist Selection (Conservative Default)

Per D-10 (conservative out of the box), use these lists only:

```yaml
# pi/config/blocklists.yaml
blocklists:
  # Steven Black Unified (ads + malware) -- 90K domains, well-maintained
  - url: "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
    name: "StevenBlack Unified"
    enabled: true

  # OISD Light -- curated, low false-positive rate
  - url: "https://small.oisd.nl/"
    name: "OISD Light"
    enabled: true

  # Do NOT include aggressive lists by default:
  # - EasyList (too many false positives for aviation)
  # - OISD Full (too aggressive, blocks cloud services)
  # - Energized (blocks legitimate CDNs)
```

### Aviation Bypass Domain List

```yaml
# pi/config/bypass-domains.yaml
# Domains that MUST route directly to Starlink, never through proxy or DNS blocking.
# Safety-critical: false negatives here degrade pilot situational awareness.

bypass_domains:
  # ForeFlight
  - "foreflight.com"
  - "*.foreflight.com"

  # Garmin Pilot / Garmin Aviation
  - "garmin.com"
  - "*.garmin.com"
  - "fly.garmin.com"

  # FAA / Aviation Weather
  - "aviationweather.gov"
  - "*.aviationweather.gov"
  - "tfr.faa.gov"
  - "notams.faa.gov"
  - "*.weather.gov"

  # ADS-B Services
  - "flightaware.com"
  - "*.flightaware.com"
  - "flightradar24.com"
  - "*.flightradar24.com"
  - "adsbexchange.com"
  - "*.adsbexchange.com"

  # AOPA
  - "aopa.org"
  - "*.aopa.org"

  # FltPlan Go
  - "fltplan.com"
  - "*.fltplan.com"

  # SkyVector / VFR charts
  - "skyvector.com"
  - "*.skyvector.com"

  # Apple/Google connectivity checks (required for captive portal detection in Phase 2)
  - "captive.apple.com"
  - "connectivitycheck.gstatic.com"
  - "www.msftconnecttest.com"
  - "www.msftncsi.com"
```

### First-Boot Setup (Minimal)

```bash
#!/bin/bash
# /opt/skygate/firstboot.sh
# Runs once on first boot. Sets SSID and password, then disables itself.

FIRSTBOOT_FLAG="/data/skygate/.firstboot-complete"
if [ -f "$FIRSTBOOT_FLAG" ]; then
    exit 0
fi

# Generate random password (8 alphanumeric chars)
DEFAULT_PASSWORD=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 8)

# For Phase 1: prompt via serial console or use defaults
# Phase 2+ will add a web-based wizard

echo "============================================"
echo "  SkyGate First Boot Setup"
echo "============================================"
echo ""
read -p "Enter WiFi network name (SSID): " SSID
if [ -z "$SSID" ]; then
    SSID="SkyGate"
fi

read -p "Enter WiFi password [${DEFAULT_PASSWORD}]: " PASSWORD
if [ -z "$PASSWORD" ]; then
    PASSWORD="$DEFAULT_PASSWORD"
    echo "Using generated password: $PASSWORD"
    echo "Write this down! It will be needed to connect devices."
fi

# Update hostapd config (remount rw temporarily if needed)
mount -o remount,rw /
sed -i "s/^ssid=.*/ssid=${SSID}/" /etc/hostapd/hostapd.conf
sed -i "s/^wpa_passphrase=.*/wpa_passphrase=${PASSWORD}/" /etc/hostapd/hostapd.conf
mount -o remount,ro /

# Save credentials to writable partition for reference
echo "SSID=${SSID}" > /data/skygate/wifi-credentials
echo "PASSWORD=${PASSWORD}" >> /data/skygate/wifi-credentials
chmod 600 /data/skygate/wifi-credentials

# Mark first boot complete
touch "$FIRSTBOOT_FLAG"

# Restart hostapd with new config
systemctl restart hostapd

echo ""
echo "Setup complete! Devices can now connect to: ${SSID}"
echo "============================================"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| iptables + ipset | nftables + native sets | Debian 11+ (2021) | nftables is default on Bookworm. iptables-nft compatibility layer exists but adds confusion. |
| dhcpcd for networking | NetworkManager | Bookworm (2023) | hostapd users must explicitly exclude AP interface from NM. |
| Pi-hole v5 (separate dnsmasq) | Pi-hole v6 (FTL embeds dnsmasq) | 2024-2025 | No more separate dnsmasq config. Custom config via `/etc/dnsmasq.d/` with `misc.etc_dnsmasq_d=true` in pihole.toml. |
| ipset utility | nftables native sets (nftset in dnsmasq) | dnsmasq 2.87+ (2022) | `--nftset` replaces `--ipset`. But Pi-hole FTL does NOT compile with HAVE_NFTSET. |
| fq_codel for QoS | CAKE qdisc | Kernel 4.19+ (2018) | CAKE handles per-flow fairness + bandwidth shaping in one qdisc. Superior for variable-rate links. |
| pi-gen for custom images | rpi-image-gen | 2025 | Official Raspberry Pi tool. YAML-based config. Faster builds. Not needed for Phase 1 (Ansible deploys to stock image). |

**Deprecated/outdated:**
- `iptables` (use nftables)
- `dhcpcd.conf` (use NetworkManager on Bookworm)
- `ipset` utility (use nftables native sets)
- `wpa_supplicant.conf` (use nmcli on Bookworm)

## Open Questions

1. **Pi-hole v6 DHCP stability on custom interface**
   - What we know: Pi-hole v6 has a built-in DHCP server configurable via pihole.toml. Some users report the DHCP config resetting to inactive on updates.
   - What's unclear: Whether Pi-hole DHCP is rock-solid on a hostapd-managed interface (wlan1) rather than the typical eth0 scenario.
   - Recommendation: Test thoroughly. Fallback is standalone dnsmasq for DHCP (disable Pi-hole DHCP, run dnsmasq with `--except-interface=lo` on a non-standard port).

2. **OverlayFS + Pi-hole gravity database updates**
   - What we know: Pi-hole updates its gravity database via `pihole -g` (weekly cron). This writes to `/etc/pihole/gravity.db`.
   - What's unclear: Whether the symlink-to-data-partition approach works cleanly with Pi-hole v6's FTL database expectations.
   - Recommendation: Test early. Store `/etc/pihole/` on the data partition via bind mount or symlink. If problematic, temporarily remount rw for gravity updates.

3. **cake-autorate Starlink-specific tuning**
   - What we know: The 3-parameter algorithm (min/base/max rate) is sound. Starlink baseline RTT is ~40-60ms with 15s satellite switch spikes.
   - What's unclear: Optimal threshold values, decrease factor, and ramp-up timing for aviation Starlink (which may differ from ground Starlink).
   - Recommendation: Start with conservative defaults (in code example above). Tune with real Starlink data from remote friend (D-04).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go compiler | Bypass daemon cross-compile | Yes | 1.26.1 (darwin/arm64) | -- |
| Docker | QEMU cross-testing (optional) | Yes | 28.3.3 | Test directly on Pi |
| Ansible | Pi provisioning (D-02) | **No** | -- | `pip3 install ansible` |
| SSH | Pi deployment | Yes | 10.2p1 | -- |
| fping | Autorate testing | Yes | installed | Only needed on Pi |
| tcpdump | Network debugging | Yes | 4.99.1 | -- |
| QEMU (aarch64) | ARM64 testing on Mac | **No** | -- | Test directly on Pi via SSH |
| Wireshark | Network analysis | **No** | -- | tcpdump (available) |
| iperf3 | Throughput testing | **No** | -- | `brew install iperf3` |
| Raspberry Pi 5 | Integration testing | **Unknown** | -- | Must acquire hardware |
| MT7612U USB adapter | AP testing | **Unknown** | -- | Must acquire hardware |
| Starlink connection | Real-world testing (D-04) | Available (remote friend) | -- | tc netem simulation |

**Missing dependencies with no fallback:**
- **Ansible**: Must install (`pip3 install ansible ansible-lint`) -- locked decision D-02
- **Raspberry Pi 5 + MT7612U adapter**: Must acquire hardware for integration testing

**Missing dependencies with fallback:**
- **QEMU aarch64**: Not installed, but can test directly on Pi via SSH (preferred anyway)
- **Wireshark**: Not installed, tcpdump available as alternative
- **iperf3**: Not installed, `brew install iperf3` to add

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (daemon) + BATS (bash scripts) + Ansible `--check` + integration shell scripts |
| Config file | None yet -- Wave 0 creates go.mod, test files, and BATS setup |
| Quick run command | `go test ./... -short` (Go) + `bats pi/scripts/tests/` (bash) |
| Full suite command | `make test` (runs Go + BATS + Ansible lint) |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NET-01 | WiFi AP accepts connections, WPA2 auth | integration (Pi) | `ssh pi 'hostapd_cli status && iwconfig wlan1'` | No -- Wave 0 |
| NET-02 | DHCP assigns IPs, DNS routes to Pi-hole | integration (Pi) | `ssh pi 'cat /var/lib/misc/dnsmasq.leases && dig @192.168.4.1 google.com'` | No -- Wave 0 |
| DNS-01 | Pi-hole blocks known ad domains | integration (Pi) | `ssh pi 'dig @192.168.4.1 doubleclick.net \| grep NXDOMAIN'` | No -- Wave 0 |
| ROUTE-01 | Bypass domains resolve and populate nftsets | unit (Go) + integration (Pi) | `go test ./cmd/bypass-daemon/ -run TestResolveAndPopulate -short` | No -- Wave 0 |
| ROUTE-01 | Bypass traffic gets fwmark 0x1, routes via eth0 | integration (Pi) | `ssh pi 'nft list set inet skygate bypass_v4 && ip rule show'` | No -- Wave 0 |
| QOS-01 | CAKE qdisc is active on eth0 | integration (Pi) | `ssh pi 'tc qdisc show dev eth0 \| grep cake'` | No -- Wave 0 |
| QOS-01 | Autorate adjusts bandwidth on latency change | unit (bash/BATS) | `bats pi/scripts/tests/test_autorate.bats` | No -- Wave 0 |
| -- | Read-only root survives power loss | manual + integration | `ssh pi 'mount \| grep "on / " \| grep ro'` | No -- Wave 0 |
| -- | Ansible playbook is idempotent | lint + check | `ansible-playbook pi/ansible/playbook.yml --check --diff` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./... -short && ansible-lint pi/ansible/`
- **Per wave merge:** `make test` (full suite including integration if Pi available)
- **Phase gate:** All integration tests pass on physical Pi before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `go.mod` -- Go module initialization
- [ ] `cmd/bypass-daemon/main_test.go` -- unit tests for DNS resolution and nftables set management
- [ ] `pi/scripts/tests/test_autorate.bats` -- BATS tests for autorate script logic
- [ ] `pi/ansible/playbook.yml` -- skeleton playbook for `ansible-lint`
- [ ] `Makefile` -- top-level build/test/deploy targets
- [ ] Install BATS on dev machine: `brew install bats-core`
- [ ] Install Ansible: `pip3 install ansible ansible-lint`

## Sources

### Primary (HIGH confidence)
- [Pi-hole FTL HAVE_NFTSET issue #1508](https://github.com/pi-hole/FTL/issues/1508) -- confirmed NOT compiled with nftset support, closed as "not planned"
- [Pi-hole v6 custom dnsmasq configuration](https://discourse.pi-hole.net/t/dnsmasq-custom-configurations-in-v6/68469) -- `misc.etc_dnsmasq_d=true` in pihole.toml
- [morrownr/USB-WiFi](https://github.com/morrownr/USB-WiFi/blob/main/home/USB_WiFi_Adapters_that_are_supported_with_Linux_in-kernel_drivers.md) -- MT7612U recommended for AP mode, RTL8812BU warned against
- [morrownr/7612u](https://github.com/morrownr/7612u) -- MT7612U in-kernel driver since 4.19, first-class AP mode support
- [Domain-based split tunneling with WireGuard](https://starsandmanifolds.xyz/blog/domain-based-split-tunneling-using-wireguard) -- complete nftables + fwmark + policy routing implementation
- [dnsmasq + nftables](https://www.monotux.tech/posts/2024/08/dnsmasq-netfilter/) -- nftset syntax: `nftset=/domain/4#inet#table#set`
- [nftables wiki: moving from ipset](https://wiki.nftables.org/wiki-nftables/index.php/Moving_from_ipset_to_nftables) -- native set replacement for ipset
- [tc-cake man page](https://man7.org/linux/man-pages/man8/tc-cake.8.html) -- CAKE qdisc documentation
- [cake-autorate GitHub](https://github.com/lynxthecat/cake-autorate) -- OpenWrt/Asus Merlin only, NOT Raspberry Pi OS
- [Chris Dzombak: read-only Raspberry Pi](https://www.dzombak.com/blog/2024/03/running-a-raspberry-pi-with-a-read-only-root-filesystem/) -- detailed OverlayFS + tmpfs + fstab guide
- [Raspberry Pi Forums: OverlayFS Bookworm issue](https://github.com/raspberrypi/bookworm-feedback/issues/137) -- OverlayFS makes all partitions read-only

### Secondary (MEDIUM confidence)
- [Raspberry Pi AP on Bookworm](https://raspberrytips.com/access-point-setup-raspberry-pi/) -- NetworkManager vs hostapd coexistence
- [Pi-hole + hostapd integration](https://amedeos.github.io/hostapd/2020/05/21/hostapd-and-pihole-a-perfect-union.html) -- practical setup guide
- [Ansible Raspberry Pi best practices](https://opensource.com/article/20/9/raspberry-pi-ansible) -- fleet management patterns
- [CAKE for Starlink](https://www.bufferbloat.net/projects/codel/wiki/Cake/) -- Bufferbloat.net CAKE documentation
- [lradaelli85/dnsmasq-nftset](https://github.com/lradaelli85/dnsmasq-nftset) -- dnsmasq nftset domain blocking reference

### Tertiary (LOW confidence)
- [Pi-hole v6 DHCP instability reports](https://discourse.pi-hole.net/t/pi-hole-v6-dhcp-server-configuration-keeps-being-disabled/81123) -- anecdotal, may be fixed in 6.3+
- cake-autorate Starlink-specific tuning values -- no authoritative source for aviation Starlink; ground Starlink values used as starting point

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all components are well-established Linux networking tools with years of production use
- Architecture: HIGH -- policy routing with fwmark is the canonical Linux split-routing pattern; Pi-hole FTL nftset limitation is verified
- Pitfalls: HIGH -- SD card corruption, WiFi instability, NM conflicts all well-documented with clear mitigations
- QoS tuning: MEDIUM -- CAKE is proven, but Starlink-specific autorate values need empirical tuning
- Pi-hole v6 DHCP: MEDIUM -- v6 is relatively new, edge cases on custom interfaces not fully documented

**Research date:** 2026-03-23
**Valid until:** 2026-05-23 (60 days -- stable Linux networking stack, slow-moving ecosystem)
