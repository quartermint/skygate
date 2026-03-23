package main

import (
	"bytes"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// HandlerChain orchestrates Content-Type based response transformation.
type HandlerChain struct {
	transcoder *Transcoder
	minifier   *Minifier
	db         *DB   // nil-safe: if nil, logging is skipped
	verbose    bool
}

// NewHandlerChain creates a HandlerChain from the given components.
func NewHandlerChain(transcoder *Transcoder, minifier *Minifier, db *DB, verbose bool) *HandlerChain {
	return &HandlerChain{
		transcoder: transcoder,
		minifier:   minifier,
		db:         db,
		verbose:    verbose,
	}
}

// HandleResponse applies Content-Type based transformation to the response.
// Routes image/* to transcoder, text/html+css+js to minifier, everything else passthrough.
// Per D-07 and D-08.
func (h *HandlerChain) HandleResponse(resp *http.Response) *http.Response {
	// Guard: nil response or nil body -> passthrough.
	if resp == nil {
		return nil
	}
	if resp.Body == nil {
		return resp
	}

	// Extract and parse Content-Type.
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		return resp
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		// Unparseable Content-Type: passthrough.
		return resp
	}

	switch {
	case isImage(mediaType):
		return h.handleImage(resp, mediaType)
	case isMinifiable(mediaType):
		return h.handleMinify(resp, mediaType)
	default:
		return resp
	}
}

// handleImage reads the response body, transcodes to WebP, and replaces the body.
func (h *HandlerChain) handleImage(resp *http.Response, mediaType string) *http.Response {
	// Read full body with size limit (anti-pattern: unbounded reads).
	maxSize := int64(10485760) // 10MB default
	if h.transcoder.MaxSizeBytes > 0 {
		maxSize = int64(h.transcoder.MaxSizeBytes)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	result := h.transcoder.Transcode(body, mediaType)
	if result != nil {
		// Transcoding succeeded: replace body.
		domain := extractDomain(resp)
		if h.db != nil {
			if err := h.db.LogCompression(domain, result.ContentType, "", len(body), len(result.Data)); err != nil && h.verbose {
				log.Printf("WARNING: failed to log compression: %v", err)
			}
		}
		resp.Body = io.NopCloser(bytes.NewReader(result.Data))
		resp.ContentLength = int64(len(result.Data))
		resp.Header.Set("Content-Type", result.ContentType)
		resp.Header.Set("Content-Length", strconv.Itoa(len(result.Data)))
		resp.Header.Del("Content-Encoding")
		return resp
	}

	// Transcoding skipped/failed: restore original body.
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return resp
}

// handleMinify reads the response body, decompresses if needed, minifies, and replaces the body.
func (h *HandlerChain) handleMinify(resp *http.Response, mediaType string) *http.Response {
	// Read full body.
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	// Per Pitfall 1: decompress before minifying.
	contentEncoding := resp.Header.Get("Content-Encoding")
	decompressed, err := DecompressIfNeeded(body, contentEncoding)
	if err != nil {
		if h.verbose {
			log.Printf("WARNING: decompression failed: %v", err)
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		return resp
	}

	// Minify the decompressed content.
	minified, err := h.minifier.Minify(decompressed, mediaType)
	if err != nil {
		// Should not happen (Minify returns original on error), but handle gracefully.
		resp.Body = io.NopCloser(bytes.NewReader(decompressed))
		resp.ContentLength = int64(len(decompressed))
		resp.Header.Del("Content-Encoding")
		resp.Header.Set("Content-Length", strconv.Itoa(len(decompressed)))
		return resp
	}

	// Log compression stats.
	domain := extractDomain(resp)
	if h.db != nil {
		if logErr := h.db.LogCompression(domain, mediaType, "", len(decompressed), len(minified)); logErr != nil && h.verbose {
			log.Printf("WARNING: failed to log compression: %v", logErr)
		}
	}

	// Replace body, update headers.
	resp.Body = io.NopCloser(bytes.NewReader(minified))
	resp.ContentLength = int64(len(minified))
	resp.Header.Set("Content-Length", strconv.Itoa(len(minified)))
	// Per Pitfall 1: remove Content-Encoding after decompression.
	resp.Header.Del("Content-Encoding")
	return resp
}

// isImage returns true for image content types that should be transcoded.
// False for image/gif (D-04: passthrough) and image/svg+xml (D-05: routes to minifier).
func isImage(mediaType string) bool {
	return strings.HasPrefix(mediaType, "image/") &&
		mediaType != "image/gif" &&
		mediaType != "image/svg+xml"
}

// isMinifiable returns true for content types that should be minified.
func isMinifiable(mediaType string) bool {
	switch mediaType {
	case "text/html", "text/css",
		"application/javascript", "text/javascript", "application/x-javascript",
		"image/svg+xml", "application/json":
		return true
	}
	return false
}

// extractDomain returns the hostname from the response's original request URL.
func extractDomain(resp *http.Response) string {
	if resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.Hostname()
	}
	return "unknown"
}
