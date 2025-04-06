package service

import (
	"context"
	"math"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/trading/config"
	"sort"
)

type SequenceDetector struct {
	moveThresholdBullish float64
	moveThresholdBearish float64
	trendDetector        *TrendDetector
	moveAnalyzer         *MoveAnalyzer
}

func NewSequenceDetector(thresholdBullish float64,
	thresholdBearish float64,
	technicalIndicators *TechnicalIndicatorService,
	tradingCalendar *TradingCalendarService,
	config config.MambaFilterConfig) *SequenceDetector {
	return &SequenceDetector{
		moveThresholdBullish: thresholdBullish,
		moveThresholdBearish: thresholdBearish,
		trendDetector:        NewTrendDetector(technicalIndicators, tradingCalendar),
		moveAnalyzer:         NewMoveAnalyzer(config),
	}
}

// DetectMove determines if a day's movement qualifies as a trend-aligned Mamba move
func (d *SequenceDetector) DetectMove(candle domain.Candle, trend domain.TrendType,
	avgVolume float64) domain.DailyMove {

	// With this:
	var basePrice float64
	if candle.Open < candle.Close {
		basePrice = candle.Open // For bullish candles, use Open as base
	} else {
		basePrice = candle.Close // For bearish candles, use Close as base
	}
	movePercentage := math.Abs((candle.Close-candle.Open)/basePrice) * 100
	isBullish := candle.Close > candle.Open

	move := domain.DailyMove{
		Date:        candle.Timestamp,
		PriceChange: movePercentage,
		HighPrice:   candle.High,
		LowPrice:    candle.Low,
		OpenPrice:   candle.Open,
		ClosePrice:  candle.Close,
		Volume:      float64(candle.Volume),
		IsBullish:   isBullish,
	}

	// Analyze move strength
	strength := d.moveAnalyzer.AnalyzeMoveStrength(move, avgVolume)

	// Determine if the move is a valid Mamba move based on trend alignment and strength
	if strength.IsStrong {
		if trend == domain.BullishTrend && isBullish && movePercentage > d.moveThresholdBullish {
			move.IsBullishMamba = true
			move.IsMamba = true
			move.MoveStrength = strength.OverallStrength
		} else if trend == domain.BearishTrend && !isBullish && movePercentage > d.moveThresholdBearish {
			move.IsBearishMamba = true
			move.IsMamba = true
			move.MoveStrength = strength.OverallStrength
		}
	}

	return move
}

// BuildSequences analyzes candles and builds sequences based on trend
func (d *SequenceDetector) BuildSequences(
	ctx context.Context,
	stock domain.StockUniverse,
	candles []domain.Candle) domain.SequenceAnalysis {

	if len(candles) == 0 {
		return domain.SequenceAnalysis{Stock: stock}
	}

	// Calculate average volume for strength comparison
	avgVolume := calculateAverageVolume(candles)

	// Determine trend first
	trendAnalysis := d.trendDetector.AnalyzeTrend(ctx, candles)

	// If trend is not strong enough, return empty analysis
	if !d.trendDetector.ValidateTrendStrength(trendAnalysis) {
		return domain.SequenceAnalysis{
			Stock: stock,
			Trend: trendAnalysis,
		}
	}

	// Sort candles by timestamp
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].Timestamp.Before(candles[j].Timestamp)
	})

	sequences := d.buildMoveSequences(candles, trendAnalysis.Type, avgVolume)

	return d.generateSequenceAnalysis(stock, sequences, trendAnalysis)
}

func (d *SequenceDetector) buildMoveSequences(candles []domain.Candle,
	trend domain.TrendType, avgVolume float64) []domain.MoveSequence {

	var sequences []domain.MoveSequence
	var currentMoves []domain.DailyMove

	for i, candle := range candles {
		move := d.DetectMove(candle, trend, avgVolume)

		if i == 0 || (move.IsMamba == currentMoves[len(currentMoves)-1].IsMamba) {
			// Continue current sequence
			currentMoves = append(currentMoves, move)
		} else {
			// End current sequence and start new one
			if len(currentMoves) > 0 {
				sequence := domain.MoveSequence{
					Type:      d.determineSequenceType(currentMoves[0].IsMamba),
					Length:    len(currentMoves),
					StartDate: currentMoves[0].Date,
					EndDate:   currentMoves[len(currentMoves)-1].Date,
					Moves:     currentMoves,
					Strength:  calculateSequenceStrength(currentMoves),
				}
				sequences = append(sequences, sequence)
			}
			currentMoves = []domain.DailyMove{move}
		}
	}

	// Add final sequence
	if len(currentMoves) > 0 {
		sequence := domain.MoveSequence{
			Type:      d.determineSequenceType(currentMoves[0].IsMamba),
			Length:    len(currentMoves),
			StartDate: currentMoves[0].Date,
			EndDate:   currentMoves[len(currentMoves)-1].Date,
			Moves:     currentMoves,
			Strength:  calculateSequenceStrength(currentMoves),
		}
		sequences = append(sequences, sequence)
	}

	return sequences
}

func (d *SequenceDetector) generateSequenceAnalysis(
	stock domain.StockUniverse,
	sequences []domain.MoveSequence,
	trendAnalysis domain.TrendAnalysis) domain.SequenceAnalysis {

	var mambaLengths []int
	var nonMambaLengths []int
	totalMamba := 0
	totalNonMamba := 0

	for _, seq := range sequences {
		if seq.Type == domain.MambaSequence {
			mambaLengths = append(mambaLengths, seq.Length)
			totalMamba += seq.Length
		} else {
			nonMambaLengths = append(nonMambaLengths, seq.Length)
			totalNonMamba += seq.Length
		}
	}

	var currentSequence domain.MoveSequence
	if len(sequences) > 0 {
		currentSequence = sequences[len(sequences)-1]
	}

	return domain.SequenceAnalysis{
		Stock:              stock,
		Trend:              trendAnalysis,
		MambaSequences:     mambaLengths,
		NonMambaSequences:  nonMambaLengths,
		AverageMambaLen:    calculateAverage(mambaLengths),
		AverageNonMambaLen: calculateAverage(nonMambaLengths),
		CurrentSequence:    currentSequence,
		TotalMambaDays:     totalMamba,
		TotalNonMambaDays:  totalNonMamba,
		Sequences:          sequences,
	}
}

func calculateAverage(lengths []int) float64 {
	if len(lengths) == 0 {
		return 0
	}
	var sum int
	for _, length := range lengths {
		sum += length
	}
	return float64(sum) / float64(len(lengths))
}

func (d *SequenceDetector) determineSequenceType(isMamba bool) domain.SequenceType {
	if isMamba {
		return domain.MambaSequence
	}
	return domain.NonMambaSequence
}

func calculateSequenceStrength(moves []domain.DailyMove) float64 {
	if len(moves) == 0 {
		return 0
	}

	totalStrength := 0.0
	for _, move := range moves {
		totalStrength += move.MoveStrength
	}
	return totalStrength / float64(len(moves))
}

func calculateAverageVolume(candles []domain.Candle) float64 {
	if len(candles) == 0 {
		return 0
	}

	totalVolume := 0.0
	for _, candle := range candles {
		totalVolume += float64(candle.Volume)
	}
	return totalVolume / float64(len(candles))
}
