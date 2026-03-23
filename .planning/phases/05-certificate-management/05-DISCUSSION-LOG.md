# Phase 5: Certificate Management - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md -- this log preserves the alternatives considered.

**Date:** 2026-03-23
**Phase:** 05-certificate-management
**Mode:** Auto (all gray areas resolved with recommended defaults)
**Areas discussed:** Mode selection UX, CA certificate lifecycle, per-platform install flows, cert-pinning bypass, proxy-side cert handling

---

## Mode Selection UX

### Default Mode Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Quick Connect as default (Recommended) | Pre-selected option, zero friction. Passengers who just tap Continue get DNS-only | ✓ |
| Max Savings as default | Encourage max savings by default, require opt-out | |
| No default (force choice) | Require explicit selection of either mode before proceeding | |

**Auto-selected:** Quick Connect as default
**Rationale:** Matches PROJECT.md's "just works like an Eero" UX bar. The two-layer TLS strategy (ARCHITECTURE.md Pattern 3) explicitly states Layer 1 alone provides massive value with zero friction. Forcing cert install as default would create friction, trust concerns, and app breakage for passengers who don't understand the implications. PITFALLS.md Security Mistake 4 warns against MITM CA cert without clear consent flow.

### Mode Switching Mid-Flight

| Option | Description | Selected |
|--------|-------------|----------|
| Dashboard toggle (Recommended) | Switch modes from dashboard settings page, no WiFi reconnect needed | ✓ |
| Reconnect required | Must disconnect and reconnect to WiFi to change mode | |
| No switching | Mode locked after initial selection | |

**Auto-selected:** Dashboard toggle
**Rationale:** Phase 2 dashboard (D-18, D-19) already establishes a settings page pattern. Adding a mode toggle is natural. Requiring reconnect creates friction -- a passenger who discovers their banking app is broken needs an immediate fix, not a WiFi dance (PITFALLS.md Pitfall 7 recovery).

### Per-Device vs Per-Network Mode

| Option | Description | Selected |
|--------|-------------|----------|
| Per-device (Recommended) | Each device selects its own mode, tracked by MAC | ✓ |
| Per-network | Single mode for all devices, set by pilot | |
| Pilot override | Per-device default with pilot ability to force all devices to one mode | |

**Auto-selected:** Per-device
**Rationale:** Different passengers have different risk tolerance. A tech-savvy passenger may want Max Savings while a pilot's spouse just wants Quick Connect. MAC-based tracking is already established (Phase 2 D-10 captive portal MAC whitelist). nftables sets support per-MAC metadata.

---

## CA Certificate Lifecycle

### CA Key Location

| Option | Description | Selected |
|--------|-------------|----------|
| Per-device generation on first boot (Recommended) | Each appliance generates unique CA keypair on first boot, stored on data partition | ✓ |
| Shared CA in SD card image | Single CA baked into the image for all SkyGate devices | |
| Cloud-provisioned CA | CA generated and signed by a SkyGate root CA hosted remotely | |

**Auto-selected:** Per-device generation on first boot
**Rationale:** PITFALLS.md Security Mistake 1 explicitly warns: "Anyone who downloads the image has the CA key. They can MITM any device that installed the SkyGate cert. Complete compromise of the MITM security model." Per-device generation eliminates this attack vector entirely. Cloud-provisioned adds dependency on external infrastructure that doesn't exist in v1.

### CA Validity Period

| Option | Description | Selected |
|--------|-------------|----------|
| 3 years (Recommended) | Reasonable balance between security and maintenance burden | ✓ |
| 1 year | More secure but requires cert rotation infrastructure | |
| 10 years | Set and forget but poor security practice | |

**Auto-selected:** 3 years
**Rationale:** SkyGate is a personal appliance, not a public CA. 1-year validity requires rotation automation that adds v1 complexity. 10 years is excessive and raises questions about security posture. 3 years matches typical enterprise internal CA practice and covers the likely hardware lifecycle of a Pi-based appliance.

