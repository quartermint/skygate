# Phase 3: Tunnel Infrastructure - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md -- this log preserves the alternatives considered.

**Date:** 2026-03-23
**Phase:** 03-tunnel-infrastructure
**Mode:** Auto (--auto flag, all defaults auto-selected)
**Areas discussed:** WireGuard Configuration, Remote Server Deployment, Key Management, Policy-Based Routing, Tunnel Resilience & Fallback, CAKE QoS on WireGuard

---

## WireGuard Configuration

[auto] Selected all gray areas: WireGuard Configuration, Remote Server Deployment, Key Management, Policy-Based Routing, Tunnel Resilience & Fallback, CAKE QoS on WireGuard.

### MTU Setting

| Option | Description | Selected |
|--------|-------------|----------|
| 1420 (conservative IPv4) | Starlink 1500 - WireGuard 80 byte overhead. Safe default, may leave ~20 bytes on the table | :white_check_mark: |
| 1280 (ultra-conservative) | Minimum IPv6 MTU, guaranteed to work everywhere but wastes bandwidth | |
| 1440 (aggressive) | Assumes minimal overhead, risk of fragmentation on some paths | |

**User's choice:** [auto] 1420 (recommended default -- conservative and well-documented for Starlink + WireGuard)
**Notes:** PITFALLS.md recommends "1280 (conservative) or calculate precisely: Starlink MTU (typically 1500) minus WireGuard overhead (60 bytes for IPv4, 80 for IPv6) = 1420-1440." Going with 1420 as the safe middle ground.

### PersistentKeepalive

| Option | Description | Selected |
|--------|-------------|----------|
| 25 seconds | Standard recommendation for NAT traversal, documented in WireGuard docs | :white_check_mark: |
| 15 seconds | More aggressive, higher overhead but faster recovery | |
| Disabled | Only works if both sides have public IPs (not the case with Starlink CGNAT) | |

**User's choice:** [auto] 25 seconds (recommended default -- standard WireGuard recommendation, mandatory for Starlink CGNAT per PITFALLS.md and ARCHITECTURE.md)
**Notes:** Both PITFALLS.md and ARCHITECTURE.md explicitly state PersistentKeepalive = 25 is mandatory for Starlink.

### Config Management

| Option | Description | Selected |
|--------|-------------|----------|
| wg-quick + Ansible template | Standard approach, Jinja2 templated wg0.conf deployed by Ansible, managed via wg-quick@wg0 systemd service | :white_check_mark: |
| vishvananda/netlink Go library | Programmatic WireGuard setup from Go daemon, more control but more code | |
| Manual wg set commands | No config file, ephemeral -- not appropriate for an appliance | |

**User's choice:** [auto] wg-quick + Ansible template (recommended default -- follows established Ansible role pattern from Phase 1, minimal code)
**Notes:** Consistent with Phase 1 patterns: Ansible roles with Jinja2 templates, systemd services.

---

## Remote Server Deployment

### Deployment Method

| Option | Description | Selected |
|--------|-------------|----------|
| Docker Compose | One-command `docker compose up -d`, standard for self-hosted appliances, linuxserver/wireguard image | :white_check_mark: |
| Bare metal install script | Direct WireGuard install on VPS, simpler but less reproducible | |
| Terraform + Ansible | Full IaC, overkill for single-tenant v1 | |

**User's choice:** [auto] Docker Compose (recommended default -- per CLAUDE.md stack decisions and PROJECT.md requirements)
**Notes:** PROJECT.md explicitly requires "One-command Docker Compose deployment for remote server."

### WireGuard Docker Image

| Option | Description | Selected |
|--------|-------------|----------|
| linuxserver/wireguard | Mature, well-documented, handles key generation, widely used | :white_check_mark: |
| wg-easy | Web UI for peer management, more features but more surface area | |
| Custom Dockerfile | Full control but maintenance burden | |

**User's choice:** [auto] linuxserver/wireguard (recommended default -- per STACK.md recommendation)
**Notes:** STACK.md recommends "Use `linuxserver/wireguard` Docker image or `wg-easy` for web UI peer management." For v1 single-tenant, the simpler option is preferred.

---

## Key Management

### Key Exchange Workflow

| Option | Description | Selected |
|--------|-------------|----------|
| Manual exchange via Ansible vars | Server generates keys, operator copies public key + endpoint to Ansible vars, Ansible provisions Pi | :white_check_mark: |
| Automated via setup script | Script SSHes to both sides and configures peers automatically | |
| Web-based key registration API | Pi registers with server via HTTP API, automated but requires API server | |

