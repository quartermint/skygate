package main

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"time"

	"github.com/disintegration/imaging"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

// Transcoder handles image transcoding to WebP with resize and timeout.
type Transcoder struct {
	Quality      int           // WebP quality (0-100), default 30
	MaxWidth     int           // Max width in pixels, default 800
	Timeout      time.Duration // Per-image timeout, default 500ms
	MaxSizeBytes int           // Skip images larger than this
	MinSizeBytes int           // Skip images smaller than this (1024 = 1KB)
	sem          chan struct{} // Concurrency limiter
}

// NewTranscoder creates a Transcoder from ImageConfig.
func NewTranscoder(cfg ImageConfig) *Transcoder {
	concurrentLimit := cfg.ConcurrentLimit
	if concurrentLimit <= 0 {
		concurrentLimit = 4
	}
	return &Transcoder{
		Quality:      cfg.Quality,
		MaxWidth:     cfg.MaxWidth,
		Timeout:      time.Duration(cfg.TimeoutMS) * time.Millisecond,
		MaxSizeBytes: cfg.MaxSizeBytes,
		MinSizeBytes: 1024, // 1KB minimum -- smaller images get larger after transcoding
		sem:          make(chan struct{}, concurrentLimit),
	}
}

// TranscodeResult holds the transcoded image data, or nil if skipped/failed.
type TranscodeResult struct {
	Data        []byte
	ContentType string // always "image/webp" if successful
}

// Transcode decodes the image from body, resizes to MaxWidth if wider,
// encodes to WebP at Quality. Returns nil result (not error) on timeout,
// skip (too small/too large), or if result is larger than original.
func (t *Transcoder) Transcode(body []byte, contentType string) *TranscodeResult {
	// Skip tiny images (Pitfall 2: overhead makes them larger).
	if len(body) < t.MinSizeBytes {
		return nil
	}

	// Skip oversized images (Pitfall 3: memory safety).
	if t.MaxSizeBytes > 0 && len(body) > t.MaxSizeBytes {
		return nil
	}

	// Acquire semaphore slot for concurrency limiting (Pitfall 3).
	t.sem <- struct{}{}
	defer func() { <-t.sem }()

	// Use context.WithTimeout for 500ms limit (D-03).
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	type result struct {
		data []byte
	}
	ch := make(chan result, 1)

	go func() {
		data := t.doTranscode(body)
		ch <- result{data: data}
	}()

	select {
	case r := <-ch:
		if r.data == nil {
			return nil
		}
		// Pitfall 2: transcoded image larger than original -- skip.
		if len(r.data) >= len(body) {
			return nil
		}
		return &TranscodeResult{
			Data:        r.data,
			ContentType: "image/webp",
		}
	case <-ctx.Done():
		// Timeout: return nil (graceful skip, D-03).
		return nil
	}
}

// doTranscode performs the actual image decode, resize, and WebP encode.
func (t *Transcoder) doTranscode(body []byte) []byte {
	// Decode the image (supports JPEG, PNG via blank imports).
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	// Resize if wider than MaxWidth (preserve aspect ratio).
	bounds := img.Bounds()
	if bounds.Dx() > t.MaxWidth {
		img = imaging.Resize(img, t.MaxWidth, 0, imaging.Lanczos)
	}

	// Encode to WebP at the configured quality.
	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, float32(t.Quality))
	if err != nil {
		return nil
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, options); err != nil {
		return nil
	}

	return buf.Bytes()
}
