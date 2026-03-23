//go:build linux

package main

import (
	"fmt"
	"log"
	"os/exec"
)

// UpdateNftSet adds all given IPs to the nftables bypass_v4 set.
// Each IP gets a 1-hour timeout (refreshed each cycle).
func UpdateNftSet(ips []string) error {
	for _, ip := range ips {
		args := FormatNftCommand(ip, 1)
		cmd := exec.Command("nft", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Printf("WARN: nft add element failed for %s: %v (output: %s)", ip, err, string(output))
			// Continue with other IPs -- don't fail the entire set update
		}
	}
	return nil
}

// FlushNftSet removes all elements from the bypass set.
func FlushNftSet() error {
	cmd := exec.Command("nft", "flush", "set", nftTable, nftFamily, nftSet)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("flushing nft set: %v (output: %s)", err, string(output))
	}
	return nil
}
