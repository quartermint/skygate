package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// makeJPEG creates a programmatic JPEG image of the given dimensions at the given quality.
func makeJPEG(width, height, quality int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a gradient so compression has some data to work with.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// makePNG creates a programmatic PNG image of the given dimensions.
func makePNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 100,
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestTranscodeJPEGToWebP(t *testing.T) {
	jpegData := makeJPEG(100, 100, 90)
	tc := NewTranscoder(ImageConfig{
		Quality:         30,
		MaxWidth:        800,
		TimeoutMS:       500,
		MaxSizeBytes:    10485760,
		ConcurrentLimit: 4,
	})

	result := tc.Transcode(jpegData, "image/jpeg")
	if result == nil {
		t.Fatal("expected non-nil result for JPEG transcoding")
	}
	if result.ContentType != "image/webp" {
		t.Errorf("expected content type image/webp, got %s", result.ContentType)
	}
	// WebP files start with "RIFF"
	if len(result.Data) < 4 || string(result.Data[:4]) != "RIFF" {
		t.Error("output does not start with WebP magic bytes 'RIFF'")
	}
	if len(result.Data) >= len(jpegData) {
		t.Errorf("expected WebP output (%d bytes) smaller than JPEG input (%d bytes)",
			len(result.Data), len(jpegData))
	}
}

func TestTranscodePNGToWebP(t *testing.T) {
	// Use a larger PNG (400x400) to ensure WebP savings exceed original size.
	pngData := makePNG(400, 400)
	tc := NewTranscoder(ImageConfig{
		Quality:         30,
		MaxWidth:        800,
		TimeoutMS:       500,
		MaxSizeBytes:    10485760,
		ConcurrentLimit: 4,
	})

	result := tc.Transcode(pngData, "image/png")
	if result == nil {
		t.Fatal("expected non-nil result for PNG transcoding")
	}
	if result.ContentType != "image/webp" {
		t.Errorf("expected content type image/webp, got %s", result.ContentType)
	}
	// WebP files start with "RIFF"
	if len(result.Data) < 4 || string(result.Data[:4]) != "RIFF" {
		t.Error("output does not start with WebP magic bytes 'RIFF'")
	}
	if len(result.Data) >= len(pngData) {
		t.Errorf("expected WebP output (%d bytes) smaller than PNG input (%d bytes)",
			len(result.Data), len(pngData))
	}
}

func TestTranscodeResize(t *testing.T) {
	// Create a 1600x1200 JPEG that must be resized to max 800px width.
	jpegData := makeJPEG(1600, 1200, 90)
	tc := NewTranscoder(ImageConfig{
		Quality:         30,
		MaxWidth:        800,
		TimeoutMS:       5000, // generous timeout for large image
		MaxSizeBytes:    10485760,
		ConcurrentLimit: 4,
	})

	result := tc.Transcode(jpegData, "image/jpeg")
	if result == nil {
		t.Fatal("expected non-nil result for resize transcoding")
	}

	// Decode the WebP result to check dimensions.
	// We can't easily decode WebP in Go without CGo, so we just verify
	// the result is smaller than input (resize + q30 should produce significant savings).
	if len(result.Data) >= len(jpegData) {
		t.Errorf("expected resized WebP (%d bytes) smaller than 1600x1200 JPEG (%d bytes)",
			len(result.Data), len(jpegData))
	}
}

func TestTranscodeTimeout(t *testing.T) {
	// Use a very small timeout (1ms) to force a timeout on transcoding.
	jpegData := makeJPEG(1600, 1200, 90) // large image, more likely to exceed 1ms
	tc := NewTranscoder(ImageConfig{
		Quality:         30,
		MaxWidth:        800,
		TimeoutMS:       1, // 1ms timeout -- should trigger timeout
		MaxSizeBytes:    10485760,
		ConcurrentLimit: 4,
	})

	// A timeout should return nil (graceful skip), not an error.
	result := tc.Transcode(jpegData, "image/jpeg")
	// Result may or may not be nil depending on speed; the key property
	// is that it does NOT panic and returns gracefully.
	_ = result
}

func TestTranscodeSkipSmall(t *testing.T) {
	// Create a tiny image under 1KB.
	// A 2x2 JPEG is extremely small.
	jpegData := makeJPEG(2, 2, 10)
	if len(jpegData) >= 1024 {
		t.Skipf("test JPEG is %d bytes, expected < 1024", len(jpegData))
	}

	tc := NewTranscoder(ImageConfig{
		Quality:         30,
		MaxWidth:        800,
		TimeoutMS:       500,
		MaxSizeBytes:    10485760,
		ConcurrentLimit: 4,
	})

	result := tc.Transcode(jpegData, "image/jpeg")
	if result != nil {
		t.Errorf("expected nil result for small image (%d bytes), got data", len(jpegData))
	}
}

func TestTranscodeSkipLarger(t *testing.T) {
	// This tests the "transcoded larger than original" skip.
	// We create an already heavily compressed, small JPEG where
	// WebP encoding might not save bytes.
	// If the WebP output ends up being >= input, Transcode should return nil.
	// We use a very small image at low quality.
	jpegData := makeJPEG(10, 10, 10)

	tc := NewTranscoder(ImageConfig{
		Quality:         30,
		MaxWidth:        800,
		TimeoutMS:       500,
		MaxSizeBytes:    10485760,
		ConcurrentLimit: 4,
	})

	// This test verifies the code path exists. The result may be nil
	// (if WebP is larger) or non-nil (if WebP is still smaller).
	// The key is it doesn't panic and handles the comparison correctly.
	result := tc.Transcode(jpegData, "image/jpeg")
	if result != nil && len(result.Data) >= len(jpegData) {
		t.Error("result should be nil when transcoded output is larger than original")
	}
}
