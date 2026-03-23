package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

// newTestHandlerChain creates a HandlerChain with real Transcoder and Minifier, nil DB.
func newTestHandlerChain() *HandlerChain {
	tc := NewTranscoder(ImageConfig{
		Quality:         30,
		MaxWidth:        800,
		TimeoutMS:       500,
		MaxSizeBytes:    10485760,
		ConcurrentLimit: 4,
	})
	m := NewMinifier(MinifyConfig{
		Enabled: true,
		HTML:    true,
		CSS:     true,
		JS:      true,
		SVG:     true,
		JSON:    true,
	})
	return NewHandlerChain(tc, m, nil, false)
}

// makeResponse creates a mock *http.Response with the given Content-Type and body.
func makeResponse(contentType string, body []byte) *http.Response {
	return &http.Response{
		StatusCode:    200,
		Header:        http.Header{"Content-Type": {contentType}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request: &http.Request{
			URL: &url.URL{Scheme: "https", Host: "example.com", Path: "/test"},
		},
	}
}

func TestHandleResponse_JPEG(t *testing.T) {
	hc := newTestHandlerChain()
	jpegData := makeJPEG(200, 200, 90)
	resp := makeResponse("image/jpeg", jpegData)

	result := hc.HandleResponse(resp)

	// Should be transcoded to WebP.
	ct := result.Header.Get("Content-Type")
	if ct != "image/webp" {
		t.Errorf("expected Content-Type image/webp, got %s", ct)
	}

	body, _ := io.ReadAll(result.Body)
	if len(body) >= len(jpegData) {
		t.Errorf("expected WebP body (%d) smaller than JPEG (%d)", len(body), len(jpegData))
	}

	// Content-Length should match actual body.
	cl := result.Header.Get("Content-Length")
	if cl != strconv.Itoa(len(body)) {
		t.Errorf("expected Content-Length %d, got %s", len(body), cl)
	}
}

func TestHandleResponse_PNG(t *testing.T) {
	hc := newTestHandlerChain()
	pngData := makePNG(400, 400)
	resp := makeResponse("image/png", pngData)

	result := hc.HandleResponse(resp)

	ct := result.Header.Get("Content-Type")
	if ct != "image/webp" {
		t.Errorf("expected Content-Type image/webp, got %s", ct)
	}

	body, _ := io.ReadAll(result.Body)
	if len(body) >= len(pngData) {
		t.Errorf("expected WebP body (%d) smaller than PNG (%d)", len(body), len(pngData))
	}
}

func TestHandleResponse_GIF(t *testing.T) {
	hc := newTestHandlerChain()
	gifData := []byte("GIF89a fake gif content for passthrough test")
	resp := makeResponse("image/gif", gifData)

	result := hc.HandleResponse(resp)

	// GIF should pass through unchanged (D-04).
	ct := result.Header.Get("Content-Type")
	if ct != "image/gif" {
		t.Errorf("expected Content-Type image/gif (passthrough), got %s", ct)
	}

	body, _ := io.ReadAll(result.Body)
	if !bytes.Equal(body, gifData) {
		t.Error("GIF body should pass through unchanged")
	}
}

func TestHandleResponse_SVG(t *testing.T) {
	hc := newTestHandlerChain()
	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg" >
		<!-- comment -->
		<circle  cx="50"  cy="50"  r="40"  fill="red"  />
	</svg>`)
	resp := makeResponse("image/svg+xml", svgData)

	result := hc.HandleResponse(resp)

	// SVG should route to minifier (D-05), not transcoder.
	body, _ := io.ReadAll(result.Body)
	if len(body) >= len(svgData) {
		t.Errorf("expected minified SVG (%d) shorter than original (%d)", len(body), len(svgData))
	}
}

func TestHandleResponse_JS(t *testing.T) {
	hc := newTestHandlerChain()
	jsData := []byte("var x   =   1 ;  var y   =   2 ;")
	resp := makeResponse("application/javascript", jsData)

	result := hc.HandleResponse(resp)

	body, _ := io.ReadAll(result.Body)
	if len(body) >= len(jsData) {
		t.Errorf("expected minified JS (%d) shorter than original (%d)", len(body), len(jsData))
	}
}

func TestHandleResponse_CSS(t *testing.T) {
	hc := newTestHandlerChain()
	cssData := []byte("body  {  color :  red ;  background :  blue ; }")
	resp := makeResponse("text/css", cssData)

	result := hc.HandleResponse(resp)

	body, _ := io.ReadAll(result.Body)
	if len(body) >= len(cssData) {
		t.Errorf("expected minified CSS (%d) shorter than original (%d)", len(body), len(cssData))
	}
}

func TestHandleResponse_HTML(t *testing.T) {
	hc := newTestHandlerChain()
	htmlData := []byte("<html>  <body>  <p>  hello  world  </p>  </body>  </html>")
	resp := makeResponse("text/html; charset=utf-8", htmlData)

	result := hc.HandleResponse(resp)

	body, _ := io.ReadAll(result.Body)
	if len(body) >= len(htmlData) {
		t.Errorf("expected minified HTML (%d) shorter than original (%d)", len(body), len(htmlData))
	}
}

func TestHandleResponse_Passthrough(t *testing.T) {
	hc := newTestHandlerChain()
	binData := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	resp := makeResponse("application/octet-stream", binData)

	result := hc.HandleResponse(resp)

	// Should pass through unchanged.
	body, _ := io.ReadAll(result.Body)
	if !bytes.Equal(body, binData) {
		t.Error("binary data should pass through unchanged")
	}
	ct := result.Header.Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("expected Content-Type application/octet-stream, got %s", ct)
	}
}

func TestHandleResponse_NilBody(t *testing.T) {
	hc := newTestHandlerChain()

	// Test nil response.
	result := hc.HandleResponse(nil)
	if result != nil {
		t.Error("nil response should return nil")
	}

	// Test response with nil body.
	resp := &http.Response{
		StatusCode: 204,
		Header:     http.Header{},
		Body:       nil,
		Request: &http.Request{
			URL: &url.URL{Scheme: "https", Host: "example.com", Path: "/test"},
		},
	}
	result = hc.HandleResponse(resp)
	if result != resp {
		t.Error("response with nil body should return unchanged")
	}
}

func TestHandleResponse_GzipJS(t *testing.T) {
	hc := newTestHandlerChain()

	// Create gzip-compressed JS content.
	jsOriginal := []byte("var x   =   1 ;  var y   =   2 ;  var z   =   3 ;")
	var gzipBuf bytes.Buffer
	gz := gzip.NewWriter(&gzipBuf)
	gz.Write(jsOriginal)
	gz.Close()

	resp := &http.Response{
		StatusCode:    200,
		Header:        http.Header{"Content-Type": {"application/javascript"}, "Content-Encoding": {"gzip"}},
		Body:          io.NopCloser(bytes.NewReader(gzipBuf.Bytes())),
		ContentLength: int64(gzipBuf.Len()),
		Request: &http.Request{
			URL: &url.URL{Scheme: "https", Host: "example.com", Path: "/app.js"},
		},
	}

	result := hc.HandleResponse(resp)

	// Content-Encoding should be removed after decompression.
	ce := result.Header.Get("Content-Encoding")
	if ce != "" {
		t.Errorf("expected Content-Encoding removed, got %q", ce)
	}

	body, _ := io.ReadAll(result.Body)
	if len(body) >= len(jsOriginal) {
		t.Errorf("expected minified JS (%d) shorter than original (%d)", len(body), len(jsOriginal))
	}
}

func TestHandleResponse_ContentLengthUpdated(t *testing.T) {
	hc := newTestHandlerChain()
	cssData := []byte("body  {  color :  red ;  background :  blue ;  margin :  0 ; }")
	resp := makeResponse("text/css", cssData)

	result := hc.HandleResponse(resp)

	body, _ := io.ReadAll(result.Body)

	// Content-Length header must match actual body.
	cl := result.Header.Get("Content-Length")
	expected := strconv.Itoa(len(body))
	if cl != expected {
		t.Errorf("Content-Length header %s doesn't match body length %s", cl, expected)
	}

	// resp.ContentLength field must also match.
	if result.ContentLength != int64(len(body)) {
		t.Errorf("resp.ContentLength %d doesn't match body length %d",
			result.ContentLength, len(body))
	}
}
