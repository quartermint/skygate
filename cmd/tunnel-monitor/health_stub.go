//go:build !linux

package main

import (
	"fmt"
	"log"
	"time"
)

// GetHandshakeOutput returns simulated healthy handshake output on non-Linux platforms.
func GetHandshakeOutput(iface string) (string, error) {
	ts := time.Now().Unix() - 30 // simulate 30-second-old handshake
	log.Printf("INFO: health stub -- simulating healthy handshake for %s", iface)
	return fmt.Sprintf("stub-public-key\t%d\n", ts), nil
}
