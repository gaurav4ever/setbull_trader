# V2 Strategy Development Guide

## Overview

This guide explains how to develop new strategies for the V2 Strategy Engine using the Go/Gota DataFrame framework. The V2 system provides a clean, extensible architecture for implementing trading strategies with superior performance compared to the Python-based V1 system.

## Architecture

### Strategy Interface

All V2 strategies must implement the `StrategyV2` interface:

```go
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
```

### Base Strategy

Use `BaseStrategy` to get default implementations for common methods:

```go
type MyStrategy struct {
    *v2.BaseStrategy
    // Strategy-specific fields
}
```

## Creating a New Strategy

### 1. Strategy Structure

```go
package strategies

import (
    "time"
    v2 "setbull_trader/internal/strategy/v2"
    "github.com/go-gota/gota/dataframe"
    "github.com/go-gota/gota/series"
)

type MyStrategyV2 struct {
    *v2.BaseStrategy
    
    // Strategy parameters
    param1 float64
    param2 int
    
    // Strategy state
    state1 bool
    state2 int
}

func NewMyStrategyV2() *MyStrategyV2 {
    metadata := v2.StrategyMetadata{
        Name:        "My_Strategy_V2",
        Version:     "1.0.0",
        Description: "Description of my strategy",
        Author:      "Your Name",
        Tags:        []string{"technical_analysis", "momentum"},
    }

    base := v2.NewBaseStrategy(metadata)
    strategy := &MyStrategyV2{
        BaseStrategy: base,
        param1:       0.5,  // Default value
        param2:       20,   // Default value
    }

    // Set default configuration
    strategy.Configure(map[string]interface{}{
        "enabled":     true,
        "timeout":     10 * time.Second,
        "max_retries": 3,
        "parameters": map[string]interface{}{
            "param1": 0.5,
            "param2": 20,
        },
    })

    return strategy
}
```

### 2. Core Processing Logic

```go
func (s *MyStrategyV2) Process(df *dataframe.DataFrame) (*dataframe.DataFrame, error) {
    if df.Nrow() == 0 {
        return df, v2.ErrInvalidDataFrame
    }

    // Clone the DataFrame to avoid modifying the original
    result := df.Copy()

    // Get required series
    closeSeries := result.Col("close")
    if closeSeries.Err != nil {
        return nil, closeSeries.Err
    }

    // Perform calculations
    calculatedValues := s.calculateMyIndicator(closeSeries)

    // Add new columns to the DataFrame
    result = result.Mutate(series.New(calculatedValues, series.Float, "my_indicator"))

    return &result, nil
}

func (s *MyStrategyV2) calculateMyIndicator(closeSeries series.Series) []float64 {
    closeValues := closeSeries.Float()
    n := len(closeValues)
    result := make([]float64, n)

    for i := 0; i < n; i++ {
        if i < s.param2-1 {
            // Not enough data for calculation
            result[i] = math.NaN()
            continue
        }

        // Your calculation logic here
        // Example: Simple moving average
        sum := 0.0
        for j := i - s.param2 + 1; j <= i; j++ {
            sum += closeValues[j]
        }
        result[i] = sum / float64(s.param2)
    }

    return result
}
```

### 3. Required Methods

```go
// GetRequiredHistory returns the number of historical candles required
func (s *MyStrategyV2) GetRequiredHistory() int {
    return s.param2 + 10 // Your calculation period + buffer
}

// Configure handles strategy-specific configuration
func (s *MyStrategyV2) Configure(config map[string]interface{}) error {
    // Call base configuration first
    if err := s.BaseStrategy.Configure(config); err != nil {
        return err
    }

    // Handle strategy-specific parameters
    if params, ok := config["parameters"].(map[string]interface{}); ok {
        if param1, ok := params["param1"].(float64); ok {
            s.param1 = param1
        }
        if param2, ok := params["param2"].(int); ok {
            s.param2 = param2
        }
    }

    return nil
}

// ValidateConfiguration validates strategy-specific configuration
func (s *MyStrategyV2) ValidateConfiguration() error {
    // Call base validation first
    if err := s.BaseStrategy.ValidateConfiguration(); err != nil {
        return err
    }

    // Validate strategy-specific parameters
    if s.param1 <= 0 {
        return v2.ErrInvalidTimeout // Reusing error for simplicity
    }
    if s.param2 < 2 {
        return v2.ErrInvalidTimeout
    }

    return nil
}

// Performance estimates
func (s *MyStrategyV2) GetEstimatedProcessingTime() time.Duration {
    return 50 * time.Millisecond
}

func (s *MyStrategyV2) GetMemoryRequirements() int64 {
    return 2 * 1024 * 1024 // 2MB
}
```

