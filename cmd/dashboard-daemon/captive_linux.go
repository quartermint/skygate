//go:build linux

package main

import (
	"os"
	"strings"
)

// AcceptDevice adds a device's MAC to the nftables allowed_macs set,
// granting it internet access through the captive portal.
func AcceptDevice(mac, ip string) error {
	return AddAllowedMAC(mac)
}

// lookupARPTable reads /proc/net/arp to find the MAC for a given IP.
// Returns empty string if not found.
func lookupARPTable(ip string) string {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == ip {
			mac := fields[3]
			if mac != "00:00:00:00:00:00" {
				return mac
			}
		}
	}
	return ""
}
