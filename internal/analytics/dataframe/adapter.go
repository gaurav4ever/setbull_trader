package dataframe

import (
	"strconv"
	"time"

	"setbull_trader/internal/domain"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// TimestampContext defines the source context of timestamps for proper handling
type TimestampContext int

const (
	TimestampFromDatabase  TimestampContext = iota // IST - No conversion needed
	TimestampFromExternal                          // Unknown - May need conversion
	TimestampFromGenerated                         // Local - Convert to IST
	TimestampForAlignment                          // Special - Alignment logic
)

// CandleDataFrame wraps gota DataFrame for candle data operations
type CandleDataFrame struct {
	df       dataframe.DataFrame
	interval string
}

// NewCandleDataFrame creates a new CandleDataFrame from candles (backward compatible)
func NewCandleDataFrame(candles []domain.Candle, timeZoneConversion bool) *CandleDataFrame {
	context := TimestampFromDatabase // Safe default - assume database source
	if timeZoneConversion {
		context = TimestampFromExternal // Legacy behavior for external sources
	}
	return NewCandleDataFrameWithContext(candles, context)
}

// NewCandleDataFrameWithContext creates a new CandleDataFrame with explicit timestamp context
func NewCandleDataFrameWithContext(candles []domain.Candle, context TimestampContext) *CandleDataFrame {
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

	// IST location for timezone operations
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback to fixed zone if loading fails
		ist = time.FixedZone("IST", 5*3600+30*60)
	}

	for i, candle := range candles {
		symbols[i] = candle.InstrumentKey

		// Handle timestamps based on context
		switch context {
		case TimestampFromDatabase:
			// Database timestamps are already in IST - preserve as-is
			timestamps[i] = candle.Timestamp.Format("2006-01-02T15:04:05")
		case TimestampFromExternal:
			// External timestamps may need conversion to IST
			timestamps[i] = candle.Timestamp.In(ist).Format("2006-01-02T15:04:05")
		case TimestampFromGenerated:
			// Generated timestamps should be converted to IST
			timestamps[i] = candle.Timestamp.In(ist).Format("2006-01-02T15:04:05")
		case TimestampForAlignment:
			// Alignment preserves timezone but may adjust time values
			if isIST(candle.Timestamp) {
				timestamps[i] = candle.Timestamp.Format("2006-01-02T15:04:05")
			} else {
				timestamps[i] = candle.Timestamp.In(ist).Format("2006-01-02T15:04:05")
			}
		default:
			// Default to database behavior (safe)
			timestamps[i] = candle.Timestamp.Format("2006-01-02T15:04:05")
		}

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

// SetInterval sets the time interval for the DataFrame
func (c *CandleDataFrame) SetInterval(interval string) {
	c.interval = interval
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

	// Get IST location for proper timezone assignment
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.FixedZone("IST", 5*3600+30*60)
	}

	for i := 0; i < timestampCol.Len(); i++ {
		timeStr := timestampCol.Elem(i).String()
		if t, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
			// Assume parsed time is in IST and assign proper location
			timestamps[i] = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), ist)
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
			TimeInterval:  c.interval,
		}
	}

	return result
}

// isIST checks if a timestamp is already in IST timezone
func isIST(t time.Time) bool {
	location := t.Location()
	if location == nil {
		return false
	}

	// Check if location name contains IST indicators
	locationName := location.String()
	return locationName == "Asia/Kolkata" ||
		locationName == "IST" ||
		locationName == "Local" && isLocalIST() ||
		(locationName != "UTC" && hasISTOffset(t))
}

// isLocalIST checks if local timezone is IST (for systems running in India)
func isLocalIST() bool {
	now := time.Now()
	_, offset := now.Zone()
	// IST is UTC+5:30 = 19800 seconds
	return offset == 19800
}

// hasISTOffset checks if the timestamp has IST offset (+05:30)
func hasISTOffset(t time.Time) bool {
	_, offset := t.Zone()
	// IST is UTC+5:30 = 19800 seconds
	return offset == 19800
}
