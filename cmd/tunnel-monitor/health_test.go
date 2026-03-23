package main

import (
	"fmt"
	"testing"
	"time"
)

func TestCheckHandshake_Healthy(t *testing.T) {
	// Recent handshake (30 seconds ago)
	ts := time.Now().Unix() - 30
	output := fmt.Sprintf("abcdef1234567890abcdef1234567890abcdef1234=\t%d\n", ts)
	healthy, age, err := CheckHandshake(output, 180*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !healthy {
		t.Error("expected healthy, got unhealthy")
	}
	if age > 60*time.Second {
		t.Errorf("expected age < 60s, got %s", age)
	}
}

func TestCheckHandshake_Unhealthy(t *testing.T) {
	// Old handshake (300 seconds ago, maxAge=180s)
	ts := time.Now().Unix() - 300
	output := fmt.Sprintf("abcdef1234567890abcdef1234567890abcdef1234=\t%d\n", ts)
	healthy, age, err := CheckHandshake(output, 180*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if healthy {
		t.Error("expected unhealthy, got healthy")
	}
	if age < 180*time.Second {
		t.Errorf("expected age > 180s, got %s", age)
	}
}

func TestCheckHandshake_NoHandshake(t *testing.T) {
	// Timestamp 0 means never connected
	output := "abcdef1234567890abcdef1234567890abcdef1234=\t0\n"
	healthy, _, err := CheckHandshake(output, 180*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if healthy {
		t.Error("expected unhealthy for timestamp 0 (never connected)")
	}
}

func TestCheckHandshake_EmptyOutput(t *testing.T) {
	_, _, err := CheckHandshake("", 180*time.Second)
	if err == nil {
		t.Error("expected error for empty output")
	}
}

func TestCheckHandshake_MalformedOutput(t *testing.T) {
	_, _, err := CheckHandshake("garbage", 180*time.Second)
	if err == nil {
		t.Error("expected error for malformed output")
	}
}

func TestStateTransition_HealthyToDegraded(t *testing.T) {
	mon := NewMonitor(3, 3)
	if mon.State != StateHealthy {
		t.Fatalf("expected initial state StateHealthy, got %s", mon.State)
	}

	// 3 consecutive failures should trigger transition
	for i := 0; i < 2; i++ {
		changed := mon.Update(false)
		if changed {
			t.Errorf("unexpected state change at failure %d", i+1)
		}
		if mon.State != StateHealthy {
			t.Errorf("expected StateHealthy at failure %d, got %s", i+1, mon.State)
		}
	}

	// Third failure triggers transition
	changed := mon.Update(false)
	if !changed {
		t.Error("expected state change on 3rd failure")
	}
	if mon.State != StateDegraded {
		t.Errorf("expected StateDegraded after 3 failures, got %s", mon.State)
	}
}

func TestStateTransition_DegradedToHealthy(t *testing.T) {
	mon := NewMonitor(3, 3)
	// Force to degraded state
	mon.State = StateDegraded

	// 3 consecutive successes should trigger recovery
	for i := 0; i < 2; i++ {
		changed := mon.Update(true)
		if changed {
			t.Errorf("unexpected state change at success %d", i+1)
		}
		if mon.State != StateDegraded {
			t.Errorf("expected StateDegraded at success %d, got %s", i+1, mon.State)
		}
	}

	// Third success triggers recovery
	changed := mon.Update(true)
	if !changed {
		t.Error("expected state change on 3rd success")
	}
	if mon.State != StateHealthy {
		t.Errorf("expected StateHealthy after 3 successes, got %s", mon.State)
	}
}

func TestStateTransition_NoFlapping(t *testing.T) {
	mon := NewMonitor(3, 3)

	// 2 failures, then 1 success should reset fail counter
	mon.Update(false)
	mon.Update(false)
	mon.Update(true) // resets fail counter

	// Now 2 more failures -- should NOT trigger degraded (counter was reset)
	mon.Update(false)
	mon.Update(false)
	if mon.State != StateHealthy {
		t.Errorf("expected StateHealthy (no flapping), got %s", mon.State)
	}
}

func TestStateTransition_Hysteresis(t *testing.T) {
	mon := NewMonitor(3, 3)

	// Alternating healthy/unhealthy should never trigger transition
	for i := 0; i < 20; i++ {
		healthy := i%2 == 0
		mon.Update(healthy)
		if mon.State != StateHealthy {
			t.Fatalf("unexpected state change on iteration %d (alternating)", i)
		}
	}
}
