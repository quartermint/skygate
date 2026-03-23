package main

import (
	"strings"
	"testing"
)

func TestFormatNftCommand(t *testing.T) {
	args := FormatNftCommand("1.2.3.4", 1)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "add element") {
		t.Errorf("expected 'add element' in command, got: %s", joined)
	}
	if !strings.Contains(joined, "inet skygate bypass_v4") {
		t.Errorf("expected 'inet skygate bypass_v4' in command, got: %s", joined)
	}
	if !strings.Contains(joined, "1.2.3.4") {
		t.Errorf("expected IP in command, got: %s", joined)
	}
	if !strings.Contains(joined, "timeout 1h") {
		t.Errorf("expected 'timeout 1h' in command, got: %s", joined)
	}
}
