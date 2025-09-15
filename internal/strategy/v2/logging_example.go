package v2

import (
	"fmt"
	"time"
)

// ExampleUsage demonstrates how to use the Phase 1 logging system
func ExampleUsage() {
	// Initialize the global logger
	err := InitializeGlobalLogger("./logs", "v2_strategy_engine.log")
	if err != nil {
		fmt.Printf("Failed to initialize global logger: %v\n", err)
		return
	}

	// Get the global logger
	logger := GetGlobalLogger()

	// Generate a trace ID for this processing session
	traceID := GenerateTraceID()

	// Create a log helper for easier logging
	helper := NewLogHelper(logger, traceID, "job_example_001")

	// Example 1: System startup logging
	helper.LogInfo(SYSTEM, "App", "Startup", "", "", "V2 Strategy Engine starting up", map[string]interface{}{
		"version": "2.0.0",
		"config":  "production",
	})

	// Example 2: Engine initialization logging
	helper.LogInfo(ENGINE, "ProductionReadyEngineV2", "Initialize", "", "", "Engine initialization started", map[string]interface{}{
		"worker_count": 8,
		"queue_size":   1000,
	})

	// Example 3: Strategy execution logging
	stockGroupID := "NIFTY50"
	strategyName := "1ST_ENTRY"

	// Log strategy execution start
	startTime := time.Now()
	helper.LogInfo(STRATEGY, "FirstEntryStrategyV2", "Process", stockGroupID, strategyName,
		"Strategy execution started", map[string]interface{}{
			"candles_count": 100,
			"parameters":    map[string]interface{}{"ema_period": 20, "rsi_period": 14},
		})

	// Simulate some processing time
	time.Sleep(50 * time.Millisecond)

	// Log strategy execution completion with performance metrics
	duration := time.Since(startTime)
	helper.LogPerformance("FirstEntryStrategyV2", "Process", stockGroupID, strategyName,
		"Strategy execution completed", duration, 128, map[string]interface{}{
			"signals_generated":    2,
			"parameters_extracted": 5,
		})

	// Example 4: Error logging
	testError := fmt.Errorf("database connection failed")
	helper.LogError(ERROR_CAT, "DatabaseService", "Connect", stockGroupID, strategyName,
		"Database connection error", map[string]interface{}{
			"retry_count": 3,
			"timeout_ms":  5000,
		}, testError)

	// Example 5: Progress logging
	helper.LogInfo(PROGRESS, "JobProcessor", "UpdateProgress", stockGroupID, strategyName,
		"Job progress updated", map[string]interface{}{
			"completion_percentage": 75,
			"jobs_remaining":        25,
			"eta_seconds":           30,
		})

	// Example 6: Performance monitoring
	helper.LogInfo(PERFORMANCE, "MemoryManager", "CheckUsage", stockGroupID, strategyName,
		"Memory usage check", map[string]interface{}{
			"current_usage_mb": 256,
			"max_usage_mb":     1024,
			"usage_percentage": 25,
		})

	fmt.Println("Logging examples completed. Check the log file for detailed output.")
}

// ExampleIntegration shows how to integrate logging into existing V2 engine components
func ExampleIntegration() {
	// Initialize logger
	err := InitializeGlobalLogger("./logs", "v2_integration.log")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	logger := GetGlobalLogger()
	traceID := GenerateTraceID()

	// Example: Integrating with ProductionReadyEngineV2
	exampleEngineLogging(logger, traceID)

	// Example: Integrating with ParallelProcessorV2
	exampleParallelProcessorLogging(logger, traceID)

	// Example: Integrating with StrategyV2 implementations
	exampleStrategyLogging(logger, traceID)
}

// exampleEngineLogging shows how to add logging to the ProductionReadyEngineV2
func exampleEngineLogging(logger DebugLogger, traceID string) {
	helper := NewLogHelper(logger, traceID, "engine_job_001")

	// Log engine startup
	helper.LogInfo(ENGINE, "ProductionReadyEngineV2", "ProcessStockGroups", "", "",
		"Starting stock group processing", map[string]interface{}{
			"stock_groups_count": 10,
			"strategies_count":   3,
		})

	// Log historical data fetching
	helper.LogInfo(ENGINE, "ProductionReadyEngineV2", "fetchHistoricalData", "", "",
		"Fetching historical data", map[string]interface{}{
			"time_range_hours": 24,
			"candle_interval":  "5min",
		})

	// Log processing completion
	helper.LogPerformance("ProductionReadyEngineV2", "ProcessStockGroups", "", "",
		"Stock group processing completed", 2*time.Second, 512, map[string]interface{}{
			"groups_processed":    10,
			"strategies_executed": 30,
		})
}

