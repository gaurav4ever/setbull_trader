package service

import (
	"math"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/trading/config"
)

// MoveAnalyzer handles the analysis of price movements
type MoveAnalyzer struct {
	config config.MambaFilterConfig
}

type MoveStrength struct {
	PriceStrength   float64
	VolumeStrength  float64
	OverallStrength float64
	IsStrong        bool
}

// NewMoveAnalyzer creates a new instance of MoveAnalyzer
func NewMoveAnalyzer(config config.MambaFilterConfig) *MoveAnalyzer {
	return &MoveAnalyzer{
		config: config,
	}
}

func (ma *MoveAnalyzer) AnalyzeMoveStrength(
	move domain.DailyMove,
	avgVolume float64) MoveStrength {

	// Calculate price strength based on move size
	priceStrength := math.Min(move.PriceChange/10.0, 1.0) // Normalize to 0-1

	// Calculate volume strength
	volumeStrength := 0.0
	if avgVolume > 0 {
		volumeRatio := move.Volume / avgVolume
		volumeStrength = math.Min(volumeRatio-1.0, 1.0) // Normalize to 0-1
	}

	// Use config values
	overallStrength := (priceStrength * (1 - ma.config.MoveAnalyzer.VolumeWeight)) +
		(volumeStrength * ma.config.MoveAnalyzer.VolumeWeight)

	return MoveStrength{
		PriceStrength:   priceStrength,
		VolumeStrength:  volumeStrength,
		OverallStrength: overallStrength,
		IsStrong:        overallStrength >= ma.config.MoveAnalyzer.StrengthThreshold,
	}
}
