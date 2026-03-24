package main

import (
	"encoding/json"
	"fmt"
)

// nftables constants for the SkyGate firewall table.
const (
	nftFamily     = "inet"
	nftTable      = "skygate"
	nftDeviceSet  = "device_counters"
	nftAllowedSet     = "allowed_macs"
	nftMaxSavingsSet  = "maxsavings_macs"
)

// nftResult represents the top-level nft -j JSON output.
type nftResult struct {
	Nftables []json.RawMessage `json:"nftables"`
}

// nftSetElem represents a single set element with counter from nft -j output.
type nftSetElem struct {
	Elem struct {
		Val     string `json:"val"`
		Counter struct {
			Packets uint64 `json:"packets"`
			Bytes   uint64 `json:"bytes"`
		} `json:"counter"`
	} `json:"elem"`
}

// DeviceCounters tracks previous and current counter readings for delta computation.
type DeviceCounters struct {
	Previous map[string]uint64
	Current  map[string]uint64
}

// ParseNftCounters parses JSON output from `nft -j list set inet skygate device_counters`.
// Returns a map of MAC address -> total bytes.
func ParseNftCounters(data []byte) (map[string]uint64, error) {
	var result nftResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing nft JSON: %w", err)
	}

	counters := make(map[string]uint64)
	for _, raw := range result.Nftables {
		var elem nftSetElem
		if json.Unmarshal(raw, &elem) == nil && elem.Elem.Val != "" {
			counters[elem.Elem.Val] = elem.Elem.Counter.Bytes
		}
	}
	return counters, nil
}

// ComputeDeltas calculates the byte delta between two counter readings.
// For each MAC in curr:
//   - If MAC exists in prev and curr >= prev: delta = curr - prev
//   - If MAC exists in prev and curr < prev (counter reset): delta = curr
//   - If MAC not in prev (new device): delta = curr
func ComputeDeltas(prev, curr map[string]uint64) map[string]uint64 {
	deltas := make(map[string]uint64)
	for mac, currBytes := range curr {
		prevBytes, ok := prev[mac]
		if !ok {
			// New device: full value as delta.
			deltas[mac] = currBytes
		} else if currBytes >= prevBytes {
			// Normal case: delta is the difference.
			deltas[mac] = currBytes - prevBytes
		} else {
			// Counter reset (reboot): use current as delta.
			deltas[mac] = currBytes
		}
	}
	return deltas
}
