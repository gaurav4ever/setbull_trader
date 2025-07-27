package main

import (
	"fmt"

	"setbull_trader/internal/service"
)

func main() {
	// Create a BatchFetchService to test interval configurations
	batchService := service.NewBatchFetchService(nil, nil, 5)

	fmt.Println("=== Testing Interval Configuration ===")
	fmt.Println()

	// Test different intervals
	intervals := []string{"1minute", "day", "5minute", "15minute", "1hour", "unknown"}

	for _, interval := range intervals {
		days := batchService.GetOptimalIntervalDays(interval)
		fmt.Printf("Interval: %-10s -> Optimal Days: %d\n", interval, days)
	}

	fmt.Println()
	fmt.Println("=== Configuration Summary ===")
	fmt.Println("• 1-minute data: 20 days per batch (as requested)")
	fmt.Println("• Daily data: 100 days per batch (as requested)")
	fmt.Println("• 5-minute data: 30 days per batch")
	fmt.Println("• 15-minute data: 40 days per batch")
	fmt.Println("• Hourly data: 60 days per batch")
	fmt.Println("• Unknown intervals: 20 days (conservative default)")
	fmt.Println()
	fmt.Println("Note: Cache optimization may reduce these values by 25% for recent data")
}
