# Phase 5: Certificate Management - Research

**Researched:** 2026-03-23
**Domain:** TLS certificate lifecycle, iOS/Android cert distribution, MITM proxy intermediate CA delegation, nftables per-device mode routing
**Confidence:** HIGH

## Summary

Phase 5 bridges two existing systems -- the captive portal (Phase 2) and the content compression proxy (Phase 4) -- with a certificate lifecycle layer that lets passengers choose their savings level. The core work is: (1) extending the captive portal with mode selection UI and cert download endpoints, (2) implementing intermediate CA delegation so the remote proxy can sign leaf certs without possessing the root CA key, (3) building iOS .mobileconfig profile generation and Android .crt download handlers, and (4) extending nftables with per-device mode sets that control whether traffic flows through the MITM proxy or bypasses it.

The technical risk is LOW. All certificate operations use Go's standard `crypto/x509` and `crypto/tls` packages, which are battle-tested. The .mobileconfig format is stable XML/plist. The nftables `ether_addr` set type for per-device MAC tracking is well-documented. The primary complexity is UX: making the two-tier mode selection dead simple for non-technical passengers and providing clear, platform-specific cert installation instructions.

**Primary recommendation:** Use Go stdlib `crypto/x509` for all certificate generation (root CA, intermediate CA, leaf certs). Generate .mobileconfig profiles as Go templates producing XML plist. Extend the existing `cmd/dashboard-daemon` with mode selection endpoints and cert download handlers. Extend the existing `cmd/proxy-server/certgen.go` to support intermediate CA loading. Add a new nftables set (`maxsavings_macs`) of type `ether_addr` for per-device mode tracking.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Captive portal presents two clearly labeled options after terms acceptance: "Quick Connect" (DNS blocking only, zero setup, works immediately) and "Max Savings" (install CA cert for proxy compression, requires 2-3 extra taps per platform)
- **D-02:** "Quick Connect" is the default and pre-selected option -- passengers who just tap "Continue" get DNS-only mode. "Max Savings" requires deliberate opt-in
- **D-03:** Mode selection is per-device -- different passengers on the same flight can choose different modes. Tracked by MAC address in nftables sets
- **D-04:** Passengers can switch modes mid-flight from the dashboard settings without reconnecting to WiFi. Switching from "Max Savings" to "Quick Connect" takes effect immediately (proxy bypass). Switching from "Quick Connect" to "Max Savings" requires cert install flow
- **D-05:** Each SkyGate appliance generates its own unique CA keypair on first boot -- no shared CA key across devices
- **D-06:** CA keypair stored at `/data/skygate/ca/` (writable data partition, survives read-only root) with restrictive permissions (0600, root only)
- **D-07:** CA certificate is a self-signed root CA with reasonable validity (3 years). Common Name includes the appliance's SSID for user recognition
- **D-08:** The Go proxy on the remote server does NOT need the CA key -- it generates ephemeral leaf certs signed by a delegated intermediate cert
- **D-09:** iOS: Generate `.mobileconfig` profile containing the CA certificate with step-by-step screenshots for Settings > Profile > Certificate Trust Settings
- **D-10:** Android: Direct `.crt` (DER format) download with step-by-step guide. Android 11+ user-installed CAs only trusted by browser traffic
- **D-11:** macOS/Windows laptops: Direct `.crt` download with platform-specific instructions (lower priority)
- **D-12:** Post-flight cert removal instructions via dashboard and QR code on printed card
- **D-13:** Bypass list is a YAML config file containing domains/patterns that are NEVER intercepted by MITM
- **D-14:** Hardcoded "never-MITM" categories: banking/financial, authentication/2FA, government, health, payment processors -- cannot be removed by user config
- **D-15:** User-extensible bypass list via dashboard settings or YAML config edit
- **D-16:** Bypass implementation: domains added to nftables set that routes traffic through tunnel but skips MITM proxy (TCP passthrough)
- **D-17:** Remote Go proxy generates ephemeral TLS certificates signed by intermediate CA that chains to Pi's root CA
- **D-18:** Intermediate CA cert generated on the Pi, pushed to remote server via WireGuard control channel
- **D-19:** Cert cache on the remote proxy with TTL matching original cert validity or 24 hours (whichever shorter)

### Claude's Discretion
- Exact CA certificate parameters (key size, signature algorithm, extensions, validity period)
- `.mobileconfig` XML profile structure and signing (unsigned vs self-signed profile)
- Intermediate CA delegation mechanism (how the Pi provisions signing authority to the remote proxy)
- nftables set structure for per-device mode tracking
- Cert generation library selection for Go (crypto/x509 stdlib vs external library)
- Cert cache implementation details on the remote proxy
- Dashboard UI layout for mode switching and cert download
- Android cert trust scope limitations and user-facing messaging
- Error handling for expired/revoked intermediate certs
- Cert rotation strategy

