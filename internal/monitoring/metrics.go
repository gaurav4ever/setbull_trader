package monitoring

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// MetricsCollector aggregates performance metrics for production monitoring
type MetricsCollector struct {
	mu sync.RWMutex

	// Request metrics
	TotalRequests    int64 `json:"total_requests"`
	V1Requests       int64 `json:"v1_requests"`
	V2Requests       int64 `json:"v2_requests"`
	ErrorRequests    int64 `json:"error_requests"`
	FallbackRequests int64 `json:"fallback_requests"`

	// Performance metrics
	AvgV1ResponseTime time.Duration   `json:"avg_v1_response_time"`
	AvgV2ResponseTime time.Duration   `json:"avg_v2_response_time"`
	V1ResponseTimes   []time.Duration `json:"-"` // Keep last 1000 samples
	V2ResponseTimes   []time.Duration `json:"-"` // Keep last 1000 samples

	// Memory and resource metrics
	V1MemoryUsage int64   `json:"v1_memory_usage_mb"`
	V2MemoryUsage int64   `json:"v2_memory_usage_mb"`
	CacheHitRate  float64 `json:"cache_hit_rate"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`

	// Error tracking
	ErrorsByType map[string]int64 `json:"errors_by_type"`
	LastErrors   []ErrorEvent     `json:"last_errors"`

	// System metrics
	CPUUsage       float64 `json:"cpu_usage_percent"`
	MemoryUsage    float64 `json:"memory_usage_percent"`
	GoroutineCount int     `json:"goroutine_count"`

	// Alerting
	alertThresholds *AlertThresholds
	alertCallbacks  []AlertCallback

	startTime time.Time
}

// ErrorEvent represents an error that occurred during processing
type ErrorEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	ServiceType string    `json:"service_type"` // "v1" or "v2"
	RequestID   string    `json:"request_id"`
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	ErrorRatePercent    float64       `json:"error_rate_percent"`     // Alert if error rate > this %
	ResponseTimeP95     time.Duration `json:"response_time_p95"`      // Alert if P95 > this duration
	MemoryUsagePercent  float64       `json:"memory_usage_percent"`   // Alert if memory > this %
	CacheHitRatePercent float64       `json:"cache_hit_rate_percent"` // Alert if cache hit rate < this %
}

// AlertCallback is called when an alert threshold is exceeded
type AlertCallback func(alertType string, message string, metrics *MetricsCollector)

// NewMetricsCollector creates a new metrics collector with default alert thresholds
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		ErrorsByType: make(map[string]int64),
		LastErrors:   make([]ErrorEvent, 0, 100), // Keep last 100 errors
		alertThresholds: &AlertThresholds{
			ErrorRatePercent:    5.0,             // Alert if > 5% error rate
			ResponseTimeP95:     5 * time.Second, // Alert if P95 > 5 seconds
			MemoryUsagePercent:  80.0,            // Alert if > 80% memory usage
			CacheHitRatePercent: 70.0,            // Alert if < 70% cache hit rate
		},
		startTime: time.Now(),
	}
}

// RecordRequest records metrics for a completed request
func (m *MetricsCollector) RecordRequest(serviceType string, duration time.Duration, err error, requestID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++

	if serviceType == "v1" {
		m.V1Requests++
		m.recordResponseTime(&m.V1ResponseTimes, duration)
		m.AvgV1ResponseTime = m.calculateAverage(m.V1ResponseTimes)
	} else if serviceType == "v2" {
		m.V2Requests++
		m.recordResponseTime(&m.V2ResponseTimes, duration)
		m.AvgV2ResponseTime = m.calculateAverage(m.V2ResponseTimes)
	}

	if err != nil {
		m.ErrorRequests++
		errorType := fmt.Sprintf("%T", err)
		m.ErrorsByType[errorType]++

		// Record error event
		errorEvent := ErrorEvent{
			Timestamp:   time.Now(),
			Type:        errorType,
			Message:     err.Error(),
			ServiceType: serviceType,
			RequestID:   requestID,
		}

		m.LastErrors = append(m.LastErrors, errorEvent)
		if len(m.LastErrors) > 100 {
			m.LastErrors = m.LastErrors[1:]
		}

		// Check for alert conditions
		m.checkAlerts()
	}
}

// RecordFallback records when a V2 request falls back to V1
func (m *MetricsCollector) RecordFallback(requestID string, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.FallbackRequests++

	log.Printf("FALLBACK: Request %s fell back to V1. Reason: %s", requestID, reason)

	// Record as a specific error type
	m.ErrorsByType["fallback_to_v1"]++
}

// RecordMemoryUsage records memory usage for a service version
func (m *MetricsCollector) RecordMemoryUsage(serviceType string, memoryMB int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if serviceType == "v1" {
		m.V1MemoryUsage = memoryMB
	} else if serviceType == "v2" {
		m.V2MemoryUsage = memoryMB
	}
}

