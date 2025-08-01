package strategies

import (
	"fmt"
	"math"
	"time"

	"github.com/go-gota/gota/dataframe"

	v2 "setbull_trader/internal/strategy/v2"
)

// BBWidthEntryStrategyV2 implements the Bollinger Bands width-based entry strategy
// This strategy detects volatility squeeze conditions using BB width analysis
// Focus: Signal generation and parameter tracking only (no position sizing, risk management, or trade management)
type BBWidthEntryStrategyV2 struct {
	*v2.BaseStrategy

	// BB Width strategy specific parameters
	bbWidthThreshold   float64
	bbPeriod           int
	bbStdDev           float64
	squeezeDurationMin int
	squeezeDurationMax int

	// Strategy state (for signal generation only)
	canGenerateLong  bool
	canGenerateShort bool

	// BB values and calculations
	bbUpper            float64
	bbLower            float64
	bbMiddle           float64
	currentBBWidth     float64
	lowestBBWidth      float64
	squeezeDetected    bool
	squeezeStartTime   *time.Time
	squeezeCandleCount int

	// BB width history tracking
	bbWidthHistory   []float64
	maxHistoryLength int

	// Parameters manager for state persistence
	paramsManager *v2.StrategyParametersManager
}

// NewBBWidthEntryStrategyV2 creates a new BB width entry strategy instance
func NewBBWidthEntryStrategyV2(paramsManager *v2.StrategyParametersManager) *BBWidthEntryStrategyV2 {
	return &BBWidthEntryStrategyV2{
		BaseStrategy: v2.NewBaseStrategy(v2.StrategyMetadata{
			Name:        "BB_WIDTH_ENTRY",
			Version:     "1.0.0",
			Description: "Bollinger Bands width-based entry strategy with squeeze detection",
			Author:      "Setbull Trader",
			Tags:        []string{"bollinger-bands", "squeeze", "volatility"},
		}),

		// Default parameters
		bbWidthThreshold:   0.2, // 20% default
		bbPeriod:           20,  // 20 period default
		bbStdDev:           2.0, // 2 standard deviations default
		squeezeDurationMin: 3,   // Minimum 3 candles
		squeezeDurationMax: 5,   // Maximum 5 candles

		// Strategy state
		canGenerateLong:  true,
		canGenerateShort: true,

		// BB values
		lowestBBWidth:      math.Inf(1),
		squeezeDetected:    false,
		squeezeCandleCount: 0,

		// BB width history
		bbWidthHistory:   make([]float64, 0),
		maxHistoryLength: 50, // Keep last 50 candles

		// Parameters manager
		paramsManager: paramsManager,
	}
}

// Process implements the StrategyV2 interface
// Processes the DataFrame and generates signals based on BB width squeeze detection
func (s *BBWidthEntryStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	if df.Nrow() == 0 {
		return df, nil
	}

	// Clone the DataFrame to avoid modifying the original
	result := df.Copy()

	// Get required series
	highSeries := result.Col("high")
	lowSeries := result.Col("low")
	timestampSeries := result.Col("timestamp")
	bbUpperSeries := result.Col("bb_upper_x")
	bbLowerSeries := result.Col("bb_lower_x")
	bbMiddleSeries := result.Col("bb_middle_x")

	if highSeries.Err != nil || lowSeries.Err != nil || timestampSeries.Err != nil ||
		bbUpperSeries.Err != nil || bbLowerSeries.Err != nil || bbMiddleSeries.Err != nil {
		return nil, fmt.Errorf("failed to get required series: %v", highSeries.Err)
	}

	// Load previous state from parameters
	s.loadStateFromParameters()

	// Process each candle
	for i := 0; i < df.Nrow(); i++ {
		// Get current candle data
		high := highSeries.Float()[i]
		low := lowSeries.Float()[i]
		timestamp := timestampSeries.Elem(i).String()
		bbUpper := bbUpperSeries.Float()[i]
		bbLower := bbLowerSeries.Float()[i]
		bbMiddle := bbMiddleSeries.Float()[i]

		// Parse timestamp
		candleTime, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			continue // Skip invalid timestamps
		}

		// Get or create strategy parameters for this candle
		params, err := s.getOrCreateParameters(candleTime)
		if err != nil {
			continue // Skip if we can't get parameters
		}

		// Update strategy state from parameters
		s.updateStateFromParameters(params)

		// Validate BB data
		if !s.validateBBData(bbUpper, bbLower, bbMiddle) {
			continue // Skip invalid BB data
		}

		// Extract BB values
		s.bbUpper = bbUpper
		s.bbLower = bbLower
		s.bbMiddle = bbMiddle
		s.currentBBWidth = bbUpper - bbLower
		s.currentBBWidth = math.Round(s.currentBBWidth*100) / 100

		// Update BB width history
		s.updateBBWidthHistory(s.currentBBWidth)

		// Get lowest BB width (simplified for now - could be enhanced with CSV lookup)
		s.lowestBBWidth = s.getLowestBBWidth()

		// Check for squeeze condition
		s.checkSqueezeCondition(candleTime, params)

		// Check entry conditions if squeeze duration is within range
		if s.squeezeDetected && s.squeezeCandleCount >= s.squeezeDurationMin {
			signal := s.checkEntryConditions(high, low, candleTime, params)

			// Update parameters with new state
			s.updateParametersWithState(params, signal)

			// Save parameters
			s.saveParameters(candleTime, params)

			// Log signal if generated (for data tracking only)
			if signal != nil {
				fmt.Printf("BB_WIDTH_ENTRY Signal Generated at %s: %s %s at %.2f (BB Width: %.2f, Squeeze: %d candles)\n",
					candleTime.Format("15:04"), signal.Type, signal.Direction, signal.Price, s.currentBBWidth, s.squeezeCandleCount)
			}
		}
	}

	return &result, nil
}

