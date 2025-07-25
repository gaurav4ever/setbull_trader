package sequence

import (
	"fmt"
	"math"
	"time"

	"setbull_trader/internal/domain"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"gonum.org/v1/gonum/stat"
)

// SequenceDataFrame wraps sequence data in a DataFrame for efficient processing
type SequenceDataFrame struct {
	df       dataframe.DataFrame
	metadata *SequenceMetadata
}

type SequenceMetadata struct {
	Stock           domain.StockUniverse
	MinSequenceLen  int
	MaxGapLength    int
	AnalysisStarted time.Time
}

// AnalysisResult contains the computed sequence analysis results
type AnalysisResult struct {
	Patterns        []domain.SequencePattern
	DominantPattern domain.SequencePattern
	QualityMetrics  QualityMetrics
	VolumeProfile   domain.VolumeProfile
}

type QualityMetrics struct {
	SequenceQuality float64
	ContinuityScore float64
	PredictiveScore float64
	MomentumScore   float64
}

// NewSequenceDataFrame creates a new sequence analyzer with DataFrame processing
func NewSequenceDataFrame(sequences []domain.MoveSequence, stock domain.StockUniverse) *SequenceDataFrame {
	if len(sequences) == 0 {
		// Return empty DataFrame
		return &SequenceDataFrame{
			df: dataframe.New(),
			metadata: &SequenceMetadata{
				Stock:           stock,
				MinSequenceLen:  2,
				MaxGapLength:    5,
				AnalysisStarted: time.Now(),
			},
		}
	}

	// Create DataFrame manually to avoid LoadStructs issues
	indices := make([]int, len(sequences))
	types := make([]string, len(sequences))
	lengths := make([]int, len(sequences))
	strengths := make([]float64, len(sequences))
	volumes := make([]float64, len(sequences))
	returns := make([]float64, len(sequences))
	gaps := make([]float64, len(sequences))

	for i, seq := range sequences {
		indices[i] = i
		types[i] = string(seq.Type)
		lengths[i] = seq.Length
		strengths[i] = seq.Strength
		volumes[i] = calculateSequenceVolume(seq)
		returns[i] = calculateSequenceReturn(seq)
		gaps[i] = 0 // Will be calculated later
	}

	// Calculate gaps between sequences
	for i := 0; i < len(gaps)-1; i++ {
		gap := sequences[i+1].StartDate.Sub(sequences[i].EndDate).Hours() / 24
		gaps[i] = gap
	}

	// Create DataFrame from series
	df := dataframe.New(
		series.New(indices, series.Int, "index"),
		series.New(types, series.String, "type"),
		series.New(lengths, series.Int, "length"),
		series.New(strengths, series.Float, "strength"),
		series.New(volumes, series.Float, "volume"),
		series.New(returns, series.Float, "return"),
		series.New(gaps, series.Float, "gap_to_next"),
	)

	return &SequenceDataFrame{
		df: df,
		metadata: &SequenceMetadata{
			Stock:           stock,
			MinSequenceLen:  2,
			MaxGapLength:    5,
			AnalysisStarted: time.Now(),
		},
	}
}