### Deferred Ideas (OUT OF SCOPE)
- Automatic cert rotation before CA expiry (3-year validity, v2 concern)
- OCSP responder for real-time cert validation
- Per-passenger cert generation (currently per-appliance CA)
- Enterprise cert deployment via MDM for fleet operators
- Cert transparency logging
- Automatic detection of cert-pinned domains via TLS handshake failure patterns
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CERT-01 | Captive portal presents "Quick Connect" (DNS blocking only, zero friction) and "Max Savings" (proxy + CA cert) mode selection | Mode selection UI extends existing captive.html; nftables `maxsavings_macs` set tracks per-device mode; dashboard-daemon API handles mode switching |
| CERT-02 | Per-device CA certificate generated and downloadable via captive portal -- iOS .mobileconfig profile and Android cert install flow | Go stdlib `crypto/x509` generates root CA + intermediate CA; .mobileconfig generated as Go template (XML plist); .crt served as DER-encoded download; intermediate CA delegated to remote proxy |
| CERT-03 | Certificate pinning bypass list prevents proxy from breaking banking, auth, and cert-pinned apps | Existing `server/bypass-domains.yaml` and `BypassSet` in proxy.go already implement this; extend with hardcoded never-MITM categories that cannot be user-removed; add dashboard UI for user-extensible bypass domains |
</phase_requirements>

## Standard Stack

### Core (No New Dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `crypto/x509` (Go stdlib) | Go 1.26 | Certificate generation (root CA, intermediate CA, leaf certs) | Standard library, battle-tested, zero dependencies. `x509.CreateCertificate` handles all cert creation. Already used in Phase 4 `certgen.go`. |
| `crypto/ecdsa` (Go stdlib) | Go 1.26 | ECDSA P-256 key generation | Per Phase 4 decision: ECDSA P-256 -- smaller and faster than RSA, adequate for MITM leaf signing. Already used in `certgen.go`. |
| `encoding/pem` (Go stdlib) | Go 1.26 | PEM encoding for cert/key files | Standard format for cert files. Already used in `certgen.go`. |
| `encoding/xml` (Go stdlib) | Go 1.26 | .mobileconfig XML plist generation | iOS configuration profiles are XML property lists. Stdlib XML encoder handles the format. |
| `text/template` (Go stdlib) | Go 1.26 | HTML template for mode selection UI, cert install instructions | Already used pattern in dashboard static files. Extend for platform-specific pages. |
| `elazarl/goproxy` | v1.8.2 | MITM proxy with `TLSConfigFromCA` | Already in go.mod. `TLSConfigFromCA` accepts intermediate CA cert for leaf signing. |

