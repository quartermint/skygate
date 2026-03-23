package main

import "fmt"

// SavingsConfig holds the heuristic parameters for estimating bandwidth savings
// from DNS blocking. Values per RESEARCH.md Open Question 3 (conservative estimates).
type SavingsConfig struct {
	AvgAdPayloadBytes      uint64  // default: 153600 (150KB per D-15)
	AvgTrackerPayloadBytes uint64  // default: 5120 (5KB per D-15)
	OverageRatePerMB       float64 // from settings, default: 0.01 per D-16
}

// DefaultSavingsConfig returns the default savings configuration using
// conservative payload size estimates.
func DefaultSavingsConfig() SavingsConfig {
	return SavingsConfig{
		AvgAdPayloadBytes:      153600, // 150KB avg ad payload
		AvgTrackerPayloadBytes: 5120,   // 5KB avg tracker payload
		OverageRatePerMB:       0.01,   // $0.01/MB baseline overage rate
	}
}

// SavingsResult holds the calculated bandwidth savings from DNS blocking.
type SavingsResult struct {
	BlockedQueries      uint64  `json:"blocked_queries"`
	EstimatedBytesSaved uint64  `json:"estimated_bytes_saved"`
	DollarAmount        float64 `json:"dollar_amount"`
	FormattedAmount     string  `json:"formatted_amount"` // "$X.XX" per D-17
}

// CalcSavings estimates bandwidth savings from blocked ad and tracker DNS queries.
// Calculation: bytesSaved = (blockedAds * AvgAdPayloadBytes) + (blockedTrackers * AvgTrackerPayloadBytes)
// Dollar: dollarAmount = float64(bytesSaved) / (1024 * 1024) * OverageRatePerMB
func CalcSavings(blockedAds, blockedTrackers uint64, cfg SavingsConfig) SavingsResult {
	bytesSaved := (blockedAds * cfg.AvgAdPayloadBytes) + (blockedTrackers * cfg.AvgTrackerPayloadBytes)
	dollarAmount := float64(bytesSaved) / (1024 * 1024) * cfg.OverageRatePerMB

	return SavingsResult{
		BlockedQueries:      blockedAds + blockedTrackers,
		EstimatedBytesSaved: bytesSaved,
		DollarAmount:        dollarAmount,
		FormattedAmount:     fmt.Sprintf("$%.2f", dollarAmount),
	}
}
