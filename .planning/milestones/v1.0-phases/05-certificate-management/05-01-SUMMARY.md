---
phase: 05-certificate-management
plan: 01
subsystem: infra
tags: [x509, ecdsa, tls, pki, certificate-authority, mitm-bypass, goproxy]

# Dependency graph
requires:
  - phase: 04-content-compression-proxy
    provides: "Go proxy server with goproxy MITM, BypassSet, CertStore, LoadOrGenerateCA"
provides:
  - "GenerateRootCA with SSID-aware CN, 3-year validity, ECDSA P-256"
  - "GenerateIntermediateCA signed by root with MaxPathLen=0, 1-year validity"
  - "hardcodedBypassDomains var with banking/auth/gov/health/payment/aviation domains"
  - "BuildBypassSet merger function (hardcoded + user YAML, graceful degradation)"
  - "Config fields for IntermediateCACertPath and IntermediateCAKeyPath"
affects: [05-certificate-management, proxy-server]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Root+intermediate CA delegation model", "Hardcoded never-MITM domains in Go source", "BuildBypassSet merger with graceful degradation"]

key-files:
  created: []
  modified:
    - "cmd/proxy-server/certgen.go"
    - "cmd/proxy-server/certgen_test.go"
    - "cmd/proxy-server/proxy.go"
    - "cmd/proxy-server/proxy_test.go"
    - "cmd/proxy-server/config.go"

key-decisions:
  - "Root CA uses random 128-bit serial numbers (not sequential) for security"
  - "fileExists helper and loadRootCA extracted for reuse across certgen functions"
  - "BuildBypassSet logs warning on user file failure but never returns error (graceful degradation)"
  - "Intermediate CA cert+key paths added to Config but main.go wiring deferred to Plan 03"

patterns-established:
  - "Root+intermediate CA hierarchy: root key stays on Pi, intermediate delegates to remote proxy"
  - "Hardcoded security domains in Go source (not config) to prevent accidental removal"
  - "Idempotent cert generation: check existence first, load if present, generate if missing"

requirements-completed: [CERT-02, CERT-03]

# Metrics
duration: 6min
completed: 2026-03-24
---

# Phase 5 Plan 1: Certificate Management Foundation Summary

**Root CA (SSID-aware, 3-year, ECDSA P-256) + intermediate CA (MaxPathLen=0, 1-year) delegation model with hardcoded never-MITM bypass domains for banking/auth/gov/health/payments**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-24T00:31:50Z
- **Completed:** 2026-03-24T00:38:03Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Root CA generation with appliance SSID in CommonName, 3-year validity, ECDSA P-256, idempotent load-or-generate
- Intermediate CA signed by root with MaxPathLen=0 constraint, 1-year validity, chain validation and leaf signing verified
- Hardcoded never-MITM bypass domains embedded in Go source covering banking, auth, government, health, payment, and aviation categories
- BuildBypassSet merger that combines hardcoded domains with user YAML config, with graceful degradation on missing/invalid user file

## Task Commits

Each task was committed atomically:

1. **Task 1: Root CA and intermediate CA generation** - `c82da59` (feat)
2. **Task 2: Hardcoded bypass domains and BuildBypassSet** - `2f00588` (feat)

## Files Created/Modified
- `cmd/proxy-server/certgen.go` - Added GenerateRootCA, GenerateIntermediateCA, helper functions (fileExists, loadRootCA, randomSerial)
- `cmd/proxy-server/certgen_test.go` - 5 new tests: root CA generation, idempotent load, intermediate CA, chain validation, leaf signing
- `cmd/proxy-server/proxy.go` - Added hardcodedBypassDomains var (28 domains) and BuildBypassSet merger function
- `cmd/proxy-server/proxy_test.go` - 4 new tests: hardcoded domains verification, merge behavior, empty user file, user-cannot-remove-hardcoded
- `cmd/proxy-server/config.go` - Added IntermediateCACertPath and IntermediateCAKeyPath fields to Config struct

## Decisions Made
- Root CA uses random 128-bit serial numbers from crypto/rand (not sequential big.NewInt(1)) for cryptographic uniqueness across appliances
- Extracted fileExists and loadRootCA as private helpers to keep GenerateRootCA readable and support future reuse
- BuildBypassSet returns (*BypassSet, error) but never returns an error in practice -- user file failures are logged as warnings and hardcoded domains are used as fallback
- Config fields for intermediate CA paths added now but main.go wiring intentionally deferred to Plan 03 (integration plan)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- GenerateRootCA and GenerateIntermediateCA ready for Plan 02 (captive portal cert download) and Plan 03 (main.go integration)
- BuildBypassSet ready to replace the current NewBypassSet(bypassDomains) call in main.go
- Intermediate CA config fields ready for Plan 03 wiring
- All 28 existing + 9 new tests passing (28 total in test suite)

---
*Phase: 05-certificate-management*
*Completed: 2026-03-24*