### Supporting (Already in go.mod)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `gopkg.in/yaml.v3` | v3.0.1 | YAML config parsing | Already used. Extend for cert-bypass-domains.yaml parsing (reuse existing `LoadBypassDomains` pattern). |
| `modernc.org/sqlite` | v1.47.0 | Device mode persistence | Already used. Extend DB schema with `device_modes` table for per-device mode tracking. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go stdlib `crypto/x509` | smallstep/certificates | Overkill for single-device CA. Adds dependency for features we don't need (ACME, SCEP, provisioners). |
| Go stdlib `encoding/xml` | howett.net/plist | More plist-native but adds a dependency. Stdlib XML is sufficient for the simple .mobileconfig structure. |
| Unsigned .mobileconfig | Signed .mobileconfig (via PKCS#7) | Signing requires a separate Apple-trusted code signing cert or S/MIME cert. Unsigned profiles work fine -- iOS shows "Unsigned" warning but allows install. Signing is a v2 polish item. |

**Installation:** No new packages needed. All dependencies already in `go.mod`.

## Architecture Patterns

### Recommended Changes to Project Structure

```
cmd/dashboard-daemon/
    mode.go              # Mode selection API: POST /api/mode, GET /api/mode
    mode_linux.go         # nftables maxsavings_macs set operations
    mode_stub.go          # macOS dev stub
    mode_test.go          # Mode selection tests
    certdownload.go       # CA cert download handlers: /ca.crt, /ca.mobileconfig
    certdownload_test.go  # Cert download tests

cmd/proxy-server/
    certgen.go            # EXTEND: add LoadIntermediateCA(), update LoadOrGenerateCA()
    certgen_test.go       # EXTEND: add intermediate CA chain tests
    proxy.go              # EXTEND: TLSConfigFromCA uses intermediate cert

pi/static/
    captive.html          # EXTEND: add mode selection after terms acceptance
    mode-select.html      # NEW: dedicated mode selection page
    cert-install-ios.html # NEW: iOS cert install step-by-step guide
    cert-install-android.html # NEW: Android cert install guide
    cert-remove.html      # NEW: Post-flight cert removal guide

pi/config/
    cert-bypass-domains.yaml  # NEW: user-extensible bypass domains (separate from hardcoded)

pi/ansible/roles/
    certificate/          # NEW Ansible role for CA generation on first boot
        tasks/main.yml
        templates/
        handlers/main.yml

server/bypass-domains.yaml  # EXTEND: ensure hardcoded never-MITM categories
```

### Pattern 1: Root CA + Intermediate CA Delegation

**What:** Pi generates a root CA on first boot. It then creates an intermediate CA signed by the root. The intermediate CA (cert + key) is transferred to the remote proxy via WireGuard. The remote proxy uses the intermediate to sign leaf certs for intercepted HTTPS domains. The root CA key never leaves the Pi.

**When to use:** Always. This is the locked decision (D-08, D-17, D-18).

**Implementation:**

```go
// On Pi: generate root CA (already in certgen.go, needs adjustment)
// Store at /data/skygate/ca/root-ca.crt and /data/skygate/ca/root-ca.key

// On Pi: generate intermediate CA signed by root
func GenerateIntermediateCA(rootCert *x509.Certificate, rootKey crypto.PrivateKey, certPath, keyPath string) (*tls.Certificate, error) {
    privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

    template := &x509.Certificate{
        SerialNumber: generateSerial(),
        Subject: pkix.Name{
            Organization: []string{"SkyGate Proxy CA"},
            CommonName:   "SkyGate Intermediate CA",
        },
        NotBefore:             time.Now(),
        NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        BasicConstraintsValid: true,
        IsCA:                  true,
        MaxPathLen:            0, // Can only sign leaf certs, not sub-CAs
        MaxPathLenZero:        true,
    }

    // Sign with ROOT CA key, not self-signed
    certDER, _ := x509.CreateCertificate(rand.Reader, template, rootCert, &privKey.PublicKey, rootKey)
    // ... PEM encode and write to disk, transfer to remote proxy
}
```

**Key constraints:**
- Intermediate CA `MaxPathLen: 0` -- it can only sign leaf certs, never sub-CAs
- Intermediate CA validity: 1 year (shorter than root's 3 years, rotatable without re-distributing root)
- Root CA key permissions: 0600, root only, at `/data/skygate/ca/root-ca.key`
- Intermediate CA key: transferred to remote proxy Docker volume

### Pattern 2: iOS .mobileconfig Profile Generation

**What:** Dynamically generate a .mobileconfig XML plist containing the root CA certificate in DER/base64 format. Serve it from the dashboard daemon with correct MIME type.

**When to use:** When an iOS user selects "Max Savings" mode.

**Implementation:**

```go
// .mobileconfig is a standard Apple Property List (XML) with specific payload structure
const mobileconfigTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadCertificateFileName</key>
            <string>skygate-ca.cer</string>
            <key>PayloadContent</key>
            <data>{{.CertBase64}}</data>
            <key>PayloadDescription</key>
            <string>Adds SkyGate bandwidth optimization CA certificate</string>
            <key>PayloadDisplayName</key>
            <string>SkyGate CA Certificate</string>
            <key>PayloadIdentifier</key>
            <string>com.skygate.ca.{{.UUID1}}</string>
            <key>PayloadType</key>
            <string>com.apple.security.root</string>
            <key>PayloadUUID</key>
            <string>{{.UUID1}}</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
    <key>PayloadDescription</key>
    <string>SkyGate Max Savings - Enables bandwidth compression for browser traffic</string>
    <key>PayloadDisplayName</key>
    <string>SkyGate Max Savings</string>
    <key>PayloadIdentifier</key>
    <string>com.skygate.profile.{{.UUID2}}</string>
    <key>PayloadOrganization</key>
    <string>SkyGate</string>
    <key>PayloadRemovalDisallowed</key>
    <false/>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>{{.UUID2}}</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>`

// Handler serves the .mobileconfig with correct MIME type
func HandleMobileConfig(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/x-apple-aspen-config")
    w.Header().Set("Content-Disposition", `attachment; filename="SkyGate.mobileconfig"`)
    // ... template execution with base64-encoded DER cert data
}
```

**Key details:**
- `PayloadType: com.apple.security.root` -- installs as a root CA certificate
- `PayloadContent` contains the DER-encoded certificate data, base64-wrapped inside the plist `<data>` tag
- MIME type MUST be `application/x-apple-aspen-config` to trigger iOS profile installer
- Profile is unsigned (acceptable for self-signed CA; shows "Unsigned" in iOS installer)
- `PayloadRemovalDisallowed: false` -- user can always remove the profile
- UUIDs should be deterministic per-appliance (derived from CA cert fingerprint) so re-downloads don't create duplicate profiles

**iOS User Flow (3 steps after download):**
1. Tap download link -> iOS shows "Profile Downloaded" banner -> user taps "Close"
2. Settings > General > VPN & Device Management > SkyGate Max Savings > Install
3. Settings > General > About > Certificate Trust Settings > Enable full trust for SkyGate CA

### Pattern 3: Per-Device Mode Tracking via nftables Sets

**What:** A new nftables set `maxsavings_macs` of type `ether_addr` tracks which devices have opted into "Max Savings" mode. The prerouting chain uses this set to determine whether to route traffic through the MITM proxy or pass it through the tunnel un-intercepted.

**When to use:** Core routing decision for Phase 5.

**Implementation:**

```
# nftables.conf.j2 additions
set maxsavings_macs {
    type ether_addr
    flags timeout
    timeout 24h
}

# In prerouting chain (after bypass check, before tunnel mark):
# If device is in maxsavings set AND destination is not in bypass set,
# mark for proxy interception (fwmark 0x2 goes through tunnel to proxy)
# If device is NOT in maxsavings set, traffic still tunnels but the
# proxy passes it through without MITM (ConnectAccept in goproxy)
```

**Key insight:** The nftables set only controls which devices get their traffic tunneled. The proxy itself decides whether to MITM based on whether the device's source IP maps to a "Max Savings" MAC. The simplest approach: all tunnel traffic flows through the proxy regardless, but the proxy only performs MITM on connections from Max Savings devices. This avoids complex per-device routing and keeps the nftables rules simple.

**Revised approach (simpler):** Since all non-bypass traffic already goes through the tunnel (fwmark 0x2), and the proxy already has ConnectMitm vs ConnectAccept logic, the mode decision can be made at the proxy level. The Pi's dashboard daemon tells the proxy (via API or config reload) which MACs are in Max Savings mode. The proxy checks the source IP/MAC before deciding to MITM.

**Simplest approach:** The proxy runs on the remote server, so it doesn't see client MACs directly. Instead, the Pi must communicate mode state to the proxy. Two options:
1. **nftables routing:** Quick Connect devices get fwmark 0x2 (tunnel, proxy sees but does ConnectAccept for all). Max Savings devices get fwmark 0x3 (tunnel, proxy does ConnectMitm). The proxy checks the source port or some other marker.
2. **Simpler: All traffic through proxy, proxy always MITMs.** Quick Connect devices never install the CA cert, so MITM produces cert warnings -> users won't browse. But actually, the proxy generates certs signed by the CA, and Quick Connect devices haven't installed the CA, so browsers show errors.

**Actual simplest approach:** The proxy MITMs everything that isn't in the bypass list. Quick Connect devices haven't installed the CA cert, so their browsers see certificate warnings for HTTPS sites and refuse to load. This is terrible UX.

**Correct approach (from D-16):** Quick Connect traffic should NOT flow through the MITM proxy. It flows through the tunnel but the proxy does TCP passthrough (ConnectAccept) for Quick Connect devices. The Pi needs to signal per-device mode to the proxy. Options:
1. **X-Forwarded-For header:** Won't work for CONNECT tunnels.
2. **Source IP mapping:** The WireGuard tunnel preserves source IPs from the Pi's subnet (192.168.4.x). The Pi tells the proxy which IPs are Max Savings via a shared config file or API call. The proxy checks source IP against the Max Savings list in the HandleConnect callback.
3. **Separate ports:** Quick Connect traffic goes to proxy port A (passthrough only), Max Savings goes to port B (MITM). nftables on the Pi routes based on MAC set membership.

**Recommendation: Source IP mapping via REST API.** The dashboard daemon on the Pi maintains the authoritative mode-per-device map. It exposes a simple endpoint (or the proxy periodically fetches the list from the Pi via WireGuard). The proxy's HandleConnect checks the source IP against the current Max Savings IP list. This is the least invasive change to the existing proxy architecture.

### Pattern 4: Hardcoded Never-MITM Categories

**What:** Certain domain categories are hardcoded in the bypass list binary and cannot be removed by user configuration. The user-extensible bypass list is a separate YAML file that gets merged with the hardcoded list.

**When to use:** Always. Per D-14.

**Implementation:**

```go
// Hardcoded in Go source -- cannot be modified via config
var hardcodedBypassDomains = []string{
    // Banking & Financial
    "*.chase.com", "*.bankofamerica.com", "*.wellsfargo.com",
    "*.capitalone.com", "*.citi.com", "*.schwab.com",
    "*.fidelity.com", "*.vanguard.com", "*.usaa.com",
    // Authentication & Identity
    "*.apple.com", "accounts.google.com", "login.microsoftonline.com",
    "*.okta.com", "*.auth0.com", "*.duosecurity.com",
    // Payments
    "*.paypal.com", "*.venmo.com", "*.stripe.com", "*.square.com",
    // Government
    "*.gov", "*.mil",
    // Health
    "*.epic.com", "*.mychart.com",
    // Aviation (belt-and-suspenders with DNS bypass)
    "*.foreflight.com", "*.garmin.com", "*.faa.gov",
}

// Merge hardcoded + user YAML for final bypass set
func BuildBypassSet(userBypassPath string) (*BypassSet, error) {
    allDomains := append([]string{}, hardcodedBypassDomains...)
    if userDomains, err := LoadBypassDomains(userBypassPath); err == nil {
        allDomains = append(allDomains, userDomains...)
    }
    return NewBypassSet(allDomains), nil
}
```

### Anti-Patterns to Avoid

- **Sending root CA private key to the remote proxy:** D-08 explicitly forbids this. Use intermediate CA delegation instead.
- **Generating the same CA cert on every SkyGate device:** Compromises the entire fleet if one key leaks. Per-device generation on first boot (D-05).
- **Making .mobileconfig available before terms acceptance:** The cert download URL must only work after captive portal acceptance. Check MAC against `portal_accepted` table.
- **MITM on Quick Connect devices:** These devices haven't installed the CA cert. MITM would cause cert errors on every HTTPS site. The proxy must do TCP passthrough for Quick Connect devices.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| x509 certificate generation | Custom ASN.1 encoding | Go stdlib `crypto/x509.CreateCertificate` | Handles all DER/PEM encoding, extension marshaling, signing. Already proven in Phase 4 `certgen.go`. |
| iOS profile format | Custom XML builder | Go `text/template` with static .mobileconfig XML template | The format is stable XML plist. A template with base64 cert data injected is sufficient. |
| Certificate chain validation | Manual chain walking | Go stdlib `x509.Certificate.Verify` with `x509.CertPool` | Stdlib handles path validation, expiry checks, key usage constraints. |
| UUID generation | Custom random strings | Go stdlib `crypto/rand` + formatting | UUIDs in .mobileconfig need to be valid RFC 4122. |
| Base64 encoding of DER cert | Manual encoding | Go stdlib `encoding/base64.StdEncoding.EncodeToString` | Apple plist `<data>` tags expect standard base64. |

**Key insight:** Go's stdlib has everything needed for this phase. Zero new dependencies required.

## Common Pitfalls

### Pitfall 1: iOS Certificate Trust Requires TWO Manual Steps
**What goes wrong:** Developer installs the .mobileconfig profile and assumes the CA is trusted. It isn't -- iOS requires a separate step to enable full trust.
**Why it happens:** Since iOS 10.3, installed profiles add the cert to the device but do NOT enable trust by default. The user must go to Settings > General > About > Certificate Trust Settings and toggle the switch.
**How to avoid:** The cert install guide page MUST show all 3 steps with screenshots: (1) Download profile, (2) Install profile in Settings > VPN & Device Management, (3) Enable trust in Certificate Trust Settings. The "Max Savings" mode in the dashboard should indicate "Pending" until the device actually makes a successful HTTPS request through the proxy.
**Warning signs:** Users report "Max Savings mode is on but nothing is compressed" -- means they installed the profile but didn't enable trust.

### Pitfall 2: Android User-Installed CAs Not Trusted by Most Apps
**What goes wrong:** Android user installs the CA cert expecting all traffic to be compressed. Only browser traffic works -- native apps ignore user-installed CAs.
**Why it happens:** Since Android 7 (Nougat), apps do not trust user-installed certificates unless the app developer explicitly opts in via `networkSecurityConfig`. Since Android 14, even rooted devices cannot easily add system-level CAs. This means banking apps, social media apps, and most native apps will ignore the SkyGate CA entirely.
**How to avoid:** The Android cert install page MUST clearly state: "Max Savings mode optimizes browser traffic (Chrome, Firefox, Safari). Native apps will continue using DNS-only savings." This is honest and prevents support tickets. Browser traffic is still a significant portion of in-flight data usage.
**Warning signs:** Android users complaining "my Instagram/TikTok isn't compressed" -- correct behavior, document it upfront.

### Pitfall 3: .mobileconfig MIME Type Must Be Exact
**What goes wrong:** iOS downloads the .mobileconfig as a text file instead of triggering the profile installer.
**Why it happens:** The MIME type must be exactly `application/x-apple-aspen-config`. Any other MIME type (text/xml, application/xml, application/octet-stream) causes iOS to treat it as a regular download instead of a configuration profile.
**How to avoid:** Set the Content-Type header explicitly in the Go handler. Also set `Content-Disposition: attachment; filename="SkyGate.mobileconfig"`. Test with actual iOS device, not just Safari on macOS.
**Warning signs:** Users tap the download link but no "Profile Downloaded" banner appears.

### Pitfall 4: Intermediate CA MaxPathLen Must Be 0
**What goes wrong:** A misconfigured intermediate CA with `MaxPathLen > 0` could theoretically be used to create sub-CAs, expanding the attack surface.
**Why it happens:** Default x509 template doesn't constrain path length. Must explicitly set `MaxPathLen: 0` and `MaxPathLenZero: true`.
**How to avoid:** Set both fields in the intermediate CA template. Add a test that verifies `MaxPathLen == 0` on generated intermediate certs.
**Warning signs:** Security audit flags intermediate CA as capable of creating sub-CAs.

### Pitfall 5: Proxy Can't See Client MAC Addresses
**What goes wrong:** The proxy runs on the remote server. Traffic arrives through WireGuard tunnel. The proxy sees the WireGuard tunnel's IP, not the original client's MAC address.
**Why it happens:** WireGuard tunneling and NAT on the Pi strip the original MAC. The proxy only sees source IPs from the Pi's subnet (192.168.4.x).
**How to avoid:** The Pi's dashboard daemon maintains a MAC -> IP -> mode mapping. This mapping is communicated to the proxy via API or config. The proxy checks the source IP (which IS preserved through the tunnel, since the Pi NATs to its wlan0 subnet) against the Max Savings IP list.
**Warning signs:** Proxy MITMs all traffic or no traffic regardless of device mode selection.

### Pitfall 6: Race Condition Between Mode Switch and nftables Update
**What goes wrong:** User switches from Quick Connect to Max Savings in the dashboard. The dashboard API updates the DB, but the nftables set and proxy notification happen asynchronously. For a brief window, the user is in "Max Savings" according to the UI but traffic is still in passthrough mode.
**Why it happens:** Multiple systems need to be updated atomically: SQLite, nftables set, proxy mode list.
**How to avoid:** Accept eventual consistency. The mode switch is not safety-critical. Update DB first (source of truth), then nftables, then notify proxy. If any step fails, the UI shows the current actual state on next poll. Design the API to be idempotent.
**Warning signs:** Users switch modes but see no change for several seconds.

## Code Examples

### Example 1: Root CA Generation (Extend Existing certgen.go)

```go
// Source: Go stdlib crypto/x509 + existing SkyGate certgen.go pattern
func GenerateRootCA(certPath, keyPath, ssid string) (*x509.Certificate, crypto.PrivateKey, error) {
    privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, nil, fmt.Errorf("generating root CA key: %w", err)
    }

    serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

    template := &x509.Certificate{
        SerialNumber: serialNumber,
        Subject: pkix.Name{
            Organization: []string{"SkyGate"},
            CommonName:   fmt.Sprintf("SkyGate-%s CA", ssid),
        },
        NotBefore:             time.Now(),
        NotAfter:              time.Now().Add(3 * 365 * 24 * time.Hour), // 3 years per D-07
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        BasicConstraintsValid: true,
        IsCA:                  true,
    }

    certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
    if err != nil {
        return nil, nil, fmt.Errorf("creating root CA cert: %w", err)
    }

    // Write PEM files with restrictive permissions
    os.MkdirAll(filepath.Dir(certPath), 0755)
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
    os.WriteFile(certPath, certPEM, 0644)

    keyDER, _ := x509.MarshalECPrivateKey(privKey)
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
    os.WriteFile(keyPath, keyPEM, 0600)

    cert, _ := x509.ParseCertificate(certDER)
    return cert, privKey, nil
}
```

### Example 2: Intermediate CA Generation

```go
// Source: Go stdlib crypto/x509, pattern from github.com/Mattemagikern gist
func GenerateIntermediateCA(rootCert *x509.Certificate, rootKey crypto.PrivateKey) (*tls.Certificate, error) {
    privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

    template := &x509.Certificate{
        SerialNumber: serialNumber,
        Subject: pkix.Name{
            Organization: []string{"SkyGate"},
            CommonName:   "SkyGate Intermediate CA",
        },
        NotBefore:             time.Now(),
        NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        BasicConstraintsValid: true,
        IsCA:                  true,
        MaxPathLen:            0,    // Can ONLY sign leaf certs
        MaxPathLenZero:        true, // Explicitly set to zero
    }

    // Signed by ROOT CA, not self-signed
    certDER, _ := x509.CreateCertificate(rand.Reader, template, rootCert, &privKey.PublicKey, rootKey)

    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
    keyDER, _ := x509.MarshalECPrivateKey(privKey)
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

    tlsCert, _ := tls.X509KeyPair(certPEM, keyPEM)
    return &tlsCert, nil
}
```

### Example 3: nftables Per-Device Mode Set

```bash
# Source: nftables documentation, Red Hat nftables guide
# Add to nftables.conf.j2

# Per-device mode set -- MACs that opted into Max Savings
set maxsavings_macs {
    type ether_addr
    flags timeout
    timeout 24h
}
```

```go
// Source: existing nftables_linux.go pattern
// Add/remove MAC from maxsavings set
func SetDeviceMode(mac string, maxSavings bool) error {
    if maxSavings {
        element := fmt.Sprintf("{ %s timeout 24h }", mac)
        cmd := exec.Command("nft", "add", "element", "inet", "skygate", "maxsavings_macs", element)
        output, err := cmd.CombinedOutput()
        if err != nil {
            return fmt.Errorf("adding MAC %s to maxsavings: %v (%s)", mac, err, output)
        }
    } else {
        element := fmt.Sprintf("{ %s }", mac)
        cmd := exec.Command("nft", "delete", "element", "inet", "skygate", "maxsavings_macs", element)
        output, err := cmd.CombinedOutput()
        if err != nil {
            return fmt.Errorf("removing MAC %s from maxsavings: %v (%s)", mac, err, output)
        }
    }
    return nil
}
```

### Example 4: Proxy Mode Decision at CONNECT Time

```go
// Source: existing proxy.go HandleConnect pattern
// Extend to check source IP against Max Savings list
proxy.OnRequest().HandleConnectFunc(
    func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
        hostname := stripPort(host)

        // Always bypass cert-pinned domains
        if bypassSet.Contains(hostname) {
            return &goproxy.ConnectAction{Action: goproxy.ConnectAccept}, host
        }

        // Check if source device is in Max Savings mode
        sourceIP := extractSourceIP(ctx.Req)
        if !maxSavingsIPs.Contains(sourceIP) {
            // Quick Connect device: TCP passthrough, no MITM
            return &goproxy.ConnectAction{Action: goproxy.ConnectAccept}, host
        }

        // Max Savings device: MITM with intermediate CA
        return &goproxy.ConnectAction{
            Action:    goproxy.ConnectMitm,
            TLSConfig: goproxy.TLSConfigFromCA(intermediateCACert),
        }, host
    })
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single CA for root + signing | Root CA + Intermediate CA delegation | Industry standard, codified in CA/Browser Forum Baseline Requirements | Root key never exposed to operational systems. Intermediate can be rotated without redistributing root cert to all devices. |
| iOS auto-trusts installed profiles | iOS 10.3+: profile install + manual trust enable required | iOS 10.3 (2017) | Two-step process for cert trust. Must be documented in user guide. |
| Android trusts user-installed CAs for all apps | Android 7+: user CAs only trusted by apps that opt-in | Android 7 Nougat (2016) | Most native apps ignore user CAs. Browser traffic is the primary beneficiary of Max Savings on Android. |
| Android allows writing to system CA store | Android 14+: system CA store immutable, even for root | Android 14 (2023) | No workaround possible. User-installed CAs are the only option for non-MDM devices. |
| goproxy uses root CA directly | goproxy supports any CA via `TLSConfigFromCA(*tls.Certificate)` | Always | Can pass intermediate CA. Leaf certs chain back through intermediate to root. |

