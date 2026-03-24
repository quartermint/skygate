# Phase 4: Content Compression Proxy - Research

**Researched:** 2026-03-23
**Domain:** Go MITM proxy with image transcoding, JS/CSS minification, Docker deployment
**Confidence:** HIGH

## Summary

Phase 4 builds a remote Go-based MITM proxy server that compresses web content (images, JS, CSS, HTML) before it traverses the expensive Starlink satellite link. This is entirely remote server work -- no Pi-side changes beyond what Phase 3 established. The proxy is built on elazarl/goproxy (v1.8.2, actively maintained, used by Stripe/Google/Grafana), with image transcoding via kolesa-team/go-webp (CGo bindings to libwebp), text minification via tdewolff/minify (v2.24.10, pure Go), and image resizing via disintegration/imaging (v1.6.2, pure Go). The entire stack deploys as Docker containers alongside the existing WireGuard server from Phase 3.

The key architectural insight is that goproxy supports conditional MITM: `ConnectAccept` passes traffic through transparently (for SNI bypass domains), while `ConnectMitm` intercepts HTTPS for compression. The response handler pipeline is clean -- `OnResponse().DoFunc()` receives the full `*http.Response`, routes by Content-Type to the appropriate handler (image transcoder, text minifier, or passthrough), transforms the body, updates Content-Length, and returns. The 500ms timeout on image transcoding is implemented via `context.WithTimeout` on the transcoding goroutine, falling back to the original image if exceeded.

