//go:build !linux

package main

import "log"

// AcceptDevice is a no-op on non-Linux platforms.
func AcceptDevice(mac, ip string) error {
	log.Printf("INFO: captive stub -- would accept device MAC=%s IP=%s", mac, ip)
	return nil
}