### CA Key Storage

| Option | Description | Selected |
|--------|-------------|----------|
| /data/skygate/ca/ with 0600 perms (Recommended) | Writable data partition, root-only access | ✓ |
| Hardware security module (HSM) | TPM or USB HSM for key storage | |
| Encrypted file with passphrase | Encrypted CA key requiring unlock on boot | |

**Auto-selected:** /data/skygate/ca/ with 0600 perms
**Rationale:** Pi 5 has no TPM. USB HSM adds hardware cost and complexity. Encrypted passphrase requires interactive unlock on every boot (unacceptable for zero-config UX). File-based with restrictive permissions is the standard approach for embedded appliances. The data partition at /data/skygate is already established (Phase 1 D-76, Phase 2 D-13).

---

## Per-Platform Cert Install Flows

### iOS Install Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| .mobileconfig profile (Recommended) | Configuration profile containing CA cert, downloaded via Safari | ✓ |
| Direct .crt download | Raw certificate file, requires manual trust steps | |
| QR code to cert URL | Scan QR code with camera, opens cert download in Safari | |

**Auto-selected:** .mobileconfig profile
**Rationale:** .mobileconfig is the standard iOS mechanism for distributing certificates. It triggers the native profile installer workflow that users are somewhat familiar with from corporate MDM. Direct .crt requires more manual steps. QR code is a nice addition (noted in specifics) but the primary mechanism should be the profile download link in the captive portal.

### Android Install Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Direct .crt download (Recommended) | DER-format certificate file, Android prompts for credential storage install | ✓ |
| .p12 bundle | PKCS#12 bundle with cert, requires password | |
| ADB install instructions | Developer-focused, command-line cert install | |

**Auto-selected:** Direct .crt download
**Rationale:** Simplest path for Android. .p12 adds unnecessary password complexity. ADB requires developer mode, which is out of scope for non-technical passengers. Note the Android 11+ limitation: user-installed CAs are only trusted by the system browser and apps that explicitly opt-in to the user trust store. This is honestly communicated in the portal UI.

### Post-Flight Cert Removal

| Option | Description | Selected |
|--------|-------------|----------|
| Portal instructions + QR card (Recommended) | Dashboard shows removal steps; physical QR code on printed card links to removal guide | ✓ |
| Automatic cert expiry (short validity) | Cert expires after 24 hours, auto-untrusted | |
| Push notification reminder | Send notification to remove cert | |

**Auto-selected:** Portal instructions + QR card
**Rationale:** Short-validity certs require re-install every flight (terrible UX). Push notifications require a native app (out of scope). Clear removal instructions are the honest, practical approach. QR code on a physical card gives passengers a reference they can use after disconnecting from SkyGate WiFi.

---

## Certificate Pinning Bypass

### Bypass List Management

| Option | Description | Selected |
|--------|-------------|----------|
| YAML config + dashboard UI (Recommended) | YAML file for defaults, dashboard settings for pilot additions | ✓ |
| YAML config only | File-based management, SSH or Ansible for changes | |
| Dashboard UI only | All bypass management via web UI | |

**Auto-selected:** YAML config + dashboard UI
**Rationale:** Follows established Phase 1 pattern (bypass-domains.yaml for aviation apps). YAML provides sane defaults and version-controllable config. Dashboard UI (Phase 2 settings page integration) makes it accessible to non-technical pilots. Both together is the same pattern used for aviation bypass -- config file for defaults, web UI for runtime changes.

### Hardcoded Never-MITM Categories

| Option | Description | Selected |
|--------|-------------|----------|
| Banking + Auth + Gov + Health + Payments (Recommended) | Comprehensive hardcoded list of sensitive categories that can never be removed | ✓ |
| Banking + Auth only | Minimal hardcoded list, everything else user-configurable | |
| No hardcoded categories | Everything configurable, pilot has full control | |