**Primary recommendation:** Build a single Go binary (`cmd/proxy-server/`) that embeds the goproxy MITM proxy with response handlers for image transcoding and text minification. Deploy in Docker with CGo enabled (for libwebp). Use `network_mode: service:wireguard` to share the WireGuard container's network stack, eliminating separate networking configuration. Use modernc.org/sqlite (already in go.mod, pure Go) for compression logging to stay consistent with the dashboard daemon pattern.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** JPEG quality reduction to q30 + resize to max 800px width as the default aggressive preset
- **D-02:** PNG and JPEG transcoded to WebP using kolesa-team/go-webp (CGo bindings to libwebp)
- **D-03:** Inline compression with 500ms timeout per image -- pass original through if timeout exceeded
- **D-04:** GIF and animated images passed through unmodified in v1
- **D-05:** SVG files minified via tdewolff/minify
- **D-06:** Built on elazarl/goproxy as the MITM proxy framework
- **D-07:** Response handler chain: check Content-Type, route to handler, update Content-Length, log stats
- **D-08:** Content-Type routing: image/* to transcoder, text/html+css+js to minifier, else passthrough
- **D-09:** Certificate-pinned domains bypass MITM via configurable SNI bypass list (YAML)
- **D-10:** Request/response logging to SQLite: timestamp, domain, original_bytes, compressed_bytes, content_type, device_id
- **D-11:** Single CA certificate generated at first server startup, persisted to Docker volume
- **D-12:** CA cert file exposed via HTTP endpoint for Pi captive portal download
- **D-13:** On-the-fly leaf certificate generation per domain using goproxy built-in MITM support, LRU cache

### Claude's Discretion
- goproxy MITM configuration details (cert pool size, TLS version constraints)
- Exact SQLite schema for compression logs (tables, indexes, retention/pruning policy)
- Image transcoding pipeline internals (buffer pooling, goroutine concurrency limits)
- Minification configuration flags for tdewolff/minify
- Docker healthcheck implementation details
- Go module structure within server/ directory (flat package vs nested packages)
- Error handling strategy for malformed content
- WireGuard server container choice (linuxserver/wireguard vs wg-easy vs raw wireguard-tools)
- Content-Length recalculation approach (buffered vs chunked transfer encoding)

### Deferred Ideas (OUT OF SCOPE)
- Video content blocking/replacement with placeholder
- Social media text-only mode
- Content rules hot-reload system
- Proxy horizontal scaling for hosted multi-tenant service
- Per-peer traffic accounting for multi-tenant billing
- AVIF transcoding as alternative to WebP
- Animated GIF to animated WebP transcoding
- HTML rewriting (strip video embeds, remove tracking scripts from DOM)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PROXY-01 | Go-based MITM proxy on remote server (built on goproxy) transcodes images (JPEG quality reduction, PNG/JPEG to WebP) and minifies JS/CSS | goproxy v1.8.2 OnResponse handlers + kolesa-team/go-webp for WebP transcoding + tdewolff/minify v2.24.10 for text minification. Full API patterns documented below. |
| PROXY-02 | Remote proxy server deployable via one-command Docker Compose with WireGuard server endpoint | Extend existing server/docker-compose.yml with proxy service using `network_mode: service:wireguard`. Multi-stage Dockerfile with CGo for libwebp. |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| elazarl/goproxy | v1.8.2 (Feb 2026) | MITM proxy framework | 6.2k stars, 10+ years mature, used by Stripe/Google/Grafana/Kubernetes. Handles CONNECT tunneling, TLS interception, request/response hooks. ConnectAccept for passthrough, ConnectMitm for interception. |
| kolesa-team/go-webp | v1.0.5 (Mar 2025) | WebP encoding (lossy) | CGo bindings to Google libwebp. Supports lossy encoding with configurable quality (q30 for SkyGate). Pure-Go alternatives (nativewebp) only support lossless -- not viable for aggressive compression. |
| tdewolff/minify | v2.24.10 (Feb 2026) | HTML/CSS/JS minification | 3.6k stars, actively maintained, pure Go (no CGo). 20-70 MB/s throughput. Streaming io.Reader/io.Writer API. 10-65% compression ratios on text content. |
| disintegration/imaging | v1.6.2 | Image resizing | Pure Go, 5.1k stars. Resize images to max 800px width before WebP encoding. Lanczos resampling for quality. |
| modernc.org/sqlite | v1.47.0 (in go.mod) | Compression stats logging | Already in project go.mod. Pure Go, no CGo required. WAL mode. Consistent with dashboard-daemon pattern. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| gopkg.in/yaml.v3 | v3.0.1 (in go.mod) | Config file loading | SNI bypass list, compression settings YAML. Same pattern as bypass-daemon and tunnel-monitor. |
| crypto/tls (stdlib) | Go 1.26.1 | CA cert generation | Generate CA keypair on first startup. Load into goproxy via TLSConfigFromCA(). |
| crypto/x509 (stdlib) | Go 1.26.1 | X.509 certificate creation | Self-signed CA cert generation, persist to Docker volume. |
| sync.Pool (stdlib) | Go 1.26.1 | Buffer pooling | Reuse byte buffers for response body reading/transformation to reduce GC pressure. |
| context (stdlib) | Go 1.26.1 | Timeout enforcement | 500ms deadline on image transcoding operations. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| kolesa-team/go-webp (CGo) | HugoSmits86/nativewebp (pure Go) | nativewebp only supports lossless VP8L encoding. SkyGate needs lossy at q30 for aggressive compression. Not viable. |
| kolesa-team/go-webp (CGo) | gen2brain/webp (WASM/purego) | CGo-free via wazero WASM runtime, but slower than native CGo. Since proxy runs in Docker (not Pi), CGo overhead is acceptable. |
| modernc.org/sqlite | mattn/go-sqlite3 | go-sqlite3 requires CGo. modernc.org/sqlite is already in go.mod and proven in dashboard-daemon. Use consistently. |
| elazarl/goproxy | ScrapeOps/go-proxy-mitm | Fork of goproxy with fewer users. No reason to deviate from the well-maintained original. |
| Buffered response | Chunked transfer encoding | Buffered is simpler and required for Content-Length recalculation. Images need full body for transcoding. Memory cost is bounded (most images <5MB). |

**Installation (go.mod additions):**
```bash
go get github.com/elazarl/goproxy@v1.8.2
go get github.com/kolesa-team/go-webp@v1.0.5
go get github.com/tdewolff/minify/v2@v2.24.10
go get github.com/disintegration/imaging@v1.6.2
```

**CGo note:** kolesa-team/go-webp requires `libwebp-dev` at build time and `libwebp` at runtime. The proxy Docker image must include these. This is fine since the proxy runs on a VPS, not the Pi. The Pi-side daemons remain CGO_ENABLED=0.

## Architecture Patterns

### Recommended Project Structure

```
cmd/proxy-server/              # New daemon for Phase 4
  main.go                      # Entry point: config, CA cert, goproxy setup
  config.go                    # YAML config loader (same pattern as other daemons)
  config_test.go               # Config loading tests
  proxy.go                     # goproxy setup, MITM config, handler registration
  proxy_test.go                # Proxy handler unit tests
  handlers.go                  # Response handler dispatch (Content-Type routing)
  handlers_test.go             # Handler dispatch tests
  transcoder.go                # Image transcoding: decode, resize, WebP encode
  transcoder_test.go           # Transcoder tests with sample images
  minifier.go                  # Text minification: HTML, CSS, JS via tdewolff
  minifier_test.go             # Minifier tests
  certgen.go                   # CA certificate generation and persistence
  certgen_test.go              # Cert generation tests
  db.go                        # SQLite compression logging
  db_test.go                   # Database tests
server/
  docker-compose.yml           # Updated: add proxy service
  Dockerfile.proxy             # Multi-stage CGo build for proxy binary
  .env.example                 # Updated: add proxy config vars
  proxy-config.yaml            # Default proxy configuration
  bypass-domains.yaml          # SNI bypass list (cert-pinned domains)
```

### Pattern 1: goproxy MITM with Conditional Bypass

**What:** Configure goproxy to MITM most HTTPS traffic but passthrough (ConnectAccept) for domains on the SNI bypass list. This prevents breaking cert-pinned apps.
**When to use:** Every CONNECT request hits this handler first.
**Example:**
```go
// Source: pkg.go.dev/github.com/elazarl/goproxy (HandleConnectFunc API)
proxy.OnRequest().HandleConnectFunc(
    func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
        hostname := stripPort(host)
        if bypassSet.Contains(hostname) {
            return &goproxy.ConnectAction{Action: goproxy.ConnectAccept}, host
        }
        return &goproxy.ConnectAction{
            Action:    goproxy.ConnectMitm,
            TLSConfig: goproxy.TLSConfigFromCA(&caCert),
        }, host
    })
```

### Pattern 2: Content-Type Response Handler Dispatch

**What:** After MITM decryption, route responses to the appropriate handler based on Content-Type header.
**When to use:** Every response flowing through the MITM proxy.
**Example:**
```go
// Source: pkg.go.dev/github.com/elazarl/goproxy (OnResponse API)
proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
    if resp == nil || resp.Body == nil {
        return resp
    }
    ct := resp.Header.Get("Content-Type")
    originalSize := resp.ContentLength

    switch {
    case isImage(ct):
        return handleImage(resp, ctx)
    case isMinifiable(ct):
        return handleMinify(resp, ctx)
    default:
        return resp // passthrough
    }
})
```

### Pattern 3: Image Transcoding with Timeout

**What:** Decode image, resize to max 800px width, encode to WebP at q30, with 500ms timeout. Falls back to original on timeout or error.
**When to use:** All image/* responses (except GIF/animated).
**Example:**
```go
// Source: kolesa-team/go-webp README + disintegration/imaging docs
func handleImage(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
    ct := resp.Header.Get("Content-Type")
    if strings.Contains(ct, "gif") || strings.Contains(ct, "svg") {
        return resp // D-04: GIF passthrough, D-05: SVG goes to minifier
    }

    // Read body with size limit
    body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize))
    resp.Body.Close()
    if err != nil {
        resp.Body = io.NopCloser(bytes.NewReader(body))
        return resp
    }

    // Transcode with 500ms timeout (D-03)
    ctx2, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
    defer cancel()

    resultCh := make(chan []byte, 1)
    go func() {
        result, err := transcodeToWebP(body, 30, 800) // q30, max 800px
        if err != nil {
            resultCh <- nil
            return
        }
        resultCh <- result
    }()

    select {
    case result := <-resultCh:
        if result != nil && len(result) < len(body) {
            // Log compression stats (D-10)
            logCompression(ctx, len(body), len(result), "image/webp")
            resp.Body = io.NopCloser(bytes.NewReader(result))
            resp.ContentLength = int64(len(result))
            resp.Header.Set("Content-Type", "image/webp")
            resp.Header.Del("Content-Encoding")
            return resp
        }
    case <-ctx2.Done():
        // Timeout: pass original through
    }

    resp.Body = io.NopCloser(bytes.NewReader(body))
    resp.ContentLength = int64(len(body))
    return resp
}
```

### Pattern 4: Text Minification Pipeline

**What:** Apply tdewolff/minify to HTML, CSS, and JavaScript response bodies.
**When to use:** All text/html, text/css, application/javascript responses.
**Example:**
```go
// Source: github.com/tdewolff/minify README
import (
    "github.com/tdewolff/minify/v2"
    "github.com/tdewolff/minify/v2/css"
    "github.com/tdewolff/minify/v2/html"
    "github.com/tdewolff/minify/v2/js"
    "github.com/tdewolff/minify/v2/json"
    "github.com/tdewolff/minify/v2/svg"
)

func newMinifier() *minify.M {
    m := minify.New()
    m.AddFunc("text/css", css.Minify)
    m.AddFunc("text/html", html.Minify)
    m.AddFunc("image/svg+xml", svg.Minify)
    m.AddFunc("application/json", json.Minify)
    m.AddFuncRegexp(regexp.MustCompile(`^(application|text)/(x-)?(java|ecma)script$`), js.Minify)
    return m
}

func handleMinify(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
    ct := resp.Header.Get("Content-Type")
    mediaType, _, _ := mime.ParseMediaType(ct)

    body, err := io.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
        resp.Body = io.NopCloser(bytes.NewReader(body))
        return resp
    }

    minified, err := minifier.Bytes(mediaType, body)
    if err != nil {
        // Minification failed, return original
        resp.Body = io.NopCloser(bytes.NewReader(body))
        resp.ContentLength = int64(len(body))
        return resp
    }

    logCompression(ctx, len(body), len(minified), mediaType)
    resp.Body = io.NopCloser(bytes.NewReader(minified))
    resp.ContentLength = int64(len(minified))
    resp.Header.Del("Content-Encoding") // Remove if was gzip
    return resp
}
```

### Pattern 5: Docker network_mode: service:wireguard

**What:** The proxy container shares the WireGuard container's network stack. All traffic arriving via the WireGuard tunnel is directly accessible to the proxy on localhost.
**When to use:** Docker Compose deployment with WireGuard + proxy.
**Example:**
```yaml
# Source: linuxserver.io/blog/routing-docker-host-and-container-traffic-through-wireguard
services:
  wireguard:
    image: lscr.io/linuxserver/wireguard:latest
    ports:
      - "${SERVERPORT:-51820}:51820/udp"
      - "8443:8443"   # Proxy HTTPS listen port (on WG container)
      - "8080:8080"   # CA cert download endpoint
    # ... existing config ...

  proxy:
    build:
      context: .
      dockerfile: Dockerfile.proxy
    network_mode: "service:wireguard"
    depends_on:
      - wireguard
    volumes:
      - proxy-data:/data
      - ./proxy-config.yaml:/etc/skygate/proxy.yaml:ro
      - ./bypass-domains.yaml:/etc/skygate/bypass-domains.yaml:ro
    environment:
      - SKYGATE_CONFIG=/etc/skygate/proxy.yaml
    restart: unless-stopped
```

**Critical:** When using `network_mode: service:wireguard`, all port mappings must be on the wireguard service, not the proxy service. The proxy listens on ports inside the shared network namespace, and those ports are exposed through the wireguard container.

### Pattern 6: CA Certificate Generation and Persistence

**What:** Generate a self-signed CA certificate on first startup, persist to Docker volume.
**When to use:** First proxy server boot.
**Example:**
```go
// Source: Go crypto/x509 stdlib + goproxy TLSConfigFromCA
func loadOrGenerateCA(certPath, keyPath string) (*tls.Certificate, error) {
    // Try loading existing cert
    cert, err := tls.LoadX509KeyPair(certPath, keyPath)
    if err == nil {
        return &cert, nil
    }

    // Generate new CA
    privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    template := &x509.Certificate{
        SerialNumber:          big.NewInt(1),
        Subject:               pkix.Name{Organization: []string{"SkyGate Proxy CA"}},
        NotBefore:             time.Now(),
        NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        BasicConstraintsValid: true,
        IsCA:                  true,
    }
    certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)

    // Persist to files
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
    keyDER, _ := x509.MarshalECPrivateKey(privKey)
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
    os.WriteFile(certPath, certPEM, 0644)
    os.WriteFile(keyPath, keyPEM, 0600)

    return &tls.Certificate{...}, nil
}
```

### Anti-Patterns to Avoid

- **Running proxy on the Pi:** Full content traverses the satellite link before compression. Defeats the entire purpose. Proxy MUST be on the remote server.
- **Using compy as a dependency:** Unmaintained since 2021, 35 open issues. Study its architecture but build fresh on goproxy.
- **Transparent proxy via iptables REDIRECT:** Not applicable -- traffic arrives via WireGuard tunnel. goproxy operates as a forward proxy on the tunnel endpoint.
- **CGO_ENABLED=0 for proxy binary:** go-webp requires CGo. Set CGO_ENABLED=1 in Dockerfile only. Pi-side binaries remain CGO_ENABLED=0.
- **Unbounded response body reading:** Always use io.LimitReader with a reasonable cap (e.g., 10MB for images). Prevents OOM on malicious or huge responses.
- **Modifying Content-Encoding without decompressing first:** If the upstream sends gzip-encoded content, decompress before minifying, then let the proxy re-encode. Forgetting this produces corrupted output.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTPS MITM interception | Custom TLS proxy with CONNECT handling | elazarl/goproxy | 10 years of edge cases: TLS version negotiation, SNI extraction, connection hijacking, certificate generation, HTTP/2 downgrade. |
| Image format transcoding | Custom JPEG decoder + WebP encoder | kolesa-team/go-webp + standard library image decoders | libwebp handles color space conversion, chroma subsampling, alpha channel edge cases. |
| HTML/CSS/JS minification | Regex-based stripping | tdewolff/minify | Minification is parser-dependent. Regex breaks on template literals, CSS calc(), HTML attributes. tdewolff parses properly. |
| Image resizing with quality | Manual pixel sampling | disintegration/imaging | Lanczos resampling, proper aspect ratio maintenance, EXIF orientation handling. |
| Certificate generation | OpenSSL exec calls | Go crypto/x509 stdlib | Pure Go, no external dependency, proper ASN.1 encoding, testable. |
| SQLite with WAL mode | Direct file I/O | modernc.org/sqlite | ACID transactions, concurrent reads, proper locking, query interface. Already proven in dashboard-daemon. |

**Key insight:** The proxy pipeline is a composition of well-maintained, specialized libraries. Each handles hundreds of edge cases (TLS negotiation quirks, image color spaces, JavaScript template literal parsing, SQLite WAL checkpointing) that would take months to reimplement correctly.

## Common Pitfalls

### Pitfall 1: Content-Encoding Double-Handling

**What goes wrong:** Upstream server sends gzip-compressed response. Proxy reads the raw gzip bytes, tries to minify them as HTML, produces garbage. Or: proxy decompresses, minifies, but forgets to remove the Content-Encoding header, and the browser tries to decompress already-decompressed content.
**Why it happens:** HTTP content negotiation adds a compression layer that's invisible to response body handlers.
**How to avoid:** Before any content transformation: (1) check Content-Encoding header, (2) decompress if gzip/br/deflate, (3) transform the plaintext, (4) remove Content-Encoding header from response (let the client handle it as uncompressed, or re-compress). goproxy does NOT auto-decompress responses -- the handler must do it.
**Warning signs:** Garbled output in browser, "ERR_CONTENT_DECODING_FAILED" errors.

### Pitfall 2: Image Transcoding Causing Larger Files

**What goes wrong:** WebP encoding of an already-heavily-optimized JPEG at q30 can produce a LARGER file than the original. Small images (<5KB) and already-compressed thumbnails are common offenders.
**Why it happens:** WebP encoding overhead (header, metadata) exceeds savings on already-small files. Very low quality originals have minimal redundancy left to exploit.
**How to avoid:** Always compare output size to input size. If transcoded version is not smaller, serve the original. Also skip transcoding for images below a minimum size threshold (e.g., 1KB).
**Warning signs:** Compression stats showing negative savings for some images.

### Pitfall 3: Memory Pressure from Concurrent Large Image Transcoding

**What goes wrong:** Multiple passengers browsing image-heavy sites simultaneously. Each image is buffered fully in memory for decoding and transcoding. 10 concurrent 5MB images = 50MB+ of heap pressure, plus decoded pixel buffers.
**Why it happens:** Image transcoding requires the full decompressed pixel buffer in memory (width x height x 4 bytes for RGBA).
**How to avoid:** (1) Use sync.Pool for reusable byte buffers. (2) Limit concurrent transcoding goroutines with a semaphore (e.g., `make(chan struct{}, 4)` for max 4 concurrent). (3) Set a max image size (e.g., 10MB) -- pass through anything larger without transcoding. (4) A 1GB VPS can handle 4 concurrent transcodes comfortably.
**Warning signs:** Container OOM kills, Go runtime GC pauses >100ms, proxy latency spikes during heavy browsing.

### Pitfall 4: SNI Bypass List Stale or Incomplete

**What goes wrong:** Cert-pinned apps break because their domains are not on the bypass list. Banking apps show SSL errors. Users lose trust in the system.
**Why it happens:** New cert-pinned apps appear constantly. Domain names change. CDN rotations put cert-pinned traffic on new domains.
**How to avoid:** Ship with a generous default bypass list covering banking, auth, payments, health, and government domains (per Phase 5 decisions). Make the bypass list a YAML file that can be updated independently. Log MITM errors -- they often indicate missing bypass entries. Default to passthrough (ConnectAccept) for any domain that causes TLS errors during MITM.
**Warning signs:** Proxy logs showing TLS handshake failures, user reports of "app doesn't work."

### Pitfall 5: Docker network_mode Port Collision

**What goes wrong:** The proxy container and WireGuard container both try to listen on the same port. Since they share a network namespace via `network_mode: service:wireguard`, they are effectively on the same interface.
**Why it happens:** WireGuard uses 51820/udp. Proxy needs its own ports (e.g., 8443 for HTTPS proxy, 8080 for CA cert download). Port mappings go on the WireGuard service, not the proxy service.
**How to avoid:** Define ALL port mappings on the `wireguard` service in docker-compose.yml. The proxy just listens on its ports inside the shared namespace. Document this clearly in comments.
**Warning signs:** "bind: address already in use" errors, proxy container failing to start.

### Pitfall 6: Missing Content-Length After Transformation

**What goes wrong:** After replacing the response body with transcoded/minified content, the original Content-Length header is stale. Browser receives fewer bytes than expected and hangs waiting for more data, or truncates the response.
**Why it happens:** Changing the body without updating Content-Length. goproxy does not automatically recalculate it.
**How to avoid:** After replacing resp.Body, always set `resp.ContentLength = int64(len(newBody))`. Also delete the `Content-Length` header if switching to chunked encoding (though buffered bodies with explicit length are simpler for this use case).
**Warning signs:** Incomplete page loads, images loading partially, browser dev tools showing "pending" requests.

## Code Examples

### Complete Proxy Server Setup

```go
// Source: goproxy API + project patterns
func setupProxy(cfg *Config, caCert *tls.Certificate, bypassDomains map[string]bool) *goproxy.ProxyHttpServer {
    proxy := goproxy.NewProxyHttpServer()
    proxy.Verbose = cfg.Verbose

    // Certificate cache for MITM performance
    proxy.CertStore = &memCertStore{cache: make(map[string]*tls.Certificate)}

    // Conditional MITM: bypass cert-pinned domains
    proxy.OnRequest().HandleConnectFunc(
        func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
            hostname := stripPort(host)
            if bypassDomains[hostname] || matchBypassWildcard(hostname, bypassDomains) {
                ctx.Logf("BYPASS: %s (cert-pinned)", hostname)
                return &goproxy.ConnectAction{Action: goproxy.ConnectAccept}, host
            }
            return &goproxy.ConnectAction{
                Action:    goproxy.ConnectMitm,
                TLSConfig: goproxy.TLSConfigFromCA(caCert),
            }, host
        })

    // Response handler pipeline
    proxy.OnResponse().DoFunc(responseHandler)

    return proxy
}
```

### Docker Multi-Stage Build for CGo Proxy

```dockerfile
# Source: Docker + kolesa-team/go-webp requirements
FROM golang:1.26-bookworm AS builder

# Install libwebp for CGo build
RUN apt-get update && apt-get install -y libwebp-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /proxy-server ./cmd/proxy-server/

# Runtime image
FROM debian:bookworm-slim

# Install libwebp runtime (not -dev)
RUN apt-get update && apt-get install -y libwebp7 ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /proxy-server /usr/local/bin/proxy-server

# Data directory for CA certs and SQLite
RUN mkdir -p /data/skygate

ENTRYPOINT ["/usr/local/bin/proxy-server"]
CMD ["--config", "/etc/skygate/proxy.yaml"]
```

### SQLite Compression Log Schema

```go
// Source: dashboard-daemon db.go pattern (same project)
func (d *DB) migrate() error {
    schema := `
    CREATE TABLE IF NOT EXISTS compression_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp INTEGER NOT NULL,
        domain TEXT NOT NULL,
        content_type TEXT NOT NULL,
        original_bytes INTEGER NOT NULL,
        compressed_bytes INTEGER NOT NULL,
        device_id TEXT,
        UNIQUE(timestamp, domain, device_id)
    );
    CREATE INDEX IF NOT EXISTS idx_compression_ts ON compression_log(timestamp);
    CREATE INDEX IF NOT EXISTS idx_compression_domain ON compression_log(domain);

    CREATE TABLE IF NOT EXISTS proxy_stats (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp INTEGER NOT NULL,
        requests_total INTEGER NOT NULL,
        bytes_original_total INTEGER NOT NULL,
        bytes_compressed_total INTEGER NOT NULL,
        images_transcoded INTEGER NOT NULL,
        texts_minified INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_stats_ts ON proxy_stats(timestamp);
    `
    _, err := d.db.Exec(schema)
    return err
}
```

### YAML Config Pattern (Consistent with Other Daemons)

```yaml
# proxy-config.yaml
listen_addr: ":8443"
ca_cert_path: /data/skygate/ca.crt
ca_key_path: /data/skygate/ca.key
ca_download_addr: ":8080"
db_path: /data/skygate/proxy.db
bypass_domains_file: /etc/skygate/bypass-domains.yaml
verbose: false

# Image transcoding
image:
  quality: 30
  max_width: 800
  timeout_ms: 500
  max_size_bytes: 10485760  # 10MB
  concurrent_limit: 4

# Minification
minify:
  enabled: true
  html: true
  css: true
  js: true
  svg: true
  json: true

# Logging
log:
  retention_days: 30
  batch_interval_s: 10
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| compy binary as proxy | Custom Go proxy on goproxy | 2024+ (compy unmaintained since 2021) | Must build fresh; goproxy is well-maintained foundation |
| mattn/go-sqlite3 (CGo) | modernc.org/sqlite (pure Go) | 2023+ (modernc matured) | Already adopted in project go.mod; use consistently |
| JPEG quality reduction only | JPEG/PNG to WebP transcoding | WebP support universal in 2024+ | 25-35% additional savings over optimized JPEG at equal quality |
| iptables for packet marking | nftables native | Debian 12+ (2023) | Already adopted in Phase 1; proxy uses nftables indirectly via WireGuard |
| Separate Docker network for proxy | network_mode: service:wireguard | linuxserver/wireguard pattern | Simpler networking; proxy shares WG namespace directly |

**Deprecated/outdated:**
- compy (barnacs/compy): Unmaintained since 2021, do not use as dependency
- iptables-legacy: Replaced by nftables in Debian 12+
- OpenVPN: Replaced by WireGuard for VPN tunneling

## Open Questions

1. **Proxy listening as forward proxy vs transparent proxy**
   - What we know: WireGuard tunnel delivers raw traffic to the server. goproxy can operate as a forward proxy (clients must be configured to use it) or handle transparent proxying.
   - What's unclear: Whether the Pi's nftables configuration from Phase 3 configures the tunnel traffic to hit the proxy as a forward proxy or as a transparent endpoint.
   - Recommendation: Configure as a transparent proxy using iptables REDIRECT inside the WireGuard container (or nftables equivalent) to redirect port 80/443 to the proxy listen port. This way the Pi does not need to configure individual devices as proxy clients.

2. **CA cert download endpoint accessibility**
   - What we know: D-12 says CA cert exposed via HTTP endpoint. Phase 5 captive portal will serve it to passengers.
   - What's unclear: Whether the CA download endpoint should be accessible directly on the remote server or only via the WireGuard tunnel (Pi fetches it and serves locally).
   - Recommendation: Expose on both. The proxy serves CA cert on port 8080 inside the WireGuard namespace. The Pi's captive portal (Phase 5) fetches it via tunnel and caches locally for passenger download. Direct access is useful for manual setup.

3. **Device ID extraction from WireGuard peer**
   - What we know: D-10 requires device_id in compression logs. Traffic arrives via WireGuard.
   - What's unclear: In Phase 4's single-tenant mode, all traffic comes from one WireGuard peer (the Pi). Individual device identification would require the Pi to encode device info in the traffic somehow (X-Forwarded-For header via proxy config on Pi, or IP-based if WireGuard preserves source IPs).
   - Recommendation: For v1, log the WireGuard peer IP as device_id. True per-device attribution requires Phase 5's per-device proxy mode or a custom header injected by the Pi. Keep the schema flexible (TEXT type for device_id).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build proxy binary | Yes | 1.26.1 | -- |
| Docker | Container deployment | Yes | 28.3.3 | -- |
| Docker Compose | One-command deploy | Yes | 2.39.2 | -- |
| libwebp-dev | go-webp CGo build | In Docker image | Bookworm repo | -- |

**Missing dependencies with no fallback:** None -- all build dependencies available locally or installable in Docker.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + BATS |
| Config file | `go.mod` (Go 1.26.1) |
| Quick run command | `go test ./cmd/proxy-server/... -v -short` |
| Full suite command | `go test ./... -v -short && bats pi/scripts/tests/` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PROXY-01a | Image transcoding JPEG to WebP reduces size | unit | `go test ./cmd/proxy-server/ -run TestTranscodeJPEGToWebP -v` | Wave 0 |
| PROXY-01b | PNG to WebP transcoding works | unit | `go test ./cmd/proxy-server/ -run TestTranscodePNGToWebP -v` | Wave 0 |
| PROXY-01c | 500ms timeout falls back to original | unit | `go test ./cmd/proxy-server/ -run TestTranscodeTimeout -v` | Wave 0 |
| PROXY-01d | GIF passthrough (no transcoding) | unit | `go test ./cmd/proxy-server/ -run TestGIFPassthrough -v` | Wave 0 |
| PROXY-01e | JS minification reduces size | unit | `go test ./cmd/proxy-server/ -run TestMinifyJS -v` | Wave 0 |
| PROXY-01f | CSS minification reduces size | unit | `go test ./cmd/proxy-server/ -run TestMinifyCSS -v` | Wave 0 |
| PROXY-01g | HTML minification reduces size | unit | `go test ./cmd/proxy-server/ -run TestMinifyHTML -v` | Wave 0 |
| PROXY-01h | SNI bypass domains skip MITM | unit | `go test ./cmd/proxy-server/ -run TestBypassDomains -v` | Wave 0 |
| PROXY-01i | Content-Type routing dispatches correctly | unit | `go test ./cmd/proxy-server/ -run TestContentTypeRouting -v` | Wave 0 |
| PROXY-01j | Compression stats logged to SQLite | unit | `go test ./cmd/proxy-server/ -run TestCompressionLogging -v` | Wave 0 |
| PROXY-02a | Dockerfile builds successfully | smoke | `docker build -f server/Dockerfile.proxy -t skygate-proxy .` | Wave 0 |
| PROXY-02b | docker compose up creates all services | smoke | `cd server && docker compose up -d && docker compose ps` | Wave 0 |
| PROXY-02c | CA cert generated on first startup | integration | Manual verify: `docker exec skygate-proxy ls /data/skygate/ca.crt` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/proxy-server/... -v -short`
- **Per wave merge:** `go test ./... -v -short`
- **Phase gate:** Full suite green + Docker build + compose up verification

### Wave 0 Gaps
- [ ] `cmd/proxy-server/` directory -- new daemon, all files must be created
- [ ] `server/Dockerfile.proxy` -- multi-stage CGo build
- [ ] `server/proxy-config.yaml` -- default configuration
- [ ] `server/bypass-domains.yaml` -- SNI bypass list
- [ ] Test images (sample JPEG, PNG) embedded in tests or as testdata/
- [ ] Makefile targets: `build-proxy`, `cross-build-proxy` (note: proxy does NOT cross-compile for ARM -- it runs on amd64 VPS)

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev/github.com/elazarl/goproxy](https://pkg.go.dev/github.com/elazarl/goproxy) -- Full API: HandleConnect, OnResponse, ConnectAction, CertStore, TLSConfigFromCA, HandleBytes, ProxyCtx
- [github.com/elazarl/goproxy v1.8.2](https://github.com/elazarl/goproxy/tree/v1.8.2) -- Latest release Feb 2026, MITM examples, custom CA setup
- [github.com/kolesa-team/go-webp v1.0.5](https://github.com/kolesa-team/go-webp) -- WebP encoding API, CGo requirements, lossy quality options
- [github.com/tdewolff/minify v2.24.10](https://github.com/tdewolff/minify) -- Minifier API, content type registration, streaming support, 20-70 MB/s throughput
- [linuxserver.io WireGuard routing blog](https://www.linuxserver.io/blog/routing-docker-host-and-container-traffic-through-wireguard) -- network_mode: service:wireguard pattern, port mapping rules
- [Go proxy module versions](https://proxy.golang.org) -- Verified: goproxy v1.8.2, minify v2.24.10, go-webp v1.0.5, imaging v1.6.2
- [github.com/barnacs/compy](https://github.com/barnacs/compy) -- Architecture study: filter chain pattern, content-type routing, image transcoding approach (unmaintained, do not depend on)

### Secondary (MEDIUM confidence)
- [Docker Hub linuxserver/wireguard](https://hub.docker.com/r/linuxserver/wireguard) -- Docker Compose patterns, environment variables, volume mounts
- [github.com/HugoSmits86/nativewebp v1.2.1](https://github.com/HugoSmits86/nativewebp) -- Pure Go WebP (lossless only, NOT suitable for lossy q30 requirement)
- [Go sync.Pool patterns](https://wundergraph.com/blog/golang-sync-pool) -- Buffer pooling best practices for HTTP proxy workloads

### Tertiary (LOW confidence)
- None -- all critical claims verified against primary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all library versions verified against Go module proxy, APIs confirmed via official docs
- Architecture: HIGH -- goproxy MITM patterns well-documented, Docker networking pattern proven, existing codebase patterns (YAML config, SQLite, daemon structure) established in prior phases
- Pitfalls: HIGH -- content-encoding, image size, memory pressure, bypass lists are well-known proxy engineering concerns documented in compy issues and community forums

**Research date:** 2026-03-23
**Valid until:** 2026-04-23 (stable libraries, 30-day validity)
