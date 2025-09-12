package strategies

import (
	"time"

	v2 "setbull_trader/internal/strategy/v2"
	"setbull_trader/pkg/log"

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
	startTime := time.Now()

	// Log strategy processing start
	log.Info("MINIMAL_V2: Starting strategy processing - strategy=%s version=%s data_rows=%d",
		s.GetName(),
		s.GetVersion(),
		df.Nrow())

	if df.Nrow() == 0 {
		log.Error("MINIMAL_V2: Invalid dataframe - no rows - strategy=%s error=%s",
			s.GetName(),
			v2.ErrInvalidDataFrame.Error())
		return df, v2.ErrInvalidDataFrame
	}

	// Simple processing: just log that we processed the data
	log.Info("MINIMAL_V2: Processing data - strategy=%s rows=%d",
		s.GetName(),
		df.Nrow())

	processingTime := time.Since(startTime)

	// Log processing completion
	log.Info("MINIMAL_V2: Strategy processing completed - strategy=%s processing_time_ms=%d candles_processed=%d",
		s.GetName(),
		processingTime.Milliseconds(),
		df.Nrow())

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

// Configure configures the strategy with new parameters
func (s *MinimalStrategyV2) Configure(config map[string]interface{}) error {
	log.Info("MINIMAL_V2: Starting configuration - strategy=%s config_keys=%d",
		s.GetName(),
		len(config))

	// Call base configuration
	err := s.BaseStrategy.Configure(config)
	if err != nil {
		log.Error("MINIMAL_V2: Base configuration failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	log.Info("MINIMAL_V2: Configuration completed successfully - strategy=%s",
		s.GetName())

	return nil
}

// ValidateConfiguration validates the strategy configuration
func (s *MinimalStrategyV2) ValidateConfiguration() error {
	log.Debug("MINIMAL_V2: Starting configuration validation - strategy=%s",
		s.GetName())

	// Call base validation
	err := s.BaseStrategy.ValidateConfiguration()
	if err != nil {
		log.Error("MINIMAL_V2: Base configuration validation failed - strategy=%s error=%s",
			s.GetName(),
			err.Error())
		return err
	}

	log.Info("MINIMAL_V2: Configuration validation completed successfully - strategy=%s",
		s.GetName())

	return nil
}
