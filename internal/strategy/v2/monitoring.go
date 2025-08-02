package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
)

// MonitoringV2 provides comprehensive monitoring for the V2 strategy engine
type MonitoringV2 struct {
	config *config.StrategyEngineV2Config
	mu     sync.RWMutex

	// Metrics
	engineMetrics      *EngineMetrics
	registryMetrics    *RegistryMetrics
	schedulerMetrics   *SchedulerMetrics
	persistenceMetrics map[string]interface{}

	// Performance tracking
	processingLatency []time.Duration
	memoryUsage       []int64
	errorRates        map[string]int64

	// Health status
	lastHeartbeat     time.Time
	isHealthy         bool
	healthCheckErrors []string
}

// NewMonitoringV2 creates a new monitoring instance
func NewMonitoringV2(config *config.StrategyEngineV2Config) *MonitoringV2 {
	return &MonitoringV2{
		config:             config,
		engineMetrics:      &EngineMetrics{},
		registryMetrics:    &RegistryMetrics{},
		schedulerMetrics:   &SchedulerMetrics{},
		persistenceMetrics: make(map[string]interface{}),
		processingLatency:  make([]time.Duration, 0, 100),
		memoryUsage:        make([]int64, 0, 100),
		errorRates:         make(map[string]int64),
		lastHeartbeat:      time.Now(),
		isHealthy:          true,
		healthCheckErrors:  make([]string, 0),
	}
}

// UpdateEngineMetrics updates the engine metrics
func (m *MonitoringV2) UpdateEngineMetrics(metrics *EngineMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engineMetrics = metrics
}

// UpdateRegistryMetrics updates the registry metrics
func (m *MonitoringV2) UpdateRegistryMetrics(metrics *RegistryMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registryMetrics = metrics
}

// UpdateSchedulerMetrics updates the scheduler metrics
func (m *MonitoringV2) UpdateSchedulerMetrics(metrics *SchedulerMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedulerMetrics = metrics
}

// UpdatePersistenceMetrics updates the persistence metrics
func (m *MonitoringV2) UpdatePersistenceMetrics(metrics map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.persistenceMetrics = metrics
}

// RecordProcessingLatency records processing latency for performance tracking
func (m *MonitoringV2) RecordProcessingLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.processingLatency = append(m.processingLatency, latency)

	// Keep only last 100 measurements
	if len(m.processingLatency) > 100 {
		m.processingLatency = m.processingLatency[1:]
	}
}

// RecordMemoryUsage records memory usage for performance tracking
func (m *MonitoringV2) RecordMemoryUsage(usageBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memoryUsage = append(m.memoryUsage, usageBytes)

	// Keep only last 100 measurements
	if len(m.memoryUsage) > 100 {
		m.memoryUsage = m.memoryUsage[1:]
	}
}

// RecordError records an error occurrence
func (m *MonitoringV2) RecordError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorRates[errorType]++
}

// UpdateHealthStatus updates the health status
func (m *MonitoringV2) UpdateHealthStatus(isHealthy bool, errors []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.isHealthy = isHealthy
	m.healthCheckErrors = errors
	m.lastHeartbeat = time.Now()
}

