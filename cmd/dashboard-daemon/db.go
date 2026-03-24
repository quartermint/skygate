package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// UsageSnapshot represents a single per-device usage data point.
type UsageSnapshot struct {
	Timestamp  int64
	MAC        string
	BytesTotal uint64
	BytesDelta uint64
}

// Device represents a connected device with metadata.
type Device struct {
	MAC       string
	Hostname  string
	UserName  string
	OUIVendor string
	FirstSeen int64
	LastSeen  int64
}

// DB wraps a SQLite database connection for the dashboard daemon.
type DB struct {
	db *sql.DB
}

// NewDB opens a SQLite database at the given path, enables WAL mode,
// and initializes the schema with all required tables.
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

// migrate creates all tables and inserts default settings.
func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS device_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		mac_addr TEXT NOT NULL,
		bytes_total INTEGER NOT NULL,
		bytes_delta INTEGER NOT NULL,
		UNIQUE(timestamp, mac_addr)
	);
	CREATE INDEX IF NOT EXISTS idx_device_usage_ts ON device_usage(timestamp);
	CREATE INDEX IF NOT EXISTS idx_device_usage_mac ON device_usage(mac_addr);

	CREATE TABLE IF NOT EXISTS devices (
		mac_addr TEXT PRIMARY KEY,
		hostname TEXT,
		user_name TEXT,
		oui_vendor TEXT,
		first_seen INTEGER NOT NULL,
		last_seen INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS domain_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		domain TEXT NOT NULL,
		query_count INTEGER NOT NULL,
		blocked INTEGER NOT NULL DEFAULT 0,
		category TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_domain_stats_ts ON domain_stats(timestamp);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS savings_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER NOT NULL,
		blocked_queries INTEGER NOT NULL,
		estimated_bytes_saved INTEGER NOT NULL,
		dollar_amount REAL NOT NULL
	);

	CREATE TABLE IF NOT EXISTS portal_accepted (
		mac_addr TEXT PRIMARY KEY,
		accepted_at INTEGER NOT NULL,
		ip_addr TEXT
	);

	CREATE TABLE IF NOT EXISTS device_modes (
		mac_addr TEXT PRIMARY KEY,
		mode TEXT NOT NULL DEFAULT 'quickconnect',
		updated_at INTEGER NOT NULL
	);
	`
	if _, err := d.db.Exec(schema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}

	// Insert default settings (INSERT OR IGNORE preserves user changes).
	now := time.Now().Unix()
	defaults := []struct{ key, value string }{
		{"plan_cap_gb", "20"},
		{"billing_cycle_start", "1"},
		{"overage_rate_per_mb", "0.01"},
	}
	for _, def := range defaults {
		_, err := d.db.Exec(
			"INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES (?, ?, ?)",
			def.key, def.value, now,
		)
		if err != nil {
			return fmt.Errorf("inserting default setting %s: %w", def.key, err)
		}
	}
	return nil
}

// WriteUsageSnapshot inserts a per-device usage data point.
// Uses nanosecond-precision timestamps to avoid UNIQUE conflicts at high write rates.
func (d *DB) WriteUsageSnapshot(mac string, bytesTotal, bytesDelta uint64) error {
	now := time.Now().UnixNano()
	_, err := d.db.Exec(
		"INSERT INTO device_usage (timestamp, mac_addr, bytes_total, bytes_delta) VALUES (?, ?, ?, ?)",
		now, mac, bytesTotal, bytesDelta,
	)
	if err != nil {
		return fmt.Errorf("writing usage snapshot: %w", err)
	}
	return nil
}

// GetDeviceUsage returns the last N usage snapshots for a given MAC address,
// ordered by timestamp descending (most recent first).
func (d *DB) GetDeviceUsage(mac string, limit int) ([]UsageSnapshot, error) {
	rows, err := d.db.Query(
		"SELECT timestamp, mac_addr, bytes_total, bytes_delta FROM device_usage WHERE mac_addr = ? ORDER BY timestamp DESC LIMIT ?",
		mac, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying device usage: %w", err)
	}
	defer rows.Close()

	var snapshots []UsageSnapshot
	for rows.Next() {
		var s UsageSnapshot
		if err := rows.Scan(&s.Timestamp, &s.MAC, &s.BytesTotal, &s.BytesDelta); err != nil {
			return nil, fmt.Errorf("scanning usage row: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

// GetSettings returns all settings as a key-value map.
func (d *DB) GetSettings() (map[string]string, error) {
	rows, err := d.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, fmt.Errorf("querying settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scanning settings row: %w", err)
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

// PutSetting writes or updates a setting key-value pair.
func (d *DB) PutSetting(key, value string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(
		"INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at",
		key, value, now,
	)
	if err != nil {
		return fmt.Errorf("putting setting %s: %w", key, err)
	}
	return nil
}

// WriteDevice writes or updates device metadata.
func (d *DB) WriteDevice(mac, hostname, oui string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(
		`INSERT INTO devices (mac_addr, hostname, oui_vendor, first_seen, last_seen) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(mac_addr) DO UPDATE SET hostname=excluded.hostname, oui_vendor=excluded.oui_vendor, last_seen=excluded.last_seen`,
		mac, hostname, oui, now, now,
	)
	if err != nil {
		return fmt.Errorf("writing device: %w", err)
	}
	return nil
}

// SetDeviceMode persists a device's mode selection (quickconnect or maxsavings).
func (d *DB) SetDeviceMode(mac, mode string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(
		"INSERT INTO device_modes (mac_addr, mode, updated_at) VALUES (?, ?, ?) ON CONFLICT(mac_addr) DO UPDATE SET mode=excluded.mode, updated_at=excluded.updated_at",
		mac, mode, now,
	)
	if err != nil {
		return fmt.Errorf("setting device mode for %s: %w", mac, err)
	}
	return nil
}

// GetDeviceMode returns the mode for a device. Returns "quickconnect" for unknown MACs (default per D-02).
func (d *DB) GetDeviceMode(mac string) (string, error) {
	var mode string
	err := d.db.QueryRow("SELECT mode FROM device_modes WHERE mac_addr = ?", mac).Scan(&mode)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "quickconnect", nil
		}
		return "", fmt.Errorf("getting device mode for %s: %w", mac, err)
	}
	return mode, nil
}

// GetMaxSavingsMACs returns all MAC addresses with mode "maxsavings".
func (d *DB) GetMaxSavingsMACs() ([]string, error) {
	rows, err := d.db.Query("SELECT mac_addr FROM device_modes WHERE mode = 'maxsavings'")
	if err != nil {
		return nil, fmt.Errorf("querying maxsavings MACs: %w", err)
	}
	defer rows.Close()

	var macs []string
	for rows.Next() {
		var mac string
		if err := rows.Scan(&mac); err != nil {
			return nil, fmt.Errorf("scanning maxsavings MAC: %w", err)
		}
		macs = append(macs, mac)
	}
	if macs == nil {
		macs = []string{}
	}
	return macs, rows.Err()
}

// GetMaxSavingsIPs returns source IPs of devices with mode "maxsavings" by
// joining device_modes with portal_accepted. This is the data source for the
// remote proxy's per-device MITM decision set (per Research Pattern 3).
func (d *DB) GetMaxSavingsIPs() ([]string, error) {
	rows, err := d.db.Query(`
		SELECT DISTINCT pa.ip_addr FROM device_modes dm
		INNER JOIN portal_accepted pa ON dm.mac_addr = pa.mac_addr
		WHERE dm.mode = 'maxsavings' AND pa.ip_addr != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("querying maxsavings IPs: %w", err)
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("scanning maxsavings IP: %w", err)
		}
		ips = append(ips, ip)
	}
	if ips == nil {
		ips = []string{}
	}
	return ips, rows.Err()
}

// GetDevices returns all known devices.
func (d *DB) GetDevices() ([]Device, error) {
	rows, err := d.db.Query("SELECT mac_addr, COALESCE(hostname,''), COALESCE(user_name,''), COALESCE(oui_vendor,''), first_seen, last_seen FROM devices ORDER BY last_seen DESC")
	if err != nil {
		return nil, fmt.Errorf("querying devices: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.MAC, &d.Hostname, &d.UserName, &d.OUIVendor, &d.FirstSeen, &d.LastSeen); err != nil {
			return nil, fmt.Errorf("scanning device row: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}
