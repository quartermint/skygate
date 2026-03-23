//go:build !linux

package main

import "log"

// updateNftSet is a no-op stub for non-Linux platforms (macOS dev environment).
// On Linux, the real implementation in nft_linux.go shells out to the nft CLI.
func updateNftSet(ips []string) error {
	log.Printf("stub: would update nftables bypass_v4 set with %d IPs (non-Linux, skipping)", len(ips))
	return nil
}
