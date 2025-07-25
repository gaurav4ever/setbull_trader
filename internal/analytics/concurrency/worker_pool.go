package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"setbull_trader/pkg/log"
)

// Task represents a computation task that can be executed by workers
type Task interface {
	Execute(ctx context.Context) (interface{}, error)
	ID() string
	Priority() int
}

// Result represents the result of a task execution
type Result struct {
	TaskID string
	Data   interface{}
	Error  error
	Timing TaskTiming
}

// TaskTiming provides execution timing information
type TaskTiming struct {
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	WorkerID      int
	QueueWaitTime time.Duration
}

// WorkerPool manages a pool of workers for concurrent task execution
type WorkerPool struct {
	workerCount int
	taskQueue   chan TaskWrapper
	resultQueue chan Result
	workers     []*Worker
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	// Metrics
	tasksSubmitted   int64
	tasksCompleted   int64
	tasksInProgress  int64
	totalProcessTime int64 // in nanoseconds

	// Configuration
	queueSize       int
	maxWorkers      int
	shutdownTimeout time.Duration

	mu sync.RWMutex
}

// TaskWrapper wraps a task with submission timing
type TaskWrapper struct {
	Task        Task
	SubmittedAt time.Time
	Context     context.Context
}

// Worker represents a single worker in the pool
type Worker struct {
	ID        int
	pool      *WorkerPool
	ctx       context.Context
	taskCount int64
	isActive  int32 // atomic
	lastTask  time.Time
}

// WorkerPoolConfig configures the worker pool
type WorkerPoolConfig struct {
	MaxWorkers      int
	QueueSize       int
	ShutdownTimeout time.Duration
}

// NewWorkerPool creates a new worker pool with the specified configuration
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}
	if config.QueueSize <= 0 {
		config.QueueSize = config.MaxWorkers * 100
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workerCount:     config.MaxWorkers,
		taskQueue:       make(chan TaskWrapper, config.QueueSize),
		resultQueue:     make(chan Result, config.QueueSize),
		workers:         make([]*Worker, config.MaxWorkers),
		ctx:             ctx,
		cancel:          cancel,
		queueSize:       config.QueueSize,
		maxWorkers:      config.MaxWorkers,
		shutdownTimeout: config.ShutdownTimeout,
	}

	// Create workers
	for i := 0; i < config.MaxWorkers; i++ {
		worker := &Worker{
			ID:   i,
			pool: pool,
			ctx:  ctx,
		}
		pool.workers[i] = worker
	}

	log.Info("Created worker pool with %d workers, queue size %d", config.MaxWorkers, config.QueueSize)
	return pool
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	log.Info("Starting worker pool with %d workers", wp.workerCount)

	// Start all workers
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go worker.run()
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(ctx context.Context, task Task) error {
	select {
	case wp.taskQueue <- TaskWrapper{
		Task:        task,
		SubmittedAt: time.Now(),
		Context:     ctx,
	}:
		atomic.AddInt64(&wp.tasksSubmitted, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shut down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitBatch submits multiple tasks to the worker pool
func (wp *WorkerPool) SubmitBatch(ctx context.Context, tasks []Task) error {
	for _, task := range tasks {
		if err := wp.Submit(ctx, task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.ID(), err)
		}
	}
	return nil
}

// Results returns the result channel for reading task results
func (wp *WorkerPool) Results() <-chan Result {
	return wp.resultQueue
}

// Wait waits for all submitted tasks to complete
func (wp *WorkerPool) Wait(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for tasks to complete")
		case <-ticker.C:
			if wp.IsIdle() {
				return nil
			}
		}
	}
}

// IsIdle returns true if the worker pool has no pending or active tasks
func (wp *WorkerPool) IsIdle() bool {
	return atomic.LoadInt64(&wp.tasksInProgress) == 0 && len(wp.taskQueue) == 0
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown() error {
	log.Info("Shutting down worker pool...")

	// Cancel context to signal workers to stop
	wp.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Worker pool shut down successfully")
		return nil
	case <-time.After(wp.shutdownTimeout):
		log.Warn("Worker pool shutdown timed out after %v", wp.shutdownTimeout)
		return fmt.Errorf("shutdown timeout")
	}
}