// GetComprehensiveMetrics returns all metrics in a structured format
func (m *MonitoringV2) GetComprehensiveMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate performance statistics
	var avgLatency time.Duration
	var maxLatency time.Duration
	var avgMemoryUsage int64
	var maxMemoryUsage int64

	if len(m.processingLatency) > 0 {
		totalLatency := time.Duration(0)
		for _, latency := range m.processingLatency {
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
		}
		avgLatency = totalLatency / time.Duration(len(m.processingLatency))
	}

	if len(m.memoryUsage) > 0 {
		totalMemory := int64(0)
		for _, usage := range m.memoryUsage {
			totalMemory += usage
			if usage > maxMemoryUsage {
				maxMemoryUsage = usage
			}
		}
		avgMemoryUsage = totalMemory / int64(len(m.memoryUsage))
	}

	// Calculate error rates
	totalErrors := int64(0)
	for _, count := range m.errorRates {
		totalErrors += count
	}

	return map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"health": map[string]interface{}{
			"is_healthy":          m.isHealthy,
			"last_heartbeat":      m.lastHeartbeat.Format(time.RFC3339),
			"health_check_errors": m.healthCheckErrors,
		},
		"performance": map[string]interface{}{
			"avg_processing_latency_ms": avgLatency.Milliseconds(),
			"max_processing_latency_ms": maxLatency.Milliseconds(),
			"avg_memory_usage_mb":       avgMemoryUsage / 1024 / 1024,
			"max_memory_usage_mb":       maxMemoryUsage / 1024 / 1024,
			"total_errors":              totalErrors,
			"error_rates":               m.errorRates,
		},
		"engine": map[string]interface{}{
			"total_processing_runs":     m.engineMetrics.TotalProcessingRuns,
			"total_stocks_processed":    m.engineMetrics.TotalStocksProcessed,
			"total_strategies_executed": m.engineMetrics.TotalStrategiesExecuted,
			"avg_processing_time_ms":    m.engineMetrics.AverageProcessingTime.Milliseconds(),
			"total_errors":              m.engineMetrics.TotalErrors,
			"last_processing_time":      m.engineMetrics.LastProcessingTime.Format(time.RFC3339),
			"memory_usage_mb":           m.engineMetrics.MemoryUsageMB,
		},
		"registry": map[string]interface{}{
			"total_strategies":         m.registryMetrics.TotalStrategies,
			"active_strategies":        m.registryMetrics.ActiveStrategies,
			"total_processing_time_ms": m.registryMetrics.TotalProcessingTime.Milliseconds(),
			"total_errors":             m.registryMetrics.TotalErrors,
			"last_execution_time":      m.registryMetrics.LastExecutionTime.Format(time.RFC3339),
		},
		"scheduler": map[string]interface{}{
			"total_v1_executions":       m.schedulerMetrics.TotalV1Executions,
			"total_v2_executions":       m.schedulerMetrics.TotalV2Executions,
			"total_v1_errors":           m.schedulerMetrics.TotalV1Errors,
			"total_v2_errors":           m.schedulerMetrics.TotalV2Errors,
			"avg_v1_processing_time_ms": m.schedulerMetrics.AverageV1ProcessingTime.Milliseconds(),
			"avg_v2_processing_time_ms": m.schedulerMetrics.AverageV2ProcessingTime.Milliseconds(),
			"last_v1_execution_time":    m.schedulerMetrics.LastV1ExecutionTime.Format(time.RFC3339),
			"last_v2_execution_time":    m.schedulerMetrics.LastV2ExecutionTime.Format(time.RFC3339),
		},
		"persistence": m.persistenceMetrics,
		"configuration": map[string]interface{}{
			"enabled":                    m.config.Enabled,
			"concurrent_workers":         m.config.Processing.ConcurrentWorkers,
			"processing_timeout_seconds": m.config.Processing.ProcessingTimeout.Seconds(),
			"max_historical_candles":     m.config.Processing.MaxHistoricalCandles,
			"batch_size":                 m.config.Processing.BatchSize,
			"data_retention_days":        m.config.Processing.DataRetentionDays,
			"parallel_strategies":        m.config.Strategies.Execution.ParallelStrategies,
			"strategy_timeout_seconds":   m.config.Strategies.Execution.StrategyTimeout.Seconds(),
			"batch_insert":               m.config.Persistence.BatchInsert,
			"enable_metrics":             m.config.Monitoring.EnableMetrics,
			"enable_tracing":             m.config.Monitoring.EnableTracing,
			"log_level":                  m.config.Monitoring.LogLevel,
		},
	}
}

// GetHealthStatus returns the current health status
func (m *MonitoringV2) GetHealthStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"is_healthy":          m.isHealthy,
		"last_heartbeat":      m.lastHeartbeat.Format(time.RFC3339),
		"health_check_errors": m.healthCheckErrors,
		"uptime_seconds":      time.Since(m.lastHeartbeat).Seconds(),
	}
}

