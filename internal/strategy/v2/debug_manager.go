package v2

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// DebugFlags represents debug configuration flags
type DebugFlags struct {
	Enabled              bool `json:"enabled"`
	TraceActive          bool `json:"trace_active"`
	StateInspection      bool `json:"state_inspection"`
	MemoryInspection     bool `json:"memory_inspection"`
	PerformanceProfiling bool `json:"performance_profiling"`
	ErrorTracking        bool `json:"error_tracking"`
	ProgressTracking     bool `json:"progress_tracking"`
}

// DebugManager manages debugging features and controls
type DebugManager struct {
	flags              DebugFlags
	logger             DebugLogger
	mu                 sync.RWMutex
	stateSnapshots     map[string]*StateSnapshot
	errorTracker       *ErrorTracker
	performanceMonitor *PerformanceMonitor
	memoryMonitor      *MemoryMonitor
	progressTracker    *ProgressTracker
}

// StateSnapshot represents a snapshot of system state
type StateSnapshot struct {
	Timestamp   time.Time              `json:"timestamp"`
	Component   string                 `json:"component"`
	State       map[string]interface{} `json:"state"`
	MemoryStats *runtime.MemStats      `json:"memory_stats"`
	Goroutines  int                    `json:"goroutines"`
}

// ErrorTracker tracks errors and their patterns
type ErrorTracker struct {
	errors      []*TrackedError
	errorCounts map[string]int
	mu          sync.RWMutex
	maxErrors   int
}

// TrackedError represents a tracked error with context
type TrackedError struct {
	Timestamp   time.Time              `json:"timestamp"`
	Error       string                 `json:"error"`
	Component   string                 `json:"component"`
	Method      string                 `json:"method"`
	Context     map[string]interface{} `json:"context"`
	Stack       string                 `json:"stack"`
	Occurrences int                    `json:"occurrences"`
}

