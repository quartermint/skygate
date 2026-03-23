//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

// GetHandshakeOutput runs `wg show <iface> latest-handshakes` and returns the output.
func GetHandshakeOutput(iface string) (string, error) {
	cmd := exec.Command("wg", "show", iface, "latest-handshakes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wg show %s: %v (output: %s)", iface, err, string(output))
	}
	return string(output), nil
}
