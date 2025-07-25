package main

import (
	"fmt"
	"time"

	"setbull_trader/internal/analytics/sequence"
	"setbull_trader/internal/domain"
)

func main() {
	// Test the DataFrame creation and basic operations
	fmt.Println("Testing DataFrame-based sequence analyzer...")

	// Create test sequences
	sequences := []domain.MoveSequence{
		{
			Type:      domain.MambaSequence,
			Length:    3,
			StartDate: time.Now().AddDate(0, 0, -10),
			EndDate:   time.Now().AddDate(0, 0, -8),
			Strength:  100.0,
			Moves: []domain.DailyMove{
				{Date: time.Now().AddDate(0, 0, -10), OpenPrice: 100.0, ClosePrice: 102.0, Volume: 1000.0},
				{Date: time.Now().AddDate(0, 0, -9), OpenPrice: 102.0, ClosePrice: 104.0, Volume: 1100.0},
				{Date: time.Now().AddDate(0, 0, -8), OpenPrice: 104.0, ClosePrice: 106.0, Volume: 1200.0},
			},
		},
		{
			Type:      domain.MambaSequence,
			Length:    4,
			StartDate: time.Now().AddDate(0, 0, -7),
			EndDate:   time.Now().AddDate(0, 0, -4),
			Strength:  105.0,
			Moves: []domain.DailyMove{
				{Date: time.Now().AddDate(0, 0, -7), OpenPrice: 106.0, ClosePrice: 108.0, Volume: 1300.0},
				{Date: time.Now().AddDate(0, 0, -6), OpenPrice: 108.0, ClosePrice: 110.0, Volume: 1400.0},
				{Date: time.Now().AddDate(0, 0, -5), OpenPrice: 110.0, ClosePrice: 112.0, Volume: 1500.0},
				{Date: time.Now().AddDate(0, 0, -4), OpenPrice: 112.0, ClosePrice: 114.0, Volume: 1600.0},
			},
		},
	}

	stock := domain.StockUniverse{Symbol: "TESTSTOCK"}

	// Create sequence DataFrame
	fmt.Printf("Creating DataFrame with %d sequences...\n", len(sequences))
	sequenceDF := sequence.NewSequenceDataFrame(sequences, stock)

	// Debug DataFrame content
	fmt.Printf("DataFrame rows: %d\n", sequenceDF.Nrow())
	if sequenceDF.Nrow() > 0 {
		fmt.Printf("DataFrame columns: %v\n", sequenceDF.Names())
		fmt.Printf("DataFrame preview:\n%s\n", sequenceDF.String())
	} else {
		fmt.Println("DataFrame is empty!")
	}

	// Analyze patterns
	fmt.Println("Analyzing patterns...")
	result := sequenceDF.AnalyzePatterns()

	// Print results
	fmt.Printf("Patterns found: %d\n", len(result.Patterns))
	for i, pattern := range result.Patterns {
		fmt.Printf("Pattern %d: Type=%s, Length=%d, Frequency=%d, Strength=%.2f\n",
			i+1, pattern.Type, pattern.Length, pattern.Frequency, pattern.Strength)
	}

	fmt.Printf("Quality Metrics:\n")
	fmt.Printf("  Sequence Quality: %.2f\n", result.QualityMetrics.SequenceQuality)
	fmt.Printf("  Continuity Score: %.2f\n", result.QualityMetrics.ContinuityScore)
	fmt.Printf("  Predictive Score: %.2f\n", result.QualityMetrics.PredictiveScore)
	fmt.Printf("  Momentum Score: %.2f\n", result.QualityMetrics.MomentumScore)

	fmt.Printf("Volume Profile:\n")
	fmt.Printf("  Average Volume: %.2f\n", result.VolumeProfile.AverageVolume)
	fmt.Printf("  Volume Strength: %.2f\n", result.VolumeProfile.VolumeStrength)
	fmt.Printf("  Volume Trend: %.2f\n", result.VolumeProfile.VolumeTrend)
	fmt.Printf("  Volume Consistency: %.2f\n", result.VolumeProfile.VolumeConsistency)
}
