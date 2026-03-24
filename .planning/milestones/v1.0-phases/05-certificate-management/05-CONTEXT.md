# Phase 5: Certificate Management - Context

**Gathered:** 2026-03-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Passengers choose their savings level -- "Quick Connect" for zero-friction DNS blocking or "Max Savings" with CA cert install for full proxy compression. The captive portal presents both options with guided per-platform install flows. Cert-pinned apps (banking, auth, health, payments) never break because a maintained bypass list routes them around the MITM proxy. Per-device CA certificates are generated uniquely on each SkyGate appliance (never shared across devices/appliances). No new proxy logic, no new DNS rules, no new QoS -- just the certificate lifecycle and the user-facing mode selection that bridges the existing captive portal (Phase 2) and content compression proxy (Phase 4).

</domain>

<decisions>
## Implementation Decisions

### Two-Tier Mode Selection UX
- **D-01:** Captive portal presents two clearly labeled options after terms acceptance: "Quick Connect" (DNS blocking only, zero setup, works immediately) and "Max Savings" (install CA cert for proxy compression, requires 2-3 extra taps per platform)
- **D-02:** "Quick Connect" is the default and pre-selected option -- passengers who just tap "Continue" get DNS-only mode. "Max Savings" requires deliberate opt-in
- **D-03:** Mode selection is per-device -- different passengers on the same flight can choose different modes. Tracked by MAC address in nftables sets
- **D-04:** Passengers can switch modes mid-flight from the dashboard settings without reconnecting to WiFi. Switching from "Max Savings" to "Quick Connect" takes effect immediately (proxy bypass). Switching from "Quick Connect" to "Max Savings" requires cert install flow

### CA Certificate Generation
- **D-05:** Each SkyGate appliance generates its own unique CA keypair on first boot -- no shared CA key across devices. This prevents a compromised image from enabling MITM across all SkyGate installations
- **D-06:** CA keypair stored at `/data/skygate/ca/` (writable data partition, survives read-only root) with restrictive permissions (0600, root only). Private key never leaves the device
- **D-07:** CA certificate is a self-signed root CA with reasonable validity (3 years). Common Name includes the appliance's SSID for user recognition (e.g., "SkyGate-TailNumber CA")
- **D-08:** The Go proxy on the remote server does NOT need the CA key -- it generates ephemeral leaf certs signed by the Pi's CA for each intercepted domain. The CA key stays on the Pi; the remote proxy receives signing authority via a delegated intermediate cert or the Pi signs certs on-demand and forwards them through the WireGuard tunnel

### Per-Platform Cert Install Flow
- **D-09:** iOS: Generate `.mobileconfig` profile containing the CA certificate. Passenger taps download link in captive portal, iOS prompts to install profile, then passenger must go to Settings > General > Profile to confirm, then Settings > General > About > Certificate Trust Settings to enable full trust. Portal shows step-by-step screenshots for each iOS step
- **D-10:** Android: Direct `.crt` (DER format) download. Android prompts to install as "CA certificate" in credential storage. Portal shows step-by-step guide with screenshots. Note: Android 11+ user-installed CAs only trusted by apps that opt-in -- browser traffic works, but most native apps ignore user CAs regardless
- **D-11:** macOS/Windows laptops: Direct `.crt` download with platform-specific instructions. macOS: Keychain Access, mark as trusted. Windows: certmgr.msc import to Trusted Root. Lower priority than mobile (most passengers are on phones/tablets)
- **D-12:** Post-flight cert removal instructions displayed when passenger disconnects or via QR code on a printed card. Essential for security hygiene -- passengers should remove the CA cert after landing

### Certificate Pinning Bypass List
- **D-13:** Bypass list is a YAML config file (consistent with Phase 1 pattern) containing domains and domain patterns that are NEVER intercepted by the MITM proxy, regardless of whether the device has "Max Savings" enabled
- **D-14:** Hardcoded "never-MITM" categories that cannot be removed by user config: banking/financial (*.chase.com, *.wellsfargo.com, *.bankofamerica.com, etc.), authentication/2FA (accounts.google.com, login.microsoftonline.com, appleid.apple.com, etc.), government (*.gov), health (*.epic.com, mychart.*, etc.), payment processors (*.paypal.com, *.venmo.com, *.stripe.com, etc.)
- **D-15:** User-extensible bypass list -- pilots can add domains via dashboard settings (Phase 2 dashboard integration) or YAML config edit. When a passenger reports "my app doesn't work in Max Savings mode," the pilot adds the domain to bypass
- **D-16:** Bypass implementation: domains in the bypass list are added to an nftables set (similar to aviation bypass in Phase 1) that routes traffic directly through the tunnel but skips the MITM proxy. Traffic still goes through WireGuard (for routing) but the proxy passes it through without interception

