//go:build linux

package main

// AcceptDevice adds a device's MAC to the nftables allowed_macs set,
// granting it internet access through the captive portal.
func AcceptDevice(mac, ip string) error {
	return AddAllowedMAC(mac)
}
