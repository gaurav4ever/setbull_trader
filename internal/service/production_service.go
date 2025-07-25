package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"setbull_trader/internal/config"
	"setbull_trader/internal/deployment"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/monitoring"
)

// ProductionTradingService wraps the V1 and V2 services with production features
type ProductionTradingService struct {
	// Service instances
	v1Service *TechnicalIndicatorService
	v2Service *TechnicalIndicatorServiceV2

	// Production infrastructure
	featureFlags      *config.FeatureFlags
	metrics           *monitoring.MetricsCollector
	deploymentManager *deployment.DeploymentManager

	// Request context
	requestIDCounter int64
}

// NewProductionTradingService creates a new production trading service
func NewProductionTradingService(
	v1Service *TechnicalIndicatorService,
	v2Service *TechnicalIndicatorServiceV2,
) *ProductionTradingService {

	// Initialize feature flags from environment
	featureFlags := config.LoadFeatureFlagsFromEnv()
	if err := featureFlags.Validate(); err != nil {
		log.Printf("PRODUCTION: Feature flag validation failed: %v. Using defaults.", err)
		featureFlags = config.DefaultFeatureFlags()
	}

	// Initialize metrics collector
	metricsCollector := monitoring.NewMetricsCollector()

	// Set up alert callbacks
	metricsCollector.AddAlertCallback(func(alertType string, message string, metrics *monitoring.MetricsCollector) {
		log.Printf("ALERT [%s]: %s", alertType, message)
		// In a real implementation, this would send alerts to Slack, PagerDuty, etc.
	})

	// Initialize deployment manager
	deploymentMgr := deployment.NewDeploymentManager(featureFlags, metricsCollector)

	log.Printf("PRODUCTION: Initialized with feature flags: %s", featureFlags.LogConfiguration())

	return &ProductionTradingService{
		v1Service:         v1Service,
		v2Service:         v2Service,
		featureFlags:      featureFlags,
		metrics:           metricsCollector,
		deploymentManager: deploymentMgr,
	}
}

// ProcessCandlesWithIndicators processes candles with production monitoring and fallback
func (p *ProductionTradingService) ProcessCandlesWithIndicators(
	ctx context.Context,
	candles []domain.CandleData,
	indicators []string,
) (map[string]interface{}, error) {

	// Generate request ID for tracking
	requestID := p.generateRequestID()

	// Determine which service to use
	useV2 := p.featureFlags.ShouldUseOptimizedAnalytics(requestID)

	startTime := time.Now()
	var result map[string]interface{}
	var err error
	var serviceUsed string

	if useV2 && p.featureFlags.UseOptimizedAnalytics {
		// Try V2 service first
		serviceUsed = "v2"
		result, err = p.processWithV2Service(ctx, candles, indicators, requestID)

		// Fallback to V1 if V2 fails and fallback is enabled
		if err != nil && p.featureFlags.FallbackToV1OnError {
			log.Printf("PRODUCTION: V2 service failed for request %s: %v. Falling back to V1.", requestID, err)
			p.metrics.RecordFallback(requestID, fmt.Sprintf("V2 error: %v", err))

			serviceUsed = "v1"
			result, err = p.processWithV1Service(ctx, candles, indicators, requestID)
		}
	} else {
		// Use V1 service
		serviceUsed = "v1"
		result, err = p.processWithV1Service(ctx, candles, indicators, requestID)
	}

	// Record metrics
	duration := time.Since(startTime)
	p.metrics.RecordRequest(serviceUsed, duration, err, requestID)

	if err != nil {
		log.Printf("PRODUCTION: Request %s failed with service %s after %v: %v", requestID, serviceUsed, duration, err)
	} else {
		log.Printf("PRODUCTION: Request %s completed with service %s in %v", requestID, serviceUsed, duration)
	}

	return result, err
}

// processWithV1Service processes using the V1 service
func (p *ProductionTradingService) processWithV1Service(
	ctx context.Context,
	candles []domain.CandleData,
	indicators []string,
	requestID string,
) (map[string]interface{}, error) {

	// For this example, we'll assume the V1 service has a method like this
	// In practice, you'd adapt this to your actual V1 service interface
	log.Printf("PRODUCTION: Processing request %s with V1 service", requestID)

	// Simulate V1 processing
	// This would be replaced with actual V1 service calls
	result := map[string]interface{}{
		"service_version": "v1",
		"request_id":      requestID,
		"candle_count":    len(candles),
		"indicators":      indicators,
		"processed_at":    time.Now(),
	}

	// Record memory usage (this would be actual measurement in practice)
	p.metrics.RecordMemoryUsage("v1", 64) // 64 MB simulated

	return result, nil
}

