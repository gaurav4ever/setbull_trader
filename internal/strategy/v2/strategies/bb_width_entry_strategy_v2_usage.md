# BB_WIDTH_ENTRY Strategy Usage Guide

## Overview

The **BB_WIDTH_ENTRY** strategy is a Bollinger Bands width-based entry strategy that detects volatility squeeze conditions and generates entry signals when price breaks out of the squeeze. This strategy focuses on **signal generation and parameter tracking only** - it does not handle position sizing, risk management, or trade management.

## Strategy Logic

### Core Concept
The strategy identifies "squeeze" conditions when the Bollinger Bands width becomes very narrow, indicating low volatility. When price breaks out of this squeeze, it generates entry signals in the direction of the breakout.

### Key Components

1. **Bollinger Bands Width Calculation**
   - Calculates BB width as: `BB_Upper - BB_Lower`
   - Tracks the lowest BB width over time for threshold comparison

2. **Squeeze Detection**
   - Detects squeeze when current BB width ≤ lowest BB width × (1 + threshold)
   - Default threshold: 20% (0.2)
   - Minimum squeeze duration: 3 candles
   - Maximum squeeze duration: 5 candles

3. **Entry Conditions**
   - **Long Entry**: Price breaks above BB upper band during squeeze
   - **Short Entry**: Price breaks below BB lower band during squeeze
   - Only generates one signal per squeeze period

## Configuration

### Default Parameters
```yaml
bb_width_threshold: 0.2      # 20% threshold for squeeze detection
bb_period: 20               # Bollinger Bands period
bb_std_dev: 2.0             # Standard deviation multiplier
squeeze_duration_min: 3     # Minimum squeeze duration (candles)
squeeze_duration_max: 5     # Maximum squeeze duration (candles)
```

### Parameter Descriptions

- **bb_width_threshold**: Percentage above lowest BB width to trigger squeeze detection
- **bb_period**: Number of periods for Bollinger Bands calculation
- **bb_std_dev**: Standard deviation multiplier for BB bands
- **squeeze_duration_min**: Minimum number of candles in squeeze before entry
- **squeeze_duration_max**: Maximum number of candles in squeeze before entry

## Usage Examples

### Basic Usage
```go
// Create strategy instance
strategy := NewBBWidthEntryStrategyV2(paramsManager)

// Configure with custom parameters
config := map[string]interface{}{
    "parameters": map[string]interface{}{
        "bb_width_threshold":    0.3,
        "bb_period":             30,
        "bb_std_dev":            2.5,
        "squeeze_duration_min":  4,
        "squeeze_duration_max":  6,
    },
}
strategy.Configure(config)

// Process DataFrame
result, err := strategy.Process(&df)
if err != nil {
    log.Printf("Error processing data: %v", err)
}
```

### Integration with V2 Engine
```go
// Register strategy with V2 engine
engine := v2.NewStrategyEngineV2()
strategy := NewBBWidthEntryStrategyV2(paramsManager)
engine.RegisterStrategy(strategy)

// Process stock group
stockGroup := []string{"RELIANCE", "TCS", "INFY"}
result := engine.ProcessStockGroup(stockGroup, df)
```

## Signal Structure

### Signal Metadata
```go
type EntrySignal struct {
    Type      string                 // "BB_WIDTH_ENTRY"
    Direction string                 // "LONG" or "SHORT"
    Price     float64                // Entry price
    Timestamp time.Time              // Signal timestamp
    Metadata  map[string]interface{} // Strategy-specific data
}
```

### Signal Metadata Fields
- **entry_type**: "bb_width_entry"
- **entry_time**: Time in HH:MM format
- **signal_purpose**: "data_tracking"
- **bb_upper**: BB upper band value
- **bb_lower**: BB lower band value
- **bb_middle**: BB middle band value
- **bb_width**: Current BB width
- **lowest_bb_width**: Lowest BB width in history
- **squeeze_duration**: Number of candles in squeeze
- **squeeze_start_time**: When squeeze started
- **squeeze_detected**: Whether squeeze is active
- **squeeze_candle_count**: Current squeeze candle count

## Entry Conditions

### Long Entry
- Squeeze detected and duration ≥ minimum
- Price high > BB upper band
- Can generate long signal is true
- Direction bias is "BULLISH"

### Short Entry
- Squeeze detected and duration ≥ minimum
- Price low < BB lower band
- Can generate short signal is true
- Direction bias is "BEARISH"

## Performance Characteristics

### Processing Time
- **Estimated**: < 50ms per stock group
- **Memory**: 512KB per strategy instance
- **History Required**: 30 candles (BB period + 10)

### Scalability
- Supports parallel processing
- State persistence across candles
- Parameter caching for performance

## State Management

### Strategy State
- **canGenerateLong**: Whether long signals can be generated
- **canGenerateShort**: Whether short signals can be generated
- **squeezeDetected**: Whether squeeze is currently active
- **squeezeStartTime**: When current squeeze started
- **squeezeCandleCount**: Number of candles in current squeeze

### State Persistence
- State is saved to database for each candle
- Parameters are cached for faster retrieval
- State is restored when processing resumes

## Testing

### Unit Tests
```bash
go test ./internal/strategy/v2/strategies -run TestBBWidthEntryStrategy
```

### Test Coverage
- Strategy initialization and configuration
- BB data validation
- Squeeze detection logic
- Entry condition checking
- State management
- Performance benchmarks

## Best Practices

### Configuration
1. **Adjust threshold based on market conditions**
   - Lower threshold (0.1-0.2) for volatile markets
   - Higher threshold (0.3-0.4) for stable markets

2. **Set appropriate squeeze duration**
   - Longer duration for more reliable signals
   - Shorter duration for more frequent signals

3. **Monitor performance**
   - Track processing time per stock group
   - Monitor memory usage
   - Validate signal accuracy

### Integration
1. **Use with parameter manager**
   - Enable state persistence
   - Cache parameters for performance
   - Monitor parameter updates

2. **Combine with other strategies**
   - Use as confirmation for other signals
   - Combine with trend analysis
   - Integrate with risk management

## Troubleshooting

### Common Issues

1. **No signals generated**
   - Check BB data availability
   - Verify squeeze threshold settings
   - Ensure minimum squeeze duration

2. **Too many signals**
   - Increase squeeze duration minimum
   - Adjust BB width threshold
   - Check signal generation flags

3. **Performance issues**
   - Monitor DataFrame size
   - Check memory usage
   - Validate BB calculations

### Debug Information
- Strategy logs squeeze detection events
- Signal generation is logged with details
- BB width calculations are tracked
- State transitions are monitored

## Limitations

### Current Limitations
1. **Direction bias is hardcoded** (defaults to "BULLISH")
2. **No CSV lookup for lowest BB width** (uses in-memory history)
3. **Limited to 5-minute candles** (not configurable)
4. **No advanced filtering** (relies on basic BB data)

### Future Enhancements
1. **Configurable direction bias**
2. **External BB width data source**
3. **Multiple timeframe support**
4. **Advanced filtering options**
5. **Machine learning integration**

## Conclusion

The BB_WIDTH_ENTRY strategy provides a robust foundation for volatility squeeze-based trading. It focuses on signal generation and parameter tracking, making it suitable for integration with comprehensive trading systems. The strategy is designed for real-time processing and can handle multiple stocks efficiently.

For production use, consider:
- Implementing proper risk management
- Adding position sizing logic
- Integrating with trade management systems
- Monitoring and alerting systems
- Performance optimization for large datasets 