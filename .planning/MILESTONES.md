# Milestones

## v1.0 SkyGate Bandwidth Management Appliance (Shipped: 2026-03-24)

**Phases completed:** 5 phases, 18 plans, 38 tasks

**Key accomplishments:**

- Go module with bypass daemon (DNS resolution + nftables population), Ansible playbook skeleton with 6 roles, aviation bypass domain config (D-12), conservative blocklists (D-10), BATS test scaffold, and Makefile with 11 targets
- Ansible roles for hostapd WiFi AP (2.4GHz/WPA2/8 clients), Pi-hole v6 DHCP+DNS blocking with aviation domain whitelisting, and nftables firewall with bypass_v4 set for policy routing
- Modular Go bypass daemon with DNS resolution, nftables set population, and Ansible routing role deploying binary + systemd service with security hardening
- CAKE autorate bash script with fping RTT measurement, dynamic 5-100 Mbps bandwidth adjustment, 9 BATS tests, and Ansible QoS role with systemd service
- OverlayFS read-only root with ext4 /data partition for crash-safe persistence, plus systemd oneshot first-boot wizard for pilot WiFi SSID/password configuration
- Go dashboard daemon data layer with SQLite WAL persistence, nftables per-MAC counter parser, domain-to-category mapper, and YAML config -- 23 unit tests passing
- HTMX+SSE dashboard with Chart.js bandwidth/category charts, captive portal terms page, and settings configuration -- all static files served by Caddy with dark aviation theme
- Pi-hole API client, savings calculator, SSE streaming (6 events), REST API endpoints, captive portal accept, and daemon main.go -- 47 unit tests passing, cross-compiles for linux/arm64
- Ansible dashboard role with Caddy CNA-interception reverse proxy, nftables captive portal DNAT, systemd hardened service, and multi-daemon Makefile targets
- WireGuard tunnel infrastructure with server Docker Compose endpoint, Pi Ansible role (Table=off, MTU 1420), dual-fwmark nftables (0x1 bypass, 0x2 tunnel), and policy routing table 200
- Go tunnel-monitor daemon with WireGuard handshake health checks, hysteresis state machine, and ip rule routing fallback
- Makefile tunnel-monitor targets, Ansible playbook wireguard role, CAKE QoS on wg0, autorate WG support, and 14 BATS nftables tunnel validation tests
- YAML config loader, ECDSA CA certificate generation, and SQLite compression logging for the Go MITM proxy server
- WebP image transcoding (q30, 800px max, 500ms timeout) and JS/CSS/HTML minification with Content-Type response routing
- goproxy MITM proxy with conditional SNI bypass, LRU CertStorage, Docker Compose deployment sharing WireGuard network namespace
- Root CA (SSID-aware, 3-year, ECDSA P-256) + intermediate CA (MaxPathLen=0, 1-year) delegation model with hardcoded never-MITM bypass domains for banking/auth/gov/health/payments
- Per-device Quick Connect / Max Savings mode selection with SQLite persistence, nftables integration, IP mapping API for proxy awareness, and cert download handlers (.mobileconfig, .crt) with platform-specific install guides
- Ansible certificate role, nftables maxsavings set, proxy intermediate CA loading with per-device MaxSavingsIPSet polling, Docker Compose CA volume mount, and Makefile provisioning target

---
