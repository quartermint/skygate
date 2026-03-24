---
phase: 05-certificate-management
plan: 03
subsystem: infra
tags: [ansible, nftables, docker-compose, intermediate-ca, maxsavings-ipset, per-device-mitm, wireguard]

# Dependency graph
requires:
  - phase: 05-certificate-management
    provides: "GenerateRootCA, GenerateIntermediateCA, BuildBypassSet, hardcodedBypassDomains, intermediate CA config fields (Plan 01)"
  - phase: 05-certificate-management
    provides: "Mode selection API, /api/mode/ips endpoint, device_modes table (Plan 02)"
  - phase: 04-content-compression-proxy
    provides: "Go proxy server with goproxy MITM, BypassSet, CertStore, LoadOrGenerateCA, Makefile, Docker Compose"
provides:
  - "Ansible certificate role generating root + intermediate CA on first boot via openssl"
  - "nftables maxsavings_macs set for per-device mode tracking"
  - "User-extensible cert bypass YAML config at pi/config/cert-bypass-domains.yaml"
  - "Proxy intermediate CA loading with root CA fallback (backward compatible)"
  - "MaxSavingsIPSet polling Pi dashboard /api/mode/ips for per-device MITM decision"
  - "SetupProxy per-device mode awareness: Quick Connect gets ConnectAccept, Max Savings gets ConnectMitm"
  - "Docker Compose intermediate CA volume mount from server/ca/ directory"
  - "Makefile provision-ca helper target"
  - "DashboardAPIURL proxy config field for proxy-to-Pi communication"
affects: [05-certificate-management, proxy-server, deployment]

# Tech tracking
tech-stack:
  added: [encoding/json, time]
  patterns: ["Per-device MITM via REST API polling", "Intermediate CA delegation with root CA fallback", "Ansible first-boot idempotent CA generation"]

key-files:
  created:
    - "pi/ansible/roles/certificate/tasks/main.yml"
    - "pi/ansible/roles/certificate/templates/ca-generate.sh.j2"
    - "pi/ansible/roles/certificate/handlers/main.yml"
    - "pi/config/cert-bypass-domains.yaml"
  modified:
    - "pi/ansible/roles/networking/templates/nftables.conf.j2"
    - "cmd/dashboard-daemon/config.go"
    - "cmd/proxy-server/main.go"
    - "cmd/proxy-server/config.go"
    - "cmd/proxy-server/proxy.go"
    - "cmd/proxy-server/proxy_test.go"
    - "cmd/proxy-server/config_test.go"
    - "server/docker-compose.yml"
    - "server/proxy-config.yaml"
    - "Makefile"

key-decisions:
  - "MaxSavingsIPSet polls /api/mode/ips every 10s with 5s HTTP timeout -- graceful degradation on failure uses last known set"
  - "Empty DashboardAPIURL disables per-device mode -- MITM all non-bypass traffic (pre-Phase 5 backward compatibility)"
  - "Intermediate CA loaded via tls.LoadX509KeyPair with fallback to LoadOrGenerateCA root CA"
  - "extractSourceIP uses req.RemoteAddr (preserved through WireGuard tunnel) for source IP detection"
  - "proxy-config.yaml updated with production-tuned values (5MB max image, 7-day retention, 30s batch interval, JSON minify disabled)"

patterns-established:
  - "Per-device MITM via source IP mapping: proxy polls Pi dashboard API, checks source IP against MaxSavingsIPSet"
  - "Ansible certificate role: idempotent first-boot CA generation with restart handler"
  - "Docker Compose CA provisioning: server/ca/ directory mounted read-only into proxy container"

requirements-completed: [CERT-01, CERT-02, CERT-03]

# Metrics
duration: 8min
completed: 2026-03-24
---

# Phase 5 Plan 3: Deployment Pipeline Integration Summary

**Ansible certificate role, nftables maxsavings set, proxy intermediate CA loading with per-device MaxSavingsIPSet polling, Docker Compose CA volume mount, and Makefile provisioning target**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-24T00:44:12Z
- **Completed:** 2026-03-24T00:52:38Z
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments
- Ansible certificate role generates root + intermediate CA on first boot using openssl (ECDSA P-256, 3-year root, 1-year intermediate with pathlen:0)
- Proxy loads intermediate CA for MITM leaf signing with automatic fallback to root CA for pre-Phase 5 compatibility
- Per-device MITM decision via MaxSavingsIPSet polling Pi dashboard /api/mode/ips: Quick Connect devices get TCP passthrough (ConnectAccept), Max Savings devices get MITM (ConnectMitm)
- Complete deployment pipeline: nftables maxsavings_macs set, Docker Compose CA volume mount, Makefile provision-ca target, user-extensible bypass config

