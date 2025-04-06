package service

import (
	"setbull_trader/internal/domain"
)

type PatternAnalyzer struct {
	minPatternLength int
}

func NewPatternAnalyzer(minLength int) *PatternAnalyzer {
	return &PatternAnalyzer{
		minPatternLength: minLength,
	}
}

func (p *PatternAnalyzer) AnalyzePatterns(analysis domain.SequenceAnalysis) []domain.SequencePattern {
	patterns := make([]domain.SequencePattern, 0)

	// Group sequences by length
	lengthFrequency := make(map[int]int)
	for _, length := range analysis.MambaSequences {
		if length >= p.minPatternLength {
			lengthFrequency[length]++
		}
	}

	// Calculate average gaps between sequences of same length
	for length, freq := range lengthFrequency {
		avgGap := p.calculateAverageGap(analysis, length)
		successRate := p.calculateSuccessRate(analysis, length)

		patterns = append(patterns, domain.SequencePattern{
			Length:      length,
			Frequency:   freq,
			AverageGap:  avgGap,
			SuccessRate: successRate,
		})
	}

	return patterns
}

func (p *PatternAnalyzer) calculateAverageGap(analysis domain.SequenceAnalysis,
	sequenceLength int) float64 {
	// Implementation of gap calculation between sequences of same length
	// This is a simplified version - you might want to enhance this based on your needs
	totalGap := 0
	gapCount := 0

	for i := 0; i < len(analysis.NonMambaSequences); i++ {
		if i > 0 && analysis.MambaSequences[i] == sequenceLength &&
			analysis.MambaSequences[i-1] == sequenceLength {
			totalGap += analysis.NonMambaSequences[i]
			gapCount++
		}
	}

	if gapCount == 0 {
		return 0
	}
	return float64(totalGap) / float64(gapCount)
}

func (p *PatternAnalyzer) calculateSuccessRate(analysis domain.SequenceAnalysis,
	sequenceLength int) float64 {
	// Implementation of success rate calculation for sequences of given length
	// This could be based on price movement after sequence completion
	// Simplified version for now
	successCount := 0
	totalCount := 0

	for _, length := range analysis.MambaSequences {
		if length == sequenceLength {
			totalCount++
			// Here you could add logic to determine if the sequence led to a successful outcome
			// For now, we'll consider all sequences of this length as successful
			successCount++
		}
	}

	if totalCount == 0 {
		return 0
	}
	return float64(successCount) / float64(totalCount)
}
