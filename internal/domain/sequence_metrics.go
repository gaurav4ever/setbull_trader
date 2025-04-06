package domain

type SequenceMetrics struct {
	Stock           StockUniverse
	SequenceQuality float64
	ContinuityScore float64
	PredictiveScore float64
	MomentumScore   float64
	TrendStrength   float64
	PricePatterns   []PricePattern
	VolumeProfile   VolumeProfile
	DominantPattern SequencePattern
}

type PricePattern struct {
	Type          string
	Length        int
	Frequency     int
	AverageGap    float64
	SuccessRate   float64
	AverageReturn float64
	Strength      float64
}

type VolumeProfile struct {
	AverageVolume     float64
	VolumeStrength    float64
	VolumeTrend       float64
	VolumeConsistency float64
}

type SequencePattern struct {
	Type          SequenceType
	Length        int
	Frequency     int
	AverageGap    float64
	SuccessRate   float64
	AverageReturn float64
	Strength      float64
}
