package strategies

import (
	"fmt"
	"time"

	v2 "setbull_trader/internal/strategy/v2"

	"github.com/go-gota/gota/dataframe"
)

// MinimalStrategyV2 is a minimal working V2 strategy for testing
type MinimalStrategyV2 struct {
	*v2.BaseStrategy
}

// NewMinimalStrategyV2 creates a new minimal V2 strategy
func NewMinimalStrategyV2() *MinimalStrategyV2 {
	metadata := v2.StrategyMetadata{
		Name:        "MINIMAL_V2",
		Version:     "1.0.0",
		Description: "Minimal V2 strategy for testing compilation",
		Author:      "Setbull Trader",
		Tags:        []string{"test", "minimal"},
	}

	base := v2.NewBaseStrategy(metadata)
	strategy := &MinimalStrategyV2{
		BaseStrategy: base,
	}

	// Set default configuration
	strategy.Configure(map[string]interface{}{
		"enabled":     true,
		"timeout":     10 * time.Second,
		"max_retries": 3,
		"parameters": map[string]interface{}{
			"test_param": "test_value",
		},
	})

	return strategy
}

// Process implements the minimal strategy processing logic
func (s *MinimalStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
	if df.Nrow() == 0 {
		return df, v2.ErrInvalidDataFrame
	}

	// Simple processing: just log that we processed the data
	fmt.Printf("MinimalStrategyV2: Processed %d rows\n", df.Nrow())

	// Return the original DataFrame (no modifications for minimal strategy)
	return df, nil
}

// GetRequiredHistory returns the number of historical candles required
func (s *MinimalStrategyV2) GetRequiredHistory() int {
	return 10
}

// GetEstimatedProcessingTime returns estimated processing time
func (s *MinimalStrategyV2) GetEstimatedProcessingTime() time.Duration {
	return 50 * time.Millisecond
}

// GetMemoryRequirements returns estimated memory requirements in bytes
func (s *MinimalStrategyV2) GetMemoryRequirements() int64 {
	return 512 * 1024 // 512KB
}
