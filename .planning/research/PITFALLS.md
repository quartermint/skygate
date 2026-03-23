# Pitfalls Research

**Domain:** Pi-based network appliance for aviation satellite bandwidth management
**Researched:** 2026-03-22
**Confidence:** HIGH (hardware/networking pitfalls well-documented in community), MEDIUM (Starlink aviation-specific behavior)

## Critical Pitfalls

### Pitfall 1: SD Card Corruption from Power Loss

**What goes wrong:**
The Pi's SD card filesystem corrupts when power is cut without clean shutdown. In an aircraft, power loss happens every flight -- engine off, master switch off, done. A normally configured Raspberry Pi OS caches writes in RAM (especially on 4GB+ models) and a sudden power cut leaves filesystem structures corrupted. After a few flights, the appliance bricks itself. This is the single most common failure mode for Pi-based appliances in the field.

**Why it happens:**
Raspberry Pi OS defaults to ext4 with write-back caching. The OS accumulates dirty pages in RAM for performance. Aircraft power is binary -- avionics master switch goes off and 12V bus drops instantly. There is no graceful shutdown signal. Pilots will not SSH in and run `sudo shutdown -h now` before cutting the master.

**How to avoid:**
- Use read-only root filesystem with OverlayFS and tmpfs overlay from day one. Raspberry Pi OS has built-in support via `raspi-config > Performance Options > Overlay File System`. All writes go to RAM and are discarded on reboot -- corruption is impossible.
- Store mutable data (usage logs, config changes) on a separate small partition with `sync` mount option, or use SQLite in WAL mode on a dedicated partition with `noatime,data=journal` mount options.
- Test by pulling power 50+ times during operation and verifying clean boot every time.

