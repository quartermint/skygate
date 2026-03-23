package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection for compression logging.
type DB struct {
	db *sql.DB
}

// ProxyStats holds aggregated compression statistics.
type ProxyStats struct {
	RequestsTotal        int64
	BytesOriginalTotal   int64
	BytesCompressedTotal int64
	ImagesTranscoded     int64
	TextsMinified        int64
}

// NewDB opens a SQLite database at the given path, enables WAL mode,
// and initializes the schema with compression logging tables.
// Use ":memory:" for in-memory testing.
func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database %s: %w", path, err)
	}

	// Enable WAL mode for concurrent reads/writes.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	// Performance and reliability pragmas.
	if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting synchronous: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting busy_timeout: %w", err)
	}

	d := &DB{db: db}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// migrate creates all tables and indexes for compression logging.
func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS compression_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		domain TEXT NOT NULL,
		content_type TEXT NOT NULL,
		original_bytes INTEGER NOT NULL,
		compressed_bytes INTEGER NOT NULL,
		device_id TEXT DEFAULT '',
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
	if _, err := d.db.Exec(schema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	return nil
}

// LogCompression inserts a compression log entry.
// Uses nanosecond-precision timestamps to avoid UNIQUE constraint collisions
// at high write rates (per Phase 2 nanosecond timestamp decision).
func (d *DB) LogCompression(domain, contentType, deviceID string, originalBytes, compressedBytes int) error {
	now := time.Now().UnixNano()
	_, err := d.db.Exec(
		"INSERT INTO compression_log (timestamp, domain, content_type, original_bytes, compressed_bytes, device_id) VALUES (?, ?, ?, ?, ?, ?)",
		now, domain, contentType, originalBytes, compressedBytes, deviceID,
	)
	if err != nil {
		return fmt.Errorf("logging compression: %w", err)
	}
	return nil
}

// GetStats returns aggregated compression statistics since the given time.
func (d *DB) GetStats(since time.Time) (*ProxyStats, error) {
	sinceNano := since.UnixNano()
	var stats ProxyStats
	err := d.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(original_bytes), 0),
			COALESCE(SUM(compressed_bytes), 0),
			COALESCE(SUM(CASE WHEN content_type LIKE 'image/%' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN content_type NOT LIKE 'image/%' THEN 1 ELSE 0 END), 0)
		FROM compression_log
		WHERE timestamp >= ?
	`, sinceNano).Scan(
		&stats.RequestsTotal,
		&stats.BytesOriginalTotal,
		&stats.BytesCompressedTotal,
		&stats.ImagesTranscoded,
		&stats.TextsMinified,
	)
	if err != nil {
		return nil, fmt.Errorf("getting stats: %w", err)
	}
	return &stats, nil
}