**Auto-selected:** Banking + Auth + Gov + Health + Payments
**Rationale:** PITFALLS.md Pitfall 7 states: "Never, ever MITM .gov domains, banking domains, or health domains. Hardcode these as bypass-always." These categories represent services where MITM interception could cause financial loss, identity theft, or health data exposure. Pilots should never accidentally remove these from bypass. The comprehensive list is the safe default.

### Bypass Implementation Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| nftables set + proxy passthrough (Recommended) | Bypass domains added to nftables set; proxy checks set and passes through without interception | ✓ |
| DNS-based routing (like aviation bypass) | Bypass domains route around the tunnel entirely | |
| Proxy-only bypass (no nftables) | Proxy decides internally which domains to intercept | |

**Auto-selected:** nftables set + proxy passthrough
**Rationale:** Aviation bypass (Phase 1) routes traffic around the WireGuard tunnel entirely -- appropriate for ForeFlight etc. that need minimum latency. Cert-pinning bypass is different: the traffic should still route through WireGuard (for bandwidth accounting and monitoring) but the proxy should NOT intercept the TLS connection. This means the bypass list informs the proxy to use TCP passthrough (CONNECT tunnel) rather than MITM for matching domains. The nftables set provides a fast lookup path on the proxy side.

---

## Proxy-Side Cert Handling

### Remote Proxy Signing Authority

| Option | Description | Selected |
|--------|-------------|----------|
| Intermediate CA delegated to proxy (Recommended) | Pi generates an intermediate CA cert, provisions it to the remote proxy. Proxy signs leaf certs with intermediate | ✓ |
| Root CA key on proxy | Send root CA private key to remote server | |
| Pi signs all leaf certs on-demand | Pi generates leaf certs per-domain, sends through tunnel to proxy | |

**Auto-selected:** Intermediate CA delegated to proxy
**Rationale:** Sending the root CA key to the remote server violates the security principle that the CA key never leaves the device. On-demand signing from the Pi adds latency to every new HTTPS connection (Pi must generate cert, send through tunnel, proxy waits). Intermediate CA delegation is the standard PKI approach: Pi generates and signs an intermediate cert, provisions it to the remote proxy once. Proxy uses intermediate to sign leaf certs locally with zero latency.

### Leaf Cert Caching

| Option | Description | Selected |
|--------|-------------|----------|
| Domain-keyed cache with 24h TTL (Recommended) | Cache generated leaf certs by domain, expire after 24 hours or original cert validity, whichever is shorter | ✓ |
| No caching | Generate fresh cert for every connection | |
| Persistent cache (survive proxy restart) | Write cert cache to disk | |

**Auto-selected:** Domain-keyed cache with 24h TTL
**Rationale:** Cert generation involves RSA/ECDSA key generation which is CPU-intensive. Caching eliminates repeated generation for frequently visited domains. 24h TTL balances cache freshness against generation overhead. No caching wastes CPU. Persistent disk cache adds complexity for minimal benefit -- most flights are under 6 hours, and in-memory cache is rebuilt quickly.

---

## Claude's Discretion

- CA certificate parameters (key size, signature algorithm, X.509 extensions, validity period details)
- .mobileconfig XML profile structure and optional signing
- Intermediate CA delegation mechanism (provisioning flow from Pi to remote proxy)
- nftables set structure for per-device mode tracking
- Go crypto/x509 stdlib usage for cert generation (vs external libraries)
- Cert cache implementation on remote proxy
- Dashboard UI layout for mode switching and cert download pages
- Android cert trust scope messaging
- Error handling for expired/revoked intermediate certs
- Cert rotation strategy

## Deferred Ideas

- Automatic cert rotation before CA expiry
- OCSP responder
- Per-passenger individual certs
- Enterprise MDM cert deployment
- CT logging
- Automatic cert-pinning detection via TLS handshake failure analysis
