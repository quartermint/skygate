package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"regexp"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
)

// Minifier wraps tdewolff/minify with SkyGate configuration.
type Minifier struct {
	m   *minify.M
	cfg MinifyConfig
}

// NewMinifier creates a configured Minifier from MinifyConfig.
// Registers handlers only for enabled content types.
func NewMinifier(cfg MinifyConfig) *Minifier {
	m := minify.New()

	if cfg.CSS {
		m.AddFunc("text/css", css.Minify)
	}
	if cfg.HTML {
		m.AddFunc("text/html", html.Minify)
	}
	if cfg.SVG {
		m.AddFunc("image/svg+xml", svg.Minify)
	}
	if cfg.JSON {
		m.AddFunc("application/json", json.Minify)
	}
	if cfg.JS {
		m.AddFuncRegexp(regexp.MustCompile(`^(application|text)/(x-)?(java|ecma)script$`), js.Minify)
	}

	return &Minifier{
		m:   m,
		cfg: cfg,
	}
}

// Minify applies minification to the body based on the media type.
// Returns the original body unchanged on error or if the media type is not registered.
func (m *Minifier) Minify(body []byte, mediaType string) ([]byte, error) {
	result, err := m.m.Bytes(mediaType, body)
	if err != nil {
		// Minification failed; return original unchanged.
		return body, nil
	}
	return result, nil
}

// CanMinify returns true if the given media type has a registered minifier.
func (m *Minifier) CanMinify(mediaType string) bool {
	// Attempt a zero-byte minification to check if the media type is registered.
	// The minify library returns an error for unregistered types.
	_, err := m.m.Bytes(mediaType, []byte{})
	return err == nil
}

// DecompressIfNeeded decompresses the body if the Content-Encoding indicates compression.
// Per Pitfall 1: MUST decompress before minifying.
//   - "gzip": decompress via gzip.NewReader
//   - "br": return as-is with logged warning (brotli decompression deferred to v2)
//   - "" or "identity": return as-is
func DecompressIfNeeded(body []byte, contentEncoding string) ([]byte, error) {
	switch contentEncoding {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}
		defer reader.Close()
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("decompressing gzip: %w", err)
		}
		return decompressed, nil
	case "br":
		log.Println("WARNING: brotli decompression not yet implemented, passing through compressed body")
		return body, nil
	case "", "identity":
		return body, nil
	default:
		log.Printf("WARNING: unknown Content-Encoding %q, passing through unchanged", contentEncoding)
		return body, nil
	}
}
