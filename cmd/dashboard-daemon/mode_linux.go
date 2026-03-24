//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

// AddMaxSavingsMAC adds a MAC address to the maxsavings_macs nftables set with a 24h timeout.
func AddMaxSavingsMAC(mac string) error {
	element := fmt.Sprintf("{ %s timeout 24h }", mac)
	cmd := exec.Command("nft", "add", "element", nftFamily, nftTable, nftMaxSavingsSet, element)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("adding MAC %s to maxsavings set: %v (output: %s)", mac, err, string(output))
	}
	return nil
}

// RemoveMaxSavingsMAC removes a MAC address from the maxsavings_macs nftables set.
func RemoveMaxSavingsMAC(mac string) error {
	element := fmt.Sprintf("{ %s }", mac)
	cmd := exec.Command("nft", "delete", "element", nftFamily, nftTable, nftMaxSavingsSet, element)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removing MAC %s from maxsavings set: %v (output: %s)", mac, err, string(output))
	}
	return nil
}
