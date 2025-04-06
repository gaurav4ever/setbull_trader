package service

import (
	"fmt"
	"math"
	"setbull_trader/internal/domain"
	"sort"
)

type SequenceAnalyzer struct {
	minSequenceLength int
	maxGapLength      int
}

type AdvancedAnalysis struct {
	Patterns        []domain.SequencePattern
	DominantPattern domain.SequencePattern
	SequenceQuality float64
	ContinuityScore float64
	PredictiveScore float64
	MomentumScore   float64
	VolumeProfile   VolumeProfile
}

type VolumeProfile struct {
	AverageVolume     float64
	VolumeStrength    float64
	VolumeTrend       float64
	VolumeConsistency float64
}

func NewSequenceAnalyzer() *SequenceAnalyzer {
	return &SequenceAnalyzer{
		minSequenceLength: 2,
		maxGapLength:      5,
	}
}

func (sa *SequenceAnalyzer) AnalyzeSequences(analysis domain.SequenceAnalysis) domain.SequenceMetrics {
	advancedAnalysis := AdvancedAnalysis{}

	// Analyze patterns
	advancedAnalysis.Patterns = sa.identifyPatterns(analysis.Sequences)

	// Find dominant pattern
	if len(advancedAnalysis.Patterns) > 0 {
		advancedAnalysis.DominantPattern = sa.findDominantPattern(advancedAnalysis.Patterns)
	}

	// Calculate sequence quality metrics
	advancedAnalysis.SequenceQuality = sa.calculateSequenceQuality(analysis)
	advancedAnalysis.ContinuityScore = sa.calculateContinuityScore(analysis.Sequences)
	advancedAnalysis.PredictiveScore = sa.calculatePredictiveScore(analysis.Sequences)
	advancedAnalysis.MomentumScore = sa.calculateMomentumScore(analysis.Sequences)
	advancedAnalysis.VolumeProfile = sa.analyzeVolumeProfile(analysis.Sequences)

	return domain.SequenceMetrics{
		Stock:           analysis.Stock,
		SequenceQuality: advancedAnalysis.SequenceQuality,
		ContinuityScore: advancedAnalysis.ContinuityScore,
		PredictiveScore: advancedAnalysis.PredictiveScore,
		MomentumScore:   advancedAnalysis.MomentumScore,
	}
}

func (sa *SequenceAnalyzer) identifyPatterns(sequences []domain.MoveSequence) []domain.SequencePattern {
	patterns := make(map[string]domain.SequencePattern)

	// Group sequences by length and type
	for i, seq := range sequences {
		if seq.Length < sa.minSequenceLength {
			continue
		}

		key := fmt.Sprintf("%s-%d", seq.Type, seq.Length)
		pattern, exists := patterns[key]
		if !exists {
			pattern = domain.SequencePattern{
				Type:   seq.Type,
				Length: seq.Length,
			}
		}

		pattern.Frequency++
		pattern.Strength += seq.Strength

		// Calculate return
		if len(seq.Moves) > 0 {
			startPrice := seq.Moves[0].OpenPrice
			endPrice := seq.Moves[len(seq.Moves)-1].ClosePrice
			returnPct := ((endPrice - startPrice) / startPrice) * 100
			pattern.AverageReturn = (pattern.AverageReturn*float64(pattern.Frequency-1) +
				returnPct) / float64(pattern.Frequency)
		}

		// Calculate gap to next sequence
		if i < len(sequences)-1 {
			gap := sequences[i+1].StartDate.Sub(seq.EndDate).Hours() / 24
			pattern.AverageGap = (pattern.AverageGap*float64(pattern.Frequency-1) +
				gap) / float64(pattern.Frequency)
		}

		patterns[key] = pattern
	}

	// Convert map to slice and calculate success rates
	result := make([]domain.SequencePattern, 0, len(patterns))
	for _, pattern := range patterns {
		pattern.Strength /= float64(pattern.Frequency)
		pattern.SuccessRate = sa.calculatePatternSuccessRate(sequences, pattern)
		result = append(result, pattern)
	}

	// Sort by frequency and strength
	sort.Slice(result, func(i, j int) bool {
		if result[i].Frequency == result[j].Frequency {
			return result[i].Strength > result[j].Strength
		}
		return result[i].Frequency > result[j].Frequency
	})

	return result
}

