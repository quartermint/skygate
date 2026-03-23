//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// ReadPerMACCounters executes nft -j and parses per-MAC byte counters.
func ReadPerMACCounters() (map[string]uint64, error) {
	cmd := exec.Command("nft", "-j", "list", "set", nftFamily, nftTable, nftDeviceSet)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("executing nft list set: %w", err)
	}
	return ParseNftCounters(output)
}

// AddAllowedMAC adds a MAC address to the allowed_macs nftables set with a 24h timeout.
func AddAllowedMAC(mac string) error {
	element := fmt.Sprintf("{ %s timeout 24h }", mac)
	cmd := exec.Command("nft", "add", "element", nftFamily, nftTable, nftAllowedSet, element)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("adding MAC %s to allowed set: %v (output: %s)", mac, err, string(output))
	}
	return nil
}

// RemoveAllowedMAC removes a MAC address from the allowed_macs nftables set.
func RemoveAllowedMAC(mac string) error {
	element := fmt.Sprintf("{ %s }", mac)
	cmd := exec.Command("nft", "delete", "element", nftFamily, nftTable, nftAllowedSet, element)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removing MAC %s from allowed set: %v (output: %s)", mac, err, string(output))
	}
	return nil
}

// IsAllowedMAC checks whether a MAC address is in the allowed_macs nftables set.
func IsAllowedMAC(mac string) (bool, error) {
	cmd := exec.Command("nft", "-j", "list", "set", nftFamily, nftTable, nftAllowedSet)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("listing allowed set: %w", err)
	}

	var result nftResult
	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("parsing allowed set JSON: %w", err)
	}

	for _, raw := range result.Nftables {
		var elem nftSetElem
		if json.Unmarshal(raw, &elem) == nil && elem.Elem.Val == mac {
			return true, nil
		}
	}
	return false, nil
}
