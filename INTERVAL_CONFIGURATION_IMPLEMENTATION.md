# Interval Configuration and Date Range Control Implementation

## Overview
Implemented intelligent interval configuration that ensures your software maintains full control over API request sizes to the Upstox API, regardless of the date range requested via your API.

## Configuration Summary

### Base Interval Configurations
- **1-minute data**: 20 days per batch (as requested)
- **Daily data**: 100 days per batch (as requested)
- **5-minute data**: 30 days per batch
- **15-minute data**: 40 days per batch
- **1-hour data**: 60 days per batch
- **Unknown intervals**: 20 days (conservative default)

### Cache-Based Optimization
When cached data indicates recent data exists, the system reduces batch sizes for safety:
- **1-minute recent data**: 15 days per batch (25% reduction)
- **Daily recent data**: 75 days per batch (25% reduction)
- **Other intervals**: 25% reduction from base configuration

## Smart Date Range Control

### Automatic Range Division
When a date range requested via API exceeds the allowed interval days:

1. **Detection**: System calculates total requested days vs. max allowed days
2. **Division**: Automatically divides the range into multiple batches
3. **Sequential Processing**: Processes each batch sequentially with controlled delays
4. **Logging**: Provides detailed logs about batch division and processing

### Example Scenarios

#### Scenario 1: 1-minute data for 60 days requested
- **Requested**: 60 days of 1-minute data
- **Max allowed**: 20 days per batch
- **Result**: Automatically divided into 3 batches (20 + 20 + 20 days)
- **Processing**: 3 sequential API calls with 1-second delays between them

#### Scenario 2: Daily data for 300 days requested
- **Requested**: 300 days of daily data
- **Max allowed**: 100 days per batch
- **Result**: Automatically divided into 3 batches (100 + 100 + 100 days)
- **Processing**: 3 sequential API calls with 1-second delays between them

## Rate Limiting and API Control

### Fixed Delay Strategy
- **Base delay**: 1 second between all API batches
- **Error recovery**: Additional 1 second delay after failed requests
- **No variable delays**: Consistent timing regardless of success/failure
- **Full control**: Your software controls all API timing, not dependent on external factors

### Key Benefits
1. **Predictable API usage**: Consistent 1-second intervals
2. **Rate limit compliance**: Never exceeds your configured limits
3. **Error resilience**: Graceful handling of failed requests
4. **Scalable**: Works for any date range size
5. **Transparent**: Detailed logging of all batch operations

## Implementation Details

### Code Locations
- **Interval configuration**: `getOptimalIntervalDays()` method in `BatchFetchService`
- **Range validation**: Date range calculation in `processInstrumentWithIntervals()`
- **Batch processing**: Sequential processing loop with controlled delays
- **Cache optimization**: Smart interval reduction based on existing data

### Logging Examples
```
[BATCH] Total requested days: 60, Max allowed per batch: 20 days for 1minute interval
[BATCH] Requested range (60 days) exceeds max allowed (20 days). Will process in 3 batches with 1s delays
[BATCH] Processing optimized interval for STOCK: 2025-01-01 to 2025-01-21 (20 days)
[BATCH] Applying controlled 1-second delay between API batches for STOCK
[BATCH] Processing optimized interval for STOCK: 2025-01-22 to 2025-02-11 (20 days)
```

## Configuration Validation

The system now ensures:
- ✅ 1-minute data: Maximum 20 days per API call
- ✅ Daily data: Maximum 100 days per API call
- ✅ Automatic range division for oversized requests
- ✅ Consistent 1-second delays between API calls
- ✅ Error recovery with additional delays
- ✅ Cache-based optimization for recent data
- ✅ Detailed logging for monitoring and debugging

## Usage Impact

### For API Users
- Can request any date range via your API
- System automatically handles chunking and rate limiting
- Transparent processing with detailed progress logs
- No need to worry about Upstox API limits

### For System Operations
- Full control over API call frequency
- Predictable load on Upstox API
- Better error handling and recovery
- Improved monitoring and debugging capabilities

This implementation ensures your software maintains complete control over the interaction with the Upstox API while providing a seamless experience for users requesting data of any date range.
