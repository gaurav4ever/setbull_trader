package v2

import (
	"fmt"
	"sync"
	"time"
)

// ProgressStatus represents the status of a progress operation
type ProgressStatus string

const (
	StatusPending   ProgressStatus = "pending"
	StatusRunning   ProgressStatus = "running"
	StatusPaused    ProgressStatus = "paused"
	StatusCompleted ProgressStatus = "completed"
	StatusFailed    ProgressStatus = "failed"
	StatusCancelled ProgressStatus = "cancelled"
)

// ProgressLevel represents the level of progress detail
type ProgressLevel string

const (
	LevelBasic    ProgressLevel = "basic"
	LevelDetailed ProgressLevel = "detailed"
	LevelVerbose  ProgressLevel = "verbose"
)

// ProgressTrackerManager represents the enhanced progress tracking system
type ProgressTrackerManager struct {
	trackers     map[string]*ProgressTrackerV3
	mu           sync.RWMutex
	logger       DebugLogger
	debugManager *DebugManager
	callbacks    map[string][]ProgressCallback
	history      map[string][]*ProgressSnapshot
	maxHistory   int
}

// ProgressTrackerV3 represents a single progress tracker
type ProgressTrackerV3 struct {
	ID             string                 `json:"id"`
	Component      string                 `json:"component"`
	Status         ProgressStatus         `json:"status"`
	Progress       float64                `json:"progress"`
	TotalItems     int                    `json:"total_items"`
	CompletedItems int                    `json:"completed_items"`
	FailedItems    int                    `json:"failed_items"`
	StartTime      time.Time              `json:"start_time"`
	LastUpdate     time.Time              `json:"last_update"`
	ETA            time.Time              `json:"eta"`
	Context        map[string]interface{} `json:"context"`
	Level          ProgressLevel          `json:"level"`
	SubTasks       []*SubTask             `json:"sub_tasks"`
	Performance    *ProgressPerformance   `json:"performance"`
	ErrorHistory   []*ProgressError       `json:"error_history"`
	Checkpoints    []*ProgressCheckpoint  `json:"checkpoints"`
	Persistence    *ProgressPersistence   `json:"persistence"`
}

