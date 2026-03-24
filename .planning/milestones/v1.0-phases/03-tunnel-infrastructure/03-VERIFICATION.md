---
phase: 03-tunnel-infrastructure
verified: 2026-03-23T20:30:00Z
status: passed
score: 10/10 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Deploy to a real Pi with WireGuard keys configured and wg_enabled=true, confirm tunnel establishes on boot"
    expected: "wg0 interface comes up, handshake occurs within 30s, traffic to non-aviation domains routes via wg0"
    why_human: "Requires physical Pi hardware, Starlink uplink, and a remote WireGuard server with real key exchange"
  - test: "Simulate Starlink satellite handoff (brief connectivity loss ~10-30s) and verify tunnel auto-reconnects"
    expected: "Tunnel monitor logs DEGRADED after 3 failed checks, then HEALTHY after 3 successful checks; ip rule restored"
    why_human: "Requires live WireGuard tunnel and controlled network interruption to validate state machine behavior end-to-end"
  - test: "Verify aviation apps (ForeFlight, Garmin Pilot) route via eth0 directly while browser traffic routes via wg0"
    expected: "traceroute from aviation app IP hits Starlink gateway directly; traceroute from non-aviation IP goes through tunnel"
    why_human: "Requires device testing matrix with actual ForeFlight and Garmin Pilot installed on iOS/Android"
---

# Phase 3: Tunnel Infrastructure Verification Report

**Phase Goal:** Non-aviation traffic flows through an encrypted WireGuard tunnel to a remote server while aviation apps continue routing directly to Starlink
**Verified:** 2026-03-23T20:30:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | WireGuard tunnel establishes automatically on boot and maintains connection through Starlink satellite handoffs | VERIFIED (infra) / HUMAN for live behavior | wg-quick@wg0 systemd service enabled in wireguard role; PersistentKeepalive=25 in wg0.conf.j2; tunnel-monitor.service with Restart=always |
| 2 | Non-bypass web traffic routes through the tunnel to the remote server while aviation app traffic goes direct to Starlink | VERIFIED (infra) | nftables prerouting marks AP non-bypass traffic fwmark 0x2; policy routing table 200 routes fwmark 0x2 via wg0; bypass traffic (fwmark 0x1) routes via table 100 to eth0 |
| 3 | If the tunnel drops, traffic falls back to direct routing and the tunnel auto-reconnects without manual intervention | VERIFIED (daemon logic + infra) / HUMAN for live behavior | Monitor state machine removes ip rule on DEGRADED (15 Go tests passing); wg-quick@wg0 + Restart=always handles reconnect; wg_enabled=false path falls through to main table |