// validateBBData validates that required BB data is present and valid
func (s *BBWidthEntryStrategyV2) validateBBData(bbUpper, bbLower, bbMiddle float64) bool {
	// Check for invalid values
	if bbUpper <= 0 || bbLower <= 0 || bbMiddle <= 0 {
		return false
	}

	// Check for NaN or infinite values
	if math.IsNaN(bbUpper) || math.IsNaN(bbLower) || math.IsNaN(bbMiddle) ||
		math.IsInf(bbUpper, 0) || math.IsInf(bbLower, 0) || math.IsInf(bbMiddle, 0) {
		return false
	}

	// Validate BB relationships
	if bbUpper < bbLower {
		return false
	}

	if !(bbLower < bbMiddle && bbMiddle < bbUpper) {
		return false
	}

	return true
}

// updateBBWidthHistory updates BB width history for lowest calculation
func (s *BBWidthEntryStrategyV2) updateBBWidthHistory(bbWidth float64) {
	if bbWidth > 0 {
		s.bbWidthHistory = append(s.bbWidthHistory, bbWidth)

		// Keep only the last maxHistoryLength values
		if len(s.bbWidthHistory) > s.maxHistoryLength {
			s.bbWidthHistory = s.bbWidthHistory[len(s.bbWidthHistory)-s.maxHistoryLength:]
		}
	}
}

// getLowestBBWidth gets the lowest BB width from history
func (s *BBWidthEntryStrategyV2) getLowestBBWidth() float64 {
	if len(s.bbWidthHistory) == 0 {
		return 0.001 // Default fallback value
	}

	lowest := s.bbWidthHistory[0]
	for _, width := range s.bbWidthHistory {
		if width < lowest {
			lowest = width
		}
	}

	return lowest
}

// checkSqueezeCondition checks for squeeze condition based on BB width
func (s *BBWidthEntryStrategyV2) checkSqueezeCondition(candleTime time.Time, params *v2.StrategyParameters) {
	// Calculate squeeze threshold
	squeezeThreshold := s.lowestBBWidth * (1 + s.bbWidthThreshold)
	squeezeThreshold = math.Round(squeezeThreshold*100) / 100

	if s.currentBBWidth <= squeezeThreshold {
		// Squeeze condition detected
		if !s.squeezeDetected {
			// Start new squeeze
			s.squeezeDetected = true
			s.squeezeStartTime = &candleTime
			s.squeezeCandleCount = 1
			fmt.Printf("BB_WIDTH_ENTRY Squeeze detected at %s: Width=%.6f, Threshold=%.6f\n",
				candleTime.Format("15:04"), s.currentBBWidth, squeezeThreshold)
		} else {
			// Continue existing squeeze
			s.squeezeCandleCount++
		}
	} else {
		// No squeeze condition
		if s.squeezeDetected {
			fmt.Printf("BB_WIDTH_ENTRY Squeeze ended at %s: Width=%.6f\n",
				candleTime.Format("15:04"), s.currentBBWidth)
		}
		s.squeezeDetected = false
		s.squeezeCandleCount = 0
		s.squeezeStartTime = nil
	}
}

