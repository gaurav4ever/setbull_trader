package service

import (
	"setbull_trader/internal/analytics/sequence"
	"setbull_trader/internal/domain"
)

// SequenceAnalyzerV2 provides DataFrame-based sequence analysis
type SequenceAnalyzerV2 struct {
	minSequenceLength int
	maxGapLength      int
}

// NewSequenceAnalyzerV2 creates a new DataFrame-powered sequence analyzer
func NewSequenceAnalyzerV2() *SequenceAnalyzerV2 {
	return &SequenceAnalyzerV2{
		minSequenceLength: 2,
		maxGapLength:      5,
	}
}

// AnalyzeSequences performs comprehensive sequence analysis using DataFrame operations
func (sa *SequenceAnalyzerV2) AnalyzeSequences(analysis domain.SequenceAnalysis) domain.SequenceMetrics {
	// Create DataFrame-based analyzer
	sequenceDF := sequence.NewSequenceDataFrame(analysis.Sequences, analysis.Stock)

	// Perform pattern analysis
	result := sequenceDF.AnalyzePatterns()

	// Convert to domain model
	return domain.SequenceMetrics{
		Stock:           analysis.Stock,
		SequenceQuality: result.QualityMetrics.SequenceQuality,
		ContinuityScore: result.QualityMetrics.ContinuityScore,
		PredictiveScore: result.QualityMetrics.PredictiveScore,
		MomentumScore:   result.QualityMetrics.MomentumScore,
		VolumeProfile:   result.VolumeProfile,
		DominantPattern: result.DominantPattern,
		PricePatterns:   convertToPricePatterns(result.Patterns),
	}
}

// SetMinSequenceLength configures the minimum sequence length for analysis
func (sa *SequenceAnalyzerV2) SetMinSequenceLength(length int) {
	sa.minSequenceLength = length
}

// SetMaxGapLength configures the maximum gap length between sequences
func (sa *SequenceAnalyzerV2) SetMaxGapLength(length int) {
	sa.maxGapLength = length
}

// convertToPricePatterns converts sequence patterns to price patterns
func convertToPricePatterns(patterns []domain.SequencePattern) []domain.PricePattern {
	pricePatterns := make([]domain.PricePattern, len(patterns))

	for i, pattern := range patterns {
		pricePatterns[i] = domain.PricePattern{
			Type:          string(pattern.Type),
			Length:        pattern.Length,
			Frequency:     pattern.Frequency,
			AverageGap:    pattern.AverageGap,
			SuccessRate:   pattern.SuccessRate,
			AverageReturn: pattern.AverageReturn,
			Strength:      pattern.Strength,
		}
	}

	return pricePatterns
}
