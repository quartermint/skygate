# Phase 2: Usage Dashboard - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-23
**Phase:** 02-usage-dashboard
**Areas discussed:** Dashboard layout, Captive portal flow, Data collection, Savings calculation, Plan cap configuration
**Mode:** Auto (--auto flag — all decisions auto-selected using recommended defaults)

---

## Dashboard Layout & Information Hierarchy

| Option | Description | Selected |
|--------|-------------|----------|
| Single-page overview | All sections visible on one page, scroll to explore | ✓ |
| Multi-tab interface | Separate tabs for devices, bandwidth, savings | |
| Card-based modules | Draggable/reorderable dashboard cards | |

**User's choice:** [auto] Single-page overview (recommended — simplest for non-technical pilots)

| Option | Description | Selected |
|--------|-------------|----------|
| Table with bars | Device name/MAC, bytes, top domain, bar chart | ✓ |
| Card per device | Individual card for each connected device | |
| Compact list | Minimal list with expandable details | |

**User's choice:** [auto] Table with bars (recommended — most information-dense)

| Option | Description | Selected |
|--------|-------------|----------|
| Streaming line chart (5min) | SSE-driven, last 5 minutes of throughput | ✓ |
| Sparkline per device | Small inline graphs per device row | |
| Aggregate bar chart | Per-minute bars for last hour | |

**User's choice:** [auto] Streaming line chart via HTMX SSE (recommended — matches tech stack)

| Option | Description | Selected |
|--------|-------------|----------|
| Pie chart | Category breakdown as pie/donut chart | ✓ |
| Treemap | Proportional rectangles by category | |
| Horizontal bar chart | Ranked categories with bars | |

**User's choice:** [auto] Pie chart (recommended — matches PROJECT.md "whoa moment is the pie chart")

---

## Captive Portal Flow

| Option | Description | Selected |
|--------|-------------|----------|
| HTTP intercept → terms → dashboard | Standard captive portal pattern | ✓ |
| Immediate dashboard (no terms) | Skip terms, show dashboard directly | |
| Terms + opt-in dashboard link | Terms page with optional dashboard link | |

**User's choice:** [auto] HTTP intercept → terms → dashboard (recommended — terms required per DASH-04)

| Option | Description | Selected |
|--------|-------------|----------|
| HTTP captive portal check intercept | Intercept OS-specific check URLs | ✓ |
| DNS-based redirect | DNS for captive portal domain points to Pi | |
| DHCP option 114 | Captive portal URI via DHCP | |

**User's choice:** [auto] HTTP-based detection (recommended — widest cross-platform support)

---

## Data Collection & Persistence

| Option | Description | Selected |
|--------|-------------|----------|
| nftables per-MAC counters | Extend existing nftables with per-MAC rules | ✓ |
| /sys/class/net statistics | Read kernel network stats per interface | |
| Conntrack-based | Parse connection tracking for per-device | |

**User's choice:** [auto] nftables per-MAC counters (recommended — nftables already in place)

| Option | Description | Selected |
|--------|-------------|----------|
| Pi-hole FTL query log | DNS query log for domain breakdown | ✓ |
| Packet capture (tcpdump) | Full packet inspection for domain stats | |
| DNS proxy sidecar | Custom DNS forwarder logging queries | |

**User's choice:** [auto] Pi-hole FTL query log (recommended — already logging, no extra capture)

---

## Savings Calculation Model

| Option | Description | Selected |
|--------|-------------|----------|
| DNS blocking heuristic | Estimate blocked domain payload sizes | ✓ |
| Before/after comparison | Compare with proxy savings (Phase 4) | |
| Industry benchmarks | Use published ad/tracker size averages | |

**User's choice:** [auto] DNS blocking savings with payload heuristics (recommended — only measurable savings in Phase 2)

---

## Plan Cap Configuration

| Option | Description | Selected |
|--------|-------------|----------|
| Settings page in dashboard | Dedicated settings accessible from nav | ✓ |
| First-boot wizard integration | Configure during initial setup | |
| Config file only | Edit YAML on Pi via SSH | |

**User's choice:** [auto] Settings page in dashboard (recommended — dashboard is the central interface)

| Option | Description | Selected |
|--------|-------------|----------|
| Dashboard banner with color escalation | Non-intrusive, always visible, color-coded | ✓ |
| Modal popup at thresholds | Interruptive alert at each threshold | |
| Email/push notification | External notification system | |

**User's choice:** [auto] Dashboard banner with color escalation (recommended — non-intrusive)

---

## Claude's Discretion

Areas deferred to Claude's judgment during planning/implementation:
- nftables per-MAC counter implementation details
- Pi-hole FTL log parsing approach
- Domain-to-category mapping design
- Caddy configuration
- HTMX component structure
- Device name resolution strategy
- SQLite schema design
- Captive portal intercept implementation
- Responsive layout approach
- Chart library selection

## Deferred Ideas

- WiFi password change via web UI
- Bypass list management via web UI
- Video CDN / update / sync blocking toggles
- Per-device bandwidth throttling
- Flight-aware session tracking
- Panic button for instant filter disable
