//go:build linux

package main

import (
	"fmt"
	"log"
	"os/exec"
)

// ExecIPRule executes an `ip` command with the given arguments.
func ExecIPRule(args []string) error {
	cmd := exec.Command("ip", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip %v: %v (output: %s)", args, err, string(output))
	}
	log.Printf("ip rule: %v", args)
	return nil
}