// PerformanceMonitor tracks performance metrics
type PerformanceMonitor struct {
	metrics   map[string]*PerformanceMetric
	mu        sync.RWMutex
	startTime time.Time
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Component     string        `json:"component"`
	Method        string        `json:"method"`
	TotalCalls    int64         `json:"total_calls"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
	LastCall      time.Time     `json:"last_call"`
}

// MemoryMonitor tracks memory usage
type MemoryMonitor struct {
	snapshots    []*MemorySnapshot
	mu           sync.RWMutex
	maxSnapshots int
}

// MemorySnapshot represents a memory usage snapshot
type MemorySnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	Alloc        uint64    `json:"alloc"`
	TotalAlloc   uint64    `json:"total_alloc"`
	Sys          uint64    `json:"sys"`
	NumGC        uint32    `json:"num_gc"`
	PauseTotalNs uint64    `json:"pause_total_ns"`
	HeapAlloc    uint64    `json:"heap_alloc"`
	HeapSys      uint64    `json:"heap_sys"`
	HeapIdle     uint64    `json:"heap_idle"`
	HeapInuse    uint64    `json:"heap_inuse"`
	HeapReleased uint64    `json:"heap_released"`
	HeapObjects  uint64    `json:"heap_objects"`
	StackInuse   uint64    `json:"stack_inuse"`
	StackSys     uint64    `json:"stack_sys"`
	Goroutines   int       `json:"goroutines"`
}

// ProgressTracker tracks progress of operations
type ProgressTracker struct {
	trackers map[string]*ProgressInfo
	mu       sync.RWMutex
}

// ProgressInfo represents progress information
type ProgressInfo struct {
	ID             string                 `json:"id"`
	Component      string                 `json:"component"`
	Status         string                 `json:"status"`
	Progress       float64                `json:"progress"`
	TotalItems     int                    `json:"total_items"`
	CompletedItems int                    `json:"completed_items"`
	StartTime      time.Time              `json:"start_time"`
	LastUpdate     time.Time              `json:"last_update"`
	ETA            time.Time              `json:"eta"`
	Context        map[string]interface{} `json:"context"`
}

// NewDebugManager creates a new debug manager
func NewDebugManager(logger DebugLogger) *DebugManager {
	dm := &DebugManager{
		logger: logger,
		flags: DebugFlags{
			Enabled:              false,
			TraceActive:          false,
			StateInspection:      false,
			MemoryInspection:     false,
			PerformanceProfiling: false,
			ErrorTracking:        false,
			ProgressTracking:     false,
		},
		stateSnapshots:     make(map[string]*StateSnapshot),
		errorTracker:       NewErrorTracker(1000), // Keep last 1000 errors
		performanceMonitor: NewPerformanceMonitor(),
		memoryMonitor:      NewMemoryMonitor(100), // Keep last 100 snapshots
		progressTracker:    NewProgressTracker(),
	}

	// Log debug manager creation
	if logger != nil {
		helper := NewLogHelper(logger, GenerateTraceID(), "debug_manager")
		helper.LogInfo(SYSTEM, "DebugManager", "NewDebugManager", "", "",
			"Debug manager created", map[string]interface{}{
				"enabled": dm.flags.Enabled,
			})
	}

	return dm
}

// EnableDebug enables all debug features
func (dm *DebugManager) EnableDebug() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.flags.Enabled = true
	dm.flags.TraceActive = true
	dm.flags.StateInspection = true
	dm.flags.MemoryInspection = true
	dm.flags.PerformanceProfiling = true
	dm.flags.ErrorTracking = true
	dm.flags.ProgressTracking = true

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogInfo(SYSTEM, "DebugManager", "EnableDebug", "", "",
			"Debug mode enabled", map[string]interface{}{
				"flags": dm.flags,
			})
	}
}

// DisableDebug disables all debug features
func (dm *DebugManager) DisableDebug() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.flags.Enabled = false
	dm.flags.TraceActive = false
	dm.flags.StateInspection = false
	dm.flags.MemoryInspection = false
	dm.flags.PerformanceProfiling = false
	dm.flags.ErrorTracking = false
	dm.flags.ProgressTracking = false

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogInfo(SYSTEM, "DebugManager", "DisableDebug", "", "",
			"Debug mode disabled", map[string]interface{}{
				"flags": dm.flags,
			})
	}
}

// SetDebugFlag sets a specific debug flag
func (dm *DebugManager) SetDebugFlag(flag string, value bool) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	switch flag {
	case "enabled":
		dm.flags.Enabled = value
	case "trace_active":
		dm.flags.TraceActive = value
	case "state_inspection":
		dm.flags.StateInspection = value
	case "memory_inspection":
		dm.flags.MemoryInspection = value
	case "performance_profiling":
		dm.flags.PerformanceProfiling = value
	case "error_tracking":
		dm.flags.ErrorTracking = value
	case "progress_tracking":
		dm.flags.ProgressTracking = value
	}

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogInfo(SYSTEM, "DebugManager", "SetDebugFlag", "", "",
			"Debug flag set", map[string]interface{}{
				"flag":  flag,
				"value": value,
			})
	}
}

// GetDebugFlags returns current debug flags
func (dm *DebugManager) GetDebugFlags() DebugFlags {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.flags
}

// IsDebugEnabled checks if debug mode is enabled
func (dm *DebugManager) IsDebugEnabled() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.flags.Enabled
}

// IsFlagEnabled checks if a specific flag is enabled
func (dm *DebugManager) IsFlagEnabled(flag string) bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	switch flag {
	case "enabled":
		return dm.flags.Enabled
	case "trace_active":
		return dm.flags.TraceActive
	case "state_inspection":
		return dm.flags.StateInspection
	case "memory_inspection":
		return dm.flags.MemoryInspection
	case "performance_profiling":
		return dm.flags.PerformanceProfiling
	case "error_tracking":
		return dm.flags.ErrorTracking
	case "progress_tracking":
		return dm.flags.ProgressTracking
	default:
		return false
	}
}

// TakeStateSnapshot takes a snapshot of system state
func (dm *DebugManager) TakeStateSnapshot(component string, state map[string]interface{}) {
	if !dm.IsFlagEnabled("state_inspection") {
		return
	}

	snapshot := &StateSnapshot{
		Timestamp:  time.Now(),
		Component:  component,
		State:      state,
		Goroutines: runtime.NumGoroutine(),
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	snapshot.MemoryStats = &memStats

	dm.mu.Lock()
	dm.stateSnapshots[component] = snapshot
	dm.mu.Unlock()

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogDebug(SYSTEM, "DebugManager", "TakeStateSnapshot", "", "",
			"State snapshot taken", map[string]interface{}{
				"component":       component,
				"goroutines":      snapshot.Goroutines,
				"memory_alloc_mb": snapshot.MemoryStats.Alloc / 1024 / 1024,
			})
	}
}

// GetStateSnapshot gets a state snapshot for a component
func (dm *DebugManager) GetStateSnapshot(component string) *StateSnapshot {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.stateSnapshots[component]
}

// GetAllStateSnapshots gets all state snapshots
func (dm *DebugManager) GetAllStateSnapshots() map[string]*StateSnapshot {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	snapshots := make(map[string]*StateSnapshot)
	for k, v := range dm.stateSnapshots {
		snapshots[k] = v
	}
	return snapshots
}

// TrackError tracks an error for analysis
func (dm *DebugManager) TrackError(component, method string, err error, context map[string]interface{}) {
	if !dm.IsFlagEnabled("error_tracking") {
		return
	}

	dm.errorTracker.TrackError(component, method, err, context)

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogError(ERROR_CAT, "DebugManager", "TrackError", "", "",
			"Error tracked", context, err)
	}
}

// GetErrorStats gets error statistics
func (dm *DebugManager) GetErrorStats() map[string]interface{} {
	return dm.errorTracker.GetStats()
}

// TrackPerformance tracks performance metrics
func (dm *DebugManager) TrackPerformance(component, method string, duration time.Duration) {
	if !dm.IsFlagEnabled("performance_profiling") {
		return
	}

	dm.performanceMonitor.TrackMetric(component, method, duration)

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogPerformance("DebugManager", "TrackPerformance", "", "",
			"Performance tracked", duration, 0, map[string]interface{}{
				"component": component,
				"method":    method,
			})
	}
}

// GetPerformanceStats gets performance statistics
func (dm *DebugManager) GetPerformanceStats() map[string]*PerformanceMetric {
	return dm.performanceMonitor.GetStats()
}

// TakeMemorySnapshot takes a memory usage snapshot
func (dm *DebugManager) TakeMemorySnapshot() {
	if !dm.IsFlagEnabled("memory_inspection") {
		return
	}

	snapshot := dm.memoryMonitor.TakeSnapshot()

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogInfo(PERFORMANCE, "DebugManager", "TakeMemorySnapshot", "", "",
			"Memory snapshot taken", map[string]interface{}{
				"alloc_mb":       snapshot.Alloc / 1024 / 1024,
				"total_alloc_mb": snapshot.TotalAlloc / 1024 / 1024,
				"sys_mb":         snapshot.Sys / 1024 / 1024,
				"goroutines":     snapshot.Goroutines,
				"heap_objects":   snapshot.HeapObjects,
			})
	}
}

// GetMemoryStats gets memory statistics
func (dm *DebugManager) GetMemoryStats() []*MemorySnapshot {
	return dm.memoryMonitor.GetSnapshots()
}

// StartProgressTracking starts tracking progress for an operation
func (dm *DebugManager) StartProgressTracking(id, component string, totalItems int, context map[string]interface{}) {
	if !dm.IsFlagEnabled("progress_tracking") {
		return
	}

	dm.progressTracker.StartTracking(id, component, totalItems, context)

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogInfo(PROGRESS, "DebugManager", "StartProgressTracking", "", "",
			"Progress tracking started", map[string]interface{}{
				"id":          id,
				"component":   component,
				"total_items": totalItems,
			})
	}
}

// UpdateProgress updates progress for an operation
func (dm *DebugManager) UpdateProgress(id string, completedItems int, status string, context map[string]interface{}) {
	if !dm.IsFlagEnabled("progress_tracking") {
		return
	}

	dm.progressTracker.UpdateProgress(id, completedItems, status, context)

	if dm.logger != nil {
		helper := NewLogHelper(dm.logger, GenerateTraceID(), "debug_manager")
		helper.LogInfo(PROGRESS, "DebugManager", "UpdateProgress", "", "",
			"Progress updated", map[string]interface{}{
				"id":              id,
				"completed_items": completedItems,
				"status":          status,
			})
	}
}

// GetProgressInfo gets progress information
func (dm *DebugManager) GetProgressInfo(id string) *ProgressInfo {
	return dm.progressTracker.GetProgress(id)
}

// GetAllProgressInfo gets all progress information
func (dm *DebugManager) GetAllProgressInfo() map[string]*ProgressInfo {
	return dm.progressTracker.GetAllProgress()
}

// GetDebugSummary gets a comprehensive debug summary
func (dm *DebugManager) GetDebugSummary() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	summary := map[string]interface{}{
		"flags":             dm.flags,
		"state_snapshots":   len(dm.stateSnapshots),
		"error_stats":       dm.errorTracker.GetStats(),
		"performance_stats": dm.performanceMonitor.GetStats(),
		"memory_snapshots":  len(dm.memoryMonitor.GetSnapshots()),
		"progress_trackers": len(dm.progressTracker.GetAllProgress()),
		"goroutines":        runtime.NumGoroutine(),
	}

	// Add current memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	summary["current_memory"] = map[string]interface{}{
		"alloc_mb":       memStats.Alloc / 1024 / 1024,
		"total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
		"sys_mb":         memStats.Sys / 1024 / 1024,
		"heap_objects":   memStats.HeapObjects,
	}

	return summary
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker(maxErrors int) *ErrorTracker {
	return &ErrorTracker{
		errors:      make([]*TrackedError, 0),
		errorCounts: make(map[string]int),
		maxErrors:   maxErrors,
	}
}

// TrackError tracks an error
func (et *ErrorTracker) TrackError(component, method string, err error, context map[string]interface{}) {
	et.mu.Lock()
	defer et.mu.Unlock()

	errorKey := fmt.Sprintf("%s:%s:%s", component, method, err.Error())

	// Update error count
	et.errorCounts[errorKey]++

	// Create tracked error
	trackedError := &TrackedError{
		Timestamp:   time.Now(),
		Error:       err.Error(),
		Component:   component,
		Method:      method,
		Context:     context,
		Stack:       string(debug.Stack()),
		Occurrences: et.errorCounts[errorKey],
	}

	// Add to errors list
	et.errors = append(et.errors, trackedError)

	// Keep only maxErrors
	if len(et.errors) > et.maxErrors {
		et.errors = et.errors[1:]
	}
}

// GetStats gets error statistics
func (et *ErrorTracker) GetStats() map[string]interface{} {
	et.mu.RLock()
	defer et.mu.RUnlock()

	return map[string]interface{}{
		"total_errors":  len(et.errors),
		"error_counts":  et.errorCounts,
		"recent_errors": et.errors,
	}
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:   make(map[string]*PerformanceMetric),
		startTime: time.Now(),
	}
}

// TrackMetric tracks a performance metric
func (pm *PerformanceMonitor) TrackMetric(component, method string, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", component, method)

	metric, exists := pm.metrics[key]
	if !exists {
		metric = &PerformanceMetric{
			Component:   component,
			Method:      method,
			MinDuration: duration,
		}
		pm.metrics[key] = metric
	}

	metric.TotalCalls++
	metric.TotalDuration += duration
	metric.AvgDuration = metric.TotalDuration / time.Duration(metric.TotalCalls)
	metric.LastCall = time.Now()

	if duration < metric.MinDuration {
		metric.MinDuration = duration
	}
	if duration > metric.MaxDuration {
		metric.MaxDuration = duration
	}
}

// GetStats gets performance statistics
func (pm *PerformanceMonitor) GetStats() map[string]*PerformanceMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]*PerformanceMetric)
	for k, v := range pm.metrics {
		stats[k] = v
	}
	return stats
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(maxSnapshots int) *MemoryMonitor {
	return &MemoryMonitor{
		snapshots:    make([]*MemorySnapshot, 0),
		maxSnapshots: maxSnapshots,
	}
}

// TakeSnapshot takes a memory snapshot
func (mm *MemoryMonitor) TakeSnapshot() *MemorySnapshot {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	snapshot := &MemorySnapshot{
		Timestamp:  time.Now(),
		Goroutines: runtime.NumGoroutine(),
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	snapshot.Alloc = memStats.Alloc
	snapshot.TotalAlloc = memStats.TotalAlloc
	snapshot.Sys = memStats.Sys
	snapshot.NumGC = memStats.NumGC
	snapshot.PauseTotalNs = memStats.PauseTotalNs
	snapshot.HeapAlloc = memStats.HeapAlloc
	snapshot.HeapSys = memStats.HeapSys
	snapshot.HeapIdle = memStats.HeapIdle
	snapshot.HeapInuse = memStats.HeapInuse
	snapshot.HeapReleased = memStats.HeapReleased
	snapshot.HeapObjects = memStats.HeapObjects
	snapshot.StackInuse = memStats.StackInuse
	snapshot.StackSys = memStats.StackSys

	mm.snapshots = append(mm.snapshots, snapshot)

	// Keep only maxSnapshots
	if len(mm.snapshots) > mm.maxSnapshots {
		mm.snapshots = mm.snapshots[1:]
	}

	return snapshot
}

// GetSnapshots gets all memory snapshots
func (mm *MemoryMonitor) GetSnapshots() []*MemorySnapshot {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	snapshots := make([]*MemorySnapshot, len(mm.snapshots))
	copy(snapshots, mm.snapshots)
	return snapshots
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		trackers: make(map[string]*ProgressInfo),
	}
}

// StartTracking starts tracking progress
func (pt *ProgressTracker) StartTracking(id, component string, totalItems int, context map[string]interface{}) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.trackers[id] = &ProgressInfo{
		ID:             id,
		Component:      component,
		Status:         "started",
		Progress:       0.0,
		TotalItems:     totalItems,
		CompletedItems: 0,
		StartTime:      time.Now(),
		LastUpdate:     time.Now(),
		Context:        context,
	}
}

// UpdateProgress updates progress
func (pt *ProgressTracker) UpdateProgress(id string, completedItems int, status string, context map[string]interface{}) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if tracker, exists := pt.trackers[id]; exists {
		tracker.CompletedItems = completedItems
		tracker.Status = status
		tracker.LastUpdate = time.Now()

		if tracker.TotalItems > 0 {
			tracker.Progress = float64(completedItems) / float64(tracker.TotalItems) * 100.0
		}

		// Calculate ETA
		if tracker.Progress > 0 {
			elapsed := time.Since(tracker.StartTime)
			estimatedTotal := elapsed * time.Duration(tracker.TotalItems) / time.Duration(completedItems)
			tracker.ETA = tracker.StartTime.Add(estimatedTotal)
		}

		// Update context
		for k, v := range context {
			tracker.Context[k] = v
		}
	}
}

// GetProgress gets progress for an ID
func (pt *ProgressTracker) GetProgress(id string) *ProgressInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.trackers[id]
}

// GetAllProgress gets all progress information
func (pt *ProgressTracker) GetAllProgress() map[string]*ProgressInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress := make(map[string]*ProgressInfo)
	for k, v := range pt.trackers {
		progress[k] = v
	}
	return progress
}