**Deprecated/outdated:**
- **compy's cert approach:** compy uses a single hardcoded CA for all instances. SkyGate generates per-device CAs. Do not follow compy's pattern.
- **Pre-iOS 10.3 cert trust:** Old guides show one-step profile install. Current iOS requires the separate Certificate Trust Settings step.

## Open Questions

1. **Proxy-side per-device mode awareness**
   - What we know: The proxy sees source IPs from the Pi's subnet (192.168.4.x) because the Pi NATs outbound traffic. The proxy needs to know which IPs are in Max Savings mode.
   - What's unclear: Best transport for mode state updates -- REST API poll from proxy to Pi? Push from Pi to proxy? Shared config file on Docker volume?
   - Recommendation: REST API endpoint on the Pi's dashboard daemon (`GET /api/mode/ips`) that returns current Max Savings IPs. Proxy polls every 10 seconds. Simple, stateless, uses existing HTTP infrastructure. Alternatively, since the proxy and WireGuard share a Docker network, the Pi can push updates to a proxy admin API endpoint.

2. **Intermediate CA provisioning timing**
   - What we know: The intermediate CA cert+key must be on the remote proxy before any Max Savings traffic flows.
   - What's unclear: When exactly does the Pi generate and transfer the intermediate? First boot? Each tunnel establishment? On demand?
   - Recommendation: Generate intermediate CA during first-boot CA generation. Transfer to remote server during initial WireGuard setup (SCP over tunnel, or bake into Docker volume during initial provisioning). Re-generate only when intermediate expires (1 year).