### Proxy-Side Cert Handling
- **D-17:** The remote Go proxy (Phase 4) generates ephemeral TLS certificates for each intercepted HTTPS domain, signed by a signing certificate that chains to the Pi's root CA. The signing cert (intermediate CA) is provisioned to the remote server during WireGuard tunnel setup
- **D-18:** Intermediate CA cert generated on the Pi, pushed to remote server via WireGuard control channel (or pre-provisioned during initial server setup). This avoids sending the root CA private key to the remote server while still enabling on-the-fly cert generation
- **D-19:** Cert cache on the remote proxy -- generated leaf certs are cached by domain (with TTL matching the original cert's validity or 24 hours, whichever is shorter) to avoid repeated generation overhead

### Claude's Discretion
- Exact CA certificate parameters (key size, signature algorithm, extensions, validity period)
- `.mobileconfig` XML profile structure and signing (unsigned vs self-signed profile)
- Intermediate CA delegation mechanism (how the Pi provisions signing authority to the remote proxy)
- nftables set structure for per-device mode tracking ("Quick Connect" vs "Max Savings")
- Cert generation library selection for Go (crypto/x509 stdlib vs external library)
- Cert cache implementation details on the remote proxy
- Dashboard UI layout for mode switching and cert download
- Android cert trust scope limitations and user-facing messaging
- Error handling for expired/revoked intermediate certs
- Cert rotation strategy (when/how to regenerate the CA before expiry)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` -- Full project vision, constraints, two-layer TLS strategy decision
- `.planning/REQUIREMENTS.md` -- v1 requirements: CERT-01, CERT-02, CERT-03
- `CLAUDE.md` -- Technology stack, Go proxy on goproxy, elazarl/goproxy for MITM

### Prior Phase Foundations
- `.planning/phases/01-pi-network-foundation/01-CONTEXT.md` -- Network foundation: nftables sets, YAML config format, Go daemon patterns, systemd services, data partition at /data/skygate
- `.planning/phases/02-usage-dashboard/02-CONTEXT.md` -- Captive portal flow (D-06 through D-10), dashboard layout, Caddy reverse proxy, HTMX+SSE patterns

### Research
- `.planning/research/ARCHITECTURE.md` -- Pattern 3 (Two-Layer TLS Strategy), Pattern 2 (Captive Portal with MAC-Based Auth), data flow diagrams, CA cert integration points
- `.planning/research/PITFALLS.md` -- Pitfall 7 (MITM breaks cert-pinned apps), Security Mistake 1 (shared CA key), Security Mistake 4 (MITM CA cert without consent)
- `.planning/research/FEATURES.md` -- Two-tier TLS strategy as differentiator, cert-pinning bypass as table stakes

### Existing Codebase (by Phase 5 execution time)
- `cmd/bypass-daemon/` -- Go daemon pattern: YAML config, platform build tags, nftables integration
- `pi/config/bypass-domains.yaml` -- YAML config format for domain lists (extend for cert-pinning bypass)
- `pi/ansible/roles/networking/templates/nftables.conf.j2` -- nftables rules to extend with per-device mode sets
- Phase 2 captive portal Go code -- extend with mode selection and cert download endpoints
- Phase 4 proxy Go code -- extend with MITM cert generation using intermediate CA

### Design Document
- `~/.gstack/projects/skygate/ryanstern-unknown-design-20260322-161803.md` -- Full approved design doc

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets (available by Phase 5)
- **Go module** (`github.com/quartermint/skygate`): YAML config loader, platform build tags, nftables integration
- **Ansible roles structure**: new `certificate` role follows existing pattern
- **nftables template**: extend with per-device mode set (mac_addr type with "quickconnect" vs "maxsavings" metadata)
- **Captive portal (Phase 2)**: extend with mode selection page and cert download handlers
- **Go proxy (Phase 4)**: extend with MITM TLS interception using intermediate CA from Pi
- **systemd service templates**: follow existing pattern for any cert-related daemons
- **Makefile**: extend with cert generation and proxy MITM build targets

### Established Patterns
- Go daemons with platform-specific build tags (linux vs stub for macOS dev)
- YAML for config files (bypass-domains.yaml pattern extends to cert-bypass-domains.yaml)
- Cross-compile: GOOS=linux GOARCH=arm64 CGO_ENABLED=0
- Data persistence in /data/skygate (CA keys stored here)
- nftables sets for domain-based routing decisions (extend for per-device mode)
- Captive portal MAC-based authentication release (extend with mode tracking)

### Integration Points
- Captive portal (Phase 2) -> mode selection UI -> cert download endpoint
- nftables per-device mode set -> proxy routing decision (MITM vs passthrough)
- Pi CA keypair -> intermediate CA -> remote proxy cert generation
- WireGuard tunnel (Phase 3) -> intermediate CA provisioning channel
- Dashboard (Phase 2) -> mode switch UI -> nftables set update
- Cert-pinning bypass YAML -> nftables set -> proxy passthrough

</code_context>

<specifics>
## Specific Ideas

- The two-tier mode selection is the key UX innovation -- it must be dead simple. Two big buttons, clear labels, no jargon
- iOS .mobileconfig is the smoothest cert install path on any platform -- leverage it fully with clear step-by-step screenshots
- Android's user CA trust limitations (Android 11+) mean "Max Savings" on Android primarily benefits browser traffic, not native apps. Be honest about this in the UI messaging
- The cert-pinning bypass list will never be complete. Ship with a generous initial list and make it trivially easy for pilots to add domains. Log proxy errors prominently so "this domain failed" is immediately visible
- CA key generation on first boot ties into Phase 1's first-boot setup flow (serial console TTY input per D-78 in Phase 1). CA generation can be automatic (no user input needed)
- Post-flight cert removal is a trust/reputation issue. Make removal instructions highly visible -- QR code on printed card, dashboard reminder when device disconnects

</specifics>

<deferred>
## Deferred Ideas

- Automatic cert rotation before CA expiry (3-year validity means this is a v2 concern)
- OCSP responder for real-time cert validation (overkill for a single-appliance setup)
- Per-passenger cert generation (currently per-appliance CA; per-passenger would enable revocation but adds complexity)
- Enterprise cert deployment via MDM for fleet operators (v2, multi-tenant hosted service)
- Cert transparency logging (CT logs are for public CAs, not private MITM CAs)
- Automatic detection of cert-pinned domains via TLS handshake failure patterns (would auto-populate bypass list)

</deferred>

---

*Phase: 05-certificate-management*
*Context gathered: 2026-03-23*