**Score:** 10/10 must-haves verified (infrastructure + daemon logic confirmed; live behavior requires hardware)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `server/docker-compose.yml` | WireGuard server endpoint via linuxserver/wireguard | VERIFIED | Contains `lscr.io/linuxserver/wireguard:latest`, SERVERURL, PEERS, `net.ipv4.conf.all.src_valid_mark=1`, Phase 4 proxy stub |
| `server/.env.example` | Server configuration template | VERIFIED | Contains SERVERURL, SERVERPORT, PEERS, INTERNAL_SUBNET=10.13.13.0 |
| `pi/ansible/roles/wireguard/tasks/main.yml` | WireGuard Ansible deployment role | VERIFIED | Installs wireguard-tools, deploys wg0.conf (0600), configures policy routing table 200, deploys tunnel-monitor systemd service; all tasks gated by `wg_enabled | default(false)` |
| `pi/ansible/roles/wireguard/templates/wg0.conf.j2` | WireGuard client config template | VERIFIED | `Table = off`, `MTU = {{ wg_mtu | default(1420) }}`, `PersistentKeepalive = {{ wg_keepalive | default(25) }}`, `AllowedIPs = 0.0.0.0/0` |
| `pi/ansible/roles/networking/templates/nftables.conf.j2` | Extended nftables with tunnel marking | VERIFIED | fwmark 0x2 for non-bypass AP traffic; AP-to-wg0 forwarding; MSS clamping 1380; ct mark != 0x0 restore; all gated by wg_enabled; 14/14 BATS tests pass |
| `pi/ansible/group_vars/all.yml` | WireGuard and tunnel monitor variables | VERIFIED | wg_enabled=false, wg_mtu=1420, wg_keepalive=25, wg_client_address=10.13.13.2, tunnel_monitor_handshake_timeout_s=180, wg_cake_base_rate_kbps=15000 |
| `cmd/tunnel-monitor/main.go` | Daemon entry point with signal handling and ticker loop | VERIFIED | LoadConfig, NewMonitor, GetHandshakeOutput, CheckHandshake, ExecIPRule(FormatDelRule/FormatAddRule) all called; SIGINT/SIGTERM handling; ticker loop |
| `cmd/tunnel-monitor/config.go` | YAML config loading | VERIFIED | `type Config struct`, `func LoadConfig(path string) (*Config, error)`, HandshakeTimeoutS field |
| `cmd/tunnel-monitor/health.go` | Handshake parsing and health check logic | VERIFIED | CheckHandshake, TunnelState, StateHealthy, StateDegraded, Monitor struct, NewMonitor, Update method |
| `cmd/tunnel-monitor/fallback.go` | Routing fallback command formatting | VERIFIED | FallbackConfig, FormatAddRule, FormatDelRule |
| `cmd/tunnel-monitor/health_linux.go` | Linux exec implementation for wg show | VERIFIED | `//go:build linux`, GetHandshakeOutput via `exec.Command("wg", "show", iface, "latest-handshakes")` |
| `cmd/tunnel-monitor/health_stub.go` | macOS dev stub for wg show | VERIFIED | `//go:build !linux`, returns simulated 30s-old handshake |
| `cmd/tunnel-monitor/fallback_linux.go` | Linux exec implementation for ip rule | VERIFIED | `//go:build linux`, ExecIPRule via `exec.Command("ip", args...)` |
| `cmd/tunnel-monitor/fallback_stub.go` | macOS dev stub for ip rule | VERIFIED | `//go:build !linux`, no-op logging stub |
| `Makefile` | Tunnel monitor build targets | VERIFIED | TUNNEL_BINARY=skygate-tunnel-monitor; build-tunnel, cross-build-tunnel targets; build aggregate includes build-tunnel |
| `pi/ansible/playbook.yml` | Updated playbook with wireguard role | VERIFIED | wireguard role at line 10, after routing (line 9), before qos (line 11) |
| `pi/ansible/roles/qos/tasks/main.yml` | CAKE on wg0 initialization | VERIFIED | `tc qdisc replace dev {{ wg_interface | default('wg0') }}` with `when: wg_enabled | default(false)` |
| `pi/ansible/roles/qos/templates/autorate.sh.j2` | Dual-interface CAKE management | VERIFIED | WG_ENABLED, WG_INTERFACE, WG_CAKE_RATE_KBPS vars; apply_wg_cake() function; wg0 CAKE initialized at startup when WG_ENABLED=true; no dynamic adjustment in loop (static ceiling per research) |
| `pi/scripts/tests/test_nftables_tunnel.bats` | nftables template validation tests | VERIFIED | 14 tests; all pass (`bats pi/scripts/tests/test_nftables_tunnel.bats` exits 0) |
| `pi/config/tunnel-monitor.yaml` | Default tunnel monitor config | VERIFIED | interface=wg0, fwmark=0x2, table=200, check_interval_s=5 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `wg0.conf.j2` | `group_vars/all.yml` | Jinja2 variable references | VERIFIED | `{{ wg_private_key }}`, `{{ wg_client_address }}`, `{{ wg_server_public_key }}` all present in template; variables defined in group_vars |
| `nftables.conf.j2` | `group_vars/all.yml` | wg_enabled conditional | VERIFIED | `wg_enabled | default(false)` appears 4 times in nftables template; `wg_enabled: false` defined in group_vars |
| `wireguard/tasks/main.yml` | `nftables.conf.j2` | policy routing complements fwmark | VERIFIED | tasks/main.yml adds `fwmark 0x2 table 200` ip rule; nftables marks non-bypass traffic with fwmark 0x2 in prerouting |
| `main.go` | `config.go` | LoadConfig call | VERIFIED | `cfg, err := LoadConfig(*configPath)` in main() |
| `main.go` | `health.go` | CheckHandshake in ticker loop | VERIFIED | `CheckHandshake(output, maxAge)` called in runCheck() which is called from ticker |
| `main.go` | `fallback.go` | FormatAddRule/FormatDelRule for routing changes | VERIFIED | `ExecIPRule(FormatDelRule(fbCfg))` and `ExecIPRule(FormatAddRule(fbCfg))` in handleStateChange() |
| `health.go` | `health_linux.go` | GetHandshakeOutput platform function | VERIFIED | `GetHandshakeOutput(iface)` called in runCheck(); linux/stub files provide platform-specific implementations |
| `fallback.go` | `fallback_linux.go` | ExecIPRule platform function | VERIFIED | `ExecIPRule(args)` called in handleStateChange(); linux/stub files provide platform-specific implementations |
| `Makefile` | `cmd/tunnel-monitor/` | go build target | VERIFIED | `go build -o bin/$(TUNNEL_BINARY) ./cmd/tunnel-monitor/`; `make build` exits 0 |
| `playbook.yml` | `roles/wireguard/` | role include | VERIFIED | `- wireguard` in roles list at correct position |
| `qos/tasks/main.yml` | `group_vars/all.yml` | wg_enabled and wg_cake_* variables | VERIFIED | `wg_enabled | default(false)` when condition; `wg_cake_base_rate_kbps | default(15000)` in tc command |