3. **Captive portal CNA compatibility with mode selection**
   - What we know: iOS CNA (Captive Network Assistant) has strict behavior -- it auto-closes, limited JS support, no downloads. Phase 2 captive portal was designed to work within CNA constraints.
   - What's unclear: Can mode selection happen within the CNA, or must the user be redirected to a full browser for cert download?
   - Recommendation: Keep terms acceptance in CNA (existing flow). After acceptance, redirect to full Safari for mode selection and cert download. The CNA auto-closes after a successful captive portal response anyway. Mode selection is a separate page at `/mode-select` accessible from the dashboard.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (stdlib, no config file needed) |
| Quick run command | `go test ./cmd/dashboard-daemon/ ./cmd/proxy-server/ -v -short -run "Test(Mode\|Cert\|Bypass)"` |
| Full suite command | `go test ./... -v -short` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CERT-01 | Mode selection API (POST /api/mode, GET /api/mode) returns and persists device mode | unit | `go test ./cmd/dashboard-daemon/ -v -short -run TestMode` | Wave 0 |
| CERT-01 | nftables maxsavings_macs set operations (add/remove MAC) | unit | `go test ./cmd/dashboard-daemon/ -v -short -run TestSetDeviceMode` | Wave 0 |
| CERT-01 | Mode selection page renders with two options | unit | `go test ./cmd/dashboard-daemon/ -v -short -run TestModeSelectPage` | Wave 0 |
| CERT-02 | Root CA generation with SSID in CN, 3-year validity, ECDSA P-256 | unit | `go test ./cmd/proxy-server/ -v -short -run TestGenerateRootCA` | Wave 0 (extend certgen_test.go) |
| CERT-02 | Intermediate CA signed by root, MaxPathLen=0, 1-year validity | unit | `go test ./cmd/proxy-server/ -v -short -run TestGenerateIntermediateCA` | Wave 0 |
| CERT-02 | .mobileconfig generation with correct PayloadType and base64 cert data | unit | `go test ./cmd/dashboard-daemon/ -v -short -run TestMobileConfig` | Wave 0 |
| CERT-02 | .crt DER download with correct Content-Type | unit | `go test ./cmd/dashboard-daemon/ -v -short -run TestCertDownloadDER` | Wave 0 |
| CERT-02 | TLSConfigFromCA works with intermediate cert for leaf signing | unit | `go test ./cmd/proxy-server/ -v -short -run TestIntermediateCALeafSigning` | Wave 0 |
| CERT-03 | Hardcoded bypass domains cannot be removed by user config | unit | `go test ./cmd/proxy-server/ -v -short -run TestHardcodedBypass` | Wave 0 |
| CERT-03 | User bypass domains merged with hardcoded list | unit | `go test ./cmd/proxy-server/ -v -short -run TestMergedBypassSet` | Wave 0 |
| CERT-03 | Bypass domains API (GET/POST /api/bypass) for dashboard management | unit | `go test ./cmd/dashboard-daemon/ -v -short -run TestBypassAPI` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/dashboard-daemon/ ./cmd/proxy-server/ -v -short`
- **Per wave merge:** `go test ./... -v -short`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/dashboard-daemon/mode_test.go` -- covers CERT-01 (mode selection, nftables set ops)
- [ ] `cmd/dashboard-daemon/certdownload_test.go` -- covers CERT-02 (.mobileconfig, .crt download)
- [ ] `cmd/proxy-server/certgen_test.go` -- EXTEND with intermediate CA and chain validation tests (CERT-02)
- [ ] `cmd/proxy-server/proxy_test.go` -- EXTEND with per-device mode bypass test (CERT-03)

