package monitoring

import (
	"context"
	"time"
)

// V2ServiceMetrics represents metrics for V2 services
type V2ServiceMetrics struct {
	// Technical Indicator Service V2 metrics
	TechnicalIndicators TechnicalIndicatorMetrics `json:"technical_indicators"`

	// Candle Aggregation Service V2 metrics
	CandleAggregation CandleAggregationMetrics `json:"candle_aggregation"`

	// Sequence Analyzer V2 metrics
	SequenceAnalyzer SequenceAnalyzerMetrics `json:"sequence_analyzer"`

	// Overall system metrics
	System SystemMetrics `json:"system"`
}

// TechnicalIndicatorMetrics tracks performance of TechnicalIndicatorServiceV2
type TechnicalIndicatorMetrics struct {
	CalculationLatency     time.Duration `json:"calculation_latency"`
	CacheHitRate           float64       `json:"cache_hit_rate"`
	CacheMissRate          float64       `json:"cache_miss_rate"`
	ErrorRate              float64       `json:"error_rate"`
	MemoryUsage            int64         `json:"memory_usage"`
	ConcurrentCalculations int           `json:"concurrent_calculations"`
	TotalCalculations      int64         `json:"total_calculations"`
	GoNumOptimizationUsage float64       `json:"gonum_optimization_usage"`
}

// CandleAggregationMetrics tracks performance of CandleAggregationServiceV2
type CandleAggregationMetrics struct {
	AggregationLatency    time.Duration `json:"aggregation_latency"`
	AnalyticsEngineHits   int64         `json:"analytics_engine_hits"`
	DataFrameProcessTime  time.Duration `json:"dataframe_process_time"`
	WorkerPoolUtilization float64       `json:"worker_pool_utilization"`
	MemoryUsage           int64         `json:"memory_usage"`
	TotalAggregations     int64         `json:"total_aggregations"`
	ErrorRate             float64       `json:"error_rate"`
}

// SequenceAnalyzerMetrics tracks performance of SequenceAnalyzerV2
type SequenceAnalyzerMetrics struct {
	PatternDetectionTime   time.Duration `json:"pattern_detection_time"`
	SequenceQualityScore   float64       `json:"sequence_quality_score"`
	PatternsDetected       int64         `json:"patterns_detected"`
	DataFrameOperationTime time.Duration `json:"dataframe_operation_time"`
	MemoryUsage            int64         `json:"memory_usage"`
	TotalAnalyses          int64         `json:"total_analyses"`
	ErrorRate              float64       `json:"error_rate"`
}

// SystemMetrics tracks overall system performance with V2 services
type SystemMetrics struct {
	OverallLatency   time.Duration `json:"overall_latency"`
	TotalMemoryUsage int64         `json:"total_memory_usage"`
	CPUUsage         float64       `json:"cpu_usage"`
	ErrorRate        float64       `json:"error_rate"`
	ThroughputPerSec float64       `json:"throughput_per_sec"`
	CacheEfficiency  float64       `json:"cache_efficiency"`
}

// V2MetricsCollector defines interface for collecting V2 service metrics
type V2MetricsCollector interface {
	CollectTechnicalIndicatorMetrics(ctx context.Context) (*TechnicalIndicatorMetrics, error)
	CollectCandleAggregationMetrics(ctx context.Context) (*CandleAggregationMetrics, error)
	CollectSequenceAnalyzerMetrics(ctx context.Context) (*SequenceAnalyzerMetrics, error)
	CollectSystemMetrics(ctx context.Context) (*SystemMetrics, error)
	GetOverallMetrics(ctx context.Context) (*V2ServiceMetrics, error)
}

// AlertCondition represents conditions that trigger alerts
type AlertCondition struct {
	MetricName    string    `json:"metric_name"`
	Threshold     float64   `json:"threshold"`
	Operator      string    `json:"operator"` // >, <, >=, <=, ==
	Severity      string    `json:"severity"` // critical, high, medium, low
	Description   string    `json:"description"`
	LastTriggered time.Time `json:"last_triggered"`
}

// AlertManager manages alerting for V2 services
type AlertManager interface {
	RegisterAlertCondition(condition AlertCondition) error
	CheckAlerts(ctx context.Context, metrics *V2ServiceMetrics) ([]Alert, error)
	NotifyAlert(ctx context.Context, alert Alert) error
	GetActiveAlerts(ctx context.Context) ([]Alert, error)
}

// Alert represents a triggered alert
type Alert struct {
	ID          string         `json:"id"`
	Condition   AlertCondition `json:"condition"`
	TriggeredAt time.Time      `json:"triggered_at"`
	Value       float64        `json:"value"`
	Message     string         `json:"message"`
	Resolved    bool           `json:"resolved"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
}

// PreDefinedAlertConditions returns the alert conditions from the integration plan
func PreDefinedAlertConditions() []AlertCondition {
	return []AlertCondition{
		{
			MetricName:  "cache_hit_rate",
			Threshold:   0.70,
			Operator:    "<",
			Severity:    "medium",
			Description: "Cache hit rate below 70%",
		},
		{
			MetricName:  "calculation_latency_seconds",
			Threshold:   5.0,
			Operator:    ">",
			Severity:    "high",
			Description: "Calculation latency exceeds 5 seconds",
		},
		{
			MetricName:  "error_rate",
			Threshold:   0.05,
			Operator:    ">",
			Severity:    "high",
			Description: "Error rate exceeds 5%",
		},
		{
			MetricName:  "memory_usage_percent",
			Threshold:   0.90,
			Operator:    ">",
			Severity:    "critical",
			Description: "Memory usage exceeds 90% of allocated",
		},
		{
			MetricName:  "worker_pool_utilization",
			Threshold:   0.95,
			Operator:    ">",
			Severity:    "high",
			Description: "Analytics engine worker pool utilization exceeds 95%",
		},
		{
			MetricName:  "dataframe_aggregation_time_seconds",
			Threshold:   10.0,
			Operator:    ">",
			Severity:    "medium",
			Description: "DataFrame aggregation time exceeds 10 seconds",
		},
		{
			MetricName:  "sequence_analysis_failure_rate",
			Threshold:   0.03,
			Operator:    ">",
			Severity:    "medium",
			Description: "Sequence analysis failure rate exceeds 3%",
		},
	}
}

// RollbackTriggers returns conditions that should trigger immediate rollback
func RollbackTriggers() []AlertCondition {
	return []AlertCondition{
		{
			MetricName:  "error_rate",
			Threshold:   0.10,
			Operator:    ">",
			Severity:    "critical",
			Description: "Error rate exceeds 10% - immediate rollback required",
		},
		{
			MetricName:  "performance_degradation_percent",
			Threshold:   0.50,
			Operator:    ">",
			Severity:    "critical",
			Description: "Performance degradation exceeds 50% - immediate rollback required",
		},
		{
			MetricName:  "memory_leak_rate_mb_per_hour",
			Threshold:   100.0,
			Operator:    ">",
			Severity:    "critical",
			Description: "Memory leak detected - immediate rollback required",
		},
	}
}
