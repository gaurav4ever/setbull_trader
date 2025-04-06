package service

import (
	"setbull_trader/pkg/log"
	"sync"
	"time"
)

// PipelineMonitor handles monitoring and statistics for the filter pipeline
type PipelineMonitor struct {
	metrics        map[string][]PipelineMetrics
	dailyStats     map[string]FilterStatistics
	mutex          sync.RWMutex
	maxHistoryDays int
}

type FilterStatistics struct {
	TotalRuns      int
	AvgProcessTime time.Duration
	SuccessRate    float64
	BullishRate    float64
	BearishRate    float64
	ErrorRate      float64
}

func NewPipelineMonitor() *PipelineMonitor {
	return &PipelineMonitor{
		metrics:        make(map[string][]PipelineMetrics),
		dailyStats:     make(map[string]FilterStatistics),
		maxHistoryDays: 30,
	}
}

// RecordMetrics records metrics from a pipeline run
func (m *PipelineMonitor) RecordMetrics(date string, metrics PipelineMetrics) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Store metrics
	m.metrics[date] = append(m.metrics[date], metrics)

	// Update daily statistics
	stats := m.dailyStats[date]
	stats.TotalRuns++
	stats.AvgProcessTime += (metrics.ProcessingTime - stats.AvgProcessTime) / time.Duration(stats.TotalRuns)

	if len(m.metrics) > m.maxHistoryDays {
		m.cleanOldMetrics()
	}

	// Calculate success rates
	for filterName, metric := range metrics.FilterMetrics {
		successRate := float64(metric.Passed) / float64(metric.Processed)
		bullishRate := float64(metric.Bullish) / float64(metric.Passed)
		bearishRate := float64(metric.Bearish) / float64(metric.Passed)

		log.Info("Filter %s statistics:", filterName)
		log.Info("- Success Rate: %.2f%%", successRate*100)
		log.Info("- Bullish Rate: %.2f%%", bullishRate*100)
		log.Info("- Bearish Rate: %.2f%%", bearishRate*100)
	}
}

// cleanOldMetrics removes metrics older than maxHistoryDays
func (m *PipelineMonitor) cleanOldMetrics() {
	cutoffDate := time.Now().AddDate(0, 0, -m.maxHistoryDays).Format("2006-01-02")
	for date := range m.metrics {
		if date < cutoffDate {
			delete(m.metrics, date)
			delete(m.dailyStats, date)
		}
	}
}

// GetFilterStatistics returns statistics for a specific filter
func (m *PipelineMonitor) GetFilterStatistics(filterName string) FilterStatistics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var stats FilterStatistics
	var totalMetrics int

	for _, dailyMetrics := range m.metrics {
		for _, metrics := range dailyMetrics {
			if filterMetric, exists := metrics.FilterMetrics[filterName]; exists {
				totalMetrics++
				stats.SuccessRate += float64(filterMetric.Passed) / float64(filterMetric.Processed)
				stats.BullishRate += float64(filterMetric.Bullish) / float64(filterMetric.Passed)
				stats.BearishRate += float64(filterMetric.Bearish) / float64(filterMetric.Passed)
				stats.AvgProcessTime += filterMetric.Duration
			}
		}
	}

	if totalMetrics > 0 {
		stats.SuccessRate /= float64(totalMetrics)
		stats.BullishRate /= float64(totalMetrics)
		stats.BearishRate /= float64(totalMetrics)
		stats.AvgProcessTime /= time.Duration(totalMetrics)
		stats.TotalRuns = totalMetrics
	}

	return stats
}