// SubTask represents a sub-task within a progress tracker
type SubTask struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Status         ProgressStatus         `json:"status"`
	Progress       float64                `json:"progress"`
	TotalItems     int                    `json:"total_items"`
	CompletedItems int                    `json:"completed_items"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Context        map[string]interface{} `json:"context"`
}

// ProgressPerformance represents performance metrics for progress tracking
type ProgressPerformance struct {
	ItemsPerSecond     float64       `json:"items_per_second"`
	AverageItemTime    time.Duration `json:"average_item_time"`
	EstimatedTotalTime time.Duration `json:"estimated_total_time"`
	ActualElapsedTime  time.Duration `json:"actual_elapsed_time"`
	RemainingTime      time.Duration `json:"remaining_time"`
	SpeedFactor        float64       `json:"speed_factor"`
	Efficiency         float64       `json:"efficiency"`
}

// ProgressError represents an error during progress tracking
type ProgressError struct {
	Timestamp   time.Time              `json:"timestamp"`
	Error       string                 `json:"error"`
	ItemIndex   int                    `json:"item_index"`
	Context     map[string]interface{} `json:"context"`
	Recoverable bool                   `json:"recoverable"`
}

// ProgressCheckpoint represents a checkpoint in progress
type ProgressCheckpoint struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Progress    float64                `json:"progress"`
	Completed   int                    `json:"completed"`
	Total       int                    `json:"total"`
	Context     map[string]interface{} `json:"context"`
	Description string                 `json:"description"`
}

// ProgressPersistence represents persistence settings for progress
type ProgressPersistence struct {
	Enabled      bool          `json:"enabled"`
	StoragePath  string        `json:"storage_path"`
	AutoSave     bool          `json:"auto_save"`
	SaveInterval time.Duration `json:"save_interval"`
}

// ProgressCallback represents a callback function for progress updates
type ProgressCallback func(progress *ProgressTrackerV3)

// ProgressSnapshot represents a snapshot of progress at a point in time
type ProgressSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Progress  float64                `json:"progress"`
	Status    ProgressStatus         `json:"status"`
	Completed int                    `json:"completed"`
	Total     int                    `json:"total"`
	ETA       time.Time              `json:"eta"`
	Context   map[string]interface{} `json:"context"`
}

// NewProgressTrackerManager creates a new enhanced progress tracker manager
func NewProgressTrackerManager(logger DebugLogger, debugManager *DebugManager) *ProgressTrackerManager {
	ptm := &ProgressTrackerManager{
		trackers:     make(map[string]*ProgressTrackerV3),
		logger:       logger,
		debugManager: debugManager,
		callbacks:    make(map[string][]ProgressCallback),
		history:      make(map[string][]*ProgressSnapshot),
		maxHistory:   1000, // Keep last 1000 snapshots per tracker
	}

	if logger != nil {
		helper := NewLogHelper(logger, GenerateTraceID(), "progress_tracker")
		helper.LogInfo(SYSTEM, "ProgressTrackerManager", "NewProgressTrackerManager", "", "",
			"Enhanced progress tracker manager created", map[string]interface{}{
				"max_history": ptm.maxHistory,
			})
	}

	return ptm
}

// CreateTracker creates a new progress tracker
func (ptm *ProgressTrackerManager) CreateTracker(id, component string, totalItems int, level ProgressLevel, context map[string]interface{}) *ProgressTrackerV3 {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	tracker := &ProgressTrackerV3{
		ID:             id,
		Component:      component,
		Status:         StatusPending,
		Progress:       0.0,
		TotalItems:     totalItems,
		CompletedItems: 0,
		FailedItems:    0,
		StartTime:      time.Now(),
		LastUpdate:     time.Now(),
		Context:        make(map[string]interface{}),
		Level:          level,
		SubTasks:       make([]*SubTask, 0),
		Performance:    &ProgressPerformance{},
		ErrorHistory:   make([]*ProgressError, 0),
		Checkpoints:    make([]*ProgressCheckpoint, 0),
		Persistence: &ProgressPersistence{
			Enabled:      false,
			StoragePath:  "./progress",
			AutoSave:     false,
			SaveInterval: 30 * time.Second,
		},
	}

	// Initialize context if provided
	if context != nil {
		for k, v := range context {
			tracker.Context[k] = v
		}
	}

	ptm.trackers[id] = tracker

	// Log tracker creation
	if ptm.logger != nil {
		helper := NewLogHelper(ptm.logger, GenerateTraceID(), "progress_tracker")
		helper.LogInfo(PROGRESS, "ProgressTrackerManager", "CreateTracker", "", "",
			"Progress tracker created", map[string]interface{}{
				"id":          id,
				"component":   component,
				"total_items": totalItems,
				"level":       level,
			})
	}

	return tracker
}

// StartTracker starts a progress tracker
func (ptm *ProgressTrackerManager) StartTracker(id string, context map[string]interface{}) error {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	tracker, exists := ptm.trackers[id]
	if !exists {
		return fmt.Errorf("progress tracker not found: %s", id)
	}

	tracker.Status = StatusRunning
	tracker.StartTime = time.Now()
	tracker.LastUpdate = time.Now()

	// Update context
	if context != nil {
		for k, v := range context {
			tracker.Context[k] = v
		}
	}

	// Create initial checkpoint
	ptm.createCheckpoint(tracker, "start", "Tracker started")

	// Log tracker start
	if ptm.logger != nil {
		helper := NewLogHelper(ptm.logger, GenerateTraceID(), "progress_tracker")
		helper.LogInfo(PROGRESS, "ProgressTrackerManager", "StartTracker", "", "",
			"Progress tracker started", map[string]interface{}{
				"id":         id,
				"start_time": tracker.StartTime.Format(time.RFC3339),
			})
	}

	// Notify callbacks
	ptm.notifyCallbacks(tracker)

	return nil
}

// UpdateProgress updates progress for a tracker
func (ptm *ProgressTrackerManager) UpdateProgress(id string, completedItems int, failedItems int, status ProgressStatus, context map[string]interface{}) error {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	tracker, exists := ptm.trackers[id]
	if !exists {
		return fmt.Errorf("progress tracker not found: %s", id)
	}

	// Update tracker
	tracker.CompletedItems = completedItems
	tracker.FailedItems = failedItems
	tracker.Status = status
	tracker.LastUpdate = time.Now()

	// Calculate progress
	if tracker.TotalItems > 0 {
		tracker.Progress = float64(completedItems) / float64(tracker.TotalItems) * 100.0
	}

	// Update context
	if context != nil {
		for k, v := range context {
			tracker.Context[k] = v
		}
	}

	// Update performance metrics
	ptm.updatePerformance(tracker)

	// Calculate ETA
	ptm.calculateETA(tracker)

	// Create snapshot for history
	ptm.createSnapshot(tracker)

	// Create checkpoint if significant progress
	if ptm.shouldCreateCheckpoint(tracker) {
		ptm.createCheckpoint(tracker, "progress", fmt.Sprintf("Progress: %.1f%%", tracker.Progress))
	}

	// Log progress update
	if ptm.logger != nil {
		helper := NewLogHelper(ptm.logger, GenerateTraceID(), "progress_tracker")
		helper.LogInfo(PROGRESS, "ProgressTrackerManager", "UpdateProgress", "", "",
			"Progress updated", map[string]interface{}{
				"id":        id,
				"progress":  tracker.Progress,
				"completed": completedItems,
				"failed":    failedItems,
				"status":    status,
				"eta":       tracker.ETA.Format("15:04:05"),
			})
	}

	// Notify callbacks
	ptm.notifyCallbacks(tracker)

	return nil
}

// CompleteTracker completes a progress tracker
func (ptm *ProgressTrackerManager) CompleteTracker(id string, context map[string]interface{}) error {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	tracker, exists := ptm.trackers[id]
	if !exists {
		return fmt.Errorf("progress tracker not found: %s", id)
	}

	tracker.Status = StatusCompleted
	tracker.Progress = 100.0
	tracker.CompletedItems = tracker.TotalItems
	tracker.LastUpdate = time.Now()

	// Update context
	if context != nil {
		for k, v := range context {
			tracker.Context[k] = v
		}
	}

	// Update performance metrics
	ptm.updatePerformance(tracker)

	// Create final checkpoint
	ptm.createCheckpoint(tracker, "complete", "Tracker completed")

	// Create final snapshot
	ptm.createSnapshot(tracker)

	// Log completion
	if ptm.logger != nil {
		helper := NewLogHelper(ptm.logger, GenerateTraceID(), "progress_tracker")
		helper.LogInfo(PROGRESS, "ProgressTrackerManager", "CompleteTracker", "", "",
			"Progress tracker completed", map[string]interface{}{
				"id":            id,
				"total_time":    tracker.Performance.ActualElapsedTime.String(),
				"items_per_sec": tracker.Performance.ItemsPerSecond,
			})
	}

	// Notify callbacks
	ptm.notifyCallbacks(tracker)

	return nil
}

// GetTracker gets a progress tracker by ID
func (ptm *ProgressTrackerManager) GetTracker(id string) *ProgressTrackerV3 {
	ptm.mu.RLock()
	defer ptm.mu.RUnlock()
	return ptm.trackers[id]
}

// GetAllTrackers gets all progress trackers
func (ptm *ProgressTrackerManager) GetAllTrackers() map[string]*ProgressTrackerV3 {
	ptm.mu.RLock()
	defer ptm.mu.RUnlock()

	trackers := make(map[string]*ProgressTrackerV3)
	for k, v := range ptm.trackers {
		trackers[k] = v
	}
	return trackers
}

// GetProgressSummary gets a summary of all progress trackers
func (ptm *ProgressTrackerManager) GetProgressSummary() map[string]interface{} {
	ptm.mu.RLock()
	defer ptm.mu.RUnlock()

	summary := map[string]interface{}{
		"total_trackers":     len(ptm.trackers),
		"active_trackers":    0,
		"completed_trackers": 0,
		"failed_trackers":    0,
		"paused_trackers":    0,
		"trackers":           make(map[string]interface{}),
	}

	for id, tracker := range ptm.trackers {
		summary["trackers"].(map[string]interface{})[id] = map[string]interface{}{
			"component":   tracker.Component,
			"status":      tracker.Status,
			"progress":    tracker.Progress,
			"completed":   tracker.CompletedItems,
			"total":       tracker.TotalItems,
			"eta":         tracker.ETA.Format("15:04:05"),
			"start_time":  tracker.StartTime.Format("15:04:05"),
			"last_update": tracker.LastUpdate.Format("15:04:05"),
		}

		switch tracker.Status {
		case StatusRunning:
			summary["active_trackers"] = summary["active_trackers"].(int) + 1
		case StatusCompleted:
			summary["completed_trackers"] = summary["completed_trackers"].(int) + 1
		case StatusFailed:
			summary["failed_trackers"] = summary["failed_trackers"].(int) + 1
		case StatusPaused:
			summary["paused_trackers"] = summary["paused_trackers"].(int) + 1
		}
	}

	return summary
}

// RegisterCallback registers a callback for progress updates
func (ptm *ProgressTrackerManager) RegisterCallback(trackerID string, callback ProgressCallback) {
	ptm.mu.Lock()
	defer ptm.mu.Unlock()

	if ptm.callbacks[trackerID] == nil {
		ptm.callbacks[trackerID] = make([]ProgressCallback, 0)
	}
	ptm.callbacks[trackerID] = append(ptm.callbacks[trackerID], callback)
}

// Helper methods

// updatePerformance updates performance metrics for a tracker
func (ptm *ProgressTrackerManager) updatePerformance(tracker *ProgressTrackerV3) {
	if tracker.CompletedItems > 0 {
		elapsed := time.Since(tracker.StartTime)
		tracker.Performance.ActualElapsedTime = elapsed
		tracker.Performance.ItemsPerSecond = float64(tracker.CompletedItems) / elapsed.Seconds()
		tracker.Performance.AverageItemTime = elapsed / time.Duration(tracker.CompletedItems)

		if tracker.Performance.ItemsPerSecond > 0 {
			remainingItems := tracker.TotalItems - tracker.CompletedItems
			tracker.Performance.EstimatedTotalTime = time.Duration(float64(remainingItems)/tracker.Performance.ItemsPerSecond) * time.Second
			tracker.Performance.RemainingTime = tracker.Performance.EstimatedTotalTime
		}
	}
}

// calculateETA calculates the estimated time of arrival
func (ptm *ProgressTrackerManager) calculateETA(tracker *ProgressTrackerV3) {
	if tracker.Performance.ItemsPerSecond > 0 && tracker.CompletedItems < tracker.TotalItems {
		remainingItems := tracker.TotalItems - tracker.CompletedItems
		remainingTime := time.Duration(float64(remainingItems)/tracker.Performance.ItemsPerSecond) * time.Second
		tracker.ETA = time.Now().Add(remainingTime)
		tracker.Performance.RemainingTime = remainingTime
	}
}

// createSnapshot creates a snapshot of the tracker's current state
func (ptm *ProgressTrackerManager) createSnapshot(tracker *ProgressTrackerV3) {
	snapshot := &ProgressSnapshot{
		Timestamp: time.Now(),
		Progress:  tracker.Progress,
		Status:    tracker.Status,
		Completed: tracker.CompletedItems,
		Total:     tracker.TotalItems,
		ETA:       tracker.ETA,
		Context:   make(map[string]interface{}),
	}

	// Copy context
	for k, v := range tracker.Context {
		snapshot.Context[k] = v
	}

	// Add to history
	if ptm.history[tracker.ID] == nil {
		ptm.history[tracker.ID] = make([]*ProgressSnapshot, 0)
	}
	ptm.history[tracker.ID] = append(ptm.history[tracker.ID], snapshot)

	// Keep only maxHistory snapshots
	if len(ptm.history[tracker.ID]) > ptm.maxHistory {
		ptm.history[tracker.ID] = ptm.history[tracker.ID][1:]
	}
}

// createCheckpoint creates a checkpoint for the tracker
func (ptm *ProgressTrackerManager) createCheckpoint(tracker *ProgressTrackerV3, id, description string) {
	checkpoint := &ProgressCheckpoint{
		ID:          id,
		Timestamp:   time.Now(),
		Progress:    tracker.Progress,
		Completed:   tracker.CompletedItems,
		Total:       tracker.TotalItems,
		Description: description,
		Context:     make(map[string]interface{}),
	}

	// Copy context
	for k, v := range tracker.Context {
		checkpoint.Context[k] = v
	}

	tracker.Checkpoints = append(tracker.Checkpoints, checkpoint)
}

// shouldCreateCheckpoint determines if a checkpoint should be created
func (ptm *ProgressTrackerManager) shouldCreateCheckpoint(tracker *ProgressTrackerV3) bool {
	// Create checkpoint every 10% progress
	return int(tracker.Progress)%10 == 0 && tracker.Progress > 0
}

// notifyCallbacks notifies all registered callbacks for a tracker
func (ptm *ProgressTrackerManager) notifyCallbacks(tracker *ProgressTrackerV3) {
	callbacks, exists := ptm.callbacks[tracker.ID]
	if !exists {
		return
	}

	for _, callback := range callbacks {
		go callback(tracker)
	}
}