func (sa *SequenceAnalyzer) findDominantPattern(patterns []domain.SequencePattern) domain.SequencePattern {
	var dominant domain.SequencePattern
	maxScore := 0.0

	for _, pattern := range patterns {
		score := float64(pattern.Frequency) * pattern.Strength * pattern.SuccessRate
		if score > maxScore {
			maxScore = score
			dominant = pattern
		}
	}

	return dominant
}

func (sa *SequenceAnalyzer) calculateSequenceQuality(
	analysis domain.SequenceAnalysis) float64 {

	if len(analysis.Sequences) == 0 {
		return 0
	}

	// Factors for quality calculation
	lengthScore := sa.calculateLengthScore(analysis.Sequences)
	strengthScore := sa.calculateStrengthScore(analysis.Sequences)
	consistencyScore := sa.calculateConsistencyScore(analysis.Sequences)

	// Weighted average of factors
	return (lengthScore*0.3 + strengthScore*0.4 + consistencyScore*0.3)
}

func (sa *SequenceAnalyzer) calculateContinuityScore(
	sequences []domain.MoveSequence) float64 {

	if len(sequences) < 2 {
		return 0
	}

	totalGaps := 0.0
	gapCount := 0

	for i := 0; i < len(sequences)-1; i++ {
		gap := sequences[i+1].StartDate.Sub(sequences[i].EndDate).Hours() / 24
		if gap <= float64(sa.maxGapLength) {
			totalGaps += gap
			gapCount++
		}
	}

	if gapCount == 0 {
		return 0
	}

	avgGap := totalGaps / float64(gapCount)
	// Convert average gap to a 0-1 score (lower gaps = higher score)
	return math.Max(0, 1-(avgGap/float64(sa.maxGapLength)))
}

func (sa *SequenceAnalyzer) calculatePredictiveScore(
	sequences []domain.MoveSequence) float64 {

	if len(sequences) < 3 {
		return 0
	}

	correctPredictions := 0
	totalPredictions := 0

	for i := 0; i < len(sequences)-2; i++ {
		// Look for pattern repetition
		if sequences[i].Type == sequences[i+1].Type {
			totalPredictions++
			if sequences[i+1].Type == sequences[i+2].Type {
				correctPredictions++
			}
		}
	}

	if totalPredictions == 0 {
		return 0
	}

	return float64(correctPredictions) / float64(totalPredictions)
}

func (sa *SequenceAnalyzer) calculateMomentumScore(
	sequences []domain.MoveSequence) float64 {

	if len(sequences) == 0 {
		return 0
	}

	// Focus on recent sequences (last 5 or fewer)
	recentCount := min(5, len(sequences))
	recentSequences := sequences[len(sequences)-recentCount:]

	totalStrength := 0.0
	weightedSum := 0.0
	weightSum := 0.0

	for i, seq := range recentSequences {
		weight := float64(i + 1) // More recent sequences get higher weight
		weightedSum += seq.Strength * weight
		weightSum += weight
		totalStrength += seq.Strength
	}

	// Combine recent weighted strength with overall trend
	averageStrength := totalStrength / float64(len(recentSequences))
	weightedStrength := weightedSum / weightSum

	return (weightedStrength*0.7 + averageStrength*0.3)
}

func (sa *SequenceAnalyzer) analyzeVolumeProfile(
	sequences []domain.MoveSequence) VolumeProfile {

	if len(sequences) == 0 {
		return VolumeProfile{}
	}

	var totalVolume, volumeStrength float64
	volumes := make([]float64, 0)

	// Calculate average volume and collect volume data
	for _, seq := range sequences {
		for _, move := range seq.Moves {
			volumes = append(volumes, move.Volume)
			totalVolume += move.Volume
		}
	}

	avgVolume := totalVolume / float64(len(volumes))

	// Calculate volume trend and consistency
	volumeTrend := sa.calculateVolumeTrend(volumes)
	volumeConsistency := sa.calculateVolumeConsistency(volumes, avgVolume)

	// Calculate volume strength relative to sequence type
	for _, seq := range sequences {
		if seq.Type == domain.MambaSequence {
			seqVol := calculateAverageSequenceVolume(seq)
			volumeStrength += seqVol / avgVolume
		}
	}
	volumeStrength /= float64(len(sequences))

	return VolumeProfile{
		AverageVolume:     avgVolume,
		VolumeStrength:    volumeStrength,
		VolumeTrend:       volumeTrend,
		VolumeConsistency: volumeConsistency,
	}
}

