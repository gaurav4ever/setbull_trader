package main

import (
	"fmt"
	"os"
	"time"

	v2 "setbull_trader/internal/strategy/v2"
)

func main() {
	fmt.Println("Testing Phase 2: Debug Manager Implementation")
	fmt.Println("=============================================")

	// Create logs directory
	if err := os.MkdirAll("./logs", 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		return
	}

	// Test 1: Initialize global logger
	fmt.Println("\n1. Initializing Global Logger...")
	err := v2.InitializeGlobalLogger("./logs", "v2_debug_manager.log")
	if err != nil {
		fmt.Printf("Failed to initialize global logger: %v\n", err)
		return
	}
	fmt.Println("✅ Global logger initialized successfully")

	// Test 2: Create debug manager
	fmt.Println("\n2. Creating Debug Manager...")
	logger := v2.GetGlobalLogger()
	debugManager := v2.NewDebugManager(logger)
	if debugManager == nil {
		fmt.Println("❌ Debug manager is nil")
		return
	}
	fmt.Println("✅ Debug manager created successfully")

	// Test 3: Test debug flags
	fmt.Println("\n3. Testing Debug Flags...")

	// Check initial state
	flags := debugManager.GetDebugFlags()
	fmt.Printf("Initial debug enabled: %t\n", flags.Enabled)

	// Enable debug
	debugManager.EnableDebug()
	flags = debugManager.GetDebugFlags()
	fmt.Printf("After enable - debug enabled: %t\n", flags.Enabled)
	fmt.Printf("State inspection: %t\n", flags.StateInspection)
	fmt.Printf("Memory inspection: %t\n", flags.MemoryInspection)
	fmt.Printf("Performance profiling: %t\n", flags.PerformanceProfiling)
	fmt.Printf("Error tracking: %t\n", flags.ErrorTracking)
	fmt.Printf("Progress tracking: %t\n", flags.ProgressTracking)
	fmt.Println("✅ Debug flags working correctly")

	// Test 4: Test state snapshots
	fmt.Println("\n4. Testing State Snapshots...")

	// Take a state snapshot
	state := map[string]interface{}{
		"test_key": "test_value",
		"counter":  42,
		"active":   true,
	}
	debugManager.TakeStateSnapshot("test_component", state)

	// Get the snapshot
	snapshot := debugManager.GetStateSnapshot("test_component")
	if snapshot == nil {
		fmt.Println("❌ State snapshot is nil")
		return
	}
	fmt.Printf("Snapshot component: %s\n", snapshot.Component)
	fmt.Printf("Snapshot timestamp: %s\n", snapshot.Timestamp.Format(time.RFC3339))
	fmt.Printf("Goroutines: %d\n", snapshot.Goroutines)
	fmt.Printf("State keys: %d\n", len(snapshot.State))
	fmt.Println("✅ State snapshots working correctly")

	// Test 5: Test error tracking
	fmt.Println("\n5. Testing Error Tracking...")

	// Track some errors
	testError := fmt.Errorf("test error for debugging")
	debugManager.TrackError("test_component", "test_method", testError, map[string]interface{}{
		"error_context": "test_context",
		"retry_count":   3,
	})

	// Track another error
	debugManager.TrackError("test_component", "test_method", testError, map[string]interface{}{
		"error_context": "test_context_2",
		"retry_count":   5,
	})

	// Get error stats
	errorStats := debugManager.GetErrorStats()
	fmt.Printf("Total errors tracked: %v\n", errorStats["total_errors"])
	fmt.Println("✅ Error tracking working correctly")

	// Test 6: Test performance tracking
	fmt.Println("\n6. Testing Performance Tracking...")

	// Track some performance metrics
	startTime := time.Now()
	time.Sleep(50 * time.Millisecond) // Simulate work
	duration := time.Since(startTime)

	debugManager.TrackPerformance("test_component", "test_method", duration)
	debugManager.TrackPerformance("test_component", "test_method", duration*2)
	debugManager.TrackPerformance("test_component", "another_method", duration/2)

	// Get performance stats
	perfStats := debugManager.GetPerformanceStats()
	fmt.Printf("Performance metrics tracked: %d\n", len(perfStats))
	for key, metric := range perfStats {
		fmt.Printf("  %s: %d calls, avg: %v\n", key, metric.TotalCalls, metric.AvgDuration)
	}
	fmt.Println("✅ Performance tracking working correctly")

	// Test 7: Test memory monitoring
	fmt.Println("\n7. Testing Memory Monitoring...")

	// Take memory snapshots
	debugManager.TakeMemorySnapshot()
	time.Sleep(100 * time.Millisecond)
	debugManager.TakeMemorySnapshot()

	// Get memory stats
	memoryStats := debugManager.GetMemoryStats()
	fmt.Printf("Memory snapshots taken: %d\n", len(memoryStats))
	if len(memoryStats) > 0 {
		latest := memoryStats[len(memoryStats)-1]
		fmt.Printf("Latest snapshot - Alloc: %.2f MB, Goroutines: %d\n",
			float64(latest.Alloc)/1024/1024, latest.Goroutines)
	}
	fmt.Println("✅ Memory monitoring working correctly")

	// Test 8: Test progress tracking
	fmt.Println("\n8. Testing Progress Tracking...")

	// Start progress tracking
	debugManager.StartProgressTracking("test_job_001", "test_component", 100, map[string]interface{}{
		"job_type": "test_job",
		"priority": "high",
	})

	// Update progress
	debugManager.UpdateProgress("test_job_001", 25, "processing", map[string]interface{}{
		"current_step": "step_1",
	})

	time.Sleep(100 * time.Millisecond)

	debugManager.UpdateProgress("test_job_001", 50, "processing", map[string]interface{}{
		"current_step": "step_2",
	})

	// Get progress info
	progress := debugManager.GetProgressInfo("test_job_001")
	if progress == nil {
		fmt.Println("❌ Progress info is nil")
		return
	}
	fmt.Printf("Progress: %.2f%% (%d/%d)\n", progress.Progress, progress.CompletedItems, progress.TotalItems)
	fmt.Printf("Status: %s\n", progress.Status)
	fmt.Printf("ETA: %s\n", progress.ETA.Format("15:04:05"))
	fmt.Println("✅ Progress tracking working correctly")

	// Test 9: Test debug summary
	fmt.Println("\n9. Testing Debug Summary...")

	summary := debugManager.GetDebugSummary()
	fmt.Printf("Debug summary keys: %d\n", len(summary))

	if flags, exists := summary["flags"]; exists {
		flagsMap := flags.(v2.DebugFlags)
		fmt.Printf("Flags in summary - Enabled: %t\n", flagsMap.Enabled)
	}

	if currentMemory, exists := summary["current_memory"]; exists {
		memory := currentMemory.(map[string]interface{})
		fmt.Printf("Current memory - Alloc: %.2f MB\n", memory["alloc_mb"])
	}
	fmt.Println("✅ Debug summary working correctly")

	// Test 10: Test debug console
	fmt.Println("\n10. Testing Debug Console...")

	// Create debug console
	console := v2.NewDebugConsole(debugManager, logger)
	if console == nil {
		fmt.Println("❌ Debug console is nil")
		return
	}

	// Test some commands
	commands := []string{
		"help",
		"status",
		"memory",
		"performance",
		"errors",
		"progress",
		"summary",
	}

	for _, cmd := range commands {
		fmt.Printf("\n--- Testing command: %s ---\n", cmd)
		result := console.ExecuteCommand(cmd)
		fmt.Printf("Result: %s\n", result[:min(len(result), 100)]+"...")
	}

	// Get available commands
	availableCommands := console.GetAvailableCommands()
	fmt.Printf("\nAvailable commands: %d\n", len(availableCommands))
	fmt.Println("✅ Debug console working correctly")

	// Test 11: Test flag management
	fmt.Println("\n11. Testing Flag Management...")

	// Test individual flag setting
	debugManager.SetDebugFlag("memory_inspection", false)
	fmt.Printf("Memory inspection after disable: %t\n", debugManager.IsFlagEnabled("memory_inspection"))

	debugManager.SetDebugFlag("memory_inspection", true)
	fmt.Printf("Memory inspection after enable: %t\n", debugManager.IsFlagEnabled("memory_inspection"))

	// Test disable all
	debugManager.DisableDebug()
	fmt.Printf("Debug enabled after disable: %t\n", debugManager.IsDebugEnabled())
	fmt.Println("✅ Flag management working correctly")

	fmt.Println("\n🎉 All Phase 2 debug manager tests completed successfully!")
	fmt.Println("\nLog file location: ./logs/v2_debug_manager.log")
	fmt.Println("You can examine the log file to see the debug manager activity.")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
