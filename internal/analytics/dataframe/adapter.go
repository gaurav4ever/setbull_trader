package dataframe

import (
	"strconv"
	"time"

	"setbull_trader/internal/domain"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// CandleDataFrame wraps gota DataFrame for candle data operations
type CandleDataFrame struct {
	df dataframe.DataFrame
}

// NewCandleDataFrame creates a new CandleDataFrame from candles
func NewCandleDataFrame(candles []domain.Candle) *CandleDataFrame {
	if len(candles) == 0 {
		// Return empty DataFrame
		df := dataframe.New()
		return &CandleDataFrame{df: df}
	}

	// Prepare data for DataFrame
	symbols := make([]string, len(candles))
	timestamps := make([]string, len(candles))
	opens := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	volumes := make([]float64, len(candles))

	for i, candle := range candles {
		symbols[i] = candle.InstrumentKey
		timestamps[i] = candle.Timestamp.Format("2006-01-02T15:04:05")
		opens[i] = candle.Open
		highs[i] = candle.High
		lows[i] = candle.Low
		closes[i] = candle.Close
		volumes[i] = float64(candle.Volume)
	}

	// Create DataFrame
	df := dataframe.New(
		series.New(symbols, series.String, "Symbol"),
		series.New(timestamps, series.String, "Timestamp"),
		series.New(opens, series.Float, "Open"),
		series.New(highs, series.Float, "High"),
		series.New(lows, series.Float, "Low"),
		series.New(closes, series.Float, "Close"),
		series.New(volumes, series.Float, "Volume"),
	)

	return &CandleDataFrame{df: df}
}

// DataFrame returns the underlying gota DataFrame
func (c *CandleDataFrame) DataFrame() dataframe.DataFrame {
	return c.df
}

// Empty returns true if the DataFrame is empty
func (c *CandleDataFrame) Empty() bool {
	return c.df.Nrow() == 0
}

// GetTimestamps returns the timestamp column as time.Time slice
func (c *CandleDataFrame) GetTimestamps() []time.Time {
	if c.Empty() {
		return []time.Time{}
	}

	timestampCol := c.df.Col("Timestamp")
	timestamps := make([]time.Time, timestampCol.Len())

	for i := 0; i < timestampCol.Len(); i++ {
		timeStr := timestampCol.Elem(i).String()
		if t, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
			timestamps[i] = t
		}
	}

	return timestamps
}

// GetSymbols returns the symbol column as string slice
func (c *CandleDataFrame) GetSymbols() []string {
	if c.Empty() {
		return []string{}
	}

	symbolCol := c.df.Col("Symbol")
	symbols := make([]string, symbolCol.Len())

	for i := 0; i < symbolCol.Len(); i++ {
		symbols[i] = symbolCol.Elem(i).String()
	}

	return symbols
}

// GetOpens returns the open column as float64 slice
func (c *CandleDataFrame) GetOpens() []float64 {
	return c.getFloatColumn("Open")
}

// GetHighs returns the high column as float64 slice
func (c *CandleDataFrame) GetHighs() []float64 {
	return c.getFloatColumn("High")
}

// GetLows returns the low column as float64 slice
func (c *CandleDataFrame) GetLows() []float64 {
	return c.getFloatColumn("Low")
}

// GetCloses returns the close column as float64 slice
func (c *CandleDataFrame) GetCloses() []float64 {
	return c.getFloatColumn("Close")
}

// GetVolumes returns the volume column as float64 slice
func (c *CandleDataFrame) GetVolumes() []float64 {
	return c.getFloatColumn("Volume")
}

// getFloatColumn helper method to extract float columns
func (c *CandleDataFrame) getFloatColumn(colName string) []float64 {
	if c.Empty() {
		return []float64{}
	}

	col := c.df.Col(colName)
	values := make([]float64, col.Len())

	for i := 0; i < col.Len(); i++ {
		if val, err := strconv.ParseFloat(col.Elem(i).String(), 64); err == nil {
			values[i] = val
		}
	}

	return values
}

// ToCandles converts the DataFrame back to domain.Candle slice
func (c *CandleDataFrame) ToCandles() []domain.Candle {
	if c.Empty() {
		return []domain.Candle{}
	}

	rowCount := c.df.Nrow()
	candles := make([]domain.Candle, rowCount)

	timestamps := c.GetTimestamps()
	symbols := c.GetSymbols()
	opens := c.GetOpens()
	highs := c.GetHighs()
	lows := c.GetLows()
	closes := c.GetCloses()
	volumes := c.GetVolumes()

	for i := 0; i < rowCount; i++ {
		candles[i] = domain.Candle{
			InstrumentKey: symbols[i],
			Timestamp:     timestamps[i],
			Open:          opens[i],
			High:          highs[i],
			Low:           lows[i],
			Close:         closes[i],
			Volume:        int64(volumes[i]),
		}
	}

	return candles
}

// ToAggregatedCandles converts the DataFrame to AggregatedCandle slice
func (c *CandleDataFrame) ToAggregatedCandles() []domain.AggregatedCandle {
	if c.Empty() {
		return []domain.AggregatedCandle{}
	}

	rowCount := c.df.Nrow()
	result := make([]domain.AggregatedCandle, rowCount)

	timestamps := c.GetTimestamps()
	opens := c.GetOpens()
	highs := c.GetHighs()
	lows := c.GetLows()
	closes := c.GetCloses()
	volumes := c.GetVolumes()

	// Extract instrument key from first row if available
	instrumentKey := ""
	if symbols := c.GetSymbols(); len(symbols) > 0 {
		instrumentKey = symbols[0]
	}

	for i := 0; i < rowCount; i++ {
		result[i] = domain.AggregatedCandle{
			InstrumentKey: instrumentKey,
			Timestamp:     timestamps[i],
			Open:          opens[i],
			High:          highs[i],
			Low:           lows[i],
			Close:         closes[i],
			Volume:        int64(volumes[i]),
			OpenInterest:  0,
			TimeInterval:  "",
		}
	}

	return result
}