// Helper functions
func (sa *SequenceAnalyzer) calculateLengthScore(
	sequences []domain.MoveSequence) float64 {

	if len(sequences) == 0 {
		return 0
	}

	totalScore := 0.0
	for _, seq := range sequences {
		if seq.Type == domain.MambaSequence {
			// Score increases with length but caps at reasonable value
			lengthScore := math.Min(float64(seq.Length)/5.0, 1.0)
			totalScore += lengthScore
		}
	}

	return totalScore / float64(len(sequences))
}

func (sa *SequenceAnalyzer) calculateStrengthScore(
	sequences []domain.MoveSequence) float64 {

	if len(sequences) == 0 {
		return 0
	}

	totalStrength := 0.0
	mambaCount := 0

	for _, seq := range sequences {
		if seq.Type == domain.MambaSequence {
			totalStrength += seq.Strength
			mambaCount++
		}
	}

	if mambaCount == 0 {
		return 0
	}

	return totalStrength / float64(mambaCount)
}

func (sa *SequenceAnalyzer) calculateConsistencyScore(
	sequences []domain.MoveSequence) float64 {

	if len(sequences) < 2 {
		return 0
	}

	// Calculate variance in sequence lengths
	lengths := make([]float64, 0)
	for _, seq := range sequences {
		if seq.Type == domain.MambaSequence {
			lengths = append(lengths, float64(seq.Length))
		}
	}

	if len(lengths) < 2 {
		return 0
	}

	variance := calculateVariance(lengths)
	// Convert variance to a 0-1 score (lower variance = higher consistency)
	return math.Max(0, 1-math.Min(variance/10.0, 1.0))
}

func (sa *SequenceAnalyzer) calculatePatternSuccessRate(
	sequences []domain.MoveSequence, pattern domain.SequencePattern) float64 {

	successCount := 0
	totalCount := 0

	for i := 0; i < len(sequences)-1; i++ {
		if sequences[i].Type == pattern.Type && sequences[i].Length == pattern.Length {
			totalCount++
			// Consider it successful if next sequence continues the trend
			if i < len(sequences)-1 && sequences[i+1].Type == pattern.Type {
				successCount++
			}
		}
	}

	if totalCount == 0 {
		return 0
	}

	return float64(successCount) / float64(totalCount)
}

func (sa *SequenceAnalyzer) calculateVolumeTrend(volumes []float64) float64 {
	if len(volumes) < 2 {
		return 0
	}

	// Simple linear regression on volume
	n := float64(len(volumes))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, volume := range volumes {
		x := float64(i)
		sumX += x
		sumY += volume
		sumXY += x * volume
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	// Normalize slope to -1 to 1 range
	return math.Tanh(slope / 1000000) // Adjust scaling factor as needed
}

func (sa *SequenceAnalyzer) calculateVolumeConsistency(
	volumes []float64, avgVolume float64) float64 {

	if len(volumes) < 2 {
		return 0
	}

	// Calculate coefficient of variation
	variance := calculateVariance(volumes)
	stdDev := math.Sqrt(variance)
	cv := stdDev / avgVolume

	// Convert to a 0-1 score (lower CV = higher consistency)
	return math.Max(0, 1-math.Min(cv, 1.0))
}

func calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	return sumSquares / float64(len(values)-1)
}

func calculateAverageSequenceVolume(seq domain.MoveSequence) float64 {
	if len(seq.Moves) == 0 {
		return 0
	}

	total := 0.0
	for _, move := range seq.Moves {
		total += move.Volume
	}
	return total / float64(len(seq.Moves))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
