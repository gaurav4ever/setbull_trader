package service

import (
	"context"
	"fmt"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
)

const (
	LookbackPeriod   = 21  // Days to analyze
	BullishThreshold = 5.0 // 5% threshold for bullish mamba
	BearishThreshold = 3.0 // 3% threshold for bearish mamba
)

type MambaFilter struct {
	candleRepo       repository.CandleRepository
	sequenceDetector *SequenceDetector
	config           config.MambaFilterConfig
}

type MambaFilterConfig struct {
	LookbackPeriod       int     // Number of days to analyze
	MoveThresholdBullish float64 // Threshold for Mamba move detection in bullish trend
	MoveThresholdBearish float64 // Threshold for Mamba move detection in bearish trend
	MinSequenceLength    int     // Minimum sequence length to consider
	MaxGapDays           int     // Maximum allowed gap between sequences
	MinMambaRatio        float64 // Minimum ratio of Mamba days to total days
}

// NewMambaFilter creates a new instance of MambaFilter
func NewMambaFilter(repo repository.CandleRepository,
	config config.MambaFilterConfig,
	technicalIndicators *TechnicalIndicatorService,
	tradingCalendar *TradingCalendarService) *MambaFilter {
	return &MambaFilter{
		candleRepo:       repo,
		sequenceDetector: NewSequenceDetector(config.MoveThresholdBullish, config.MoveThresholdBearish, technicalIndicators, tradingCalendar, config),
		config:           config,
	}
}

// Filter implements the StockFilter interface
func (f *MambaFilter) Filter(ctx context.Context, stocks interface{}) (bullish, bearish []domain.FilteredStock, err error) {
	// Handle different input types
	switch input := stocks.(type) {
	case []domain.StockUniverse:
		// Convert to FilteredStock
		var filteredStocks []domain.FilteredStock
		for _, stock := range input {
			filteredStocks = append(filteredStocks, domain.FilteredStock{
				Stock:         stock,
				FilterResults: make(map[string]bool),
				FilterReasons: make(map[string]string),
			})
		}
		return f.processStocks(ctx, filteredStocks)

	case []domain.FilteredStock:
		return f.processStocks(ctx, input)
	}

	return nil, nil, fmt.Errorf("unsupported input type")
}

// processStocks handles the actual Mamba moves filtering logic
func (f *MambaFilter) processStocks(ctx context.Context, stocks []domain.FilteredStock) (bullish, bearish []domain.FilteredStock, err error) {
	var bullishStocks, bearishStocks []domain.FilteredStock
	var skippedStocks int

	log.Info("Starting Mamba moves filter for %d stocks", len(stocks))

	// Process each stock
	for _, stock := range stocks {
		candles, err := f.candleRepo.GetNDailyCandlesByTimeframe(ctx,
			stock.Stock.InstrumentKey,
			"day",
			f.config.LookbackPeriod,
		)
		if err != nil {
			log.Warn("Failed to get candles for %s: %v", stock.Stock.InstrumentKey, err)
			skippedStocks++
			continue
		}

		if len(candles) < f.config.LookbackPeriod {
			log.Debug("Insufficient candle data for %s: got %d days, need %d",
				stock.Stock.InstrumentKey, len(candles), f.config.LookbackPeriod)
			skippedStocks++
			continue
		}

		// Build and analyze the move series
		analysis := f.sequenceDetector.BuildSequences(ctx, stock.Stock, candles)

		// Apply filtering criteria
		stock.SequenceAnalysis = analysis
		if f.meetsFilterCriteria(analysis) {
			stock.FilterResults["mamba_filter"] = true
			stock.FilterReasons["mamba_filter"] = fmt.Sprintf("PASSED")

			if analysis.Trend.Type == domain.BullishTrend {
				stock.IsBullish = true
				bullishStocks = append(bullishStocks, stock)
			} else {
				stock.IsBearish = true
				bearishStocks = append(bearishStocks, stock)
			}
		} else {
			stock.FilterReasons["mamba_filter"] = fmt.Sprintf("REJECTED: %+v", analysis)
			skippedStocks++
		}
	}

	log.Info("Mamba filter completed. Bullish: %d, Bearish: %d, Skipped: %d",
		len(bullishStocks), len(bearishStocks), skippedStocks)

	return bullishStocks, bearishStocks, nil
}

func (f *MambaFilter) meetsFilterCriteria(analysis domain.SequenceAnalysis) bool {
	// Calculate Mamba day ratio
	totalDays := analysis.TotalMambaDays + analysis.TotalNonMambaDays
	if totalDays == 0 {
		return false
	}
	mambaRatio := float64(analysis.TotalMambaDays) / float64(totalDays)

	// Check if current sequence meets minimum length
	currentSeqMeetsLength := analysis.CurrentSequence.Length >= f.config.MinSequenceLength

	// Check for significant sequences
	hasSignificantSequences := false
	for _, length := range analysis.MambaSequences {
		if length >= f.config.MinSequenceLength {
			hasSignificantSequences = true
			break
		}
	}

	// Check sequence gaps
	hasAcceptableGaps := f.checkSequenceGaps(analysis)

	return mambaRatio >= f.config.MinMambaRatio &&
		(currentSeqMeetsLength || hasSignificantSequences) &&
		hasAcceptableGaps
}

func (f *MambaFilter) checkSequenceGaps(analysis domain.SequenceAnalysis) bool {
	if len(analysis.MambaSequences) <= 1 {
		return true // Not enough sequences to check gaps
	}

	gapCount := 0
	for i := 0; i < len(analysis.NonMambaSequences); i++ {
		if analysis.NonMambaSequences[i] > f.config.MaxGapDays {
			gapCount++
		}
	}

	// Allow some flexibility in gap occurrences
	return gapCount <= len(analysis.MambaSequences)/3
}
