//go:build !linux

package main

import "log"

// ReadPerMACCounters returns mock data on non-Linux platforms (macOS dev).
func ReadPerMACCounters() (map[string]uint64, error) {
	log.Println("INFO: nftables stub -- returning mock per-MAC counters")
	return map[string]uint64{
		"aa:bb:cc:dd:ee:01": 1048576, // 1 MB
		"aa:bb:cc:dd:ee:02": 524288,  // 512 KB
		"aa:bb:cc:dd:ee:03": 262144,  // 256 KB
	}, nil
}

// AddAllowedMAC is a no-op on non-Linux platforms.
func AddAllowedMAC(mac string) error {
	log.Printf("INFO: nftables stub -- would add MAC %s to allowed set", mac)
	return nil
}

// RemoveAllowedMAC is a no-op on non-Linux platforms.
func RemoveAllowedMAC(mac string) error {
	log.Printf("INFO: nftables stub -- would remove MAC %s from allowed set", mac)
	return nil
}

// IsAllowedMAC always returns true on non-Linux platforms (all allowed in dev).
func IsAllowedMAC(mac string) (bool, error) {
	return true, nil
}
