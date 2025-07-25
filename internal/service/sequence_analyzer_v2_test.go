package service

import (
	"testing"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestSequenceAnalyzerV2_AnalyzeSequences(t *testing.T) {
	analyzer := NewSequenceAnalyzerV2()

	// Create test data
	testSequences := createTestSequences()
	analysis := domain.SequenceAnalysis{
		Stock:     domain.StockUniverse{Symbol: "TESTSTOCK"},
		Sequences: testSequences,
	}

	// Perform analysis
	result := analyzer.AnalyzeSequences(analysis)

	// Verify results
	assert.Equal(t, "TESTSTOCK", result.Stock.Symbol)
	assert.Greater(t, result.SequenceQuality, 0.0)
	assert.GreaterOrEqual(t, result.ContinuityScore, 0.0)
	assert.GreaterOrEqual(t, result.PredictiveScore, 0.0)
	assert.Greater(t, result.MomentumScore, 0.0)
	assert.Greater(t, result.VolumeProfile.AverageVolume, 0.0)
	assert.Greater(t, len(result.PricePatterns), 0)
}

func TestSequenceAnalyzerV2_PatternIdentification(t *testing.T) {
	analyzer := NewSequenceAnalyzerV2()

	// Create sequences with clear patterns
	sequences := []domain.MoveSequence{
		createMambaSequence(3, 100.0, 1000.0, time.Now().AddDate(0, 0, -10)),
		createMambaSequence(3, 95.0, 1100.0, time.Now().AddDate(0, 0, -8)),
		createMambaSequence(4, 110.0, 1200.0, time.Now().AddDate(0, 0, -5)),
		createNonMambaSequence(2, 80.0, 800.0, time.Now().AddDate(0, 0, -2)),
	}

	analysis := domain.SequenceAnalysis{
		Stock:     domain.StockUniverse{Symbol: "PATTERN"},
		Sequences: sequences,
	}

	result := analyzer.AnalyzeSequences(analysis)

	// Should identify patterns
	assert.Greater(t, len(result.PricePatterns), 0)

	// Find Mamba patterns
	var mambaPattern *domain.PricePattern
	for _, pattern := range result.PricePatterns {
		if pattern.Type == string(domain.MambaSequence) {
			mambaPattern = &pattern
			break
		}
	}

	assert.NotNil(t, mambaPattern)
	assert.Greater(t, mambaPattern.Frequency, 0)
	assert.Greater(t, mambaPattern.Strength, 0.0)
}

func TestSequenceAnalyzerV2_VolumeAnalysis(t *testing.T) {
	analyzer := NewSequenceAnalyzerV2()

	// Create sequences with varying volumes
	sequences := []domain.MoveSequence{
		createMambaSequence(3, 100.0, 1000.0, time.Now().AddDate(0, 0, -10)),
		createMambaSequence(3, 100.0, 1500.0, time.Now().AddDate(0, 0, -8)),
		createMambaSequence(3, 100.0, 2000.0, time.Now().AddDate(0, 0, -5)),
	}

	analysis := domain.SequenceAnalysis{
		Stock:     domain.StockUniverse{Symbol: "VOLUME"},
		Sequences: sequences,
	}

	result := analyzer.AnalyzeSequences(analysis)

	// Volume profile should show increasing trend
	assert.Greater(t, result.VolumeProfile.AverageVolume, 0.0)
	assert.Greater(t, result.VolumeProfile.VolumeTrend, 0.0) // Positive trend
	assert.Greater(t, result.VolumeProfile.VolumeConsistency, 0.0)
}

func TestSequenceAnalyzerV2_QualityMetrics(t *testing.T) {
	analyzer := NewSequenceAnalyzerV2()

	// Create high-quality sequences with small gaps for good continuity
	// Each sequence is 5 days long, with 1-day gaps between them
	baseTime := time.Now().AddDate(0, 0, -20)
	sequences := []domain.MoveSequence{
		createMambaSequence(3, 120.0, 1000.0, baseTime),                   // Days 0-2
		createMambaSequence(3, 115.0, 1000.0, baseTime.AddDate(0, 0, 4)),  // Days 4-6 (1 day gap)
		createMambaSequence(3, 110.0, 1000.0, baseTime.AddDate(0, 0, 8)),  // Days 8-10 (1 day gap)
		createMambaSequence(3, 105.0, 1000.0, baseTime.AddDate(0, 0, 12)), // Days 12-14 (1 day gap)
		createMambaSequence(3, 100.0, 1000.0, baseTime.AddDate(0, 0, 16)), // Days 16-18 (1 day gap)
	}

	analysis := domain.SequenceAnalysis{
		Stock:     domain.StockUniverse{Symbol: "QUALITY"},
		Sequences: sequences,
	}

	result := analyzer.AnalyzeSequences(analysis)

	// Debug output to understand what's happening
	t.Logf("SequenceQuality: %.2f", result.SequenceQuality)
	t.Logf("ContinuityScore: %.2f", result.ContinuityScore)
	t.Logf("MomentumScore: %.2f", result.MomentumScore)
	t.Logf("PredictiveScore: %.2f", result.PredictiveScore)

	// Should have high quality scores
	assert.Greater(t, result.SequenceQuality, 0.5)
	assert.Greater(t, result.ContinuityScore, 0.5) // Should be good with 1-day gaps
	assert.Greater(t, result.MomentumScore, 0.5)
	// Predictive score should be high with consistent patterns (all MAMBA sequences)
	assert.Greater(t, result.PredictiveScore, 0.5)
}

func TestSequenceAnalyzerV2_Configuration(t *testing.T) {
	analyzer := NewSequenceAnalyzerV2()

	// Test configuration
	analyzer.SetMinSequenceLength(3)
	analyzer.SetMaxGapLength(10)

	assert.Equal(t, 3, analyzer.minSequenceLength)
	assert.Equal(t, 10, analyzer.maxGapLength)
}

// Benchmark tests for performance comparison
func BenchmarkSequenceAnalyzerV2_AnalyzeSequences(b *testing.B) {
	analyzer := NewSequenceAnalyzerV2()
	sequences := createLargeTestSequenceSet(1000) // 1000 sequences
	analysis := domain.SequenceAnalysis{
		Stock:     domain.StockUniverse{Symbol: "BENCHMARK"},
		Sequences: sequences,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeSequences(analysis)
	}
}

func BenchmarkSequenceAnalyzerV2_PatternExtraction(b *testing.B) {
	analyzer := NewSequenceAnalyzerV2()
	sequences := createLargeTestSequenceSet(500)
	analysis := domain.SequenceAnalysis{
		Stock:     domain.StockUniverse{Symbol: "PATTERN_BENCH"},
		Sequences: sequences,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := analyzer.AnalyzeSequences(analysis)
		_ = result.PricePatterns // Access patterns to ensure computation
	}
}

// Helper functions for creating test data
func createTestSequences() []domain.MoveSequence {
	return []domain.MoveSequence{
		createMambaSequence(3, 100.0, 1000.0, time.Now().AddDate(0, 0, -10)),
		createMambaSequence(4, 105.0, 1200.0, time.Now().AddDate(0, 0, -7)),
		createNonMambaSequence(2, 90.0, 800.0, time.Now().AddDate(0, 0, -5)),
		createMambaSequence(3, 110.0, 1100.0, time.Now().AddDate(0, 0, -2)),
	}
}

func createMambaSequence(length int, strength, volume float64, startDate time.Time) domain.MoveSequence {
	moves := make([]domain.DailyMove, length)
	for i := 0; i < length; i++ {
		moves[i] = domain.DailyMove{
			Date:       startDate.AddDate(0, 0, i),
			OpenPrice:  100.0 + float64(i),
			ClosePrice: 102.0 + float64(i),
			Volume:     volume,
		}
	}

	return domain.MoveSequence{
		Type:      domain.MambaSequence,
		Length:    length,
		StartDate: startDate,
		EndDate:   startDate.AddDate(0, 0, length-1),
		Moves:     moves,
		Strength:  strength,
	}
}

func createNonMambaSequence(length int, strength, volume float64, startDate time.Time) domain.MoveSequence {
	moves := make([]domain.DailyMove, length)
	for i := 0; i < length; i++ {
		moves[i] = domain.DailyMove{
			Date:       startDate.AddDate(0, 0, i),
			OpenPrice:  100.0 - float64(i)*0.5,
			ClosePrice: 99.0 - float64(i)*0.5,
			Volume:     volume,
		}
	}

	return domain.MoveSequence{
		Type:      domain.NonMambaSequence,
		Length:    length,
		StartDate: startDate,
		EndDate:   startDate.AddDate(0, 0, length-1),
		Moves:     moves,
		Strength:  strength,
	}
}

func createLargeTestSequenceSet(count int) []domain.MoveSequence {
	sequences := make([]domain.MoveSequence, count)
	baseDate := time.Now().AddDate(0, 0, -count)

	for i := 0; i < count; i++ {
		seqType := domain.MambaSequence
		if i%3 == 0 {
			seqType = domain.NonMambaSequence
		}

		length := 3 + (i % 5)                    // Length between 3-7
		strength := 80.0 + float64(i%40)         // Strength between 80-120
		volume := 1000.0 + float64(i%500)        // Volume between 1000-1500
		startDate := baseDate.AddDate(0, 0, i*2) // Sequences every 2 days

		if seqType == domain.MambaSequence {
			sequences[i] = createMambaSequence(length, strength, volume, startDate)
		} else {
			sequences[i] = createNonMambaSequence(length, strength, volume, startDate)
		}
	}

	return sequences
}