**User's choice:** [auto] Manual exchange via Ansible vars (recommended default -- simplest for single-tenant v1, no additional infrastructure)
**Notes:** Automated key exchange is a convenience feature that adds complexity. For v1 with one Pi and one server, manual config is fine. Deferred to future.

---

## Policy-Based Routing

### Routing Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Extend existing nftables + ip rule | Add fwmark 0x2 to prerouting, add table 200 -> wg0, minimal change to Phase 1 foundation | :white_check_mark: |
| WireGuard AllowedIPs-based routing | Use WireGuard's built-in routing (AllowedIPs = 0.0.0.0/0), bypass via excluded subnets | |
| Network namespace isolation | Separate namespace for tunnel traffic, strongest isolation but more complex | |

**User's choice:** [auto] Extend existing nftables + ip rule (recommended default -- builds directly on Phase 1 nftables infrastructure, well-documented in ARCHITECTURE.md Pattern 1)
**Notes:** ARCHITECTURE.md explicitly documents the fwmark 0x1/0x2 + policy routing approach. Phase 1 already has fwmark 0x1 for bypass. Adding 0x2 for tunnel is the natural extension.

### Forward Chain Changes

| Option | Description | Selected |
|--------|-------------|----------|
| Add wg0 as allowed output interface | Extend forward chain: `iifname ap_interface oifname wg0 accept` alongside existing eth0 rule | :white_check_mark: |
| Allow all forwarding, rely on policy routing | Remove interface restrictions, trust fwmark to route correctly | |

**User's choice:** [auto] Add wg0 as allowed output interface (recommended default -- explicit allow is more secure, follows existing pattern)
**Notes:** Phase 1 forward chain explicitly allows AP-to-eth0. Adding AP-to-wg0 follows the same pattern.

---

## Tunnel Resilience & Fallback

### Fallback Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Handshake-based monitoring + route swap | Check `wg show wg0 latest-handshakes`, if stale > threshold swap default route to direct | :white_check_mark: |
| Active probing through tunnel | Ping a target through wg0, detect failure, swap routes | |
| No fallback (tunnel or nothing) | Simplest but violates graceful degradation principle | |

**User's choice:** [auto] Handshake-based monitoring + route swap (recommended default -- lightweight, no additional traffic, uses WireGuard's built-in state)
**Notes:** PITFALLS.md Pitfall 6 emphasizes "implement instant fallback: if the WireGuard tunnel is unreachable for >2 seconds, route all traffic directly to Starlink." Handshake monitoring is the lightest-weight approach.

### Monitor Implementation

| Option | Description | Selected |
|--------|-------------|----------|
| Lightweight bash script + systemd timer | Simple, follows autorate script pattern from Phase 1 | :white_check_mark: |
| Standalone Go binary | More robust error handling, follows bypass-daemon pattern | |
| Integrated into bypass daemon | Fewer binaries but scope creep on existing daemon | |

**User's choice:** [auto] Lightweight bash script + systemd timer (recommended default -- tunnel monitoring is simple logic, bash is appropriate, follows Phase 1 autorate pattern)
**Notes:** The autorate script from Phase 1 QoS is already a bash script with a systemd service. Tunnel monitoring is comparable in complexity.

---

## CAKE QoS on WireGuard

### QoS on wg0

| Option | Description | Selected |
|--------|-------------|----------|
| CAKE on wg0 + extend autorate | Apply CAKE qdisc to wg0, extend autorate script to monitor and shape both interfaces | :white_check_mark: |
| CAKE on wg0 with static rate | Fixed bandwidth ceiling on wg0, no dynamic adjustment | |
| No QoS on wg0 (eth0 only) | Rely on eth0 CAKE to handle all shaping | |

**User's choice:** [auto] CAKE on wg0 + extend autorate (recommended default -- tunnel traffic needs its own shaping to prevent bufferbloat within the encrypted path)
**Notes:** CLAUDE.md stack section mentions "Configure CAKE on WireGuard interface (wg0) and WiFi interface (wlan0/wlan1)."

---

## Claude's Discretion

The following areas were left to Claude's judgment:
- Exact WireGuard AllowedIPs configuration for split tunneling
- Tunnel health check thresholds (handshake timeout, recovery detection timing)
- Whether tunnel monitor is standalone or integrated
- Docker Compose structure (single vs multi-service)
- ip rule priority values for policy routing tables
- Server-side firewall rules
- Exact fallback routing implementation details

## Deferred Ideas

- Dashboard tunnel status indicator (Phase 2 enhancement)
- Automated key exchange (convenience feature for future)
- Multi-peer WireGuard server (hosted service, Phase C)
- WireGuard over TCP fallback (niche requirement)
- Tunnel traffic metrics in dashboard (future dashboard enhancement)
- Server-side content proxy container (Phase 4)
