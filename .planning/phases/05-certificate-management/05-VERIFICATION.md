---
phase: 05-certificate-management
verified: 2026-03-24T00:00:00Z
status: passed
score: 14/14 must-haves verified
re_verification: false
---

# Phase 5: Certificate Management Verification Report

**Phase Goal:** Passengers choose their savings level -- "Quick Connect" for zero-friction DNS blocking or "Max Savings" with CA cert install for full proxy compression -- and cert-pinned apps never break
**Verified:** 2026-03-24
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| #  | Truth | Status | Evidence |
|----|-------|--------|---------|
| 1  | Captive portal presents Quick Connect and Max Savings as two distinct options | ✓ VERIFIED | `pi/static/mode-select.html` lines 130, 138: "Quick Connect" and "Max Savings" cards with distinct CTAs |
| 2  | Quick Connect is the default requiring zero extra taps | ✓ VERIFIED | `db.go` GetDeviceMode returns "quickconnect" for unknown MACs; mode-select.html pre-selects Quick Connect path |
| 3  | Max Savings triggers per-platform cert install guide (iOS .mobileconfig, Android .crt) | ✓ VERIFIED | `certdownload.go` HandleMobileConfig and HandleCertDownloadDER; mode-select.html JS redirects to cert-install-ios.html or cert-install-android.html |
| 4  | Per-device mode tracked by MAC and persisted in SQLite | ✓ VERIFIED | `db.go` device_modes table (line 125-129), SetDeviceMode/GetDeviceMode; all tests pass |
| 5  | nftables maxsavings_macs set updated when device switches to Max Savings | ✓ VERIFIED | `mode_linux.go` AddMaxSavingsMAC/RemoveMaxSavingsMAC; nftables.go nftMaxSavingsSet constant; mode.go calls both on HandleSetMode |
| 6  | iOS users can download .mobileconfig with correct MIME type | ✓ VERIFIED | `certdownload.go` line 100: Content-Type "application/x-apple-aspen-config"; tests pass |
| 7  | Android users can download .crt DER with correct MIME type | ✓ VERIFIED | `certdownload.go` line 115: Content-Type "application/x-x509-ca-cert"; tests pass |
| 8  | Banking/auth/gov/health/payment domains are hardcoded bypass -- cert-pinned apps never break | ✓ VERIFIED | `proxy.go` hardcodedBypassDomains var (lines 19-35): 28 domains across 5 categories; cannot be removed by config |
| 9  | User-extensible bypass domains merge with hardcoded list without overriding | ✓ VERIFIED | `proxy.go` BuildBypassSet (lines 41-55): copies hardcoded slice, appends user YAML; TestBuildBypassSet_UserCannotRemoveHardcoded passes |
| 10 | Root CA generated with appliance SSID in CN (3-year, ECDSA P-256) | ✓ VERIFIED | `certgen.go` GenerateRootCA (line 136): CN = "SkyGate-<ssid> CA", 3-year NotAfter; TestGenerateRootCA passes |
| 11 | Intermediate CA signed by root with MaxPathLen=0 (1-year) | ✓ VERIFIED | `certgen.go` GenerateIntermediateCA (lines 222-223): MaxPathLen=0, MaxPathLenZero=true; TestIntermediateCAChainValidation and TestIntermediateCALeafSigning pass |
| 12 | Proxy uses intermediate CA for leaf signing, not root CA | ✓ VERIFIED | `main.go` (lines 37-55): IntermediateCACertPath loaded first, falls back to root CA; mitmCert passed to SetupProxy |
| 13 | Quick Connect devices get TCP passthrough (no MITM), Max Savings devices get MITM | ✓ VERIFIED | `proxy.go` SetupProxy HandleConnectFunc (lines 285-293): maxSavingsIPs.Contains(sourceIP) gates ConnectMitm vs ConnectAccept; TestSetupProxy_QuickConnectPassthrough and TestSetupProxy_MaxSavingsMITM pass |
| 14 | Ansible certificate role generates root + intermediate CA on first boot | ✓ VERIFIED | `pi/ansible/roles/certificate/tasks/main.yml`: idempotent first-boot check, openssl generation script deployed |

**Score:** 14/14 truths verified

### Required Artifacts

