package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"setbull_trader/internal/analytics/concurrency"
	"setbull_trader/internal/domain"
)

// Simple demo showing V3 service concurrency integration
func main() {
	fmt.Println("=== TechnicalIndicatorServiceV3 Concurrency Integration Demo ===")

	// 1. Test Worker Pool
	fmt.Println("\n1. Testing Worker Pool...")
	testWorkerPool()

	// 2. Test Pipeline
	fmt.Println("\n2. Testing Pipeline...")
	testPipeline()

	// 3. Test Indicator Tasks
	fmt.Println("\n3. Testing Indicator Tasks...")
	testIndicatorTasks()

	fmt.Println("\n=== Demo Complete ===")
}

func testWorkerPool() {
	config := concurrency.WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	}

	pool := concurrency.NewWorkerPool(config)
	defer pool.Shutdown()

	// Submit a test task
	task := &concurrency.MockTask{
		TaskID:   "test-task-1",
		TaskData: "Test data",
		Duration: 100 * time.Millisecond,
	}

	result, err := pool.Submit(context.Background(), task)
	if err != nil {
		log.Printf("Error submitting task: %v", err)
		return
	}

	fmt.Printf("Worker Pool Task Result: %v\n", result.Data)
	fmt.Printf("Worker Pool Metrics: %+v\n", pool.GetMetrics())
}

func testPipeline() {
	config := concurrency.PipelineConfig{
		WorkerPoolConfig: concurrency.WorkerPoolConfig{
			MaxWorkers:      2,
			QueueSize:       10,
			ShutdownTimeout: 5 * time.Second,
		},
		BatchSize:      5,
		MaxConcurrency: 2,
		Timeout:        10 * time.Second,
		CacheSize:      1, // 1MB
	}

	pipeline := concurrency.NewPipeline(config)

	// Test single instrument processing
	sampleCandles := generateSampleCandles("TEST_STOCK", 50)
	indicators := []concurrency.IndicatorRequest{
		{Type: "EMA", Parameters: map[string]interface{}{"period": 5}},
		{Type: "RSI", Parameters: map[string]interface{}{"period": 14}},
	}

	instruments := []string{"TEST_STOCK"}
	candleData := map[string][]domain.Candle{
		"TEST_STOCK": sampleCandles,
	}

	result, err := pipeline.ProcessBatch(context.Background(), instruments, indicators, candleData)
	if err != nil {
		log.Printf("Error in pipeline processing: %v", err)
		return
	}

	fmt.Printf("Pipeline Result Keys: %v\n", getMapKeys(result.Results))
	fmt.Printf("Pipeline Timing: %v\n", result.TotalDuration)
}

func testIndicatorTasks() {
	// Test EMA task
	sampleCandles := generateSampleCandles("TEST_STOCK", 20)

	emaTask := &concurrency.EMATask{
		TaskID:       "ema-test",
		TaskPriority: 1,
		CandleData:   sampleCandles,
		Period:       5,
	}

	result, err := emaTask.Execute(context.Background())
	if err != nil {
		log.Printf("Error executing EMA task: %v", err)
		return
	}

	if emaResult, ok := result.([]float64); ok {
		fmt.Printf("EMA Task Result Length: %d\n", len(emaResult))
		if len(emaResult) > 0 {
			fmt.Printf("First EMA Value: %.4f\n", emaResult[0])
			fmt.Printf("Last EMA Value: %.4f\n", emaResult[len(emaResult)-1])
		}
	}
}

func generateSampleCandles(instrumentKey string, count int) []domain.Candle {
	candles := make([]domain.Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * time.Minute)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		price := basePrice + float64(i%10-5)*0.1 // Small price variations
		candles[i] = domain.Candle{
			InstrumentKey: instrumentKey,
			Timestamp:     baseTime.Add(time.Duration(i) * time.Minute),
			Open:          price,
			High:          price + 0.5,
			Low:           price - 0.5,
			Close:         price + 0.1,
			Volume:        1000 + int64(i%100)*10,
		}
	}

	return candles
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
