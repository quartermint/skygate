//go:build !linux

package main

import "log"

// UpdateNftSet is a no-op on non-Linux platforms (macOS dev).
func UpdateNftSet(ips []string) error {
	log.Printf("INFO: nftset stub -- would add %d IPs (not on Linux)", len(ips))
	return nil
}

// FlushNftSet is a no-op on non-Linux platforms.
func FlushNftSet() error {
	return nil
}
