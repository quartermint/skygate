package main

import "fmt"

// FallbackConfig holds the routing fallback parameters.
type FallbackConfig struct {
	Fwmark   string // "0x2"
	Table    int    // 200
	Priority int    // 200
}

// FormatAddRule returns args for `ip rule add fwmark 0x2 table 200 priority 200`.
func FormatAddRule(cfg FallbackConfig) []string {
	return []string{
		"rule", "add",
		"fwmark", cfg.Fwmark,
		"table", fmt.Sprintf("%d", cfg.Table),
		"priority", fmt.Sprintf("%d", cfg.Priority),
	}
}

// FormatDelRule returns args for `ip rule del fwmark 0x2 table 200 priority 200`.
func FormatDelRule(cfg FallbackConfig) []string {
	return []string{
		"rule", "del",
		"fwmark", cfg.Fwmark,
		"table", fmt.Sprintf("%d", cfg.Table),
		"priority", fmt.Sprintf("%d", cfg.Priority),
	}
}
