//go:build !linux

package main

import "log"

// ExecIPRule is a no-op on non-Linux platforms (macOS dev).
func ExecIPRule(args []string) error {
	log.Printf("INFO: fallback stub -- would exec: ip %v (not on Linux)", args)
	return nil
}
