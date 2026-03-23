---
phase: 04-content-compression-proxy
plan: 03
subsystem: proxy
tags: [go, goproxy, mitm, tls, certstore, docker, docker-compose, wireguard, network-mode]

# Dependency graph
requires:
  - phase: 04-content-compression-proxy
    plan: 01
    provides: "Config struct (LoadConfig, LoadBypassDomains), CA cert generation (LoadOrGenerateCA), DB with LogCompression"
  - phase: 04-content-compression-proxy
    plan: 02
    provides: "Transcoder (NewTranscoder), Minifier (NewMinifier), HandlerChain (NewHandlerChain, HandleResponse)"
provides:
  - "SetupProxy: goproxy server with conditional MITM bypass, CertStorage, and response handler chain"
  - "BypassSet: exact and wildcard domain matching for SNI bypass (cert-pinned apps)"
  - "memCertStore: LRU-cached CertStorage for on-the-fly leaf certificate caching (D-13)"
  - "main.go: entry point wiring config, CA, DB, transcoder, minifier, handler chain, goproxy"
  - "CA cert download endpoint on /ca.crt with /health for Docker healthcheck"
  - "Dockerfile.proxy: multi-stage CGo build with libwebp for image transcoding"
  - "docker-compose.yml: WireGuard + proxy with shared network namespace (network_mode: service:wireguard)"
  - "Makefile: build-proxy, test-proxy, docker-build, docker-up, docker-down targets"
affects: [05-captive-portal]

# Tech tracking
tech-stack:
  added: []
  patterns: ["goproxy conditional MITM: HandleConnectFunc with BypassSet for SNI bypass vs MITM interception", "CertStorage interface: Fetch(host, gen) with LRU cache and gen() fallback for cache miss", "Docker network_mode: service:wireguard for shared network namespace (ports on wireguard container only)", "Multi-stage Dockerfile with CGo: golang:bookworm builder + debian:bookworm-slim runtime"]

key-files:
  created:
    - cmd/proxy-server/proxy.go
    - cmd/proxy-server/proxy_test.go
    - cmd/proxy-server/main.go
    - server/Dockerfile.proxy
  modified:
    - server/docker-compose.yml
    - server/.env.example
    - Makefile

key-decisions:
  - "goproxy.CertStorage uses Fetch(host, gen) pattern -- gen() called on cache miss, not separate Store method"
  - "LRU eviction threshold set to 1024 entries -- covers typical browsing session with headroom"
  - "Proxy ports (8443, 8080) mapped on wireguard container, NOT on proxy service (network_mode constraint)"
  - "build target remains Pi-only (CGO_ENABLED=0); build-all includes proxy (CGO_ENABLED=1)"
  - "Docker build context is repo root (..) because Go module files are at repo root, not server/"

patterns-established:
  - "goproxy conditional MITM: bypass domains -> ConnectAccept, all others -> ConnectMitm with CA cert"
  - "CertStorage with LRU: in-memory cache with insertion-order eviction at capacity"
  - "Proxy daemon main pattern: config -> CA cert -> bypass domains -> DB -> transcoder -> minifier -> chain -> goproxy -> signals"
  - "Docker network_mode: service:wireguard -- all port mappings must go on the wireguard service"

requirements-completed: [PROXY-01, PROXY-02]

# Metrics
duration: 11min
completed: 2026-03-23
---

# Phase 4 Plan 3: Proxy Server Wiring and Docker Deployment Summary

**goproxy MITM proxy with conditional SNI bypass, LRU CertStorage, Docker Compose deployment sharing WireGuard network namespace**

## Performance

- **Duration:** 11 min
- **Started:** 2026-03-23T23:28:18Z
- **Completed:** 2026-03-23T23:39:22Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- goproxy proxy server with conditional MITM: bypass domains (banking, auth, aviation) get ConnectAccept, all others get ConnectMitm with CA cert for content transformation
- memCertStore implements goproxy.CertStorage with LRU eviction at 1024 entries for on-the-fly leaf certificate caching (D-13)
- Main entry point wires all Plan 01 + Plan 02 components into a running server with graceful shutdown
- Multi-stage Docker image builds with CGo for libwebp; docker-compose.yml deploys WireGuard + proxy in shared network namespace
- 43 total tests across all 3 plans (9 new), all passing

