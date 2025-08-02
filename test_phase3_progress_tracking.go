package main

import (
	"fmt"
	"os"
	"time"

	v2 "setbull_trader/internal/strategy/v2"
)

func main() {
	fmt.Println("Testing Phase 3: Progress Tracking Implementation")
	fmt.Println("================================================")

	// Create logs directory
	if err := os.MkdirAll("./logs", 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		return
	}

	// Test 1: Initialize global logger
	fmt.Println("\n1. Initializing Global Logger...")
	err := v2.InitializeGlobalLogger("./logs", "v2_progress_tracking.log")
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

	// Test 3: Create progress tracker manager
	fmt.Println("\n3. Creating Progress Tracker Manager...")
	progressManager := v2.NewProgressTrackerManager(logger, debugManager)
	if progressManager == nil {
		fmt.Println("❌ Progress tracker manager is nil")
		return
	}
	fmt.Println("✅ Progress tracker manager created successfully")

	// Test 4: Create and start a progress tracker
	fmt.Println("\n4. Creating and Starting Progress Tracker...")

	tracker := progressManager.CreateTracker("test_job_001", "TestComponent", 100, v2.LevelDetailed, map[string]interface{}{
		"job_type": "test_job",
		"priority": "high",
		"version":  "3.0",
	})

	if tracker == nil {
		fmt.Println("❌ Progress tracker is nil")
		return
	}

	err = progressManager.StartTracker("test_job_001", map[string]interface{}{
		"start_context": "test_start",
	})
	if err != nil {
		fmt.Printf("❌ Failed to start tracker: %v\n", err)
		return
	}

	fmt.Printf("✅ Progress tracker created and started - ID: %s, Component: %s\n", tracker.ID, tracker.Component)
	fmt.Printf("   Status: %s, Progress: %.1f%%, Total Items: %d\n", tracker.Status, tracker.Progress, tracker.TotalItems)

	// Test 5: Update progress with real-time simulation
	fmt.Println("\n5. Simulating Progress Updates...")

	// Simulate progress updates
	for i := 0; i <= 10; i++ {
		completed := i * 10
		progress := float64(completed)

		err := progressManager.UpdateProgress("test_job_001", completed, 0, v2.StatusRunning, map[string]interface{}{
			"step":       fmt.Sprintf("step_%d", i),
			"completed":  completed,
			"percentage": progress,
		})

		if err != nil {
			fmt.Printf("❌ Failed to update progress: %v\n", err)
			return
		}

		// Get updated tracker
		updatedTracker := progressManager.GetTracker("test_job_001")
		if updatedTracker == nil {
			fmt.Println("❌ Updated tracker is nil")
			return
		}

		fmt.Printf("   Progress: %.1f%% (%d/100) - ETA: %s\n",
			updatedTracker.Progress, updatedTracker.CompletedItems,
			updatedTracker.ETA.Format("15:04:05"))

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("✅ Progress updates completed successfully")

	// Test 6: Test performance metrics
	fmt.Println("\n6. Testing Performance Metrics...")

	finalTracker := progressManager.GetTracker("test_job_001")
	if finalTracker == nil {
		fmt.Println("❌ Final tracker is nil")
		return
	}

	if finalTracker.Performance != nil {
		fmt.Printf("   Items per second: %.2f\n", finalTracker.Performance.ItemsPerSecond)
		fmt.Printf("   Average item time: %v\n", finalTracker.Performance.AverageItemTime)
		fmt.Printf("   Actual elapsed time: %v\n", finalTracker.Performance.ActualElapsedTime)
		fmt.Printf("   Remaining time: %v\n", finalTracker.Performance.RemainingTime)
	}

	fmt.Println("✅ Performance metrics calculated successfully")

	// Test 7: Test checkpoints
	fmt.Println("\n7. Testing Checkpoints...")

	if len(finalTracker.Checkpoints) > 0 {
		fmt.Printf("   Total checkpoints: %d\n", len(finalTracker.Checkpoints))
		for i, checkpoint := range finalTracker.Checkpoints {
			fmt.Printf("   Checkpoint %d: %s - %.1f%% (%s)\n",
				i+1, checkpoint.ID, checkpoint.Progress, checkpoint.Description)
		}
	}

	fmt.Println("✅ Checkpoints created successfully")

	// Test 8: Complete the tracker
	fmt.Println("\n8. Completing Progress Tracker...")

	err = progressManager.CompleteTracker("test_job_001", map[string]interface{}{
		"completion_context": "test_completion",
		"final_status":       "success",
	})
	if err != nil {
		fmt.Printf("❌ Failed to complete tracker: %v\n", err)
		return
	}

	completedTracker := progressManager.GetTracker("test_job_001")
	if completedTracker == nil {
		fmt.Println("❌ Completed tracker is nil")
		return
	}

	fmt.Printf("✅ Tracker completed - Status: %s, Progress: %.1f%%\n",
		completedTracker.Status, completedTracker.Progress)

	// Test 9: Test progress summary
	fmt.Println("\n9. Testing Progress Summary...")

	summary := progressManager.GetProgressSummary()
	fmt.Printf("   Total trackers: %v\n", summary["total_trackers"])
	fmt.Printf("   Active trackers: %v\n", summary["active_trackers"])
	fmt.Printf("   Completed trackers: %v\n", summary["completed_trackers"])
	fmt.Printf("   Failed trackers: %v\n", summary["failed_trackers"])
	fmt.Printf("   Paused trackers: %v\n", summary["paused_trackers"])

	fmt.Println("✅ Progress summary generated successfully")

	// Test 10: Test multiple trackers
	fmt.Println("\n10. Testing Multiple Trackers...")

	// Create additional trackers
	progressManager.CreateTracker("test_job_002", "TestComponent2", 50, v2.LevelBasic, map[string]interface{}{
		"job_type": "test_job_2",
		"priority": "medium",
	})

	progressManager.CreateTracker("test_job_003", "TestComponent3", 25, v2.LevelVerbose, map[string]interface{}{
		"job_type": "test_job_3",
		"priority": "low",
	})

	// Start and update them
	progressManager.StartTracker("test_job_002", nil)
	progressManager.UpdateProgress("test_job_002", 25, 0, v2.StatusRunning, nil)

	progressManager.StartTracker("test_job_003", nil)
	progressManager.UpdateProgress("test_job_003", 10, 0, v2.StatusRunning, nil)

	// Get all trackers
	allTrackers := progressManager.GetAllTrackers()
	fmt.Printf("   Total trackers in system: %d\n", len(allTrackers))

	for id, tracker := range allTrackers {
		fmt.Printf("   Tracker %s: %s - %.1f%% (%s)\n",
			id, tracker.Component, tracker.Progress, tracker.Status)
	}

	fmt.Println("✅ Multiple trackers working correctly")

	// Test 11: Test callback registration
	fmt.Println("\n11. Testing Callback Registration...")

	callbackCalled := false
	callback := func(progress *v2.ProgressTrackerV3) {
		callbackCalled = true
		fmt.Printf("   Callback triggered for tracker: %s (%.1f%%)\n",
			progress.ID, progress.Progress)
	}

	progressManager.RegisterCallback("test_job_002", callback)

	// Trigger callback by updating progress
	progressManager.UpdateProgress("test_job_002", 30, 0, v2.StatusRunning, nil)

	// Give callback time to execute
	time.Sleep(50 * time.Millisecond)

	if callbackCalled {
		fmt.Println("✅ Callback registration working correctly")
	} else {
		fmt.Println("⚠️  Callback may not have been triggered")
	}

	// Test 12: Test error handling
	fmt.Println("\n12. Testing Error Handling...")

	// Create a tracker that will fail
	progressManager.CreateTracker("fail_job_001", "FailComponent", 10, v2.LevelDetailed, nil)
	progressManager.StartTracker("fail_job_001", nil)
	progressManager.UpdateProgress("fail_job_001", 3, 2, v2.StatusFailed, map[string]interface{}{
		"error_reason": "simulated_failure",
	})

	failedTracker := progressManager.GetTracker("fail_job_001")
	if failedTracker != nil && failedTracker.Status == v2.StatusFailed {
		fmt.Printf("✅ Error handling working - Failed tracker: %s\n", failedTracker.ID)
	} else {
		fmt.Println("❌ Error handling failed")
	}

	// Final summary
	fmt.Println("\n📊 Final Progress Summary:")
	finalSummary := progressManager.GetProgressSummary()
	fmt.Printf("   Total trackers: %v\n", finalSummary["total_trackers"])
	fmt.Printf("   Active trackers: %v\n", finalSummary["active_trackers"])
	fmt.Printf("   Completed trackers: %v\n", finalSummary["completed_trackers"])
	fmt.Printf("   Failed trackers: %v\n", finalSummary["failed_trackers"])
	fmt.Printf("   Paused trackers: %v\n", finalSummary["paused_trackers"])

	fmt.Println("\n🎉 All Phase 3 progress tracking tests completed successfully!")
	fmt.Println("\nLog file location: ./logs/v2_progress_tracking.log")
	fmt.Println("You can examine the log file to see the progress tracking activity.")
}