**Warning signs:**
- Any `fsck` errors in boot logs
- Appliance fails to boot after a flight
- "Read-only filesystem" errors appearing in logs (this is the kernel's emergency response to detected corruption)

**Phase to address:**
Phase 1 (OS image build). The SD card image must ship with read-only root. Retrofitting read-only root onto a working system is painful -- design for it from the start.

---

### Pitfall 2: Onboard WiFi Chip Instability as Access Point

**What goes wrong:**
The Pi's onboard Broadcom WiFi chip is unreliable as a production access point. Users experience random disconnections (`AP-STA-DISCONNECTED` events), limited range (3-5 meters in some configurations), failure to respond to client keep-alive probes, and throughput degradation under load. In an aircraft cabin -- even a small GA cabin -- this means passengers get dropped from the network mid-flight.

**Why it happens:**
The onboard WiFi was designed for client connectivity, not AP duty. It has a PCB trace antenna with poor gain, limited firmware support for AP features, and the driver has known bugs with inactivity detection (the Pi never responds to empty frames from clients). Simultaneous AP+STA mode on a single radio is officially "unsupported" per RaspAP documentation and locks both interfaces to the same channel.

**How to avoid:**
- Use an external USB WiFi adapter (with proper antenna) for the AP function and the onboard WiFi or Ethernet for the uplink to Starlink. Do not try to run AP and client on the same radio.
- Select a USB adapter with confirmed `nl80211` AP mode support and a chipset with good Linux driver history (MediaTek MT7612U or Realtek RTL8812BU with community drivers). Avoid cheap Realtek adapters -- some have TX power issues limiting range to centimeters.
- For aircraft installation: Starlink Mini has an Ethernet adapter available. Use Ethernet (wired) for the Pi-to-Starlink connection. This eliminates the need for dual WiFi entirely -- one USB WiFi adapter for AP, Ethernet for uplink.

**Warning signs:**
- Devices disconnecting and reconnecting during testing
- `hostapd` logs showing `AP-STA-DISCONNECTED due to inactivity`
- Signal strength below -70 dBm at cabin distances
- Docker running on the Pi (known to conflict with wireless AP mode)

**Phase to address:**
Phase 1 (hardware architecture). USB WiFi adapter selection and Ethernet uplink to Starlink must be decided before any software work. Wrong hardware choice here requires complete rewiring.

---

### Pitfall 3: Captive Portal Detection Incompatibility Across iOS/Android

**What goes wrong:**
iOS and Android detect captive portals differently, and getting both to reliably trigger the captive portal popup (CNA on iOS, connectivity check on Android) is notoriously difficult. iOS probes `http://captive.apple.com/hotspot-detect.html`, Android probes `http://connectivitycheck.gstatic.com/generate_204`, and Windows probes `http://www.msftconnecttest.com/connecttest.txt`. If the captive portal implementation does not correctly intercept and respond to ALL of these probes, the result is: iOS shows "no internet" and refuses to join, Android connects but never shows the portal, or infinite redirect loops that frustrate non-technical pilots.

**Why it happens:**
Each platform has different expectations for the HTTP response. iOS expects a specific HTML body from its probe URL. Android expects a 204 response from its probe URL and interprets anything else as "captive portal detected." iOS 14+ also supports RFC 8910 (DHCP option 114) for captive portal URI discovery, but adoption is inconsistent. HTTPS probe URLs cannot be intercepted without MITM, causing certificate errors instead of portal redirects.

**How to avoid:**
- Intercept all known captive portal detection URLs via dnsmasq DNS hijacking (resolve `captive.apple.com`, `connectivitycheck.gstatic.com`, `www.msftconnecttest.com` to the Pi's IP).
- Serve platform-appropriate responses: return the expected HTML for Apple probes, return non-204 for Android probes.
- Implement DHCP option 114 (RFC 8910) for iOS 14+ devices as a belt-and-suspenders approach.
- Test with physical iOS and Android devices on every OS version you can find. Simulators do not accurately reproduce CNA behavior.
- Do NOT attempt HTTPS interception for captive portal detection -- it will cause certificate warnings, not portal popups.

**Warning signs:**
- Portal works on one platform but not the other during testing
- Devices connect to WiFi but show "no internet" badge
- Portal appears but then immediately closes (iOS CNA timeout issue)
- Users report they "can't get online" despite being connected

**Phase to address:**
Phase 2 (captive portal implementation). This must be tested on real hardware across iOS 16-18 and Android 13-15 before shipping. Budget significant testing time.

---

### Pitfall 4: DNS Blocking Breaks Aviation Safety Apps

**What goes wrong:**
Aggressive DNS blocklists (Pi-hole's default lists, EasyList, etc.) block domains that ForeFlight, Garmin Pilot, FltPlan Go, and other aviation apps depend on. A pilot loses weather updates, TFR notifications, or ADS-B traffic mid-flight because the bandwidth management appliance blocked a CDN or API domain the aviation app uses. This is not just a UX problem -- it is a safety problem. An appliance that degrades pilot situational awareness is worse than no appliance at all.

**Why it happens:**
Aviation apps use the same CDNs (Akamai, CloudFront, Fastly) as ad networks and trackers. Blocklists are built for home use and have no concept of "this domain serves both ads AND safety-critical weather data." ForeFlight's tile server, METAR/TAF sources, and D-ATIS APIs may share infrastructure with services that appear on blocklists. The PROJECT.md correctly identifies "aviation app bypass" as a requirement, but the devil is in the implementation.

**How to avoid:**
- Build and maintain an aviation app allowlist that takes absolute priority over any blocklist. Start with known domains for: ForeFlight (`foreflight.com`, `*.foreflight.com`), Garmin Pilot, FltPlan Go, `aviationweather.gov`, `tfr.faa.gov`, ADS-B Exchange, and AOPA.
- Use policy-based routing to send allowlisted aviation domains directly to Starlink, bypassing the proxy entirely (already in the PROJECT.md requirements -- prioritize this).
- Ship with a conservative default blocklist (ads + trackers only), not an aggressive one. No video CDN blocking by default -- let pilots opt in.
- Provide a "panic button" in the dashboard: one-tap disable of all filtering that routes everything direct for the rest of the flight.
- Log blocked domains prominently in the dashboard so pilots can self-diagnose and allowlist.

**Warning signs:**
- ForeFlight or Garmin Pilot showing "no data" or stale weather after connecting to SkyGate
- Pilot reports of "my EFB stopped updating"
- Blocked domain logs showing `*.foreflight.com`, `*.garmin.com`, `aviationweather.gov`, or any `.gov` domain

**Phase to address:**
Phase 1 (DNS filtering). The aviation allowlist is not a nice-to-have -- it must ship in the very first version. This is the one category of false positive that could kill the product (and theoretically endanger a pilot).

---

### Pitfall 5: Starlink In-Motion Speed Cap Enforcement Renders Device Useless

**What goes wrong:**
Starlink enforces a 100 mph (87 knot) speed cap on standard Roam plans as of March 2026. Most GA aircraft exceed this speed immediately after takeoff. If a pilot is on a Roam plan (not the $250-$1000/mo aviation plan), SkyGate has zero data to manage -- Starlink itself cuts the connection. The pilot blames SkyGate for "not working" even though the problem is Starlink plan restrictions.

**Why it happens:**
Starlink detects speed via GPS in the Dishy and enforces plan-level speed limits server-side. There is no workaround -- this is not a client-side check. Pilots who bought Starlink Mini before March 2026 may not realize their plan changed. The SkyGate value proposition assumes the pilot has a working Starlink connection in flight.

**How to avoid:**
- Display Starlink connection status prominently on the dashboard. If the connection drops or degrades, show a clear message: "Starlink connection lost -- check your plan supports in-flight speeds."
- Document clearly in setup instructions: "SkyGate requires a Starlink Aviation plan ($250+/mo). Standard Roam plans do not work above 100 mph."
- Implement connection monitoring that distinguishes "Starlink is down" from "SkyGate is broken" -- this is critical for support and reputation.
- Consider adding a ground-mode / taxi-mode that is useful even without in-flight connectivity (pre-flight data sync, ground-based WiFi sharing).

**Warning signs:**
- Users reporting "device doesn't work" with no further detail
- Dashboard showing zero throughput after takeoff
- Support tickets from users on Roam plans

**Phase to address:**
Phase 2 (dashboard). The connection status display and plan compatibility messaging must be prominent. Also address in documentation and marketing from day one.

---

### Pitfall 6: WireGuard Tunnel Instability Over Satellite Link

**What goes wrong:**
The WireGuard tunnel to the remote proxy server drops during satellite handoffs (every ~15 seconds as LEO satellites pass overhead), during attitude changes (banking turns that temporarily obstruct the antenna), and during transient outages. Each reconnection causes a CPU spike on the Pi, latency spikes across all connections, and potentially drops active TCP sessions being proxied. If the tunnel drops and the fallback to direct routing is not instantaneous, passengers see "loading" spinners for 5-30 seconds.

**Why it happens:**
Starlink LEO satellites move at 27,000 km/h relative to ground. Handoffs between satellites cause 30-50ms latency spikes and occasional packet loss (~1% average, with bursts). WireGuard uses UDP and is generally resilient, but the proxy server sees the tunnel as unreachable during brief outages. If routes are not properly configured, traffic blackholes during reconnection. Default MTU settings are wrong for satellite+VPN encapsulation.

**How to avoid:**
- Set WireGuard MTU to 1280 (conservative) or calculate precisely: Starlink MTU (typically 1500) minus WireGuard overhead (60 bytes for IPv4, 80 for IPv6) = 1420-1440. Test empirically.
- Implement instant fallback: if the WireGuard tunnel is unreachable for >2 seconds, route all traffic directly to Starlink (DNS filtering still works, proxy does not). This is the "degraded mode" from PROJECT.md -- make it seamless, not error-prone.
- Use WireGuard's `PersistentKeepalive = 25` to maintain NAT mappings through Starlink's network.
- Do not run TCP-based VPN protocols over satellite -- WireGuard's UDP is correct. Do not be tempted by OpenVPN.
- Pre-configure the remote proxy server's WireGuard to handle brief peer disappearances gracefully (no aggressive timeout).

**Warning signs:**
- Dashboard showing tunnel "flapping" (up/down/up/down)
- Users experiencing periodic 5-10 second freezes while browsing
- `dmesg` showing MTU-related ICMP errors
- High CPU usage during tunnel reconnection events

**Phase to address:**
Phase 3 or 4 (WireGuard + proxy integration). The tunnel resilience and fallback logic is the hardest networking problem in the project. Do not rush it.

---

### Pitfall 7: MITM Proxy Breaks Certificate-Pinned Apps

**What goes wrong:**
Installing the SkyGate CA certificate on a passenger's device and running MITM proxy breaks any app that uses certificate pinning -- banking apps (Chase, Wells Fargo), two-factor auth apps (Google Authenticator, Duo), health apps (MyChart), payment apps (Venmo, PayPal), and some social media apps. The app shows "connection not secure" or silently fails. Passengers lose access to critical apps because they installed the "Max Savings" CA cert.

**Why it happens:**
Certificate pinning is a security feature that rejects any CA not explicitly trusted by the app, regardless of whether the OS trusts it. This is by design -- these apps are protecting against exactly the kind of MITM proxy SkyGate implements. There is no workaround short of modifying the app binary (which requires jailbreak/root and is impractical). The PROJECT.md already acknowledges this with the "certificate pinning bypass list" requirement, but maintaining an accurate list is an ongoing battle.

**How to avoid:**
- Default to "Quick Connect" mode (DNS filtering only, no MITM). Only offer "Max Savings" as an opt-in with a clear warning: "Some apps (banking, payments) may not work in Max Savings mode."
- Maintain a bypass domain list for known cert-pinned services. This list will never be complete. Ship with a generous initial list and let pilots add domains.
- Make it trivially easy to switch modes mid-flight from the dashboard. A pilot who installed the CA cert and finds their banking app broken needs a one-tap fix, not SSH access.
- Never, ever MITM `.gov` domains, banking domains, or health domains. Hardcode these as bypass-always.
- Document the CA cert removal process clearly -- passengers need to uninstall it after the flight.

**Warning signs:**
- Passengers reporting "my bank app doesn't work"
- Apps showing SSL/TLS errors
- Increasing support requests about specific apps not working
- Passengers reluctant to install a CA cert from an unknown device (trust problem)

**Phase to address:**
Phase 4 or later (MITM proxy). This is why the two-layer approach (DNS first, MITM optional) is correct. Do not ship MITM until DNS-only mode is proven and trusted.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hardcoded aviation app allowlist | Ships fast, covers 90% of pilots | Breaks when apps change domains; no user customization | MVP only -- must add user-editable allowlist by v0.2 |
| Running root filesystem read-write | Easier development, simpler logging | SD card corruption within weeks of field use | Never in shipped images. Development SD cards only |
| Single WiFi radio for AP+uplink | Fewer hardware components, lower cost | Unreliable AP, single-channel lock, drops under load | Never. Budget the USB adapter from day one |
| No tunnel fallback logic | Simpler routing, fewer states to test | Complete loss of browsing when WireGuard drops | Never, but tunnel feature itself can be deferred |
| Static CAKE bandwidth settings | No autorate complexity | Wrong settings cause either bufferbloat or throughput waste | Early testing only -- Starlink bandwidth varies too much |
| Storing usage data in flat files | No database dependency | Slow queries, data loss on power cut, no aggregation | Early prototype only -- move to SQLite WAL quickly |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Starlink Mini Ethernet adapter | Assuming WiFi bridge mode works reliably for uplink | Use Ethernet adapter for wired connection. Set Starlink router to bypass mode. Wired uplink is far more reliable than WiFi client mode on the Pi |
| Pi-hole + dnsmasq | Running Pi-hole's built-in dnsmasq alongside a separate dnsmasq for DHCP/captive portal | Use Pi-hole's integrated DHCP server or configure Pi-hole to use an external dnsmasq. Two dnsmasq instances fighting over port 53 is a common beginner mistake |
| cake-autorate + Starlink | Setting static min/max bandwidth values based on speed tests | Use cake-autorate's dynamic adjustment. Starlink bandwidth varies 10x (20-200 Mbps) depending on congestion, weather, and satellite position. Static values are always wrong |
| hostapd + NetworkManager | Leaving NetworkManager running while hostapd manages the WiFi interface | Disable NetworkManager for the AP interface entirely. Multiple network managers fighting over the same interface causes random failures |
| WireGuard + Starlink NAT | Not setting PersistentKeepalive | Starlink's CGNAT drops idle UDP mappings aggressively. PersistentKeepalive = 25 is mandatory |
| compy proxy + HTTPS | Expecting compy to compress HTTPS without MITM | compy can only compress HTTP by default. HTTPS compression requires MITM mode with CA cert. DNS-level blocking is the only option for HTTPS without CA cert installation |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Pi thermal throttling in enclosed case | CPU frequency drops from 1.8 GHz to 600 MHz, proxy latency spikes, dashboard becomes sluggish | Passive heatsink + ventilated case design. Pi 5 throttles at 80-85C. PETG case needs ventilation slots. Aluminum case acts as heatsink | Sustained load in warm cabin (30C+ ambient) with sealed case |
| Too many Pi-hole blocklist entries | DNS resolution slows, Pi-hole FTL process consumes all RAM, queries time out | Limit to 2-3 well-maintained lists (Steven Black's unified + aviation-specific). Do not stack 10+ community lists | 500K+ blocked domains on Pi 4 with 2GB RAM |
| Image transcoding blocking the event loop | Page loads hang while compy recompresses large images, all proxy traffic stalls | Set 500ms timeout on image transcoding (already in PROJECT.md). Serve original if transcoding exceeds timeout. Process images in goroutine pool, not inline | Images larger than 2MB on Pi 4, or more than 3 concurrent transcoding operations |
| SQLite write contention for usage logging | Dashboard queries block data ingestion, or vice versa. Usage data gaps during heavy traffic | Use WAL mode. Separate reader and writer connections. Batch insert usage records (every 5 seconds, not per-packet) | 5+ connected devices with active browsing |
| dnsmasq DNS cache overflow | Old queries evicted, repeated upstream lookups, increased latency | Set `cache-size=10000` (default is 150). Starlink upstream DNS is slow during handoffs -- local cache is critical | More than 150 unique domains queried (happens within minutes of normal browsing) |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Shipping the CA private key on the SD card image | Anyone who downloads the image has the CA key. They can MITM any device that installed the SkyGate cert. Complete compromise of the MITM security model | Generate unique CA keypair per device on first boot. Never ship a shared CA key. Store the CA private key with restrictive permissions (0600, root only) |
| No authentication on the dashboard | Anyone on the WiFi can view all device usage data and modify settings. A passenger could disable DNS filtering or change QoS rules | Require a PIN/password for admin functions. Display-only usage stats can be public, but config changes must be authenticated |
| Logging passenger browsing data persistently | Privacy liability. Even aggregated domain logs could expose sensitive browsing. Legal risk under various privacy laws | Log only aggregate category data (bytes by category), not individual domains per device. Auto-purge detailed logs after each flight. Make logging granularity configurable |
| MITM CA cert without clear consent flow | Passengers may not understand they are installing a root CA that allows interception of all HTTPS traffic. Legal and ethical issues | Explicit opt-in with plain-language explanation. "Quick Connect" (no MITM) must be the default. CA cert installation is an active choice, never automatic |
| Exposing Pi SSH on the passenger WiFi network | Passengers can attempt to SSH into the appliance. Default Pi passwords are well-known | Firewall rules: block all traffic from passenger WLAN to Pi management ports (22, 80/admin). Only allow access from Ethernet/localhost. Change default passwords in the image build |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Requiring SSH for any configuration | Non-technical GA pilots cannot use the device. Kills adoption entirely | Every setting must be accessible via the web dashboard. SSH is for developers only, never mentioned in user docs |
| Captive portal with too many steps | Pilots and passengers abandon setup. They switch to phone hotspot instead | Maximum 2 taps: accept terms, done. "Quick Connect" vs "Max Savings" is the only choice. No registration, no email, no account creation |
| No clear indication of data savings | Pilot has no evidence the device is working. Feels like a box doing nothing | Show real-time savings prominently: "12.3 GB saved this flight" with before/after comparison. The pie chart is the hook -- make it impossible to miss |
| Dashboard only accessible via IP address | Pilots don't remember `192.168.4.1`. They try `skygate.local` or just give up | Implement mDNS (`skygate.local`) AND DNS hijacking (any domain typed in browser redirects to dashboard when not authenticated). Both are needed |
| Error messages showing Linux/networking jargon | "dnsmasq: DHCP packet received on wlan0 which has no address" means nothing to a pilot | Translate every error to plain English with an action: "WiFi connection issue -- try restarting SkyGate by unplugging for 10 seconds" |

## "Looks Done But Isn't" Checklist

- [ ] **WiFi AP:** Tested with 5+ simultaneous devices, not just 1. Throughput and stability degrade with client count
- [ ] **Captive portal:** Tested on physical iOS 16/17/18 AND Android 13/14/15 devices. Simulator behavior differs from real CNA
- [ ] **DNS blocking:** Tested with ForeFlight, Garmin Pilot, and FltPlan Go actively downloading weather, charts, and TFRs while filtering is on
- [ ] **Read-only root:** Pulled power 50+ times during active operation. Verified clean boot every time. Not just "I enabled it and it booted once"
- [ ] **Dashboard data persistence:** Verified usage data survives power cycle (stored on writable partition, not tmpfs overlay)
- [ ] **WireGuard tunnel:** Tested with simulated packet loss (5%) and latency spikes (200ms+). Not just tested on clean LAN
- [ ] **CAKE QoS:** Verified with actual Starlink connection, not wired broadband simulating satellite. Starlink's bandwidth variability is unique
- [ ] **Thermal performance:** Ran under sustained load for 2+ hours in the 3D printed case. Measured CPU temp, verified no throttling
- [ ] **12V power supply:** Verified the buck converter provides stable 5V/3A+ under load. Cheap converters cause brownouts that look like software bugs
- [ ] **Captive portal:** Verified portal still works when there is NO upstream internet (Starlink not connected yet). Portal must work offline for terms acceptance

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| SD card corruption | LOW | Reflash from published image. All config is in a small config partition or generated on first boot. No unique state on the SD card |
| WiFi AP instability (wrong adapter) | MEDIUM | Purchase correct USB WiFi adapter, update hostapd config. Hardware swap but software is portable |
| Captive portal not working on iOS | MEDIUM | Implement RFC 8910 DHCP option 114, fix probe URL responses. Requires testing matrix across iOS versions |
| DNS blocking aviation apps | LOW | Add domains to allowlist via dashboard. Immediate fix, no restart needed |
| WireGuard tunnel flapping | MEDIUM | Tune MTU, adjust PersistentKeepalive, implement fallback routing. Requires networking expertise to diagnose |
| MITM breaking cert-pinned apps | LOW | Add domain to bypass list. Immediate fix per user report |
| Thermal throttling | MEDIUM | Redesign case with ventilation, add heatsink. Hardware change required |
| CA key compromised (shared key shipped) | HIGH | Must revoke CA, generate new per-device keys, have all users reinstall cert. Reputation damage |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| SD card corruption | Phase 1 (OS image) | Pull power 50 times during operation, verify clean boot |
| WiFi AP instability | Phase 1 (hardware) | 5+ devices connected for 2 hours, zero disconnects |
| Captive portal cross-platform | Phase 2 (captive portal) | Test matrix: iOS 16/17/18 + Android 13/14/15 + Windows 11 |
| DNS blocking aviation apps | Phase 1 (DNS filtering) | ForeFlight weather + charts load with filtering active |
| Starlink speed cap confusion | Phase 2 (dashboard) | Dashboard shows clear Starlink status and plan messaging |
| WireGuard tunnel instability | Phase 3-4 (proxy) | Tunnel recovers within 2 seconds of simulated outage |
| MITM breaking apps | Phase 4+ (MITM proxy) | Banking apps work with CA cert installed (bypass list) |
| Thermal throttling | Phase 1 (case design) | 2-hour sustained load test in PETG case, CPU <75C |
| Shared CA key | Phase 4+ (MITM proxy) | Each device generates unique CA on first boot |
| No dashboard auth | Phase 2 (dashboard) | Admin functions require PIN; public stats are view-only |

## Sources

- [Raspberry Pi SD Card Corruption - Hackaday](https://hackaday.com/2022/03/09/raspberry-pi-and-the-story-of-sd-card-corruption/)
- [Read-Only Raspberry Pi - Core Electronics](https://core-electronics.com.au/guides/read-only-raspberry-pi/)
- [Running a Raspberry Pi with a read-only root filesystem - Chris Dzombak](https://www.dzombak.com/blog/2024/03/running-a-raspberry-pi-with-a-read-only-root-filesystem/)
- [Raspberry Pi Forum: AP using wlan0 is unstable](https://forums.raspberrypi.com/viewtopic.php?t=286952)
- [Raspberry Pi Forum: WiFi AP ACK problem (2026)](https://forums.raspberrypi.com/viewtopic.php?t=393741)
- [RaspAP: AP-STA mode is unsupported](https://docs.raspap.com/features-experimental/ap-sta/)
- [USB-WiFi AP Mode Guide - GitHub](https://github.com/morrownr/USB-WiFi/blob/main/home/AP_Mode/Bridged_Wireless_Access_Point.md)
- [Solving the Captive Portal Problem on iOS - Medium](https://medium.com/@rwbutler/solving-the-captive-portal-problem-on-ios-9a53ba2b381e)
- [iOS 14 Captive Network RFC 8910 - Apple Developer Forums](https://developer.apple.com/forums/thread/660827)
- [How Captive Portals Work on Raspberry Pi - Medium (2026)](https://medium.com/@jbrathnayake98/how-captive-portals-work-building-one-from-scratch-on-raspberry-pi-f7da1601719b)
- [Starlink Aviation Speed Cap Backlash - AvGeekery](https://avgeekery.com/starlink-aviation-plan-changes-spark-backlash/)
- [Starlink In-Motion Speed Limits - iPad Pilot News](https://ipadpilotnews.com/2026/03/starlink-update-new-in-motion-speed-limits-and-what-it-means-for-pilots/)
- [Starlink Pricing Bait and Switch - Flying Magazine](https://www.flyingmag.com/starlinks-pricing-shift-a-bait-and-switch-for-general-aviation/)
- [A Transport Protocol's View of Starlink - APNIC Labs](https://labs.apnic.net/index.php/2024/05/16/a-transport-protocols-view-of-starlink/)
- [Starlink Latency and Packet Loss Analysis](https://packetstorm.com/starlink-satellite-internet-in-2026-bandwidth-latency-and-packet-loss-analyzed/)
- [CAKE-autorate GitHub (Starlink config)](https://github.com/lynxthecat/cake-autorate)
- [WireGuard Performance Tuning for High Latency Networks (2026)](https://didi-thesysadmin.com/2026/02/09/wireguard-performance-tuning-high-latency/)
- [WireGuard Tunnel Reconnect Performance Issues - Netgate Forum](https://forum.netgate.com/topic/198684/wireguard-tunnel-disconnect-reconnect-events-cause-performance-issues-system-wide)
- [Raspberry Pi Thermal Throttling - XDA Developers](https://www.xda-developers.com/raspberry-pi-probably-thermal-throttling-dont-even-know/)
- [Raspberry Pi 5 Power Consumption - SunFounder](https://www.sunfounder.com/blogs/news/raspberry-pi-temperature-guide-how-to-check-throttling-limits-cooling-tips)
- [Bypassing Certificate Pinning - Approov](https://approov.io/blog/bypassing-certificate-pinning)
- [Starlink Mini Ethernet Adapter Setup - DISHYtech](https://www.dishytech.com/starlink-ethernet-adapter-setup-and-review/)
- [Starlink Bypass Mode - Starlink Insider](https://starlinkinsider.com/starlink-bypass-mode/)
- [Pi-hole DNS Blocking Modes Documentation](https://docs.pi-hole.net/ftldns/blockingmode/)

---
*Pitfalls research for: Pi-based aviation bandwidth management appliance (SkyGate)*
*Researched: 2026-03-22*
