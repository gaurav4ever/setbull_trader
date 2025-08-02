package v2

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStructuredLogger(t *testing.T) {
	// Create temporary directory for logs
	tempDir := t.TempDir()
	logFileName := "test_debug.log"

	// Initialize logger
	logger, err := NewStructuredLogger(tempDir, logFileName)
	if err != nil {
		t.Fatalf("Failed to create structured logger: %v", err)
	}
	defer logger.Close()

	// Test basic logging
	t.Run("Basic Logging", func(t *testing.T) {
		logger.Log(INFO, SYSTEM, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Test message", 100*time.Millisecond, 128, nil, nil)

		// Verify log file was created
		logFilePath := filepath.Join(tempDir, logFileName)
		if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
			t.Errorf("Log file was not created: %s", logFilePath)
		}
	})

	// Test log levels
	t.Run("Log Levels", func(t *testing.T) {
		// Set log level to DEBUG
		logger.SetLogLevel(DEBUG)

		// Test different log levels
		logger.Log(TRACE, ENGINE, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Trace message", 0, 0, nil, nil)

		logger.Log(DEBUG, ENGINE, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Debug message", 0, 0, nil, nil)

		logger.Log(INFO, ENGINE, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Info message", 0, 0, nil, nil)

		logger.Log(WARN, ENGINE, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Warning message", 0, 0, nil, nil)

		logger.Log(ERROR, ENGINE, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Error message", 0, 0, nil, nil)
	})

	// Test log categories
	t.Run("Log Categories", func(t *testing.T) {
		// Test different categories
		categories := []LogCategory{SYSTEM, ENGINE, STRATEGY, PERFORMANCE, ERROR_CAT, PROGRESS}

		for _, category := range categories {
			logger.Log(INFO, category, "TestComponent", "TestMethod", "trace_123", "job_456",
				"STOCK_GROUP_1", "1ST_ENTRY", "Category test message", 0, 0, nil, nil)
		}
	})

	// Test context logging
	t.Run("Context Logging", func(t *testing.T) {
		context := map[string]interface{}{
			"candles_processed":    100,
			"signals_generated":    2,
			"parameters_extracted": 5,
			"processing_time_ms":   45,
		}

		logger.Log(INFO, STRATEGY, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Context test message", 45*time.Millisecond, 256, context, nil)
	})

	// Test error logging
	t.Run("Error Logging", func(t *testing.T) {
		testError := fmt.Errorf("test error message")

		logger.Log(ERROR, ERROR_CAT, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Error test message", 0, 0, nil, testError)
	})
}

func TestLogHelper(t *testing.T) {
	// Create temporary directory for logs
	tempDir := t.TempDir()
	logFileName := "test_helper.log"

	// Initialize logger
	logger, err := NewStructuredLogger(tempDir, logFileName)
	if err != nil {
		t.Fatalf("Failed to create structured logger: %v", err)
	}
	defer logger.Close()

	// Create log helper
	helper := NewLogHelper(logger, "trace_789", "job_101")

	t.Run("LogHelper Methods", func(t *testing.T) {
		context := map[string]interface{}{
			"test_key": "test_value",
		}

		// Test different log helper methods
		helper.LogInfo(SYSTEM, "TestComponent", "TestMethod", "STOCK_GROUP_1", "1ST_ENTRY", "Info message", context)
		helper.LogDebug(ENGINE, "TestComponent", "TestMethod", "STOCK_GROUP_1", "1ST_ENTRY", "Debug message", context)
		helper.LogWarn(STRATEGY, "TestComponent", "TestMethod", "STOCK_GROUP_1", "1ST_ENTRY", "Warning message", context)
		helper.LogError(PERFORMANCE, "TestComponent", "TestMethod", "STOCK_GROUP_1", "1ST_ENTRY", "Error message", context, fmt.Errorf("test error"))
		helper.LogPerformance("TestComponent", "TestMethod", "STOCK_GROUP_1", "1ST_ENTRY", "Performance message", 100*time.Millisecond, 512, context)
	})
}

func TestTraceIDGenerator(t *testing.T) {
	generator := NewTraceIDGenerator()

	t.Run("Trace ID Generation", func(t *testing.T) {
		// Generate multiple trace IDs
		traceIDs := make(map[string]bool)
		for i := 0; i < 10; i++ {
			traceID := generator.GenerateTraceID()
			if traceIDs[traceID] {
				t.Errorf("Duplicate trace ID generated: %s", traceID)
			}
			traceIDs[traceID] = true
		}
	})
}

func TestLogFormatter(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	logFileName := "test_formatter.log"

	// Create log formatter
	formatter := NewLogFormatter(tempDir, logFileName, 1024, 3) // 1KB max size, 3 backup files

	t.Run("Log Directory Creation", func(t *testing.T) {
		if err := formatter.EnsureLogDir(); err != nil {
			t.Errorf("Failed to create log directory: %v", err)
		}

		// Verify directory exists
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Errorf("Log directory was not created: %s", tempDir)
		}
	})

	t.Run("Log File Path", func(t *testing.T) {
		expectedPath := filepath.Join(tempDir, logFileName)
		actualPath := formatter.GetLogFilePath()
		if actualPath != expectedPath {
			t.Errorf("Expected log file path %s, got %s", expectedPath, actualPath)
		}
	})

	t.Run("Log Rotation Check", func(t *testing.T) {
		// Initially should not need rotation
		if formatter.ShouldRotate() {
			t.Error("Log file should not need rotation initially")
		}
	})

	t.Run("Log Stats", func(t *testing.T) {
		stats, err := formatter.GetLogStats()
		if err != nil {
			t.Errorf("Failed to get log stats: %v", err)
		}

		if stats.LogDir != tempDir {
			t.Errorf("Expected log dir %s, got %s", tempDir, stats.LogDir)
		}

		if stats.LogFileName != logFileName {
			t.Errorf("Expected log file name %s, got %s", logFileName, stats.LogFileName)
		}

		if stats.MaxSize != 1024 {
			t.Errorf("Expected max size %d, got %d", 1024, stats.MaxSize)
		}

		if stats.MaxFiles != 3 {
			t.Errorf("Expected max files %d, got %d", 3, stats.MaxFiles)
		}
	})
}

func TestGlobalLogger(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	logFileName := "test_global.log"

	t.Run("Global Logger Initialization", func(t *testing.T) {
		// Initialize global logger
		err := InitializeGlobalLogger(tempDir, logFileName)
		if err != nil {
			t.Fatalf("Failed to initialize global logger: %v", err)
		}

		// Get global logger
		logger := GetGlobalLogger()
		if logger == nil {
			t.Error("Global logger is nil")
		}

		// Test global trace ID generation
		traceID := GenerateTraceID()
		if traceID == "" {
			t.Error("Generated trace ID is empty")
		}
	})

	t.Run("Global Logger Usage", func(t *testing.T) {
		logger := GetGlobalLogger()
		if logger == nil {
			t.Skip("Global logger not available")
		}

		// Test logging with global logger
		logger.Log(INFO, SYSTEM, "TestComponent", "TestMethod", "trace_123", "job_456",
			"STOCK_GROUP_1", "1ST_ENTRY", "Global logger test message", 0, 0, nil, nil)
	})
}

// Benchmark tests for performance
func BenchmarkStructuredLogger(b *testing.B) {
	// Create temporary directory
	tempDir := b.TempDir()
	logFileName := "benchmark.log"

	// Initialize logger
	logger, err := NewStructuredLogger(tempDir, logFileName)
	if err != nil {
		b.Fatalf("Failed to create structured logger: %v", err)
	}
	defer logger.Close()

	b.ResetTimer()

	b.Run("Info Logging", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Log(INFO, SYSTEM, "BenchComponent", "BenchMethod", "trace_bench", "job_bench",
				"STOCK_GROUP_BENCH", "1ST_ENTRY", "Benchmark message", 0, 0, nil, nil)
		}
	})

	b.Run("Context Logging", func(b *testing.B) {
		context := map[string]interface{}{
			"bench_key": "bench_value",
			"counter":   123,
		}

		for i := 0; i < b.N; i++ {
			logger.Log(INFO, STRATEGY, "BenchComponent", "BenchMethod", "trace_bench", "job_bench",
				"STOCK_GROUP_BENCH", "1ST_ENTRY", "Benchmark context message", 0, 0, context, nil)
		}
	})
}

func BenchmarkTraceIDGenerator(b *testing.B) {
	generator := NewTraceIDGenerator()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		generator.GenerateTraceID()
	}
}
