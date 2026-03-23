package main

import (
	"reflect"
	"testing"
)

func TestFormatAddRule(t *testing.T) {
	cfg := FallbackConfig{
		Fwmark:   "0x2",
		Table:    200,
		Priority: 200,
	}
	got := FormatAddRule(cfg)
	want := []string{"rule", "add", "fwmark", "0x2", "table", "200", "priority", "200"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FormatAddRule() = %v, want %v", got, want)
	}
}

func TestFormatDelRule(t *testing.T) {
	cfg := FallbackConfig{
		Fwmark:   "0x2",
		Table:    200,
		Priority: 200,
	}
	got := FormatDelRule(cfg)
	want := []string{"rule", "del", "fwmark", "0x2", "table", "200", "priority", "200"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FormatDelRule() = %v, want %v", got, want)
	}
}
