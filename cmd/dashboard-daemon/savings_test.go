package main

import (
	"testing"
)

func TestCalcSavings_AdsOnly(t *testing.T) {
	cfg := SavingsConfig{
		AvgAdPayloadBytes:      153600, // 150KB
		AvgTrackerPayloadBytes: 5120,   // 5KB
		OverageRatePerMB:       0.01,
	}
	result := CalcSavings(100, 0, cfg)
	// 100 ads * 150KB = 15,000KB = 15MB (approx 14.648 MB)
	expectedBytes := uint64(100 * 153600)
	if result.EstimatedBytesSaved != expectedBytes {
		t.Errorf("expected %d bytes saved, got %d", expectedBytes, result.EstimatedBytesSaved)
	}
	if result.BlockedQueries != 100 {
		t.Errorf("expected 100 blocked queries, got %d", result.BlockedQueries)
	}
	// 15360000 bytes / 1048576 = 14.6484375 MB * $0.01 = $0.1465
	expectedDollar := float64(expectedBytes) / (1024 * 1024) * 0.01
	if result.DollarAmount < expectedDollar-0.01 || result.DollarAmount > expectedDollar+0.01 {
		t.Errorf("expected ~$%.4f, got $%.4f", expectedDollar, result.DollarAmount)
	}
	if result.FormattedAmount != "$0.15" {
		t.Errorf("expected $0.15, got %s", result.FormattedAmount)
	}
}

func TestCalcSavings_MixedCategories(t *testing.T) {
	cfg := SavingsConfig{
		AvgAdPayloadBytes:      153600,
		AvgTrackerPayloadBytes: 5120,
		OverageRatePerMB:       0.01,
	}
	result := CalcSavings(80, 200, cfg)
	// 80 ads * 150KB = 12,288,000 bytes
	// 200 trackers * 5KB = 1,024,000 bytes
	// Total = 13,312,000 bytes
	expectedBytes := uint64(80*153600 + 200*5120)
	if result.EstimatedBytesSaved != expectedBytes {
		t.Errorf("expected %d bytes saved, got %d", expectedBytes, result.EstimatedBytesSaved)
	}
	if result.BlockedQueries != 280 {
		t.Errorf("expected 280 blocked queries, got %d", result.BlockedQueries)
	}
}

func TestCalcSavings_ZeroBlocked(t *testing.T) {
	cfg := SavingsConfig{
		AvgAdPayloadBytes:      153600,
		AvgTrackerPayloadBytes: 5120,
		OverageRatePerMB:       0.01,
	}
	result := CalcSavings(0, 0, cfg)
	if result.EstimatedBytesSaved != 0 {
		t.Errorf("expected 0 bytes saved, got %d", result.EstimatedBytesSaved)
	}
	if result.DollarAmount != 0 {
		t.Errorf("expected $0.00, got $%.2f", result.DollarAmount)
	}
	if result.FormattedAmount != "$0.00" {
		t.Errorf("expected $0.00, got %s", result.FormattedAmount)
	}
}

func TestCalcSavings_CustomRate(t *testing.T) {
	cfg := SavingsConfig{
		AvgAdPayloadBytes:      153600,
		AvgTrackerPayloadBytes: 5120,
		OverageRatePerMB:       0.02, // doubled rate
	}
	result := CalcSavings(100, 0, cfg)
	expectedBytes := uint64(100 * 153600)
	expectedDollar := float64(expectedBytes) / (1024 * 1024) * 0.02
	if result.DollarAmount < expectedDollar-0.01 || result.DollarAmount > expectedDollar+0.01 {
		t.Errorf("expected ~$%.4f, got $%.4f", expectedDollar, result.DollarAmount)
	}
	// $0.30 approx (14.648 * 0.02 = 0.2930)
	if result.FormattedAmount != "$0.29" {
		t.Errorf("expected $0.29, got %s", result.FormattedAmount)
	}
}