// checkEntryConditions checks for entry conditions during squeeze
func (s *BBWidthEntryStrategyV2) checkEntryConditions(high, low float64, timestamp time.Time, params *v2.StrategyParameters) *EntrySignal {
	// Check if already in a trade (simplified - just check signal generation flags)
	if !s.canGenerateLong && !s.canGenerateShort {
		return nil
	}

	// For now, use a simple direction bias (could be enhanced with configuration)
	direction := "BULLISH" // Default, could be made configurable

	// Check for long entry (price above BB upper band)
	if direction == "BULLISH" && s.canGenerateLong && high > s.bbUpper {
		s.canGenerateLong = false
		return &EntrySignal{
			Type:      "BB_WIDTH_ENTRY",
			Direction: "LONG",
			Price:     s.bbUpper,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"entry_type":           "bb_width_entry",
				"entry_time":           timestamp.Format("15:04"),
				"signal_purpose":       "data_tracking",
				"bb_upper":             s.bbUpper,
				"bb_lower":             s.bbLower,
				"bb_middle":            s.bbMiddle,
				"bb_width":             s.currentBBWidth,
				"lowest_bb_width":      s.lowestBBWidth,
				"squeeze_duration":     s.squeezeCandleCount,
				"squeeze_start_time":   s.squeezeStartTime.Format("2006-01-02 15:04:05"),
				"squeeze_detected":     s.squeezeDetected,
				"squeeze_candle_count": s.squeezeCandleCount,
			},
		}
	}

	// Check for short entry (price below BB lower band)
	if direction == "BEARISH" && s.canGenerateShort && low < s.bbLower {
		s.canGenerateShort = false
		return &EntrySignal{
			Type:      "BB_WIDTH_ENTRY",
			Direction: "SHORT",
			Price:     s.bbLower,
			Timestamp: timestamp,
			Metadata: map[string]interface{}{
				"entry_type":           "bb_width_entry",
				"entry_time":           timestamp.Format("15:04"),
				"signal_purpose":       "data_tracking",
				"bb_upper":             s.bbUpper,
				"bb_lower":             s.bbLower,
				"bb_middle":            s.bbMiddle,
				"bb_width":             s.currentBBWidth,
				"lowest_bb_width":      s.lowestBBWidth,
				"squeeze_duration":     s.squeezeCandleCount,
				"squeeze_start_time":   s.squeezeStartTime.Format("2006-01-02 15:04:05"),
				"squeeze_detected":     s.squeezeDetected,
				"squeeze_candle_count": s.squeezeCandleCount,
			},
		}
	}

	return nil
}

// getOrCreateParameters gets or creates parameters for a specific timestamp
func (s *BBWidthEntryStrategyV2) getOrCreateParameters(timestamp time.Time) (*v2.StrategyParameters, error) {
	if s.paramsManager == nil {
		// Return default parameters if no manager is available
		return &v2.StrategyParameters{
			CanGenerateLong:  s.canGenerateLong,
			CanGenerateShort: s.canGenerateShort,
		}, nil
	}

	// Try to get existing parameters
	params, err := s.paramsManager.GetParameters("", "BB_WIDTH_ENTRY", timestamp)
	if err != nil {
		// Create new parameters if none exist
		params = &v2.StrategyParameters{
			CanGenerateLong:  s.canGenerateLong,
			CanGenerateShort: s.canGenerateShort,
		}
	}

	return params, nil
}

// updateStateFromParameters updates strategy state from parameters
func (s *BBWidthEntryStrategyV2) updateStateFromParameters(params *v2.StrategyParameters) {
	s.canGenerateLong = params.CanGenerateLong
	s.canGenerateShort = params.CanGenerateShort
}

// updateParametersWithState updates parameters with current state
func (s *BBWidthEntryStrategyV2) updateParametersWithState(params *v2.StrategyParameters, signal *EntrySignal) {
	params.CanGenerateLong = s.canGenerateLong
	params.CanGenerateShort = s.canGenerateShort

	// Add strategy-specific parameters
	params.BBWidthThreshold = &s.bbWidthThreshold
	params.BBPeriod = &s.bbPeriod
	params.BBStdDev = &s.bbStdDev
	params.CurrentBBWidth = &s.currentBBWidth
	params.LowestBBWidth = &s.lowestBBWidth
	params.SqueezeDetected = s.squeezeDetected
	params.SqueezeStartTime = s.squeezeStartTime
	params.SqueezeCandleCount = &s.squeezeCandleCount
	params.BBUpper = &s.bbUpper
	params.BBLower = &s.bbLower
	params.BBMiddle = &s.bbMiddle

	// Add signal metadata if signal was generated
	if signal != nil {
		if params.Metadata == nil {
			params.Metadata = make(map[string]interface{})
		}
		params.Metadata["last_signal"] = signal
	}
}

