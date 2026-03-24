# Phase 4: Content Compression Proxy - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md -- this log preserves the alternatives considered.

**Date:** 2026-03-23
**Phase:** 04-content-compression-proxy
**Areas discussed:** Image compression strategy, Proxy pipeline architecture, MITM certificate handling, Usage/savings logging, Docker deployment structure
**Mode:** Auto (--auto flag, all gray areas auto-selected with recommended defaults)

---

## Image Compression Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Aggressive (q30, 800px, WebP) | JPEG q30 + max 800px width + WebP transcoding. Maximum satellite savings. Passengers browsing news/social don't need retina quality. | ✓ |
| Moderate (q50, 1200px, WebP) | Better visual quality, less savings. Appropriate if image quality complaints emerge. | |
| Conservative (q70, original size, JPEG only) | Minimal quality loss, minimal savings. Only useful as a fallback. | |

**User's choice:** [auto] Aggressive (q30, 800px, WebP) -- recommended default. Satellite bandwidth is expensive ($0.01/MB), visual fidelity is secondary for passenger browsing.
**Notes:** 500ms timeout per image prevents pipeline stalls. GIFs passed through unmodified in v1. SVGs minified via text pipeline.

---

## Proxy Pipeline Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| goproxy + handler chain | elazarl/goproxy for MITM, Content-Type routing to specialized handlers (image transcoder, text minifier, passthrough). Modular, testable. | ✓ |
| Custom net/http proxy | Build from scratch on net/http. Full control but reinvents TLS interception, CONNECT handling. | |
| mitmproxy (Python) | Flexible addon system but 200-500MB RAM, memory leaks on ARM, violates Go-only server constraint. | |

**User's choice:** [auto] goproxy + handler chain -- recommended default per STACK.md and PROJECT.md Key Decisions. Well-maintained (6.2k stars), built-in MITM, and response hooks map cleanly to compression pipeline.
**Notes:** SNI bypass list prevents MITM on cert-pinned domains. Same YAML config format as Pi-side bypass-domains.yaml.

---

## MITM Certificate Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Single persistent CA + on-the-fly leaf certs | One CA cert generated at first startup, persisted to Docker volume. Leaf certs generated per-domain and cached in memory (LRU). Standard MITM pattern. | ✓ |
| Per-device unique CA | Each connecting device gets its own CA. More isolation but massive complexity for cert distribution and no real security benefit in this trust model. | |
| Pre-generated certs for top domains | Pre-generate leaf certs for top 1000 domains. Faster first connection but stale, unmaintainable. | |

**User's choice:** [auto] Single persistent CA + on-the-fly leaf certs -- recommended default. Standard approach, goproxy supports this natively. CA cert exposed via HTTP endpoint for Phase 5 captive portal distribution.
**Notes:** CA cert must persist across Docker restarts (volume mount). Leaf cert LRU cache prevents memory growth.

---

## Usage/Savings Logging

| Option | Description | Selected |
|--------|-------------|----------|
| SQLite with per-request compression stats | Log timestamp, domain, original_bytes, compressed_bytes, content_type, device_id to SQLite WAL. Enables real savings reporting to Pi dashboard. | ✓ |
| Statsd/Prometheus metrics only | Export metrics for monitoring but no per-request detail. Harder to attribute savings per device. | |
| No server-side logging | Minimal footprint, but loses the ability to show actual compression ratios on dashboard. | |

**User's choice:** [auto] SQLite with per-request compression stats -- recommended default. Matches Phase 2 SQLite pattern. Per-request data enables accurate "$X saved" display replacing Phase 2's heuristic estimates.
**Notes:** Retention/pruning policy at Claude's discretion. mattn/go-sqlite3 with CGo for SQLite driver.

---

## Docker Deployment Structure

| Option | Description | Selected |
|--------|-------------|----------|
| Single docker-compose.yml with WireGuard + proxy containers | Two containers: (1) WireGuard server (linuxserver/wireguard or wg-easy), (2) Go proxy + SQLite. Shared Docker network. One `docker compose up` command. | ✓ |
| Single monolithic container | WireGuard + proxy in one container. Simpler but violates container-per-service best practice. Harder to update proxy without disrupting tunnel. | |
| Separate docker-compose files | WireGuard and proxy deployed independently. More flexible but requires coordination, not "one command." | |

**User's choice:** [auto] Single docker-compose.yml with separate WireGuard + proxy containers -- recommended default per PROXY-02 requirement ("one-command Docker Compose"). Clean separation allows independent updates while meeting the single-command deploy requirement.
**Notes:** WireGuard container choice (linuxserver/wireguard vs wg-easy vs raw) at Claude's discretion. Shared Docker network for internal routing.

---

## Claude's Discretion

- goproxy MITM configuration details (cert pool size, TLS version constraints)
- SQLite schema design (tables, indexes, retention/pruning policy)
- Image transcoding internals (buffer pooling, goroutine concurrency limits)
- Minification configuration flags for tdewolff/minify
- Docker healthcheck implementation
- Go package structure within server/ directory
- Error handling for malformed content
- WireGuard server container selection
- Content-Length recalculation approach

## Deferred Ideas

- Video blocking/replacement -- v2 content rules
- Social media text-only mode -- v2 requirement CONT-01
- Content rules hot-reload -- v2
- Multi-tenant proxy scaling -- Phase C
- AVIF transcoding -- post-WebP baseline
- Animated GIF transcoding -- low ROI for v1
- HTML DOM rewriting -- v2 content rules