*(Existing test infrastructure covers all framework needs -- Go stdlib testing with `t.TempDir()` for file operations, `httptest` for HTTP handlers, in-memory SQLite for DB operations)*

## Sources

### Primary (HIGH confidence)
- [Go crypto/x509 package docs](https://pkg.go.dev/crypto/x509) -- CreateCertificate, Certificate struct, intermediate CA fields (MaxPathLen, IsCA)
- [elazarl/goproxy](https://pkg.go.dev/github.com/elazarl/goproxy) -- TLSConfigFromCA function, CertStorage interface, ConnectAction types
- [elazarl/goproxy custom CA example](https://github.com/elazarl/goproxy/blob/master/examples/customca/README.md) -- Custom CA cert loading, PEM format requirements
- [elazarl/goproxy https.go source](https://github.com/elazarl/goproxy/blob/master/https.go) -- TLSConfigFromCA implementation, leaf cert signing flow
- Existing codebase: `cmd/proxy-server/certgen.go`, `cmd/proxy-server/proxy.go`, `cmd/dashboard-daemon/captive.go`, `cmd/dashboard-daemon/nftables_linux.go`

### Secondary (MEDIUM confidence)
- [Apple CertificateRoot documentation](https://developer.apple.com/documentation/devicemanagement/certificateroot) -- PayloadType: com.apple.security.root for .mobileconfig profiles
- [Apple Configuration Profile Reference PDF](https://developer.apple.com/business/documentation/Configuration-Profile-Reference.pdf) -- Full profile XML structure
- [Go x509 certificate chain creation gist](https://gist.github.com/Mattemagikern/328cdd650be33bc33105e26db88e487d) -- Root CA -> Intermediate CA -> Leaf pattern in Go
- [Shane Utt's CA signing blog](https://shaneutt.com/blog/golang-ca-and-signed-cert-go/) -- Go CA + leaf cert signing walkthrough
- [nftables ether_addr sets](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/8/html/securing_networks/getting-started-with-nftables_securing-networks) -- MAC address set type documentation
- [Android 14 CA certificate restrictions](https://httptoolkit.com/blog/android-14-breaks-system-certificate-installation/) -- System CA store immutable, user CAs app-opt-in only
- [Android 11 CA trust changes](https://httptoolkit.com/blog/android-11-trust-ca-certificates/) -- User-installed CAs not trusted by apps since Android 7+
- [Steven Jordan - mobileconfig cert guide](https://www.stevenjordan.net/2016/11/add-certs-to-mobile-config-xml.html) -- Base64 DER cert embedding in plist

### Tertiary (LOW confidence)
- [WireGuard/wireguard-apple mobileconfig docs](https://github.com/WireGuard/wireguard-apple/blob/master/MOBILECONFIG.md) -- .mobileconfig plist structure reference (for VPN payloads, but same outer structure applies to cert payloads)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all Go stdlib, no new dependencies, existing patterns from Phase 4
- Architecture: HIGH -- intermediate CA delegation is standard PKI practice, well-documented in Go
- Pitfalls: HIGH -- iOS cert trust flow, Android CA limitations well-documented across multiple sources
- Mode routing: MEDIUM -- proxy-side per-device mode awareness needs implementation validation (open question 1)

**Research date:** 2026-03-23
**Valid until:** 2026-04-23 (stable domain -- certificate standards and Go stdlib don't change rapidly)
