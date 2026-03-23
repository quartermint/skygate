package main

import (
	"testing"
	"time"
)

func TestNewDB_Migrate(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB error: %v", err)
	}
	defer db.Close()

	// Verify compression_log table exists.
	var tableName string
	err = db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='compression_log'").Scan(&tableName)
	if err != nil {
		t.Fatalf("compression_log table not found: %v", err)
	}

	// Verify proxy_stats table exists.
	err = db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='proxy_stats'").Scan(&tableName)
	if err != nil {
		t.Fatalf("proxy_stats table not found: %v", err)
	}
}

func TestLogCompression(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB error: %v", err)
	}
	defer db.Close()

	err = db.LogCompression("example.com", "image/jpeg", "device-1", 100000, 30000)
	if err != nil {
		t.Fatalf("LogCompression error: %v", err)
	}

	// Query the row back and verify.
	var domain, contentType, deviceID string
	var originalBytes, compressedBytes int
	err = db.db.QueryRow(
		"SELECT domain, content_type, device_id, original_bytes, compressed_bytes FROM compression_log LIMIT 1",
	).Scan(&domain, &contentType, &deviceID, &originalBytes, &compressedBytes)
	if err != nil {
		t.Fatalf("querying compression_log: %v", err)
	}

	if domain != "example.com" {
		t.Errorf("domain = %q, want %q", domain, "example.com")
	}
	if contentType != "image/jpeg" {
		t.Errorf("content_type = %q, want %q", contentType, "image/jpeg")
	}
	if deviceID != "device-1" {
		t.Errorf("device_id = %q, want %q", deviceID, "device-1")
	}
	if originalBytes != 100000 {
		t.Errorf("original_bytes = %d, want 100000", originalBytes)
	}
	if compressedBytes != 30000 {
		t.Errorf("compressed_bytes = %d, want 30000", compressedBytes)
	}
}

func TestGetStats(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB error: %v", err)
	}
	defer db.Close()

	since := time.Now().Add(-1 * time.Hour)

	// Log multiple compression entries.
	entries := []struct {
		domain, ct, device string
		orig, comp         int
	}{
		{"example.com", "image/jpeg", "d1", 100000, 30000},
		{"example.com", "image/png", "d1", 50000, 15000},
		{"cdn.example.com", "text/css", "d1", 20000, 12000},
		{"cdn.example.com", "application/javascript", "d2", 80000, 50000},
	}
	for _, e := range entries {
		if err := db.LogCompression(e.domain, e.ct, e.device, e.orig, e.comp); err != nil {
			t.Fatalf("LogCompression error: %v", err)
		}
	}

	stats, err := db.GetStats(since)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}

	if stats.RequestsTotal != 4 {
		t.Errorf("RequestsTotal = %d, want 4", stats.RequestsTotal)
	}
	if stats.BytesOriginalTotal != 250000 {
		t.Errorf("BytesOriginalTotal = %d, want 250000", stats.BytesOriginalTotal)
	}
	if stats.BytesCompressedTotal != 107000 {
		t.Errorf("BytesCompressedTotal = %d, want 107000", stats.BytesCompressedTotal)
	}
	if stats.ImagesTranscoded != 2 {
		t.Errorf("ImagesTranscoded = %d, want 2", stats.ImagesTranscoded)
	}
	if stats.TextsMinified != 2 {
		t.Errorf("TextsMinified = %d, want 2", stats.TextsMinified)
	}
}
