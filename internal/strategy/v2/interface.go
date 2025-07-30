package v2

import (
	"errors"
	"time"

	"github.com/go-gota/gota/dataframe"
)

// Error definitions
var (
	ErrInvalidTimeout    = errors.New("invalid timeout value")
	ErrInvalidMaxRetries = errors.New("invalid max retries value")
	ErrStrategyDisabled  = errors.New("strategy is disabled")
	ErrInvalidDataFrame  = errors.New("invalid dataframe")
)

// StrategyV2 defines the interface for V2 strategies
type StrategyV2 interface {
	// Core methods
	Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error)
	GetRequiredHistory() int
	GetName() string
	GetVersion() string

	// Configuration
	Configure(config map[string]interface{}) error
	ValidateConfiguration() error

	// Metadata
	GetDescription() string
	GetAuthor() string
	GetTags() []string

	// Performance
	GetEstimatedProcessingTime() time.Duration
	GetMemoryRequirements() int64

	// Status
	IsEnabled() bool
}

// StrategyMetadata contains metadata about a strategy
type StrategyMetadata struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

// StrategyConfig contains configuration for a strategy
type StrategyConfig struct {
	Enabled    bool                   `json:"enabled"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration          `json:"timeout"`
	MaxRetries int                    `json:"max_retries"`
}

// StrategyResult contains the result of strategy processing
type StrategyResult struct {
	StrategyName   string                 `json:"strategy_name"`
	ProcessingTime time.Duration          `json:"processing_time"`
	MemoryUsed     int64                  `json:"memory_used"`
	RowsProcessed  int                    `json:"rows_processed"`
	ColumnsAdded   []string               `json:"columns_added"`
	Error          error                  `json:"error,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// BaseStrategy provides a base implementation for common strategy functionality
type BaseStrategy struct {
	metadata StrategyMetadata
	config   StrategyConfig
}

// NewBaseStrategy creates a new base strategy
func NewBaseStrategy(metadata StrategyMetadata) *BaseStrategy {
	return &BaseStrategy{
		metadata: metadata,
		config: StrategyConfig{
			Enabled:    true,
			Parameters: make(map[string]interface{}),
			Timeout:    10 * time.Second,
			MaxRetries: 3,
		},
	}
}

// GetName returns the strategy name
func (b *BaseStrategy) GetName() string {
	return b.metadata.Name
}

// GetVersion returns the strategy version
func (b *BaseStrategy) GetVersion() string {
	return b.metadata.Version
}

// GetDescription returns the strategy description
func (b *BaseStrategy) GetDescription() string {
	return b.metadata.Description
}

// GetAuthor returns the strategy author
func (b *BaseStrategy) GetAuthor() string {
	return b.metadata.Author
}

// GetTags returns the strategy tags
func (b *BaseStrategy) GetTags() []string {
	return b.metadata.Tags
}

// Configure sets the strategy configuration
func (b *BaseStrategy) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		b.config.Enabled = enabled
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		b.config.Timeout = timeout
	}
	if maxRetries, ok := config["max_retries"].(int); ok {
		b.config.MaxRetries = maxRetries
	}
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		b.config.Parameters = parameters
	}
	return nil
}

// ValidateConfiguration validates the strategy configuration
func (b *BaseStrategy) ValidateConfiguration() error {
	// Base validation - can be overridden by specific strategies
	if b.config.Timeout <= 0 {
		return ErrInvalidTimeout
	}
	if b.config.MaxRetries < 0 {
		return ErrInvalidMaxRetries
	}
	return nil
}

// GetEstimatedProcessingTime returns estimated processing time
func (b *BaseStrategy) GetEstimatedProcessingTime() time.Duration {
	// Base implementation - can be overridden by specific strategies
	return 100 * time.Millisecond
}

// GetMemoryRequirements returns estimated memory requirements in bytes
func (b *BaseStrategy) GetMemoryRequirements() int64 {
	// Base implementation - can be overridden by specific strategies
	return 1024 * 1024 // 1MB default
}

// GetRequiredHistory returns the number of historical candles required
func (b *BaseStrategy) GetRequiredHistory() int {
	// Base implementation - can be overridden by specific strategies
	return 20
}

// IsEnabled returns whether the strategy is enabled
func (b *BaseStrategy) IsEnabled() bool {
	return b.config.Enabled
}

// GetConfig returns the strategy configuration
func (b *BaseStrategy) GetConfig() StrategyConfig {
	return b.config
}