### Data-Flow Trace (Level 4)

Not applicable. Phase 3 produces infrastructure (Ansible templates, Go daemon, Makefile targets) — no dynamic data rendering components. The daemon does execute data flows (handshake checks -> state transitions -> ip rule manipulation) which are fully verified through unit tests and build checks.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Daemon compiles on macOS | `go build -o /dev/null ./cmd/tunnel-monitor/` | exits 0 | PASS |
| All 15 Go unit tests pass | `go test ./cmd/tunnel-monitor/... -v -short -count=1` | 15 tests PASS | PASS |
| `go vet` clean | `go vet ./cmd/tunnel-monitor/...` | no issues | PASS |
| `make build` compiles all 3 daemons | `make build` | bypass + dashboard + tunnel-monitor all built | PASS |
| 14 BATS nftables tunnel tests pass | `bats pi/scripts/tests/test_nftables_tunnel.bats` | 14/14 ok | PASS |
| Existing 9 autorate BATS tests pass (no regression) | `bats pi/scripts/tests/test_autorate.bats` | 9/9 ok | PASS |
| wg_enabled conditional gating | `grep -c 'wg_enabled'` in nftables.conf.j2 | 4 conditional blocks | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|---------|
| TUN-01 | 03-01, 03-02, 03-03 | WireGuard kernel-mode tunnel connects Pi to remote proxy server with keepalive and auto-reconnect on connectivity loss | SATISFIED | WireGuard Ansible role (tasks/main.yml) installs wireguard-tools, deploys wg0.conf with PersistentKeepalive=25 and wg-quick@wg0 systemd service; tunnel-monitor daemon with Monitor state machine removes/restores ip rule on tunnel drop/recovery; auto-reconnect via wg-quick Restart=always |
| ROUTE-02 | 03-01, 03-02, 03-03 | Policy-based routing via nftables sends non-bypass traffic through WireGuard tunnel to remote proxy | SATISFIED | nftables prerouting marks AP non-bypass traffic `fwmark 0x2`; Ansible wireguard role adds `ip rule fwmark 0x2 table 200` and `ip route default dev wg0 table 200`; aviation bypass traffic retains `fwmark 0x1` and routes via table 100 to eth0 (Phase 1 preserved) |