// SequenceRecord represents a single sequence record for DataFrame processing
type SequenceRecord struct {
	Index     int       `json:"index"`
	Type      string    `json:"type"`
	Length    int       `json:"length"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Strength  float64   `json:"strength"`
	Volume    float64   `json:"volume"`
	Return    float64   `json:"return"`
	GapToNext float64   `json:"gap_to_next"`
}

// AnalyzePatterns performs comprehensive pattern analysis using DataFrame operations
func (sdf *SequenceDataFrame) AnalyzePatterns() *AnalysisResult {
	result := &AnalysisResult{}

	// Check if DataFrame is empty
	if sdf.df.Nrow() == 0 {
		return result
	}

	// Filter for minimum sequence length
	filtered := sdf.df.Filter(
		dataframe.F{
			Colname:    "length",
			Comparator: ">=",
			Comparando: sdf.metadata.MinSequenceLen,
		},
	)

	// If no sequences meet minimum length, return empty result
	if filtered.Nrow() == 0 {
		return result
	}

	// Group by type and length to identify patterns (filter first)
	patterns := sdf.extractPatterns(filtered)
	result.Patterns = patterns

	// Find dominant pattern
	if len(patterns) > 0 {
		result.DominantPattern = sdf.findDominantPattern(patterns)
	}

	// Calculate quality metrics
	result.QualityMetrics = sdf.calculateQualityMetrics()

	// Analyze volume profile
	result.VolumeProfile = sdf.analyzeVolumeProfile()

	return result
}

// extractPatterns extracts sequence patterns from DataFrame
func (sdf *SequenceDataFrame) extractPatterns(filtered dataframe.DataFrame) []domain.SequencePattern {
	patterns := make([]domain.SequencePattern, 0)

	// Check if filtered data is empty
	if filtered.Nrow() == 0 {
		return patterns
	}

	// Extract data using safe accessors
	patternMap := make(map[string]*domain.SequencePattern)

	// Get column data safely
	types := filtered.Col("type").Records()
	lengths := filtered.Col("length").Records()
	strengths := filtered.Col("strength").Float()
	returns := filtered.Col("return").Float()
	gaps := filtered.Col("gap_to_next").Float()

	// Process each row
	rowCount := filtered.Nrow()
	for i := 0; i < rowCount; i++ {
		// Safely extract values with bounds checking
		seqType := "MAMBA"
		if i < len(types) {
			seqType = types[i]
		}

		length := 3
		if i < len(lengths) {
			length = parseIntFromString(lengths[i])
		}

		strength := 100.0
		if i < len(strengths) {
			strength = strengths[i]
		}

		seqReturn := 0.0
		if i < len(returns) {
			seqReturn = returns[i]
		}

		gap := 1.0
		if i < len(gaps) {
			gap = gaps[i]
		}

		key := fmt.Sprintf("%s-%d", seqType, length)

		pattern, exists := patternMap[key]
		if !exists {
			pattern = &domain.SequencePattern{
				Type:   domain.SequenceType(seqType),
				Length: length,
			}
			patternMap[key] = pattern
		}

		pattern.Frequency++
		pattern.Strength += strength
		pattern.AverageReturn = (pattern.AverageReturn*float64(pattern.Frequency-1) +
			seqReturn) / float64(pattern.Frequency)
		pattern.AverageGap = (pattern.AverageGap*float64(pattern.Frequency-1) +
			gap) / float64(pattern.Frequency)
	}

	// Convert map to slice and calculate final metrics
	for _, pattern := range patternMap {
		pattern.Strength /= float64(pattern.Frequency)
		pattern.SuccessRate = sdf.calculatePatternSuccessRate(*pattern)
		patterns = append(patterns, *pattern)
	}

	return patterns
}

// findDominantPattern identifies the most significant pattern
func (sdf *SequenceDataFrame) findDominantPattern(patterns []domain.SequencePattern) domain.SequencePattern {
	var dominant domain.SequencePattern
	maxScore := 0.0

	for _, pattern := range patterns {
		// Score combines frequency, strength, and success rate
		score := float64(pattern.Frequency) * pattern.Strength * pattern.SuccessRate
		if score > maxScore {
			maxScore = score
			dominant = pattern
		}
	}

	return dominant
}

// calculateQualityMetrics computes sequence quality metrics using statistical operations
func (sdf *SequenceDataFrame) calculateQualityMetrics() QualityMetrics {
	// Check if DataFrame is empty
	if sdf.df.Nrow() == 0 {
		return QualityMetrics{}
	}

	// Use GoNum for statistical calculations with safe accessors
	lengths := sdf.df.Col("length").Float()
	strengths := sdf.df.Col("strength").Float()
	gaps := sdf.df.Col("gap_to_next").Float()

	// Handle empty data
	if len(lengths) == 0 || len(strengths) == 0 {
		return QualityMetrics{}
	}

	// Length score: statistical distribution analysis
	lengthMean := stat.Mean(lengths, nil)
	lengthScore := math.Min(lengthMean/5.0, 1.0)

	// Strength score: average strength
	strengthScore := stat.Mean(strengths, nil)

	// Consistency score: inverse of coefficient of variation
	lengthStdDev := stat.StdDev(lengths, nil)
	var consistencyScore float64
	if lengthMean > 0 {
		cv := lengthStdDev / lengthMean
		consistencyScore = math.Max(0, 1-math.Min(cv/2.0, 1.0))
	}

	// Overall sequence quality
	sequenceQuality := lengthScore*0.3 + strengthScore*0.4 + consistencyScore*0.3

	// Continuity score: based on gaps
	validGaps := make([]float64, 0)
	for _, gap := range gaps {
		if gap <= float64(sdf.metadata.MaxGapLength) && gap > 0 {
			validGaps = append(validGaps, gap)
		}
	}

	var continuityScore float64
	if len(validGaps) > 0 {
		avgGap := stat.Mean(validGaps, nil)
		continuityScore = math.Max(0, 1-(avgGap/float64(sdf.metadata.MaxGapLength)))
	}

	// Predictive score: pattern consistency
	predictiveScore := sdf.calculatePredictiveScore()

	// Momentum score: recent sequence strength
	momentumScore := sdf.calculateMomentumScore()

	return QualityMetrics{
		SequenceQuality: sequenceQuality,
		ContinuityScore: continuityScore,
		PredictiveScore: predictiveScore,
		MomentumScore:   momentumScore,
	}
}

// analyzeVolumeProfile performs volume analysis using statistical operations
func (sdf *SequenceDataFrame) analyzeVolumeProfile() domain.VolumeProfile {
	if sdf.df.Nrow() == 0 {
		return domain.VolumeProfile{}
	}

	volumes := sdf.df.Col("volume").Float()

	if len(volumes) == 0 {
		return domain.VolumeProfile{}
	}

	// Statistical volume analysis
	avgVolume := stat.Mean(volumes, nil)
	volumeStdDev := stat.StdDev(volumes, nil)

	// Volume trend using linear regression
	indices := make([]float64, len(volumes))
	for i := range indices {
		indices[i] = float64(i)
	}

	// Calculate linear regression slope for trend
	var volumeTrend float64
	if len(volumes) > 1 {
		_, beta := stat.LinearRegression(indices, volumes, nil, false)
		volumeTrend = math.Tanh(beta / 1000000) // Normalize slope
	}

	// Volume consistency: inverse coefficient of variation
	var volumeConsistency float64
	if avgVolume > 0 {
		cv := volumeStdDev / avgVolume
		volumeConsistency = math.Max(0, 1-math.Min(cv, 1.0))
	}

	// Volume strength: relative to average
	volumeStrength := 1.0 // Default strength
	mambaVolumes := sdf.getMambaVolumes()
	if len(mambaVolumes) > 0 && avgVolume > 0 {
		mambaMean := stat.Mean(mambaVolumes, nil)
		volumeStrength = mambaMean / avgVolume
	}

	return domain.VolumeProfile{
		AverageVolume:     avgVolume,
		VolumeStrength:    volumeStrength,
		VolumeTrend:       volumeTrend,
		VolumeConsistency: volumeConsistency,
	}
}

// calculatePredictiveScore analyzes pattern predictability
func (sdf *SequenceDataFrame) calculatePredictiveScore() float64 {
	if sdf.df.Nrow() == 0 {
		return 0
	}

	types := sdf.df.Col("type").Records()

	if len(types) < 3 {
		return 0
	}

	correctPredictions := 0
	totalPredictions := 0

	for i := 0; i < len(types)-2; i++ {
		if types[i] == types[i+1] {
			totalPredictions++
			if types[i+1] == types[i+2] {
				correctPredictions++
			}
		}
	}

	if totalPredictions == 0 {
		return 0
	}

	return float64(correctPredictions) / float64(totalPredictions)
}

// calculateMomentumScore analyzes recent sequence momentum
func (sdf *SequenceDataFrame) calculateMomentumScore() float64 {
	if sdf.df.Nrow() == 0 {
		return 0
	}

	strengths := sdf.df.Col("strength").Float()

	if len(strengths) == 0 {
		return 0
	}

	// Focus on recent sequences (last 5 or fewer)
	recentCount := int(math.Min(5, float64(len(strengths))))
	recentStrengths := strengths[len(strengths)-recentCount:]

	// Calculate weighted average with more weight on recent sequences
	var weightedSum, weightSum float64
	for i, strength := range recentStrengths {
		weight := float64(i + 1)
		weightedSum += strength * weight
		weightSum += weight
	}

	if weightSum == 0 {
		return 0
	}

	weightedStrength := weightedSum / weightSum
	averageStrength := stat.Mean(recentStrengths, nil)

	return weightedStrength*0.7 + averageStrength*0.3
}

// calculatePatternSuccessRate calculates success rate for a specific pattern
func (sdf *SequenceDataFrame) calculatePatternSuccessRate(pattern domain.SequencePattern) float64 {
	if sdf.df.Nrow() == 0 {
		return 0
	}

	types := sdf.df.Col("type").Records()
	lengths := sdf.df.Col("length").Records()

	if len(types) == 0 || len(lengths) == 0 {
		return 0
	}

	successCount := 0
	totalCount := 0

	for i := 0; i < len(types)-1; i++ {
		if i >= len(lengths) {
			break
		}

		currentType := types[i]
		currentLength := parseIntFromString(lengths[i])

		if currentType == string(pattern.Type) && currentLength == pattern.Length {
			totalCount++
			// Success if next sequence continues the pattern
			if i < len(types)-1 && types[i+1] == string(pattern.Type) {
				successCount++
			}
		}
	}

	if totalCount == 0 {
		return 0
	}

	return float64(successCount) / float64(totalCount)
}

// getMambaVolumes filters and returns volumes for Mamba sequences
func (sdf *SequenceDataFrame) getMambaVolumes() []float64 {
	if sdf.df.Nrow() == 0 {
		return []float64{}
	}

	mambaFilter := sdf.df.Filter(
		dataframe.F{
			Colname:    "type",
			Comparator: "==",
			Comparando: string(domain.MambaSequence),
		},
	)

	if mambaFilter.Nrow() == 0 {
		return []float64{}
	}

	return mambaFilter.Col("volume").Float()
}

// Helper functions
func calculateSequenceVolume(seq domain.MoveSequence) float64 {
	if len(seq.Moves) == 0 {
		return 0
	}

	total := 0.0
	for _, move := range seq.Moves {
		total += move.Volume
	}
	return total / float64(len(seq.Moves))
}

func calculateSequenceReturn(seq domain.MoveSequence) float64 {
	if len(seq.Moves) == 0 {
		return 0
	}

	startPrice := seq.Moves[0].OpenPrice
	endPrice := seq.Moves[len(seq.Moves)-1].ClosePrice

	if startPrice == 0 {
		return 0
	}

	return ((endPrice - startPrice) / startPrice) * 100
}

func parseIntFromString(s string) int {
	// Simple integer parsing - in production, handle errors properly
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