## Task Commits

Each task was committed atomically:

1. **Task 1: Ansible certificate role, nftables maxsavings set, user bypass config** - `6879329` (feat)
2. **Task 2: Proxy intermediate CA loading, Docker Compose, Makefile** - `adc319b` (feat)
3. **Task 3: MaxSavingsIPSet and per-device MITM decision tests** - `f6a306d` (test)

## Files Created/Modified
- `pi/ansible/roles/certificate/tasks/main.yml` - CA generation Ansible role with first-boot idempotency
- `pi/ansible/roles/certificate/templates/ca-generate.sh.j2` - openssl script for root + intermediate CA generation
- `pi/ansible/roles/certificate/handlers/main.yml` - restart dashboard handler
- `pi/config/cert-bypass-domains.yaml` - User-extensible cert-pinning bypass domain list
- `pi/ansible/roles/networking/templates/nftables.conf.j2` - Added maxsavings_macs set (ether_addr, 24h timeout)
- `cmd/dashboard-daemon/config.go` - Added CertBypassFile field with default path
- `cmd/proxy-server/main.go` - Intermediate CA loading with fallback, BuildBypassSet, MaxSavingsIPSet creation and polling
- `cmd/proxy-server/config.go` - Added DashboardAPIURL field
- `cmd/proxy-server/proxy.go` - MaxSavingsIPSet type, extractSourceIP, updated SetupProxy with per-device MITM logic
- `cmd/proxy-server/proxy_test.go` - 8 new tests for MaxSavingsIPSet and per-device MITM behavior
- `cmd/proxy-server/config_test.go` - Updated expectations for new proxy-config.yaml values
- `server/docker-compose.yml` - Added ./ca:/data/skygate/ca:ro volume mount with provisioning comments
- `server/proxy-config.yaml` - Added intermediate CA paths, dashboard API URL, tuned production values
- `Makefile` - Added provision-ca target and .PHONY entry

## Decisions Made
- MaxSavingsIPSet polls /api/mode/ips every 10 seconds with 5-second HTTP timeout; on failure, continues with last known set (graceful degradation per Pitfall 6)
- Empty DashboardAPIURL disables per-device mode awareness entirely -- all non-bypass traffic gets MITM (pre-Phase 5 backward compatibility)
- Intermediate CA loaded via tls.LoadX509KeyPair; on failure, falls back to LoadOrGenerateCA for the root/auto-generated CA
- extractSourceIP uses req.RemoteAddr which is preserved through the WireGuard tunnel from the Pi's 192.168.4.x subnet
- proxy-config.yaml updated to production-tuned values: 5MB max image size (was 10MB), 7-day log retention (was 30), 30s batch interval (was 10s), JSON minification disabled

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated config_test.go expectations for new proxy-config.yaml values**
- **Found during:** Task 2
- **Issue:** TestLoadConfig_Valid was asserting old proxy-config.yaml values (ca.crt, ca.key, 10MB max, JSON minify true, 30-day retention, 10s batch) that no longer matched the updated config file
- **Fix:** Updated test expectations to match new values (root-ca.crt, root-ca.key, 5MB, JSON false, 7 days, 30s) and added assertions for new Phase 5 fields (IntermediateCACertPath, IntermediateCAKeyPath, DashboardAPIURL)
- **Files modified:** cmd/proxy-server/config_test.go
- **Verification:** All proxy tests pass
- **Committed in:** adc319b (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Test expectations needed updating to match the proxy-config.yaml changes prescribed by the plan. No scope creep.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all code is production-ready. MaxSavingsIPSet is fully functional with enabled/disabled modes.

## Next Phase Readiness
- Complete certificate management pipeline: CA generation (Ansible) -> intermediate CA provisioning (SCP) -> proxy loading -> per-device MITM
- All Phase 5 plans (01, 02, 03) are complete -- certificate foundation, mode selection, deployment integration
- D-19 cert cache satisfied by existing memCertStore (LRU, 1024 entries) from Phase 4
- Full test suite green across all Go packages

## Self-Check: PASSED

All 14 files verified on disk. All 3 commit hashes (6879329, adc319b, f6a306d) verified in git log.

---
*Phase: 05-certificate-management*
*Completed: 2026-03-24*