#### Plan 01 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/proxy-server/certgen.go` | GenerateRootCA, GenerateIntermediateCA | ✓ VERIFIED | Both functions present, substantive (262 lines), called from main.go |
| `cmd/proxy-server/proxy.go` | hardcodedBypassDomains, BuildBypassSet | ✓ VERIFIED | 28-domain hardcoded list, BuildBypassSet merger, both wired into main.go |
| `cmd/proxy-server/certgen_test.go` | Root CA, intermediate CA, chain tests | ✓ VERIFIED | 5 tests: TestGenerateRootCA, TestGenerateRootCA_ExistingFiles, TestGenerateIntermediateCA, TestIntermediateCAChainValidation, TestIntermediateCALeafSigning |
| `cmd/proxy-server/proxy_test.go` | Hardcoded bypass and merge tests | ✓ VERIFIED | 4 bypass tests + 8 MaxSavingsIPSet/SetupProxy tests added |
| `cmd/proxy-server/config.go` | IntermediateCACertPath, IntermediateCAKeyPath, DashboardAPIURL | ✓ VERIFIED | All three fields present in Config struct |

#### Plan 02 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/dashboard-daemon/mode.go` | HandleGetMode, HandleSetMode, HandleGetMaxSavingsIPs | ✓ VERIFIED | All three handlers present; ModeQuickConnect/ModeMaxSavings constants defined |
| `cmd/dashboard-daemon/mode_linux.go` | AddMaxSavingsMAC, RemoveMaxSavingsMAC | ✓ VERIFIED | Build-tagged `//go:build linux`; calls nft command |
| `cmd/dashboard-daemon/mode_stub.go` | macOS dev stubs | ✓ VERIFIED | Build-tagged `//go:build !linux`; no-op with logging |
| `cmd/dashboard-daemon/mode_test.go` | Mode API, DB, IP join tests | ✓ VERIFIED | 11 tests covering set/get/invalid/missing/default/ip-join scenarios |
| `cmd/dashboard-daemon/certdownload.go` | HandleMobileConfig, HandleCertDownloadDER | ✓ VERIFIED | Both handlers present with correct MIME types; mobileconfigTemplate constant defined |
| `cmd/dashboard-daemon/certdownload_test.go` | Cert download tests | ✓ VERIFIED | 4 tests with real x509 cert fixture |
| `cmd/dashboard-daemon/db.go` | device_modes table, SetDeviceMode, GetDeviceMode, GetMaxSavingsMACs, GetMaxSavingsIPs | ✓ VERIFIED | device_modes table in migrate(), all 4 methods present; JOIN query for IP mapping |
| `pi/static/mode-select.html` | Quick Connect / Max Savings options | ✓ VERIFIED | Both options present with CTAs; platform detection JS redirects to correct cert install guide |
| `pi/static/cert-install-ios.html` | iOS 3-step guide with Certificate Trust Settings | ✓ VERIFIED | Step 3 explicitly references Certificate Trust Settings (line 136) |
| `pi/static/cert-install-android.html` | Android guide with browser-only caveat | ✓ VERIFIED | "browser traffic" caveat present (line 129) |
| `pi/static/cert-remove.html` | Post-flight removal for iOS/Android/macOS/Windows | ✓ VERIFIED | All 4 platform sections present |