// RecordCacheEvent records cache hit or miss
func (m *MetricsCollector) RecordCacheEvent(hit bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hit {
		m.CacheHits++
	} else {
		m.CacheMisses++
	}

	total := m.CacheHits + m.CacheMisses
	if total > 0 {
		m.CacheHitRate = float64(m.CacheHits) / float64(total) * 100
	}
}

// UpdateSystemMetrics updates system-level metrics
func (m *MetricsCollector) UpdateSystemMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Update system metrics
	m.GoroutineCount = runtime.NumGoroutine()
	m.MemoryUsage = float64(memStats.Alloc) / float64(memStats.Sys) * 100

	// CPU usage would require additional libraries in a real implementation
	// For now, we'll simulate it or leave it for external monitoring
}

// GetSnapshot returns a snapshot of current metrics
func (m *MetricsCollector) GetSnapshot() *MetricsCollector {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy
	snapshot := &MetricsCollector{}
	*snapshot = *m

	// Deep copy slices and maps
	snapshot.ErrorsByType = make(map[string]int64)
	for k, v := range m.ErrorsByType {
		snapshot.ErrorsByType[k] = v
	}

	snapshot.LastErrors = make([]ErrorEvent, len(m.LastErrors))
	copy(snapshot.LastErrors, m.LastErrors)

	return snapshot
}

// GetErrorRate returns the current error rate as a percentage
func (m *MetricsCollector) GetErrorRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRequests == 0 {
		return 0
	}

	return float64(m.ErrorRequests) / float64(m.TotalRequests) * 100
}

// GetFallbackRate returns the fallback rate as a percentage
func (m *MetricsCollector) GetFallbackRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.V2Requests == 0 {
		return 0
	}

	return float64(m.FallbackRequests) / float64(m.V2Requests) * 100
}

// GetUptime returns how long the service has been running
func (m *MetricsCollector) GetUptime() time.Duration {
	return time.Since(m.startTime)
}

// AddAlertCallback adds a callback function to be called when alerts are triggered
func (m *MetricsCollector) AddAlertCallback(callback AlertCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alertCallbacks = append(m.alertCallbacks, callback)
}

// SetAlertThresholds updates the alert thresholds
func (m *MetricsCollector) SetAlertThresholds(thresholds *AlertThresholds) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alertThresholds = thresholds
}

// checkAlerts checks if any alert thresholds have been exceeded
func (m *MetricsCollector) checkAlerts() {
	if m.alertThresholds == nil {
		return
	}

	// Check error rate
	errorRate := m.GetErrorRate()
	if errorRate > m.alertThresholds.ErrorRatePercent {
		m.triggerAlert("HIGH_ERROR_RATE", fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%", errorRate, m.alertThresholds.ErrorRatePercent))
	}

	// Check memory usage
	if m.MemoryUsage > m.alertThresholds.MemoryUsagePercent {
		m.triggerAlert("HIGH_MEMORY_USAGE", fmt.Sprintf("Memory usage %.2f%% exceeds threshold %.2f%%", m.MemoryUsage, m.alertThresholds.MemoryUsagePercent))
	}

	// Check cache hit rate
	if m.CacheHitRate < m.alertThresholds.CacheHitRatePercent && m.CacheHits+m.CacheMisses > 100 {
		m.triggerAlert("LOW_CACHE_HIT_RATE", fmt.Sprintf("Cache hit rate %.2f%% below threshold %.2f%%", m.CacheHitRate, m.alertThresholds.CacheHitRatePercent))
	}
}

// triggerAlert calls all registered alert callbacks
func (m *MetricsCollector) triggerAlert(alertType string, message string) {
	for _, callback := range m.alertCallbacks {
		go callback(alertType, message, m)
	}
}

// recordResponseTime records a response time, keeping only the last 1000 samples
func (m *MetricsCollector) recordResponseTime(times *[]time.Duration, duration time.Duration) {
	*times = append(*times, duration)
	if len(*times) > 1000 {
		*times = (*times)[1:]
	}
}

// calculateAverage calculates the average of response times
func (m *MetricsCollector) calculateAverage(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range times {
		total += t
	}

	return total / time.Duration(len(times))
}

// StartPeriodicReporting starts a goroutine that logs metrics periodically
func (m *MetricsCollector) StartPeriodicReporting(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.UpdateSystemMetrics()
			m.logCurrentMetrics()
		}
	}
}

// logCurrentMetrics logs a summary of current metrics
func (m *MetricsCollector) logCurrentMetrics() {
	snapshot := m.GetSnapshot()

	log.Printf("METRICS SUMMARY: Total=%d, V1=%d, V2=%d, Errors=%d (%.2f%%), Fallbacks=%d (%.2f%%), Cache=%.1f%%, Memory=%.1fMB/%.1fMB, Uptime=%v",
		snapshot.TotalRequests,
		snapshot.V1Requests,
		snapshot.V2Requests,
		snapshot.ErrorRequests,
		snapshot.GetErrorRate(),
		snapshot.FallbackRequests,
		snapshot.GetFallbackRate(),
		snapshot.CacheHitRate,
		float64(snapshot.V1MemoryUsage),
		float64(snapshot.V2MemoryUsage),
		snapshot.GetUptime(),
	)
}
