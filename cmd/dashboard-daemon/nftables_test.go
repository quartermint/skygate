package main

import (
	"testing"
)

// Sample nft -j output with 2 MAC entries and counters.
const sampleNftJSON = `{
	"nftables": [
		{"metainfo": {"json_schema_version": 1}},
		{"set": {"family": "inet", "name": "device_counters", "table": "skygate", "type": "ether_addr", "flags": ["dynamic"]}},
		{"elem": {"val": "aa:bb:cc:dd:ee:01", "counter": {"packets": 100, "bytes": 1048576}}},
		{"elem": {"val": "aa:bb:cc:dd:ee:02", "counter": {"packets": 50, "bytes": 524288}}}
	]
}`

// Sample nft -j output with 3 MAC entries.
const sampleNftJSON3 = `{
	"nftables": [
		{"metainfo": {"json_schema_version": 1}},
		{"set": {"family": "inet", "name": "device_counters", "table": "skygate"}},
		{"elem": {"val": "aa:bb:cc:dd:ee:01", "counter": {"packets": 100, "bytes": 1048576}}},
		{"elem": {"val": "aa:bb:cc:dd:ee:02", "counter": {"packets": 50, "bytes": 524288}}},
		{"elem": {"val": "aa:bb:cc:dd:ee:03", "counter": {"packets": 25, "bytes": 262144}}}
	]
}`

func TestParseNftCounters_Valid(t *testing.T) {
	counters, err := ParseNftCounters([]byte(sampleNftJSON3))
	if err != nil {
		t.Fatalf("ParseNftCounters failed: %v", err)
	}
	if len(counters) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(counters))
	}
	if counters["aa:bb:cc:dd:ee:01"] != 1048576 {
		t.Errorf("expected 1048576 for :01, got %d", counters["aa:bb:cc:dd:ee:01"])
	}
	if counters["aa:bb:cc:dd:ee:02"] != 524288 {
		t.Errorf("expected 524288 for :02, got %d", counters["aa:bb:cc:dd:ee:02"])
	}
	if counters["aa:bb:cc:dd:ee:03"] != 262144 {
		t.Errorf("expected 262144 for :03, got %d", counters["aa:bb:cc:dd:ee:03"])
	}
}

func TestParseNftCounters_Empty(t *testing.T) {
	emptyJSON := `{"nftables": [{"metainfo": {"json_schema_version": 1}}, {"set": {"family": "inet", "name": "device_counters", "table": "skygate"}}]}`
	counters, err := ParseNftCounters([]byte(emptyJSON))
	if err != nil {
		t.Fatalf("ParseNftCounters failed: %v", err)
	}
	if len(counters) != 0 {
		t.Errorf("expected 0 entries for empty set, got %d", len(counters))
	}
}

func TestParseNftCounters_InvalidJSON(t *testing.T) {
	_, err := ParseNftCounters([]byte("not json at all"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestComputeDeltas(t *testing.T) {
	prev := map[string]uint64{
		"aa:bb:cc:dd:ee:01": 1000,
	}
	curr := map[string]uint64{
		"aa:bb:cc:dd:ee:01": 1500,
	}
	deltas := ComputeDeltas(prev, curr)
	if deltas["aa:bb:cc:dd:ee:01"] != 500 {
		t.Errorf("expected delta 500, got %d", deltas["aa:bb:cc:dd:ee:01"])
	}
}

func TestComputeDeltas_NewDevice(t *testing.T) {
	prev := map[string]uint64{}
	curr := map[string]uint64{
		"aa:bb:cc:dd:ee:01": 2000,
	}
	deltas := ComputeDeltas(prev, curr)
	if deltas["aa:bb:cc:dd:ee:01"] != 2000 {
		t.Errorf("expected delta 2000 for new device, got %d", deltas["aa:bb:cc:dd:ee:01"])
	}
}

func TestComputeDeltas_CounterReset(t *testing.T) {
	prev := map[string]uint64{
		"aa:bb:cc:dd:ee:01": 5000,
	}
	curr := map[string]uint64{
		"aa:bb:cc:dd:ee:01": 100, // counter reset (reboot)
	}
	deltas := ComputeDeltas(prev, curr)
	// On counter reset, use current as delta (fresh start).
	if deltas["aa:bb:cc:dd:ee:01"] != 100 {
		t.Errorf("expected delta 100 on counter reset, got %d", deltas["aa:bb:cc:dd:ee:01"])
	}
}

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