#### Plan 03 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pi/ansible/roles/networking/templates/nftables.conf.j2` | maxsavings_macs set | ✓ VERIFIED | `set maxsavings_macs { type ether_addr; flags timeout; timeout 24h; }` at line 16 |
| `pi/ansible/roles/certificate/tasks/main.yml` | CA generation Ansible role | ✓ VERIFIED | Idempotent first-boot check, deploys ca-generate.sh.j2, sets permissions |
| `pi/ansible/roles/certificate/templates/ca-generate.sh.j2` | openssl CA generation script | ✓ VERIFIED | ECDSA P-256 root (3yr) + intermediate (1yr pathlen:0) generation |
| `pi/ansible/roles/certificate/handlers/main.yml` | restart dashboard handler | ✓ VERIFIED | systemd restart handler present |
| `pi/config/cert-bypass-domains.yaml` | User-extensible bypass list | ✓ VERIFIED | bypass_domains key present; commented examples |
| `cmd/proxy-server/main.go` | Intermediate CA loading, BuildBypassSet, MaxSavingsIPSet | ✓ VERIFIED | All three wired correctly (lines 37-84) |
| `server/docker-compose.yml` | Intermediate CA volume mount | ✓ VERIFIED | `./ca:/data/skygate/ca:ro` mount present with provisioning comments |
| `server/proxy-config.yaml` | intermediate_ca_cert_path, intermediate_ca_key_path, dashboard_api_url | ✓ VERIFIED | All three fields present in production config |
| `Makefile` | provision-ca target | ✓ VERIFIED | `provision-ca:` target present; in .PHONY list |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `certgen.go` | `proxy.go` / `main.go` | GenerateIntermediateCA returns *tls.Certificate passed to SetupProxy | ✓ WIRED | main.go uses tls.LoadX509KeyPair for intermediate CA, passes mitmCert to SetupProxy |
| `proxy.go` | `config.go` | BuildBypassSet calls LoadBypassDomains via cfg.BypassDomainsFile | ✓ WIRED | main.go line 58: BuildBypassSet(cfg.BypassDomainsFile) |
| `mode.go` | `mode_linux.go` | HandleSetMode calls AddMaxSavingsMAC/RemoveMaxSavingsMAC | ✓ WIRED | mode.go lines 61-67: conditional call based on mode value |
| `mode.go` | `db.go` | HandleSetMode persists mode; HandleGetMaxSavingsIPs queries JOIN | ✓ WIRED | mode.go lines 52, 106: s.db.SetDeviceMode and s.db.GetMaxSavingsIPs |
| `main.go` (dashboard) | `mode.go` | mux.HandleFunc registers /api/mode and /api/mode/ips | ✓ WIRED | main.go lines 95-107: GET/POST dispatch and HandleGetMaxSavingsIPs |
| `main.go` (dashboard) | `certdownload.go` | mux.HandleFunc registers /ca.mobileconfig and /ca.crt | ✓ WIRED | main.go lines 110-111 |
| `ansible/certificate` | `/data/skygate/ca/` | Ansible task generates root CA + intermediate CA on first boot | ✓ WIRED | tasks/main.yml: ca-generate.sh.j2 deployed and run when root-ca.crt missing |
| `docker-compose.yml` | `main.go` (proxy) | Intermediate CA volume mounted into proxy container | ✓ WIRED | `./ca:/data/skygate/ca:ro` maps to IntermediateCACertPath="/data/skygate/ca/intermediate-ca.crt" |
| `proxy.go` MaxSavingsIPSet | dashboard `/api/mode/ips` | StartPolling polls Pi dashboard every 10s for Max Savings IPs | ✓ WIRED | proxy.go StartPolling calls fetchAndUpdate with url = apiURL + "/api/mode/ips" |
| `main.go` (proxy) | `proxy.go` SetupProxy | MaxSavingsIPSet created and passed as 3rd arg to SetupProxy | ✓ WIRED | main.go lines 80-84: NewMaxSavingsIPSet, go StartPolling, SetupProxy(mitmCert, bypassSet, maxSavingsIPs, chain, verbose) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| `mode-select.html` | mode choice (Quick Connect / Max Savings) | POST /api/mode -> HandleSetMode -> db.SetDeviceMode | Yes -- persisted to SQLite device_modes | ✓ FLOWING |
| `proxy.go` MaxSavingsIPSet | ips map[string]bool | GET /api/mode/ips -> GetMaxSavingsIPs -> JOIN device_modes + portal_accepted | Yes -- real DB JOIN query | ✓ FLOWING |
| `certdownload.go` HandleMobileConfig | DER cert bytes | os.ReadFile(cfg.CACertPath) -> PEM decode -> DER | Yes -- reads from filesystem; cert path defaults to /data/skygate/ca/root-ca.crt | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| proxy-server tests pass | `CGO_ENABLED=1 go test ./cmd/proxy-server/ -v -short` | All 37 tests pass | ✓ PASS |
| dashboard-daemon tests pass | `go test ./cmd/dashboard-daemon/ -v -short` | All tests pass | ✓ PASS |
| Full suite green | `CGO_ENABLED=1 go test ./... -short` | All 4 packages pass | ✓ PASS |
| proxy binary compiles | `CGO_ENABLED=1 go build ./cmd/proxy-server/` | Exit 0 | ✓ PASS |
| dashboard binary compiles | `go build ./cmd/dashboard-daemon/` | Exit 0 | ✓ PASS |
| nftables maxsavings_macs in template | `grep maxsavings_macs pi/ansible/roles/networking/templates/nftables.conf.j2` | Line 16 matches | ✓ PASS |
| Ansible cert role exists | `test -f pi/ansible/roles/certificate/tasks/main.yml` | File exists | ✓ PASS |
| User bypass config exists | `test -f pi/config/cert-bypass-domains.yaml` | File exists | ✓ PASS |
| provision-ca in Makefile | `grep provision-ca Makefile` | Lines 1 (.PHONY) and 79 (target) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|---------|
| CERT-01 | 05-02, 05-03 | Captive portal presents Quick Connect and Max Savings mode selection | ✓ SATISFIED | mode-select.html; /api/mode endpoint registered in main.go; mode_select tests pass |
| CERT-02 | 05-01, 05-02, 05-03 | Per-device CA cert downloadable via captive portal -- iOS .mobileconfig and Android .crt | ✓ SATISFIED | certdownload.go HandleMobileConfig (application/x-apple-aspen-config) and HandleCertDownloadDER (application/x-x509-ca-cert); /ca.mobileconfig and /ca.crt registered; cert install guide pages exist |
| CERT-03 | 05-01, 05-02, 05-03 | Certificate pinning bypass prevents proxy from breaking banking, auth, and cert-pinned apps | ✓ SATISFIED | hardcodedBypassDomains (28 domains in proxy.go source); BuildBypassSet merger; SetupProxy ConnectAccept path for bypass domains; TestBuildBypassSet_UserCannotRemoveHardcoded passes |

