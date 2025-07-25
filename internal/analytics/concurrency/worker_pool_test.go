package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTask for testing
type MockTask struct {
	id            string
	priority      int
	executionTime time.Duration
	shouldError   bool
	executeCount  int32
	mu            sync.Mutex
}

func NewMockTask(id string, priority int, executionTime time.Duration, shouldError bool) *MockTask {
	return &MockTask{
		id:            id,
		priority:      priority,
		executionTime: executionTime,
		shouldError:   shouldError,
	}
}

func (mt *MockTask) ID() string {
	return mt.id
}

func (mt *MockTask) Priority() int {
	return mt.priority
}

func (mt *MockTask) Execute(ctx context.Context) (interface{}, error) {
	mt.mu.Lock()
	mt.executeCount++
	mt.mu.Unlock()

	// Simulate work
	if mt.executionTime > 0 {
		select {
		case <-time.After(mt.executionTime):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if mt.shouldError {
		return nil, assert.AnError
	}

	return map[string]interface{}{
		"task_id": mt.id,
		"result":  "success",
	}, nil
}

func (mt *MockTask) GetExecuteCount() int32 {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	return mt.executeCount
}

func TestWorkerPool_BasicOperations(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	assert.NotNil(t, pool)
	assert.Equal(t, 2, pool.workerCount)
	assert.Equal(t, 10, pool.queueSize)

	pool.Start()
	defer pool.Shutdown()

	// Test task submission
	ctx := context.Background()
	task := NewMockTask("test-1", 1, 10*time.Millisecond, false)

	err := pool.Submit(ctx, task)
	assert.NoError(t, err)

	// Wait for result
	select {
	case result := <-pool.Results():
		assert.Equal(t, "test-1", result.TaskID)
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Data)

		// Verify timing information
		assert.True(t, result.Timing.Duration > 0)
		assert.True(t, result.Timing.WorkerID >= 0)
		assert.True(t, result.Timing.WorkerID < 2)

	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

func TestWorkerPool_ConcurrentExecution(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      4,
		QueueSize:       100,
		ShutdownTimeout: 10 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	// Submit multiple tasks
	const numTasks = 20
	tasks := make([]*MockTask, numTasks)

	ctx := context.Background()
	for i := 0; i < numTasks; i++ {
		task := NewMockTask(fmt.Sprintf("task-%d", i), 1, 50*time.Millisecond, false)
		tasks[i] = task

		err := pool.Submit(ctx, task)
		require.NoError(t, err)
	}

	// Collect results
	results := make(map[string]Result)
	for i := 0; i < numTasks; i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
			assert.NoError(t, result.Error)

		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for result %d", i)
		}
	}

	// Verify all tasks completed
	assert.Len(t, results, numTasks)

	// Verify each task was executed once
	for _, task := range tasks {
		assert.Equal(t, int32(1), task.GetExecuteCount())
	}

	// Check metrics
	metrics := pool.GetMetrics()
	assert.Equal(t, int64(numTasks), metrics.TasksSubmitted)
	assert.Equal(t, int64(numTasks), metrics.TasksCompleted)
	assert.Equal(t, int64(0), metrics.TasksInProgress)
	assert.True(t, metrics.AvgProcessingTime > 0)
}

func TestWorkerPool_ErrorHandling(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	// Submit task that will error
	ctx := context.Background()
	task := NewMockTask("error-task", 1, 10*time.Millisecond, true)

	err := pool.Submit(ctx, task)
	assert.NoError(t, err)

	// Wait for result
	select {
	case result := <-pool.Results():
		assert.Equal(t, "error-task", result.TaskID)
		assert.Error(t, result.Error)
		assert.Nil(t, result.Data)

	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for error result")
	}
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Submit long-running task
	task := NewMockTask("long-task", 1, 1*time.Second, false)

	err := pool.Submit(ctx, task)
	assert.NoError(t, err)

	// Cancel context immediately
	cancel()

	// Wait for result
	select {
	case result := <-pool.Results():
		assert.Equal(t, "long-task", result.TaskID)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "context canceled")

	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for cancelled task result")
	}
}

