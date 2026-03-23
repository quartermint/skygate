package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TunnelState represents the current tunnel health state.
type TunnelState int

const (
	StateHealthy  TunnelState = iota
	StateDegraded
)

// String returns the state name.
func (s TunnelState) String() string {
	switch s {
	case StateHealthy:
		return "HEALTHY"
	case StateDegraded:
		return "DEGRADED"
	default:
		return "UNKNOWN"
	}
}

// CheckHandshake parses `wg show <iface> latest-handshakes` output and returns
// whether the tunnel is healthy based on handshake recency.
// Output format: "<public_key>\t<unix_timestamp>\n"
func CheckHandshake(output string, maxAge time.Duration) (healthy bool, age time.Duration, err error) {
	lines := strings.TrimSpace(output)
	if lines == "" {
		return false, 0, fmt.Errorf("no handshake data (tunnel may not be configured)")
	}

	parts := strings.Fields(lines)
	if len(parts) < 2 {
		return false, 0, fmt.Errorf("unexpected wg show output: %q", lines)
	}

	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false, 0, fmt.Errorf("parsing timestamp: %w", err)
	}

	if ts == 0 {
		return false, 0, nil
	}

	age = time.Since(time.Unix(ts, 0))
	return age < maxAge, age, nil
}

// Monitor tracks tunnel health state with hysteresis to prevent flapping.
type Monitor struct {
	State        TunnelState
	FailCount    int // consecutive failures
	RecoverCount int // consecutive successes
	MaxFail      int // failures before degraded
	MaxRecover   int // successes before recovery
}

// NewMonitor creates a Monitor with the given hysteresis thresholds.
func NewMonitor(maxFail, maxRecover int) *Monitor {
	return &Monitor{
		State:      StateHealthy,
		MaxFail:    maxFail,
		MaxRecover: maxRecover,
	}
}

// Update processes a health check result and returns true if state changed.
func (m *Monitor) Update(healthy bool) (changed bool) {
	switch m.State {
	case StateHealthy:
		if !healthy {
			m.FailCount++
			m.RecoverCount = 0
			if m.FailCount >= m.MaxFail {
				m.State = StateDegraded
				m.FailCount = 0
				return true
			}
		} else {
			m.FailCount = 0
		}
	case StateDegraded:
		if healthy {
			m.RecoverCount++
			m.FailCount = 0
			if m.RecoverCount >= m.MaxRecover {
				m.State = StateHealthy
				m.RecoverCount = 0
				return true
			}
		} else {
			m.RecoverCount = 0
		}
	}
	return false
}