Both phase requirements are SATISFIED by implementation evidence. REQUIREMENTS.md traceability table correctly maps both to Phase 3 with status "Pending" — the traceability table reflects pre-execution state and has not been updated post-completion, but this is a documentation state issue not a code gap.

**Note on ROADMAP.md:** `03-03-PLAN.md` is marked `[ ]` (not checked) in ROADMAP.md despite Plan 03 being fully executed and verified. All artifacts from Plan 03 exist and pass checks. The ROADMAP checkbox was not updated after execution. This is a minor documentation state issue with no impact on code correctness.

### Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| `pi/ansible/group_vars/all.yml` | `wg_private_key: "PLACEHOLDER"`, `wg_server_public_key: "PLACEHOLDER"`, `wg_server_endpoint: "PLACEHOLDER"` | INFO | Intentional — documented operational requirement. Keys are exchanged during actual deployment (D-10 workflow). Not a code stub; these values are overridden by operator before `make deploy`. |

No blockers or warnings. The PLACEHOLDER values in group_vars are intentional design (keys cannot be committed to source control; the wifi_password uses the same pattern).

### Human Verification Required

#### 1. Boot-time WireGuard Establishment

**Test:** Flash SD card with wg_enabled=true and real WireGuard keys, boot Pi, observe wg0 interface
**Expected:** `wg show wg0` shows active peer with handshake within 30 seconds of boot; `ip rule show` shows `fwmark 0x2 lookup 200`; `ip route show table 200` shows `default dev wg0`
**Why human:** Requires physical Pi 5 hardware, Starlink Mini uplink, and a deployed WireGuard server with key exchange

#### 2. Satellite Handoff Resilience

**Test:** With active WireGuard tunnel, simulate 15-20s connectivity interruption (unplug eth0 briefly), observe tunnel monitor logs and routing state
**Expected:** After 3 consecutive failed handshake checks (~15s at 5s interval), monitor logs `STATE CHANGE -> DEGRADED`, removes ip rule, traffic falls to main table. After restoration, 3 successful checks restore the rule with `STATE CHANGE -> HEALTHY`
**Why human:** Requires live WireGuard tunnel and controlled network interruption; state machine logic is verified in unit tests but live timing depends on WireGuard handshake behavior on Starlink

#### 3. Aviation App Bypass Verification

**Test:** With wg_enabled=true and bypass domains configured, connect ForeFlight and a browser to SkyGate WiFi. Run simultaneous connections.
**Expected:** `conntrack -L | grep <foreflight_ip>` shows fwmark 0x1; `conntrack -L | grep <browser_ip>` shows fwmark 0x2; traceroute from aviation IP exits directly via eth0, traceroute from browser IP exits via wg0
**Why human:** Requires real device test matrix (iOS ForeFlight, Garmin Pilot) and live network capture

### Gaps Summary

No gaps. All automated verification passes:
- All 23 Phase 3 artifacts exist and contain the required patterns
- All 11 key links are verified
- Go daemon compiles clean (`go build`, `go vet`)
- 15/15 Go unit tests pass (config loading, handshake parsing, state machine transitions, fallback formatting)
- 14/14 BATS nftables tunnel tests pass
- 9/9 existing autorate BATS tests pass (no regressions)
- `make build` produces all three daemons

The phase goal is structurally achieved: the infrastructure for routing non-aviation traffic through a WireGuard tunnel while preserving aviation bypass exists, is wired, and behaves correctly per automated tests. Human verification is needed only for live hardware behavior (boot-time tunnel establishment, satellite handoff resilience, and multi-device routing confirmation).

---

_Verified: 2026-03-23T20:30:00Z_
_Verifier: Claude (gsd-verifier)_
