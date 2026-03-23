# Phase 4: Content Compression Proxy - Context

**Gathered:** 2026-03-23
**Status:** Ready for planning

<domain>
## Phase Boundary

A remote Go-based MITM proxy server that compresses web content (images, JS, CSS, HTML) before it traverses the expensive Starlink satellite link, delivering 80-90% additional savings beyond DNS blocking. Deployed via one-command Docker Compose with WireGuard server endpoint. No Pi-side changes beyond what Phase 3 established (tunnel routing) -- this phase is entirely remote server work.

</domain>

<decisions>
## Implementation Decisions

### Image Compression Strategy
- **D-01:** JPEG quality reduction to q30 + resize to max 800px width as the default aggressive preset -- satellite bandwidth is expensive, visual fidelity is secondary for passenger browsing
- **D-02:** PNG and JPEG transcoded to WebP using kolesa-team/go-webp (CGo bindings to libwebp) -- WebP delivers 25-35% smaller files vs optimized JPEG at equivalent quality
- **D-03:** Inline compression with 500ms timeout per image -- if transcoding exceeds timeout, pass the original through rather than blocking the page load
- **D-04:** GIF and animated images passed through unmodified in v1 -- animated WebP transcoding is complex and low ROI for the GA browsing use case
- **D-05:** SVG files minified via tdewolff/minify (same as HTML/CSS/JS pipeline) rather than rasterized

### Proxy Pipeline Architecture
- **D-06:** Built on elazarl/goproxy as the MITM proxy framework -- handles CONNECT tunneling, TLS interception, and request/response hooks
- **D-07:** Response handler chain ordered by content type: (1) check Content-Type header, (2) route to appropriate handler (image transcoder, text minifier, or passthrough), (3) update Content-Length, (4) log compression stats
- **D-08:** Content-Type based routing: `image/*` routes to image transcoder, `text/html` and `text/css` and `application/javascript` route to tdewolff/minify, everything else passes through unmodified
- **D-09:** Certificate-pinned domains bypass MITM entirely -- proxy maintains a configurable SNI bypass list loaded from YAML config (same format as Pi-side bypass-domains.yaml)
- **D-10:** Request/response logging to SQLite captures: timestamp, domain, original_bytes, compressed_bytes, content_type, device_id (from WireGuard peer) -- enables savings reporting back to Pi dashboard

### MITM Certificate Handling
- **D-11:** Single CA certificate generated at first server startup, persisted to Docker volume -- all on-the-fly leaf certs signed by this CA
- **D-12:** CA cert file exposed via a simple HTTP endpoint on the server so the Pi's captive portal (Phase 5) can serve it to passengers for download
- **D-13:** On-the-fly leaf certificate generation per domain using goproxy's built-in MITM support -- certs cached in memory with LRU eviction

