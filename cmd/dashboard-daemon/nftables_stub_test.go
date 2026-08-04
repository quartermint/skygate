//go:build !linux

// These exercise the no-op stub in nftables_stub.go. On linux the real
// implementation shells out to nft and needs root, so they belong behind
// the same build tag as the code they cover.

package main

import (
	"testing"
)

func TestReadPerMACCounters_Stub(t *testing.T) {
	counters, err := ReadPerMACCounters()
	if err != nil {
		t.Fatalf("ReadPerMACCounters stub failed: %v", err)
	}
	if len(counters) < 2 {
		t.Errorf("expected at least 2 mock entries, got %d", len(counters))
	}
}

func TestAddAllowedMAC_Stub(t *testing.T) {
	err := AddAllowedMAC("aa:bb:cc:dd:ee:01")
	if err != nil {
		t.Errorf("AddAllowedMAC stub returned error: %v", err)
	}
}

func TestRemoveAllowedMAC_Stub(t *testing.T) {
	err := RemoveAllowedMAC("aa:bb:cc:dd:ee:01")
	if err != nil {
		t.Errorf("RemoveAllowedMAC stub returned error: %v", err)
	}
}

func TestAcceptDevice_Stub(t *testing.T) {
	err := AcceptDevice("aa:bb:cc:dd:ee:01", "192.168.4.100")
	if err != nil {
		t.Errorf("AcceptDevice stub returned error: %v", err)
	}
}