// exampleParallelProcessorLogging shows how to add logging to the ParallelProcessorV2
func exampleParallelProcessorLogging(logger DebugLogger, traceID string) {
	helper := NewLogHelper(logger, traceID, "parallel_job_001")

	// Log parallel processing start
	helper.LogInfo(ENGINE, "ParallelProcessorV2", "ProcessStockGroups", "", "",
		"Starting parallel processing", map[string]interface{}{
			"workers_count": 8,
			"queue_size":    1000,
		})

	// Log job creation
	helper.LogInfo(ENGINE, "ParallelProcessorV2", "createProcessingJobs", "", "",
		"Creating processing jobs", map[string]interface{}{
			"jobs_created": 10,
			"batch_size":   50,
		})

	// Log worker activity
	helper.LogInfo(ENGINE, "Worker", "processJob", "", "",
		"Worker processing job", map[string]interface{}{
			"worker_id": 1,
			"job_id":    "job_001",
		})

	// Log processing completion
	helper.LogPerformance("ParallelProcessorV2", "ProcessStockGroups", "", "",
		"Parallel processing completed", 500*time.Millisecond, 256, map[string]interface{}{
			"jobs_processed": 10,
			"errors_count":   0,
		})
}

// exampleStrategyLogging shows how to add logging to StrategyV2 implementations
func exampleStrategyLogging(logger DebugLogger, traceID string) {
	stockGroupID := "BANKNIFTY"
	strategyName := "2_30_ENTRY"

	helper := NewLogHelper(logger, traceID, "strategy_job_001")

	// Log strategy initialization
	helper.LogInfo(STRATEGY, "TwoThirtyEntryStrategyV2", "Process", stockGroupID, strategyName,
		"Strategy processing started", map[string]interface{}{
			"dataframe_rows": 100,
			"dataframe_cols": 6,
		})

	// Log entry condition check
	helper.LogDebug(STRATEGY, "TwoThirtyEntryStrategyV2", "isEntryTime", stockGroupID, strategyName,
		"Checking entry time conditions", map[string]interface{}{
			"current_time":  "14:30:00",
			"entry_time":    "14:30:00",
			"is_entry_time": true,
		})

	// Log day range calculation
	helper.LogDebug(STRATEGY, "TwoThirtyEntryStrategyV2", "calculateDayRange", stockGroupID, strategyName,
		"Calculating day range", map[string]interface{}{
			"high":  45000.0,
			"low":   44000.0,
			"range": 1000.0,
		})

	// Log signal generation
	helper.LogInfo(STRATEGY, "TwoThirtyEntryStrategyV2", "addSignalToDataFrame", stockGroupID, strategyName,
		"Signal generated", map[string]interface{}{
			"signal_type":     "BUY",
			"signal_strength": 0.8,
			"entry_price":     44500.0,
		})

	// Log strategy completion
	helper.LogPerformance("TwoThirtyEntryStrategyV2", "Process", stockGroupID, strategyName,
		"Strategy processing completed", 25*time.Millisecond, 64, map[string]interface{}{
			"signals_generated": 1,
			"columns_added":     3,
		})
}

// ExampleLogLevels demonstrates different log levels
func ExampleLogLevels() {
	err := InitializeGlobalLogger("./logs", "v2_log_levels.log")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	logger := GetGlobalLogger()
	helper := NewLogHelper(logger, GenerateTraceID(), "levels_job_001")

	// TRACE level - very detailed execution flow
	helper.Log(TRACE, STRATEGY, "ExampleStrategy", "calculateEMA", "STOCK_GROUP_1", "1ST_ENTRY",
		"Calculating EMA for period 20", 0, 0, map[string]interface{}{
			"period": 20,
			"values": []float64{100.0, 101.0, 102.0},
		}, nil)

	// DEBUG level - detailed debugging information
	helper.Log(DEBUG, ENGINE, "ExampleEngine", "processData", "STOCK_GROUP_1", "1ST_ENTRY",
		"Processing data frame", 0, 0, map[string]interface{}{
			"rows": 100,
			"cols": 6,
		}, nil)

	// INFO level - general information
	helper.Log(INFO, SYSTEM, "ExampleSystem", "startup", "STOCK_GROUP_1", "1ST_ENTRY",
		"System startup completed", 0, 0, map[string]interface{}{
			"version": "2.0.0",
		}, nil)

	// WARN level - warnings
	helper.Log(WARN, PERFORMANCE, "ExamplePerformance", "memoryCheck", "STOCK_GROUP_1", "1ST_ENTRY",
		"Memory usage high", 0, 0, map[string]interface{}{
			"usage_percent": 85,
			"threshold":     80,
		}, nil)

	// ERROR level - errors
	testError := fmt.Errorf("test error for demonstration")
	helper.Log(ERROR, ERROR_CAT, "ExampleError", "handleError", "STOCK_GROUP_1", "1ST_ENTRY",
		"Error occurred during processing", 0, 0, map[string]interface{}{
			"retry_count": 3,
		}, testError)
}
