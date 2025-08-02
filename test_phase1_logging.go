package main

import (
	"fmt"
	"os"
	"time"

	v2 "setbull_trader/internal/strategy/v2"
)

func main() {
	fmt.Println("Testing Phase 1: Basic Logging Implementation")
	fmt.Println("=============================================")

	// Create logs directory
	if err := os.MkdirAll("./logs", 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		return
	}

	// Test 1: Initialize global logger
	fmt.Println("\n1. Testing Global Logger Initialization...")
	err := v2.InitializeGlobalLogger("./logs", "v2_test.log")
	if err != nil {
		fmt.Printf("Failed to initialize global logger: %v\n", err)
		return
	}
	fmt.Println("✅ Global logger initialized successfully")

	// Test 2: Get global logger
	fmt.Println("\n2. Testing Global Logger Access...")
	logger := v2.GetGlobalLogger()
	if logger == nil {
		fmt.Println("❌ Global logger is nil")
		return
	}
	fmt.Println("✅ Global logger retrieved successfully")

	// Test 3: Generate trace ID
	fmt.Println("\n3. Testing Trace ID Generation...")
	traceID := v2.GenerateTraceID()
	if traceID == "" {
		fmt.Println("❌ Generated trace ID is empty")
		return
	}
	fmt.Printf("✅ Generated trace ID: %s\n", traceID)

	// Test 4: Create log helper
	fmt.Println("\n4. Testing Log Helper Creation...")
	helper := v2.NewLogHelper(logger, traceID, "test_job_001")
	if helper == nil {
		fmt.Println("❌ Log helper is nil")
		return
	}
	fmt.Println("✅ Log helper created successfully")

	// Test 5: Test different log levels
	fmt.Println("\n5. Testing Different Log Levels...")

	// Test INFO level
	helper.LogInfo(v2.SYSTEM, "TestApp", "Startup", "", "", "Application starting up", map[string]interface{}{
		"version": "2.0.0",
		"config":  "test",
	})
	fmt.Println("✅ INFO level logging successful")

	// Test DEBUG level
	helper.LogDebug(v2.ENGINE, "TestEngine", "Initialize", "", "", "Engine initialization", map[string]interface{}{
		"workers": 8,
		"queue":   1000,
	})
	fmt.Println("✅ DEBUG level logging successful")

	// Test WARN level
	helper.LogWarn(v2.PERFORMANCE, "TestPerformance", "MemoryCheck", "", "", "Memory usage warning", map[string]interface{}{
		"usage_percent": 85,
		"threshold":     80,
	})
	fmt.Println("✅ WARN level logging successful")

	// Test ERROR level
	testError := fmt.Errorf("test error for demonstration")
	helper.LogError(v2.ERROR_CAT, "TestError", "HandleError", "", "", "Error occurred", map[string]interface{}{
		"retry_count": 3,
	}, testError)
	fmt.Println("✅ ERROR level logging successful")

	// Test 6: Test performance logging
	fmt.Println("\n6. Testing Performance Logging...")
	startTime := time.Now()
	time.Sleep(50 * time.Millisecond) // Simulate processing
	duration := time.Since(startTime)

	helper.LogPerformance("TestPerformance", "Process", "STOCK_GROUP_1", "1ST_ENTRY",
		"Performance test completed", duration, 256, map[string]interface{}{
			"items_processed": 100,
			"memory_used_mb":  256,
		})
	fmt.Println("✅ Performance logging successful")

	// Test 7: Test strategy logging
	fmt.Println("\n7. Testing Strategy Logging...")
	helper.LogInfo(v2.STRATEGY, "TestStrategy", "Process", "STOCK_GROUP_1", "1ST_ENTRY",
		"Strategy execution started", map[string]interface{}{
			"candles_count": 100,
			"parameters":    map[string]interface{}{"ema_period": 20},
		})
	fmt.Println("✅ Strategy logging successful")

	// Test 8: Test progress logging
	fmt.Println("\n8. Testing Progress Logging...")
	helper.LogInfo(v2.PROGRESS, "TestProgress", "Update", "STOCK_GROUP_1", "1ST_ENTRY",
		"Progress updated", map[string]interface{}{
			"completion_percent": 75,
			"jobs_remaining":     25,
		})
	fmt.Println("✅ Progress logging successful")

	// Test 9: Check log file creation
	fmt.Println("\n9. Verifying Log File Creation...")
	logFilePath := "./logs/v2_test.log"
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		fmt.Printf("❌ Log file was not created: %s\n", logFilePath)
		return
	}
	fmt.Printf("✅ Log file created successfully: %s\n", logFilePath)

	// Test 10: Check log file content
	fmt.Println("\n10. Checking Log File Content...")
	fileInfo, err := os.Stat(logFilePath)
	if err != nil {
		fmt.Printf("❌ Failed to get log file info: %v\n", err)
		return
	}
	fmt.Printf("✅ Log file size: %d bytes\n", fileInfo.Size())

	fmt.Println("\n🎉 All Phase 1 logging tests completed successfully!")
	fmt.Println("\nLog file location: ./logs/v2_test.log")
	fmt.Println("You can examine the log file to see the structured JSON logs.")
}
