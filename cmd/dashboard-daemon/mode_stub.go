//go:build !linux

package main

import "log"

// AddMaxSavingsMAC is a no-op on non-Linux platforms.
func AddMaxSavingsMAC(mac string) error {
	log.Printf("INFO: nftables stub -- would add MAC %s to maxsavings set", mac)
	return nil
}

// RemoveMaxSavingsMAC is a no-op on non-Linux platforms.
func RemoveMaxSavingsMAC(mac string) error {
	log.Printf("INFO: nftables stub -- would remove MAC %s from maxsavings set", mac)
	return nil
}
