package v2

import (
	"sync"
	"time"

	"setbull_trader/pkg/log"
)

// WorkerPool manages a pool of workers for parallel processing
type WorkerPool struct {
	config        *ParallelProcessorConfig
	workers       []*Worker
	jobQueue      chan *ProcessingJob
	workerPool    chan *Worker
	shutdownChan  chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	activeWorkers int
	queueLength   int
}

// Worker represents a single worker in the pool
type Worker struct {
	id       int
	pool     *WorkerPool
	jobChan  chan *ProcessingJob
	stopChan chan struct{}
	active   bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config *ParallelProcessorConfig) *WorkerPool {
	pool := &WorkerPool{
		config:       config,
		jobQueue:     make(chan *ProcessingJob, config.QueueSize),
		workerPool:   make(chan *Worker, config.MaxWorkers),
		shutdownChan: make(chan struct{}),
		workers:      make([]*Worker, 0, config.MaxWorkers),
	}

	// Start workers
	for i := 0; i < config.MinWorkers; i++ {
		worker := pool.createWorker(i)
		pool.workers = append(pool.workers, worker)
		pool.workerPool <- worker
	}

	// Start job dispatcher
	go pool.dispatcher()

	log.Info("Worker pool created with %d initial workers", config.MinWorkers)
	return pool
}

// createWorker creates a new worker
func (p *WorkerPool) createWorker(id int) *Worker {
	worker := &Worker{
		id:       id,
		pool:     p,
		jobChan:  make(chan *ProcessingJob, 1),
		stopChan: make(chan struct{}),
		active:   true,
	}

	p.wg.Add(1)
	go worker.start()

	return worker
}

// start starts the worker
func (w *Worker) start() {
	defer w.pool.wg.Done()

	for {
		select {
		case job := <-w.jobChan:
			w.processJob(job)
			// Return worker to pool
			select {
			case w.pool.workerPool <- w:
			case <-w.stopChan:
				return
			}
		case <-w.stopChan:
			return
		}
	}
}

// processJob processes a job
func (w *Worker) processJob(job *ProcessingJob) {
	log.Debug("Worker %d processing job %s", w.id, job.ID)

	// Simulate job processing (in real implementation, this would call the actual processing logic)
	time.Sleep(10 * time.Millisecond)

	log.Debug("Worker %d completed job %s", w.id, job.ID)
}

// dispatcher dispatches jobs to available workers
func (p *WorkerPool) dispatcher() {
	for {
		select {
		case job := <-p.jobQueue:
			// Get available worker
			select {
			case worker := <-p.workerPool:
				worker.jobChan <- job
			case <-p.shutdownChan:
				return
			}
		case <-p.shutdownChan:
			return
		}
	}
}

// SubmitJob submits a job to the worker pool
func (p *WorkerPool) SubmitJob(job *ProcessingJob) error {
	select {
	case p.jobQueue <- job:
		p.mu.Lock()
		p.queueLength++
		p.mu.Unlock()
		return nil
	case <-p.shutdownChan:
		return ErrWorkerPoolShutdown
	default:
		return ErrWorkerPoolFull
	}
}

// GetActiveWorkerCount returns the number of active workers
func (p *WorkerPool) GetActiveWorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeWorkers
}

// GetQueueLength returns the current queue length
func (p *WorkerPool) GetQueueLength() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.queueLength
}

// Shutdown gracefully shuts down the worker pool
func (p *WorkerPool) Shutdown() {
	log.Info("Shutting down worker pool")

	// Signal shutdown
	close(p.shutdownChan)

	// Stop all workers
	for _, worker := range p.workers {
		close(worker.stopChan)
	}

	// Wait for all workers to finish
	p.wg.Wait()

	log.Info("Worker pool shutdown complete")
}

// ScaleUp scales up the worker pool by adding more workers
func (p *WorkerPool) ScaleUp(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	currentWorkers := len(p.workers)
	maxWorkers := p.config.MaxWorkers

	if currentWorkers+count > maxWorkers {
		count = maxWorkers - currentWorkers
	}

	for i := 0; i < count; i++ {
		worker := p.createWorker(currentWorkers + i)
		p.workers = append(p.workers, worker)
		p.workerPool <- worker
		p.activeWorkers++
	}

	log.Info("Scaled up worker pool by %d workers (total: %d)", count, len(p.workers))
}

// ScaleDown scales down the worker pool by removing workers
func (p *WorkerPool) ScaleDown(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	currentWorkers := len(p.workers)
	minWorkers := p.config.MinWorkers

	if currentWorkers-count < minWorkers {
		count = currentWorkers - minWorkers
	}

	if count <= 0 {
		return
	}

	// Remove workers from the end
	for i := 0; i < count; i++ {
		workerIndex := len(p.workers) - 1
		worker := p.workers[workerIndex]

		// Stop worker
		close(worker.stopChan)

		// Remove from slice
		p.workers = p.workers[:workerIndex]
		p.activeWorkers--
	}

	log.Info("Scaled down worker pool by %d workers (total: %d)", count, len(p.workers))
}

// AutoScale automatically scales the worker pool based on load
func (p *WorkerPool) AutoScale() {
	queueLength := p.GetQueueLength()
	currentWorkers := len(p.workers)

	// Scale up if queue is getting long
	if queueLength > p.config.QueueSize/2 && currentWorkers < p.config.MaxWorkers {
		p.ScaleUp(1)
	}

	// Scale down if queue is short and we have more than minimum workers
	if queueLength < p.config.QueueSize/10 && currentWorkers > p.config.MinWorkers {
		p.ScaleDown(1)
	}
}

// GetStats returns worker pool statistics
func (p *WorkerPool) GetStats() *WorkerPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &WorkerPoolStats{
		TotalWorkers:  len(p.workers),
		ActiveWorkers: p.activeWorkers,
		QueueLength:   p.queueLength,
		MaxWorkers:    p.config.MaxWorkers,
		MinWorkers:    p.config.MinWorkers,
		QueueSize:     p.config.QueueSize,
	}
}

// WorkerPoolStats contains statistics about the worker pool
type WorkerPoolStats struct {
	TotalWorkers  int `json:"total_workers"`
	ActiveWorkers int `json:"active_workers"`
	QueueLength   int `json:"queue_length"`
	MaxWorkers    int `json:"max_workers"`
	MinWorkers    int `json:"min_workers"`
	QueueSize     int `json:"queue_size"`
}

// Error definitions
var (
	ErrWorkerPoolShutdown = &WorkerPoolError{message: "worker pool is shutdown"}
	ErrWorkerPoolFull     = &WorkerPoolError{message: "worker pool queue is full"}
)

// WorkerPoolError represents worker pool errors
type WorkerPoolError struct {
	message string
}

func (e *WorkerPoolError) Error() string {
	return e.message
}