// GetPerformanceMetrics returns performance-specific metrics
func (m *MonitoringV2) GetPerformanceMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgLatency time.Duration
	var maxLatency time.Duration
	var avgMemoryUsage int64
	var maxMemoryUsage int64

	if len(m.processingLatency) > 0 {
		totalLatency := time.Duration(0)
		for _, latency := range m.processingLatency {
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
		}
		avgLatency = totalLatency / time.Duration(len(m.processingLatency))
	}

	if len(m.memoryUsage) > 0 {
		totalMemory := int64(0)
		for _, usage := range m.memoryUsage {
			totalMemory += usage
			if usage > maxMemoryUsage {
				maxMemoryUsage = usage
			}
		}
		avgMemoryUsage = totalMemory / int64(len(m.memoryUsage))
	}

	return map[string]interface{}{
		"processing_latency": map[string]interface{}{
			"avg_ms": avgLatency.Milliseconds(),
			"max_ms": maxLatency.Milliseconds(),
			"min_ms": func() int64 {
				if len(m.processingLatency) == 0 {
					return 0
				}
				min := m.processingLatency[0]
				for _, latency := range m.processingLatency {
					if latency < min {
						min = latency
					}
				}
				return min.Milliseconds()
			}(),
			"measurements": len(m.processingLatency),
		},
		"memory_usage": map[string]interface{}{
			"avg_mb": avgMemoryUsage / 1024 / 1024,
			"max_mb": maxMemoryUsage / 1024 / 1024,
			"min_mb": func() int64 {
				if len(m.memoryUsage) == 0 {
					return 0
				}
				min := m.memoryUsage[0]
				for _, usage := range m.memoryUsage {
					if usage < min {
						min = usage
					}
				}
				return min / 1024 / 1024
			}(),
			"measurements": len(m.memoryUsage),
		},
		"error_rates": m.errorRates,
	}
}

// ExportMetricsAsJSON exports all metrics as JSON
func (m *MonitoringV2) ExportMetricsAsJSON() ([]byte, error) {
	metrics := m.GetComprehensiveMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

// LogMetrics logs the current metrics
func (m *MonitoringV2) LogMetrics() {
	metrics := m.GetComprehensiveMetrics()

	log.Info("=== V2 Strategy Engine Metrics ===")
	log.Info("Health: %v", metrics["health"])
	log.Info("Performance: %v", metrics["performance"])
	log.Info("Engine: %v", metrics["engine"])
	log.Info("Registry: %v", metrics["registry"])
	log.Info("Scheduler: %v", metrics["scheduler"])
	log.Info("Persistence: %v", metrics["persistence"])
	log.Info("==================================")
}

// StartMetricsCollection starts periodic metrics collection
func (m *MonitoringV2) StartMetricsCollection(ctx context.Context) {
	if !m.config.Monitoring.EnableMetrics {
		log.Info("Metrics collection disabled in configuration")
		return
	}

	ticker := time.NewTicker(30 * time.Second) // Collect metrics every 30 seconds
	defer ticker.Stop()

	log.Info("Starting V2 metrics collection")

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping V2 metrics collection")
			return
		case <-ticker.C:
			m.LogMetrics()
		}
	}
}

// PerformHealthCheck performs a comprehensive health check
func (m *MonitoringV2) PerformHealthCheck(ctx context.Context) error {
	var errors []string

	// Check if engine is responding
	if m.engineMetrics == nil {
		errors = append(errors, "Engine metrics not available")
	}

	// Check if registry is healthy
	if m.registryMetrics == nil {
		errors = append(errors, "Registry metrics not available")
	}

	// Check processing latency
	if len(m.processingLatency) > 0 {
		latestLatency := m.processingLatency[len(m.processingLatency)-1]
		if latestLatency > m.config.Processing.ProcessingTimeout {
			errors = append(errors, fmt.Sprintf("Processing latency (%v) exceeds timeout (%v)",
				latestLatency, m.config.Processing.ProcessingTimeout))
		}
	}

	// Check memory usage
	if len(m.memoryUsage) > 0 {
		latestMemory := m.memoryUsage[len(m.memoryUsage)-1]
		memoryLimitMB := int64(m.config.DataFrame.MemoryLimitMB) * 1024 * 1024
		if latestMemory > memoryLimitMB {
			errors = append(errors, fmt.Sprintf("Memory usage (%d MB) exceeds limit (%d MB)",
				latestMemory/1024/1024, m.config.DataFrame.MemoryLimitMB))
		}
	}

	// Check error rates
	totalErrors := int64(0)
	for _, count := range m.errorRates {
		totalErrors += count
	}
	if totalErrors > 100 { // Arbitrary threshold
		errors = append(errors, fmt.Sprintf("High error rate: %d total errors", totalErrors))
	}

	isHealthy := len(errors) == 0
	m.UpdateHealthStatus(isHealthy, errors)

	if !isHealthy {
		log.Warn("V2 Strategy Engine health check failed: %v", errors)
		return fmt.Errorf("health check failed: %v", errors)
	}

	log.Info("V2 Strategy Engine health check passed")
	return nil
}