// processWithV2Service processes using the V2 optimized service
func (p *ProductionTradingService) processWithV2Service(
	ctx context.Context,
	candles []domain.CandleData,
	indicators []string,
	requestID string,
) (map[string]interface{}, error) {

	log.Printf("PRODUCTION: Processing request %s with V2 optimized service", requestID)

	// Convert candles to the format expected by V2 service
	// This would depend on your actual V2 service interface

	// For demonstration, let's assume we have a method on V2 service
	// In practice, you'd implement the actual calls to your V2 service

	result := map[string]interface{}{
		"service_version": "v2",
		"request_id":      requestID,
		"candle_count":    len(candles),
		"indicators":      indicators,
		"processed_at":    time.Now(),
		"optimizations": map[string]bool{
			"dataframe_enabled":   true,
			"cache_enabled":       p.featureFlags.CacheEnabled,
			"concurrency_enabled": p.featureFlags.ConcurrencyEnabled,
		},
	}

	// Record cache events (simulated)
	p.metrics.RecordCacheEvent(true) // Cache hit

	// Record memory usage (this would be actual measurement in practice)
	p.metrics.RecordMemoryUsage("v2", 32) // 32 MB simulated (optimized)

	return result, nil
}

// generateRequestID generates a unique request ID for tracking
func (p *ProductionTradingService) generateRequestID() string {
	p.requestIDCounter++
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), p.requestIDCounter)
}

// StartProduction starts the production system with monitoring and deployment management
func (p *ProductionTradingService) StartProduction(ctx context.Context) error {
	log.Printf("PRODUCTION: Starting production system")

	// Start periodic metrics reporting
	go p.metrics.StartPeriodicReporting(ctx, 30*time.Second)

	// Start deployment monitoring (if deployment is in progress)
	if p.featureFlags.UseOptimizedAnalytics && p.featureFlags.RolloutPercentage > 0 {
		go p.deploymentManager.MonitorAndProgress(ctx)
	}

	log.Printf("PRODUCTION: System started successfully")
	return nil
}

// InitiateDeployment starts the gradual rollout to V2
func (p *ProductionTradingService) InitiateDeployment(ctx context.Context) error {
	log.Printf("PRODUCTION: Initiating deployment to V2 service")

	if err := p.deploymentManager.StartDeployment(ctx); err != nil {
		return fmt.Errorf("failed to start deployment: %w", err)
	}

	// Start monitoring the deployment
	go p.deploymentManager.MonitorAndProgress(ctx)

	return nil
}

// RollbackDeployment performs a manual rollback
func (p *ProductionTradingService) RollbackDeployment(reason string) error {
	log.Printf("PRODUCTION: Manual rollback requested. Reason: %s", reason)
	return p.deploymentManager.Rollback(reason)
}

// GetSystemStatus returns comprehensive system status
func (p *ProductionTradingService) GetSystemStatus() map[string]interface{} {
	metricsSnapshot := p.metrics.GetSnapshot()
	deploymentStatus := p.deploymentManager.GetDeploymentStatus()

	return map[string]interface{}{
		"feature_flags": p.featureFlags,
		"metrics":       metricsSnapshot,
		"deployment":    deploymentStatus,
		"system_info": map[string]interface{}{
			"uptime":         metricsSnapshot.GetUptime(),
			"total_requests": metricsSnapshot.TotalRequests,
			"current_phase":  deploymentStatus["current_phase"],
			"auto_rollback":  deploymentStatus["auto_rollback"],
		},
	}
}

// UpdateFeatureFlags updates feature flags at runtime
func (p *ProductionTradingService) UpdateFeatureFlags(flags *config.FeatureFlags) error {
	if err := flags.Validate(); err != nil {
		return fmt.Errorf("invalid feature flags: %w", err)
	}

	log.Printf("PRODUCTION: Updating feature flags from %s to %s",
		p.featureFlags.LogConfiguration(),
		flags.LogConfiguration())

	p.featureFlags = flags
	return nil
}

// GetMetrics returns current metrics
func (p *ProductionTradingService) GetMetrics() *monitoring.MetricsCollector {
	return p.metrics.GetSnapshot()
}

// EnableAutoRollback enables or disables automatic rollback
func (p *ProductionTradingService) EnableAutoRollback(enabled bool) {
	p.deploymentManager.SetAutoRollback(enabled)
}

// IsHealthy returns true if the system is healthy
func (p *ProductionTradingService) IsHealthy() bool {
	metrics := p.metrics.GetSnapshot()

	// Basic health checks
	errorRate := metrics.GetErrorRate()
	if errorRate > 15.0 { // 15% error rate threshold
		return false
	}

	// Check if we're in a failed rollback state
	if p.deploymentManager.IsRolledBack() {
		return false
	}

	return true
}

// GetHealthStatus returns detailed health information
func (p *ProductionTradingService) GetHealthStatus() map[string]interface{} {
	metrics := p.metrics.GetSnapshot()

	return map[string]interface{}{
		"healthy":           p.IsHealthy(),
		"error_rate":        metrics.GetErrorRate(),
		"fallback_rate":     metrics.GetFallbackRate(),
		"cache_hit_rate":    metrics.CacheHitRate,
		"avg_response_time": metrics.AvgV2ResponseTime,
		"rollback_occurred": p.deploymentManager.IsRolledBack(),
		"uptime":            metrics.GetUptime(),
		"last_errors":       metrics.LastErrors,
	}
}
