package main

import (
	"testing"
)

func TestNewDB_CreatesSchema(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	// Check all 6 tables exist
	tables := []string{"device_usage", "devices", "domain_stats", "settings", "savings_log", "portal_accepted"}
	for _, table := range tables {
		var name string
		err := db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestNewDB_WALMode(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	var mode string
	err = db.db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode failed: %v", err)
	}
	// In-memory databases may return "memory" instead of "wal"
	// but for file-based DBs it should be "wal". Accept both for in-memory tests.
	if mode != "wal" && mode != "memory" {
		t.Errorf("expected journal_mode wal or memory, got %s", mode)
	}
}

func TestNewDB_WALMode_File(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	var mode string
	err = db.db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode failed: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected journal_mode wal, got %s", mode)
	}
}

func TestWriteUsageSnapshot(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	mac := "aa:bb:cc:dd:ee:01"
	err = db.WriteUsageSnapshot(mac, 1048576, 524288)
	if err != nil {
		t.Fatalf("WriteUsageSnapshot failed: %v", err)
	}

	snapshots, err := db.GetDeviceUsage(mac, 10)
	if err != nil {
		t.Fatalf("GetDeviceUsage failed: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
	if snapshots[0].MAC != mac {
		t.Errorf("expected MAC %s, got %s", mac, snapshots[0].MAC)
	}
	if snapshots[0].BytesTotal != 1048576 {
		t.Errorf("expected bytes_total 1048576, got %d", snapshots[0].BytesTotal)
	}
	if snapshots[0].BytesDelta != 524288 {
		t.Errorf("expected bytes_delta 524288, got %d", snapshots[0].BytesDelta)
	}
}

func TestGetDeviceUsage(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	mac := "aa:bb:cc:dd:ee:02"
	// Insert 3 snapshots
	for i := 0; i < 3; i++ {
		err = db.WriteUsageSnapshot(mac, uint64(1000*(i+1)), uint64(1000))
		if err != nil {
			t.Fatalf("WriteUsageSnapshot %d failed: %v", i, err)
		}
	}

	// Get last 2
	snapshots, err := db.GetDeviceUsage(mac, 2)
	if err != nil {
		t.Fatalf("GetDeviceUsage failed: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	// Should be ordered by timestamp desc (most recent first)
	if snapshots[0].BytesTotal < snapshots[1].BytesTotal {
		t.Errorf("expected desc order, got first=%d second=%d", snapshots[0].BytesTotal, snapshots[1].BytesTotal)
	}
}

func TestPutSetting_GetSetting(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	err = db.PutSetting("test_key", "test_value")
	if err != nil {
		t.Fatalf("PutSetting failed: %v", err)
	}

	settings, err := db.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings["test_key"] != "test_value" {
		t.Errorf("expected test_value, got %s", settings["test_key"])
	}
}

func TestDefaultSettings(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	defer db.Close()

	settings, err := db.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}

	expected := map[string]string{
		"plan_cap_gb":          "20",
		"billing_cycle_start":  "1",
		"overage_rate_per_mb":  "0.01",
	}
	for key, val := range expected {
		if settings[key] != val {
			t.Errorf("expected default %s=%s, got %s", key, val, settings[key])
		}
	}
}