func TestWorkerPool_QueueOverflow(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      1,
		QueueSize:       2, // Small queue
		ShutdownTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	ctx := context.Background()

	// Fill up the queue
	for i := 0; i < 2; i++ {
		task := NewMockTask(fmt.Sprintf("task-%d", i), 1, 100*time.Millisecond, false)
		err := pool.Submit(ctx, task)
		assert.NoError(t, err)
	}

	// This should fail due to full queue
	task := NewMockTask("overflow-task", 1, 10*time.Millisecond, false)
	err := pool.Submit(ctx, task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is full")
}

func TestWorkerPool_Shutdown(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		ShutdownTimeout: 2 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()

	// Submit some tasks
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		task := NewMockTask(fmt.Sprintf("shutdown-task-%d", i), 1, 50*time.Millisecond, false)
		err := pool.Submit(ctx, task)
		assert.NoError(t, err)
	}

	// Shutdown should complete successfully
	err := pool.Shutdown()
	assert.NoError(t, err)

	// Submitting after shutdown should fail
	task := NewMockTask("post-shutdown-task", 1, 10*time.Millisecond, false)
	err = pool.Submit(ctx, task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shut down")
}

func TestWorkerPool_Wait(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		ShutdownTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	// Submit tasks
	ctx := context.Background()
	numTasks := 5

	for i := 0; i < numTasks; i++ {
		task := NewMockTask(fmt.Sprintf("wait-task-%d", i), 1, 100*time.Millisecond, false)
		err := pool.Submit(ctx, task)
		assert.NoError(t, err)
	}

	// Wait for completion
	err := pool.Wait(3 * time.Second)
	assert.NoError(t, err)

	// Pool should be idle
	assert.True(t, pool.IsIdle())

	// All tasks should have completed
	metrics := pool.GetMetrics()
	assert.Equal(t, int64(numTasks), metrics.TasksCompleted)
}

func TestWorkerPool_Metrics(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers:      3,
		QueueSize:       20,
		ShutdownTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	// Initial metrics
	metrics := pool.GetMetrics()
	assert.Equal(t, 3, metrics.WorkerCount)
	assert.Equal(t, 0, metrics.ActiveWorkers)
	assert.Equal(t, int64(0), metrics.TasksSubmitted)
	assert.Equal(t, int64(0), metrics.TasksCompleted)
	assert.Equal(t, int64(20), metrics.QueueCapacity)

	// Submit and process some tasks
	ctx := context.Background()
	numTasks := 10

	for i := 0; i < numTasks; i++ {
		task := NewMockTask(fmt.Sprintf("metrics-task-%d", i), 1, 20*time.Millisecond, false)
		err := pool.Submit(ctx, task)
		assert.NoError(t, err)
	}

	// Wait for completion
	err := pool.Wait(2 * time.Second)
	assert.NoError(t, err)

	// Check final metrics
	finalMetrics := pool.GetMetrics()
	assert.Equal(t, int64(numTasks), finalMetrics.TasksSubmitted)
	assert.Equal(t, int64(numTasks), finalMetrics.TasksCompleted)
	assert.Equal(t, int64(0), finalMetrics.TasksInProgress)
	assert.True(t, finalMetrics.AvgProcessingTime > 0)
	assert.Equal(t, 1.0, finalMetrics.Throughput) // All submitted tasks completed
}

// Benchmark tests
func BenchmarkWorkerPool_TaskSubmission(b *testing.B) {
	config := WorkerPoolConfig{
		MaxWorkers:      runtime.NumCPU(),
		QueueSize:       1000,
		ShutdownTimeout: 10 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		taskCounter := 0
		for pb.Next() {
			task := NewMockTask(fmt.Sprintf("bench-task-%d", taskCounter), 1, 0, false)
			pool.Submit(ctx, task)
			taskCounter++
		}
	})
}

func BenchmarkWorkerPool_TaskExecution(b *testing.B) {
	config := WorkerPoolConfig{
		MaxWorkers:      runtime.NumCPU(),
		QueueSize:       b.N + 100,
		ShutdownTimeout: 30 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Shutdown()

	ctx := context.Background()

	b.ResetTimer()

	// Submit all tasks
	for i := 0; i < b.N; i++ {
		task := NewMockTask(fmt.Sprintf("bench-exec-%d", i), 1, 0, false)
		pool.Submit(ctx, task)
	}

	// Wait for all results
	for i := 0; i < b.N; i++ {
		<-pool.Results()
	}
}
