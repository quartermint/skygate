//go:build linux

package main

import (
	"fmt"
	"log"
	"os/exec"
)

// updateNftSet populates the nftables bypass_v4 set with the given IPs.
// Each IP is added with a 1-hour timeout so stale entries expire automatically.
func updateNftSet(ips []string) error {
	for _, ip := range ips {
		cmd := exec.Command("nft", "add", "element", "inet", "skygate",
			"bypass_v4", "{", ip, "timeout", "1h", "}")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("nft add element %s: %w (output: %s)", ip, err, string(out))
		}
		log.Printf("added %s to bypass_v4 set", ip)
	}
	return nil
}