// saveParameters saves parameters to the database
func (s *BBWidthEntryStrategyV2) saveParameters(timestamp time.Time, params *v2.StrategyParameters) {
	if s.paramsManager != nil {
		s.paramsManager.SaveParameters("", "BB_WIDTH_ENTRY", timestamp, params)
	}
}

// loadStateFromParameters loads strategy state from parameters
func (s *BBWidthEntryStrategyV2) loadStateFromParameters() {
	if s.paramsManager == nil {
		return
	}

	// For now, we'll skip loading previous state since GetLatestParameters is not implemented
	// In a real implementation, this would load the most recent parameters for this strategy
	// params, err := s.paramsManager.GetLatestParameters(stockID, "BB_WIDTH_ENTRY")
	// if err == nil && params != nil {
	//     s.updateStateFromParameters(params)
	// }
}

// GetRequiredHistory returns the number of candles required for this strategy
func (s *BBWidthEntryStrategyV2) GetRequiredHistory() int {
	return s.bbPeriod + 10 // BB period + some buffer for calculations
}

// GetName returns the strategy name
func (s *BBWidthEntryStrategyV2) GetName() string {
	return "BB_WIDTH_ENTRY"
}

// Configure configures the strategy with parameters
func (s *BBWidthEntryStrategyV2) Configure(config map[string]interface{}) error {
	if params, ok := config["parameters"].(map[string]interface{}); ok {
		// Parse BB width threshold
		if bbWidthThreshold, ok := params["bb_width_threshold"].(float64); ok {
			s.bbWidthThreshold = bbWidthThreshold
		}

		// Parse BB period
		if bbPeriod, ok := params["bb_period"].(int); ok {
			s.bbPeriod = bbPeriod
		}

		// Parse BB standard deviation
		if bbStdDev, ok := params["bb_std_dev"].(float64); ok {
			s.bbStdDev = bbStdDev
		}

		// Parse squeeze duration min
		if squeezeDurationMin, ok := params["squeeze_duration_min"].(int); ok {
			s.squeezeDurationMin = squeezeDurationMin
		}

		// Parse squeeze duration max
		if squeezeDurationMax, ok := params["squeeze_duration_max"].(int); ok {
			s.squeezeDurationMax = squeezeDurationMax
		}
	}

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *BBWidthEntryStrategyV2) ValidateConfiguration() error {
	if s.bbWidthThreshold <= 0 || s.bbWidthThreshold > 1.0 {
		return fmt.Errorf("bb_width_threshold must be between 0 and 1.0")
	}

	if s.bbPeriod <= 0 || s.bbPeriod > 100 {
		return fmt.Errorf("bb_period must be between 1 and 100")
	}

	if s.bbStdDev <= 0 || s.bbStdDev > 5.0 {
		return fmt.Errorf("bb_std_dev must be between 0 and 5.0")
	}

	if s.squeezeDurationMin <= 0 || s.squeezeDurationMin > s.squeezeDurationMax {
		return fmt.Errorf("squeeze_duration_min must be positive and less than squeeze_duration_max")
	}

	return nil
}

// GetEstimatedProcessingTime returns the estimated processing time
func (s *BBWidthEntryStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 50 * time.Millisecond // < 50ms per stock group
}

// GetMemoryRequirements returns the memory requirements
func (s *BBWidthEntryStrategyV2) GetMemoryRequirements() int64 {
	return 512 * 1024 // 512KB per strategy instance
}

// ResetState resets the strategy state
func (s *BBWidthEntryStrategyV2) ResetState() {
	s.canGenerateLong = true
	s.canGenerateShort = true
	s.squeezeDetected = false
	s.squeezeStartTime = nil
	s.squeezeCandleCount = 0
	s.lowestBBWidth = math.Inf(1)
	s.bbUpper = 0
	s.bbLower = 0
	s.bbMiddle = 0
	s.currentBBWidth = 0
	s.bbWidthHistory = make([]float64, 0)
}
