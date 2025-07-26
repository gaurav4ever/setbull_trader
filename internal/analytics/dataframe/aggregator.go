package dataframe

import (
	"fmt"
	"time"

	"setbull_trader/internal/domain"
)

// Aggregator handles candle aggregation operations using DataFrames
type Aggregator struct {
	// Configuration options for aggregation
	config *AggregatorConfig
}

// AggregatorConfig holds configuration for the aggregator
type AggregatorConfig struct {
	DefaultTimeframe string        `json:"default_timeframe"`
	MaxCandles       int           `json:"max_candles"`
	Timeout          time.Duration `json:"timeout"`
}

// NewAggregator creates a new DataFrame-based aggregator
func NewAggregator(config *AggregatorConfig) *Aggregator {
	if config == nil {
		config = &AggregatorConfig{
			DefaultTimeframe: "5m",
			MaxCandles:       10000,
			Timeout:          30 * time.Second,
		}
	}

	return &Aggregator{
		config: config,
	}
}

// Aggregate5MinCandles aggregates 1-minute candles to 5-minute candles
func (a *Aggregator) Aggregate5MinCandles(candles []domain.Candle) ([]domain.Candle5Min, error) {
	if len(candles) == 0 {
		return []domain.Candle5Min{}, nil
	}

	// Create DataFrame from candles - database source (IST timestamps)
	df := NewCandleDataFrameWithContext(candles, TimestampFromDatabase)
	if df.Empty() {
		return []domain.Candle5Min{}, nil
	}

	// Group by 5-minute intervals
	aggregatedCandles, err := a.groupByTimeInterval(df, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate candles: %v", err)
	}

	// Convert to domain.Candle5Min
	result := make([]domain.Candle5Min, len(aggregatedCandles))
	for i, candle := range aggregatedCandles {
		result[i] = domain.Candle5Min{
			InstrumentKey: candle.InstrumentKey,
			Timestamp:     candle.Timestamp,
			Open:          candle.Open,
			High:          candle.High,
			Low:           candle.Low,
			Close:         candle.Close,
			Volume:        candle.Volume,
			OpenInterest:  candle.OpenInterest,
			TimeInterval:  "5m",
		}
	}

	return result, nil
}

// groupByTimeInterval groups candles by specified time interval
func (a *Aggregator) groupByTimeInterval(df *CandleDataFrame, interval time.Duration) ([]domain.Candle, error) {
	candles := df.ToCandles()
	if len(candles) == 0 {
		return []domain.Candle{}, nil
	}

	var result []domain.Candle
	var currentGroup []domain.Candle

	// Start with the first candle's timestamp aligned to interval
	startTime := alignToInterval(candles[0].Timestamp, interval)
	currentInterval := startTime

	for _, candle := range candles {
		intervalForCandle := alignToInterval(candle.Timestamp, interval)

		// If this candle belongs to a new interval, process the current group
		if !intervalForCandle.Equal(currentInterval) {
			if len(currentGroup) > 0 {
				aggregated := a.aggregateGroup(currentGroup, currentInterval, interval)
				result = append(result, aggregated)
			}
			currentGroup = []domain.Candle{candle}
			currentInterval = intervalForCandle
		} else {
			currentGroup = append(currentGroup, candle)
		}
	}

	// Process the last group
	if len(currentGroup) > 0 {
		aggregated := a.aggregateGroup(currentGroup, currentInterval, interval)
		result = append(result, aggregated)
	}

	return result, nil
}

// aggregateGroup aggregates a group of candles into a single candle
func (a *Aggregator) aggregateGroup(candles []domain.Candle, intervalStart time.Time, interval time.Duration) domain.Candle {
	if len(candles) == 0 {
		return domain.Candle{}
	}

	// OHLCV aggregation
	open := candles[0].Open
	high := candles[0].High
	low := candles[0].Low
	close := candles[len(candles)-1].Close
	var volume int64
	var openInterest int64

	// Find high, low, and sum volume
	for _, candle := range candles {
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
		volume += candle.Volume
		openInterest += candle.OpenInterest
	}

	return domain.Candle{
		InstrumentKey: candles[0].InstrumentKey,
		Timestamp:     intervalStart,
		Open:          open,
		High:          high,
		Low:           low,
		Close:         close,
		Volume:        volume,
		OpenInterest:  openInterest,
		TimeInterval:  formatInterval(interval),
	}
}

// alignToInterval aligns a timestamp to the specified interval
func alignToInterval(t time.Time, interval time.Duration) time.Time {
	switch interval {
	case time.Minute:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	case 5 * time.Minute:
		minute := (t.Minute() / 5) * 5
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
	case 15 * time.Minute:
		minute := (t.Minute() / 15) * 15
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
	case time.Hour:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	default:
		// For custom intervals, use the interval in nanoseconds
		intervalNano := interval.Nanoseconds()
		alignedNano := (t.UnixNano() / intervalNano) * intervalNano
		return time.Unix(0, alignedNano).In(t.Location())
	}
}

// formatInterval formats a time.Duration to string representation
func formatInterval(interval time.Duration) string {
	switch interval {
	case time.Minute:
		return "1m"
	case 5 * time.Minute:
		return "5m"
	case 15 * time.Minute:
		return "15m"
	case time.Hour:
		return "1h"
	default:
		return fmt.Sprintf("%v", interval)
	}
}

// AggregateWithIndicators aggregates candles and calculates indicators in one operation
func (a *Aggregator) AggregateWithIndicators(candles []domain.Candle, interval time.Duration) (*CandleDataFrame, error) {
	// First aggregate the candles - candles from database are already in IST, no conversion needed
	aggregatedCandles, err := a.groupByTimeInterval(NewCandleDataFrameWithContext(candles, TimestampFromDatabase), interval)
	if err != nil {
		return nil, err
	}

	// Create DataFrame from aggregated candles - these are aligned, preserve timestamps
	df := NewCandleDataFrameWithContext(aggregatedCandles, TimestampForAlignment)

	// Set the interval on the DataFrame
	df.SetInterval(formatInterval(interval))

	// This is where we'll add indicator calculations
	// For now, return the DataFrame ready for indicator calculation
	return df, nil
}

// ValidateTimeframe validates if the timeframe is supported
func (a *Aggregator) ValidateTimeframe(timeframe string) error {
	validTimeframes := map[string]bool{
		"1m":  true,
		"3m":  true,
		"5m":  true,
		"15m": true,
		"30m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}

	if !validTimeframes[timeframe] {
		return fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	return nil
}

// ParseTimeframe converts string timeframe to time.Duration
func (a *Aggregator) ParseTimeframe(timeframe string) (time.Duration, error) {
	switch timeframe {
	case "1m":
		return time.Minute, nil
	case "3m":
		return 3 * time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}
}