No orphaned requirements for Phase 5. REQUIREMENTS.md maps CERT-01, CERT-02, CERT-03 to Phase 5 -- all three are accounted for in plans 01-02, 01-03, and 02-03 respectively.

### Anti-Patterns Found

No anti-patterns detected. Scan of all phase 5 modified files found:
- No TODO/FIXME/PLACEHOLDER comments
- No empty handler implementations (`return null`, `return []`, `return {}`)
- No hardcoded empty data flowing to user-visible output
- Stubs in mode_stub.go are intentional cross-platform dev stubs (established pattern matching nftables_stub.go from Phase 2), not production stubs

### Human Verification Required

The following behaviors require human testing -- they cannot be verified programmatically without running on actual hardware:

#### 1. iOS .mobileconfig Installation Flow

**Test:** On an iPhone/iPad, browse to the captive portal at http://skygate.local, choose "Max Savings", and tap "Download Certificate Profile".
**Expected:** iOS prompts "Profile Downloaded -- Review the profile in the Settings app". Navigate to Settings > General > VPN & Device Management > SkyGate Max Savings > Install. Then Settings > General > About > Certificate Trust Settings > enable full trust for SkyGate CA. No certificate error when browsing HTTPS sites.
**Why human:** Requires physical iOS device, live CA cert, and Safari's profile installation flow.

#### 2. Android .crt Installation Flow

**Test:** On an Android device, browse to the captive portal, choose "Max Savings", and download the .crt file.
**Expected:** Android prompts to install as a CA certificate. After installing, Chrome shows no certificate errors when browsing through the proxy.
**Why human:** Requires physical Android device with varying OEM cert installation UIs.

#### 3. Banking App Passthrough (cert-pinned app verification)

**Test:** With a device in "Max Savings" mode and the CA cert installed, open a banking app (Chase, Bank of America) or attempt to browse to a bank's HTTPS website.
**Expected:** Banking app and banking website work normally without certificate errors. Dashboard shows the domain was BYPASSED in proxy logs.
**Why human:** Requires live proxy + WireGuard tunnel, real banking domains, and actual MITM bypass verification.

#### 4. Quick Connect Device HTTPS Behavior

**Test:** Connect a device in "Quick Connect" mode (no CA cert installed). Browse to any HTTPS site.
**Expected:** No certificate errors. Browser connects normally because the proxy does TCP passthrough for this device's source IP.
**Why human:** Requires live proxy with MaxSavingsIPSet polling active and a real browser session.

#### 5. Post-Flight Cert Removal Instructions Accuracy

**Test:** Follow the cert-remove.html instructions on iOS: Settings > General > VPN & Device Management > SkyGate Max Savings > Remove Profile.
**Expected:** CA certificate is removed cleanly. Max Savings mode stops working (expected). Instructions match actual iOS UI text for the current iOS version.
**Why human:** iOS UI strings change between versions; requires physical device verification.

### Gaps Summary

No gaps. All 14 observable truths are verified. All artifacts exist, are substantive, and are wired correctly. All key links are confirmed in code. Both Go packages compile cleanly and all tests pass (37 proxy-server tests, all dashboard-daemon tests). Requirements CERT-01, CERT-02, and CERT-03 are fully satisfied.

The one area that warrants human verification is the physical cert install and bypass experience on real devices -- this is inherent to the nature of certificate management and cannot be verified programmatically.

---
_Verified: 2026-03-24_
_Verifier: Claude (gsd-verifier)_