// GetMetrics returns current worker pool metrics
func (wp *WorkerPool) GetMetrics() WorkerPoolMetrics {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	submitted := atomic.LoadInt64(&wp.tasksSubmitted)
	completed := atomic.LoadInt64(&wp.tasksCompleted)
	inProgress := atomic.LoadInt64(&wp.tasksInProgress)
	totalTime := atomic.LoadInt64(&wp.totalProcessTime)

	var avgProcessTime time.Duration
	if completed > 0 {
		avgProcessTime = time.Duration(totalTime / completed)
	}

	var throughput float64
	if submitted > 0 && completed > 0 {
		throughput = float64(completed) / float64(submitted)
	}

	activeWorkers := 0
	for _, worker := range wp.workers {
		if atomic.LoadInt32(&worker.isActive) > 0 {
			activeWorkers++
		}
	}

	return WorkerPoolMetrics{
		WorkerCount:         wp.workerCount,
		ActiveWorkers:       activeWorkers,
		TasksSubmitted:      submitted,
		TasksCompleted:      completed,
		TasksInProgress:     inProgress,
		TasksInQueue:        int64(len(wp.taskQueue)),
		QueueCapacity:       int64(wp.queueSize),
		QueueUtilization:    float64(len(wp.taskQueue)) / float64(wp.queueSize),
		AvgProcessingTime:   avgProcessTime,
		Throughput:          throughput,
		TotalProcessingTime: time.Duration(totalTime),
	}
}

// WorkerPoolMetrics contains metrics about the worker pool
type WorkerPoolMetrics struct {
	WorkerCount         int           `json:"worker_count"`
	ActiveWorkers       int           `json:"active_workers"`
	TasksSubmitted      int64         `json:"tasks_submitted"`
	TasksCompleted      int64         `json:"tasks_completed"`
	TasksInProgress     int64         `json:"tasks_in_progress"`
	TasksInQueue        int64         `json:"tasks_in_queue"`
	QueueCapacity       int64         `json:"queue_capacity"`
	QueueUtilization    float64       `json:"queue_utilization"`
	AvgProcessingTime   time.Duration `json:"avg_processing_time"`
	Throughput          float64       `json:"throughput"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.pool.wg.Done()

	log.Debug("Worker %d started", w.ID)

	for {
		select {
		case taskWrapper := <-w.pool.taskQueue:
			w.executeTask(taskWrapper)
		case <-w.ctx.Done():
			log.Debug("Worker %d stopping", w.ID)
			return
		}
	}
}

// executeTask executes a single task
func (w *Worker) executeTask(taskWrapper TaskWrapper) {
	atomic.StoreInt32(&w.isActive, 1)
	atomic.AddInt64(&w.pool.tasksInProgress, 1)
	defer func() {
		atomic.StoreInt32(&w.isActive, 0)
		atomic.AddInt64(&w.pool.tasksInProgress, -1)
		atomic.AddInt64(&w.pool.tasksCompleted, 1)
		atomic.AddInt64(&w.taskCount, 1)
		w.lastTask = time.Now()
	}()

	task := taskWrapper.Task
	startTime := time.Now()
	queueWaitTime := startTime.Sub(taskWrapper.SubmittedAt)

	// Execute the task
	data, err := task.Execute(taskWrapper.Context)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Update metrics
	atomic.AddInt64(&w.pool.totalProcessTime, int64(duration))

	// Send result
	result := Result{
		TaskID: task.ID(),
		Data:   data,
		Error:  err,
		Timing: TaskTiming{
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      duration,
			WorkerID:      w.ID,
			QueueWaitTime: queueWaitTime,
		},
	}

	select {
	case w.pool.resultQueue <- result:
		// Result sent successfully
	case <-w.ctx.Done():
		// Pool is shutting down
		return
	default:
		// Result queue is full, log warning
		log.Warn("Result queue is full, dropping result for task %s", task.ID())
	}
}