## Task Commits

Each task was committed atomically:

1. **Task 1: goproxy setup with conditional MITM, CertStore, and main entry point** - `268307d` (feat)
2. **Task 2: Docker deployment and Makefile integration** - `8e5e516` (feat)

_TDD workflow: tests written first (RED), implementation passes all tests (GREEN)._

## Files Created/Modified
- `cmd/proxy-server/proxy.go` - SetupProxy with conditional MITM, BypassSet (exact+wildcard), memCertStore (LRU CertStorage)
- `cmd/proxy-server/proxy_test.go` - 9 tests: BypassSet exact/wildcard/mixed/empty, stripPort, memCertStore set/get/LRU, SetupProxy
- `cmd/proxy-server/main.go` - Entry point: config load, CA cert, bypass domains, DB, transcoder, minifier, chain, goproxy, CA download, signals
- `server/Dockerfile.proxy` - Multi-stage CGo build: golang:1.26-bookworm + debian:bookworm-slim with libwebp
- `server/docker-compose.yml` - Added proxy service with network_mode: service:wireguard, volume mounts, port mappings on wireguard
- `server/.env.example` - Added proxy port documentation
- `Makefile` - Added build-proxy, test-proxy, build-all, test-all, docker-build, docker-up, docker-down targets

## Decisions Made
- goproxy.CertStorage interface uses `Fetch(host, gen)` pattern where gen() is called on cache miss -- different from plan's assumed Fetch/Store separate methods. Adapted implementation to match actual goproxy v1.8.2 interface.
- LRU eviction threshold of 1024 entries provides ample headroom for typical in-flight browsing sessions
- Proxy ports mapped on wireguard container per Docker network_mode: service:wireguard constraint (Pitfall 5 from RESEARCH.md)
- `build` target stays Pi-only (CGO_ENABLED=0); `build-all` includes proxy (CGO_ENABLED=1) to avoid breaking existing Pi workflows

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Adapted memCertStore to actual goproxy.CertStorage interface**
- **Found during:** Task 1 (proxy.go implementation)
- **Issue:** Plan specified `Fetch(host, *ProxyCtx)` + `Store(host, cert, *ProxyCtx)` but actual goproxy v1.8.2 CertStorage interface is `Fetch(host, gen func() (*tls.Certificate, error)) (*tls.Certificate, error)` -- a single method with generator fallback
- **Fix:** Rewrote memCertStore to implement actual CertStorage interface: check cache, on miss call gen(), store result with LRU eviction, return cert
- **Files modified:** cmd/proxy-server/proxy.go, cmd/proxy-server/proxy_test.go
- **Verification:** TestMemCertStore_SetAndGet and TestMemCertStore_LRUEviction pass
- **Committed in:** 268307d (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug -- wrong interface signature in plan)
**Impact on plan:** Interface adaptation necessary for correctness. Same LRU caching behavior, different method signature. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all functions are fully implemented with working data paths.

## Next Phase Readiness
- Complete proxy server binary ready: config, CA cert, DB, transcoder, minifier, handler chain, goproxy with conditional MITM
- Docker deployment: `docker compose up -d` in server/ starts both WireGuard and proxy
- CA cert downloadable at http://localhost:8080/ca.crt for Phase 5 captive portal integration
- PROXY-01 (Go MITM proxy with image transcoding + minification) and PROXY-02 (one-command Docker deployment) both complete
- 43 tests pass across all proxy-server modules, go vet clean

## Self-Check: PASSED

All 7 files verified on disk. Both task commits (268307d, 8e5e516) verified in git history.

---
*Phase: 04-content-compression-proxy*
*Completed: 2026-03-23*
