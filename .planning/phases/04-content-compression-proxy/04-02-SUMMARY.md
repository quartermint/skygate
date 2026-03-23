---
phase: 04-content-compression-proxy
plan: 02
subsystem: proxy
tags: [go, webp, minify, image-transcoding, cgo, goproxy, content-type-routing]

# Dependency graph
requires:
  - phase: 04-content-compression-proxy
    plan: 01
    provides: "Config struct (ImageConfig, MinifyConfig), DB with LogCompression, CA cert generation"
provides:
  - "Transcoder with JPEG/PNG to WebP at q30, max 800px resize, 500ms timeout, concurrency semaphore"
  - "Minifier wrapping tdewolff/minify for HTML, CSS, JS, SVG, JSON with per-type enable/disable"
  - "DecompressIfNeeded for gzip Content-Encoding handling before transformation"
  - "HandlerChain with Content-Type routing: image/* to transcoder, text/html+css+js to minifier, passthrough"
  - "isImage and isMinifiable helpers for Content-Type dispatch (GIF passthrough, SVG to minifier)"
affects: [04-content-compression-proxy]

# Tech tracking
tech-stack:
  added: [elazarl/goproxy v1.8.2, kolesa-team/go-webp v1.0.5, tdewolff/minify v2.24.10, disintegration/imaging v1.6.2]
  patterns: ["Image transcoding pipeline: decode -> resize -> WebP encode with context.WithTimeout", "Text minification via tdewolff/minify with per-type config flags", "Content-Type response routing: image/* -> transcoder, text/* -> minifier, else passthrough", "Gzip decompression before content transformation (Pitfall 1)", "Semaphore-based concurrency limiting for image transcoding (Pitfall 3)"]

key-files:
  created:
    - cmd/proxy-server/transcoder.go
    - cmd/proxy-server/transcoder_test.go
    - cmd/proxy-server/minifier.go
    - cmd/proxy-server/minifier_test.go
    - cmd/proxy-server/handlers.go
    - cmd/proxy-server/handlers_test.go

key-decisions:
  - "400x400 minimum PNG test size for reliable WebP savings (100x100 PNG too small for consistent compression gain)"
  - "Minifier.Minify returns original on error rather than propagating error (graceful degradation)"
  - "Brotli decompression deferred to v2 -- logged warning, body passed through unchanged"
  - "Handler chain uses nil-safe DB pointer -- if db is nil, compression logging is silently skipped"

patterns-established:
  - "TDD for proxy pipeline: programmatic test images (makeJPEG/makePNG helpers) instead of fixture files"
  - "Content-Type routing convention: image/gif excluded from transcoder (D-04), image/svg+xml routed to minifier (D-05)"
  - "Semaphore concurrency limit pattern: make(chan struct{}, N) with acquire/defer-release"
  - "io.LimitReader for bounded body reads in proxy handlers (anti-pattern: unbounded io.ReadAll)"

requirements-completed: [PROXY-01]

# Metrics
duration: 7min
completed: 2026-03-23
---

# Phase 4 Plan 2: Image Transcoding and Text Minification Pipeline Summary

**WebP image transcoding (q30, 800px max, 500ms timeout) and JS/CSS/HTML minification with Content-Type response routing**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-23T23:16:01Z
- **Completed:** 2026-03-23T23:23:46Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Image transcoder decodes JPEG/PNG, resizes to max 800px width via Lanczos, encodes to WebP at q30 with 500ms context timeout
- Text minifier wraps tdewolff/minify with per-type config flags (HTML, CSS, JS, SVG, JSON), returns original on error
- Content-Type response handler chain routes image/* to transcoder, text/html+css+js to minifier, GIF passthrough, SVG to minifier
- Gzip Content-Encoding decompressed before transformation with header cleanup (Pitfall 1)
- 23 new tests (12 transcoder/minifier + 11 handler) all passing, 34 total with Plan 01

## Task Commits

Each task was committed atomically:

1. **Task 1: Image transcoder and text minifier modules** - `0f75b28` (feat)
2. **Task 2: Response handler dispatch with Content-Type routing** - `13016d0` (feat)

_TDD workflow: tests written first (RED), implementation passes all tests (GREEN)._

## Files Created/Modified
- `cmd/proxy-server/transcoder.go` - Transcoder struct with NewTranscoder, Transcode method, doTranscode internal, semaphore concurrency
- `cmd/proxy-server/transcoder_test.go` - 6 tests: JPEG-to-WebP, PNG-to-WebP, resize, timeout, skip-small, skip-larger
- `cmd/proxy-server/minifier.go` - Minifier struct with NewMinifier, Minify, CanMinify methods, DecompressIfNeeded function
- `cmd/proxy-server/minifier_test.go` - 6 tests: JS, CSS, HTML, SVG minify, disabled config, gzip decompress
- `cmd/proxy-server/handlers.go` - HandlerChain with HandleResponse, isImage, isMinifiable, extractDomain helpers
- `cmd/proxy-server/handlers_test.go` - 11 tests: JPEG, PNG, GIF passthrough, SVG minify, JS, CSS, HTML, passthrough, nil body, gzip JS, Content-Length
- `go.mod` - Added goproxy, go-webp, minify/v2, imaging dependencies
- `go.sum` - Updated with new dependency checksums

## Decisions Made
- Minifier.Minify returns original on error rather than propagating -- graceful degradation prevents broken responses
- Brotli decompression deferred to v2 (logged warning, body passed through) -- brotli stdlib support not available without external dep
- Handler chain uses nil-safe DB pointer -- allows test construction without database setup
- Test images generated programmatically (makeJPEG/makePNG helpers) rather than fixture files -- no test data to maintain

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Adjusted PNG test dimensions for reliable WebP savings**
- **Found during:** Task 1 (transcoder tests)
- **Issue:** 100x100 PNG with gradient produced WebP output larger than the compact PNG, causing TestTranscodePNGToWebP to fail (Pitfall 2 in action)
- **Fix:** Increased test PNG dimensions to 400x400 where WebP compression reliably produces smaller output
- **Files modified:** cmd/proxy-server/transcoder_test.go
- **Verification:** TestTranscodePNGToWebP passes consistently
- **Committed in:** 0f75b28 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Test dimension adjustment necessary for reliable test assertions. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all functions are fully implemented with working data paths.

## Next Phase Readiness
- Transcoder, Minifier, and HandlerChain modules ready for goproxy integration (Plan 03)
- All 4 new Go dependencies (goproxy, go-webp, minify/v2, imaging) in go.mod
- 34 tests pass across Plan 01 + Plan 02, go vet clean
- CGo required at build time for go-webp (libwebp); Dockerfile.proxy in Plan 03 will install libwebp-dev

## Self-Check: PASSED