### Claude's Discretion
- goproxy MITM configuration details (cert pool size, TLS version constraints)
- Exact SQLite schema for compression logs (tables, indexes, retention/pruning policy)
- Image transcoding pipeline internals (buffer pooling, goroutine concurrency limits for parallel image processing)
- Minification configuration flags for tdewolff/minify (preserve comments, collapse whitespace thresholds)
- Docker healthcheck implementation details
- Go module structure within server/ directory (flat package vs nested packages)
- Error handling strategy for malformed content (corrupt images, invalid CSS)
- WireGuard server container choice (linuxserver/wireguard vs wg-easy vs raw wireguard-tools)
- Content-Length recalculation approach (buffered vs chunked transfer encoding)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` -- Full project vision, constraints, key decisions (custom Go proxy on goproxy, not compy)
- `.planning/REQUIREMENTS.md` -- v1 requirements: PROXY-01 (Go MITM proxy with image transcoding + JS/CSS minification), PROXY-02 (one-command Docker Compose deployment)
- `CLAUDE.md` -- Technology stack decisions, recommended libraries with versions, alternatives considered, "What NOT to Use" list

### Architecture & Stack Research
- `.planning/research/ARCHITECTURE.md` -- Split architecture (proxy on remote server, NOT Pi), component boundaries, data flow diagrams, Anti-Pattern 1 (never run proxy on Pi), Anti-Pattern 3 (never use cake-autorate directly), recommended project structure with `server/` directory layout
- `.planning/research/STACK.md` -- Remote server stack: goproxy v1.8+, kolesa-team/go-webp, tdewolff/minify v2.21+, disintegration/imaging, mattn/go-sqlite3, Docker Compose v2, linuxserver/wireguard. CGo cross-compilation notes.
- `.planning/research/PITFALLS.md` -- compy maintenance risk (unmaintained since 2021), mitmproxy memory leaks on ARM

### Prior Phase Context
- `.planning/phases/01-pi-network-foundation/01-CONTEXT.md` -- YAML config format, Go daemon patterns, platform build tags, nftables bypass set design
- `.planning/phases/02-usage-dashboard/02-CONTEXT.md` -- SQLite WAL mode on data partition, Go daemon SSE endpoint pattern, per-device tracking via MAC/IP, savings calculation model (Phase 4 will provide actual compression ratios vs Phase 2's heuristic estimates)

### Existing Codebase
- `cmd/bypass-daemon/` -- Go daemon pattern: YAML config, platform build tags, exported function names for testability
- `go.mod` -- Go module path: `github.com/quartermint/skygate`, Go 1.26.1
- `Makefile` -- Build/test/deploy target patterns, cross-compilation with GOOS/GOARCH

### Design Document
- `~/.gstack/projects/skygate/ryanstern-unknown-design-20260322-161803.md` -- Full approved design doc from /office-hours

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Go module** (`github.com/quartermint/skygate`): existing go.mod can host server/ code in the same module
- **YAML config loader pattern** from `cmd/bypass-daemon/config.go`: reuse for proxy server config (SNI bypass list, compression settings)
- **Makefile patterns**: extend with `server-build`, `server-test`, `docker-build` targets
- **Platform build tags**: follow `nftset_linux.go` / `nftset_stub.go` pattern if any proxy components need OS-specific behavior

### Established Patterns
- Go daemons with exported functions for testability (LoadConfig, ResolveDomains, FormatNftCommand)
- YAML for configuration files
- Cross-compile: GOOS=linux GOARCH=arm64 CGO_ENABLED=0 (note: CGO_ENABLED=1 needed for go-webp and go-sqlite3)
- Test infrastructure: Go unit tests + BATS for bash scripts
- systemd service templates for daemon lifecycle

### Integration Points
- Phase 3 WireGuard tunnel delivers traffic to the proxy server -- proxy listens on the WireGuard server's internal network
- Proxy compression stats (original vs compressed bytes) feed back to Phase 2 dashboard for real savings display (replacing heuristic estimates)
- SNI bypass list on proxy must align with Pi-side bypass-domains.yaml to ensure cert-pinned apps are never MITM'd
- Docker Compose orchestrates WireGuard server + Go proxy + SQLite as a single deployable unit

</code_context>

<specifics>
## Specific Ideas

- Proxy runs on remote server ONLY (Anti-Pattern 1 from ARCHITECTURE.md) -- full content fetched on cheap terrestrial bandwidth, compressed version sent through expensive satellite link
- Study compy's architecture for patterns but DO NOT depend on compy as a binary (unmaintained, 35 open issues, last commit 2021)
- Image quality q30 at max 800px is aggressively optimized for satellite -- passengers browsing social media and news don't need retina-quality images
- 500ms timeout on image transcoding prevents proxy from becoming a bottleneck -- pass originals through if compression is too slow
- SQLite compression logs enable future "actual savings" display on dashboard (Phase 2 currently uses heuristic estimates)
- Docker Compose one-command deploy is a v1 requirement (PROXY-02) -- must be dead simple for self-hosted users
- Remote server target: any VPS with 1GB+ RAM (Hetzner CX22: 2 vCPU, 4GB, ~$4.50/mo per STACK.md)

</specifics>

<deferred>
## Deferred Ideas

- Video content blocking/replacement with placeholder -- Phase 5 or v2 (content rules system)
- Social media text-only mode (strip Instagram/Twitter to text + compressed thumbnails) -- v2 requirement CONT-01
- Content rules hot-reload system (rules.json with tested_date, stale rule flagging) -- v2
- Proxy horizontal scaling for hosted multi-tenant service -- Phase C concern
- Per-peer traffic accounting for multi-tenant billing -- Phase C
- AVIF transcoding as alternative to WebP -- evaluate after WebP baseline is proven
- Animated GIF to animated WebP transcoding -- low ROI for v1
- HTML rewriting (strip video embeds, remove tracking scripts from DOM) -- v2 content rules

</deferred>

---

*Phase: 04-content-compression-proxy*
*Context gathered: 2026-03-23*