## DataFrame Operations

### Working with Series

```go
// Get a column
closeSeries := df.Col("close")
if closeSeries.Err != nil {
    return nil, closeSeries.Err
}

// Convert to values
closeValues := closeSeries.Float()
highValues := df.Col("high").Float()
lowValues := df.Col("low").Float()

// Create new series
newValues := make([]float64, len(closeValues))
// ... calculate values ...
newSeries := series.New(newValues, series.Float, "new_column")

// Add to DataFrame
result = result.Mutate(newSeries)
```

### Common Operations

```go
// Filtering
filtered := df.Filter(dataframe.F{
    "close", series.Greater, 100.0,
})

// Sorting
sorted := df.Arrange(dataframe.Sort("timestamp"))

// Selecting columns
selected := df.Select("timestamp", "close", "volume")

// Grouping (if applicable)
grouped := df.GroupBy("category")
```

## Testing Your Strategy

### Unit Tests

```go
func TestMyStrategyV2(t *testing.T) {
    // Create strategy
    strategy := NewMyStrategyV2()
    assert.NotNil(t, strategy)
    assert.Equal(t, "My_Strategy_V2", strategy.GetName())

    // Create test data
    df := createTestData(t)
    
    // Process data
    result, err := strategy.Process(df)
    assert.NoError(t, err)
    assert.NotNil(t, result)

    // Verify results
    assert.True(t, result.HasCol("my_indicator"))
    
    // Check specific values
    indicatorCol := result.Col("my_indicator")
    assert.NoError(t, indicatorCol.Err)
    values := indicatorCol.Float()
    assert.True(t, len(values) > 0)
}
```

### Performance Tests

```go
func BenchmarkMyStrategyV2(b *testing.B) {
    strategy := NewMyStrategyV2()
    df := createBenchmarkData(1000)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := strategy.Process(df)
        if err != nil {
            b.Fatalf("Strategy processing failed: %v", err)
        }
    }
}
```

## Configuration

### Strategy Configuration

Strategies can be configured via YAML:

```yaml
strategy_engine_v2:
  strategies:
    my_strategy:
      enabled: true
      timeout: 10s
      max_retries: 3
      parameters:
        param1: 0.5
        param2: 20
```

### Runtime Configuration

```go
// Configure strategy at runtime
config := map[string]interface{}{
    "enabled": true,
    "parameters": map[string]interface{}{
        "param1": 0.7,
        "param2": 30,
    },
}
err := strategy.Configure(config)
```

## Best Practices

### 1. Performance
- Use efficient algorithms for calculations
- Minimize memory allocations
- Profile your strategy with benchmarks
- Target < 50ms processing time for 1000 candles

### 2. Error Handling
- Always check for DataFrame errors
- Handle insufficient data gracefully
- Return meaningful error messages
- Use NaN for invalid calculations

### 3. Memory Management
- Clone DataFrames to avoid modifying originals
- Reuse slices when possible
- Estimate memory requirements accurately
- Clean up resources in long-running strategies

### 4. Testing
- Write comprehensive unit tests
- Test edge cases (empty data, insufficient data)
- Benchmark performance
- Test configuration validation

### 5. Documentation
- Document your strategy logic
- Explain parameters and their effects
- Provide usage examples
- Include performance characteristics

## Migration from Python

### Key Differences

1. **DataFrames**: Gota vs Pandas
   - Gota is more memory-efficient
   - Different API for operations
   - Strong typing with Series types

2. **Performance**: Go vs Python
   - Much faster execution
   - Lower memory usage
   - Better concurrency support

3. **Error Handling**: Explicit vs Implicit
   - Check errors explicitly
   - Return error values
   - Use NaN for invalid data

### Migration Checklist

- [ ] Port calculation logic
- [ ] Adapt DataFrame operations
- [ ] Implement error handling
- [ ] Add configuration support
- [ ] Write comprehensive tests
- [ ] Benchmark performance
- [ ] Document the strategy

## Example: BB Width Strategy

See `bb_width_enhanced_strategy.go` for a complete example of a strategy that:
- Calculates Bollinger Bands
- Detects squeeze conditions
- Generates entry signals
- Handles configuration
- Includes comprehensive testing

## Performance Targets

- **Processing Time**: < 50ms for 1000 candles
- **Memory Usage**: < 10MB per strategy
- **Concurrency**: Support parallel execution
- **Accuracy**: 100% match with Python results

## Support

For questions or issues:
1. Check existing strategy examples
2. Review test cases
3. Run performance benchmarks
4. Consult the V2 architecture documentation 