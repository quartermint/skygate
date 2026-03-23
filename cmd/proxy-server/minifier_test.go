package main

import (
	"bytes"
	"compress/gzip"
	"testing"
)

func TestMinifyJS(t *testing.T) {
	cfg := MinifyConfig{Enabled: true, HTML: true, CSS: true, JS: true, SVG: true, JSON: true}
	m := NewMinifier(cfg)

	input := []byte("var x   =   1 ;")
	output, err := m.Minify(input, "application/javascript")
	if err != nil {
		t.Fatalf("minify JS error: %v", err)
	}
	if len(output) >= len(input) {
		t.Errorf("expected minified JS (%d bytes) shorter than input (%d bytes)", len(output), len(input))
	}
	// Should contain "var x=1" or equivalent compact form.
	if !bytes.Contains(output, []byte("x=1")) && !bytes.Contains(output, []byte("x =1")) {
		t.Errorf("expected output to contain 'x=1', got: %s", string(output))
	}
}

func TestMinifyCSS(t *testing.T) {
	cfg := MinifyConfig{Enabled: true, HTML: true, CSS: true, JS: true, SVG: true, JSON: true}
	m := NewMinifier(cfg)

	input := []byte("body  {  color :  red ; }")
	output, err := m.Minify(input, "text/css")
	if err != nil {
		t.Fatalf("minify CSS error: %v", err)
	}
	if len(output) >= len(input) {
		t.Errorf("expected minified CSS (%d bytes) shorter than input (%d bytes)", len(output), len(input))
	}
}

func TestMinifyHTML(t *testing.T) {
	cfg := MinifyConfig{Enabled: true, HTML: true, CSS: true, JS: true, SVG: true, JSON: true}
	m := NewMinifier(cfg)

	input := []byte("<html>  <body>  <p>  hello  </p>  </body>  </html>")
	output, err := m.Minify(input, "text/html")
	if err != nil {
		t.Fatalf("minify HTML error: %v", err)
	}
	if len(output) >= len(input) {
		t.Errorf("expected minified HTML (%d bytes) shorter than input (%d bytes)", len(output), len(input))
	}
}

func TestMinifySVG(t *testing.T) {
	cfg := MinifyConfig{Enabled: true, HTML: true, CSS: true, JS: true, SVG: true, JSON: true}
	m := NewMinifier(cfg)

	input := []byte(`<svg xmlns="http://www.w3.org/2000/svg" >
		<!-- This is a comment -->
		<circle  cx="50"  cy="50"  r="40"  fill="red"  />
	</svg>`)
	output, err := m.Minify(input, "image/svg+xml")
	if err != nil {
		t.Fatalf("minify SVG error: %v", err)
	}
	if len(output) >= len(input) {
		t.Errorf("expected minified SVG (%d bytes) shorter than input (%d bytes)", len(output), len(input))
	}
}

func TestMinifyDisabled(t *testing.T) {
	// JS is disabled in this config.
	cfg := MinifyConfig{Enabled: true, HTML: true, CSS: true, JS: false, SVG: true, JSON: true}
	m := NewMinifier(cfg)

	input := []byte("var x   =   1 ;")
	output, err := m.Minify(input, "application/javascript")
	if err != nil {
		t.Fatalf("minify disabled JS error: %v", err)
	}
	// When JS is disabled, output should be the original unchanged.
	if !bytes.Equal(output, input) {
		t.Errorf("expected original input when JS disabled, got: %s", string(output))
	}
}

func TestDecompressGzip(t *testing.T) {
	original := []byte("body { color: red; background: blue; margin: 0; padding: 0; }")

	// Gzip compress the original.
	var gzipBuf bytes.Buffer
	gz := gzip.NewWriter(&gzipBuf)
	if _, err := gz.Write(original); err != nil {
		t.Fatalf("gzip write error: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close error: %v", err)
	}

	decompressed, err := DecompressIfNeeded(gzipBuf.Bytes(), "gzip")
	if err != nil {
		t.Fatalf("decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, original) {
		t.Errorf("expected decompressed to match original, got: %s", string(decompressed))
	}
}
