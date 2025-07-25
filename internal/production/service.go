package production

import (
	"context"
	"fmt"
	"log"
	"time"

	"setbull_trader/internal/config"
	"setbull_trader/internal/deployment"
	"setbull_trader/internal/monitoring"
)

// CandleData represents a trading candle
type CandleData struct {
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
	Time   time.Time `json:"time"`
}

// TradingServiceV1 interface for V1 service
type TradingServiceV1 interface {
	ProcessCandles(ctx context.Context, candles []CandleData, indicators []string) (map[string]interface{}, error)
}

// TradingServiceV2 interface for V2 optimized service
type TradingServiceV2 interface {
	ProcessCandlesOptimized(ctx context.Context, candles []CandleData, indicators []string) (map[string]interface{}, error)
}

// ProductionTradingService wraps the V1 and V2 services with production features
type ProductionTradingService struct {
	// Service instances
	v1Service TradingServiceV1
	v2Service TradingServiceV2
	
	// Production infrastructure
	featureFlags      *config.FeatureFlags
	metrics          *monitoring.MetricsCollector
	deploymentManager *deployment.DeploymentManager
	
	// Request context
	requestIDCounter int64
}

// NewProductionTradingService creates a new production trading service
func NewProductionTradingService(
	v1Service TradingServiceV1,
	v2Service TradingServiceV2,
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
		metrics:          metricsCollector,
		deploymentManager: deploymentMgr,
	}
}

// ProcessCandlesWithIndicators processes candles with production monitoring and fallback
func (p *ProductionTradingService) ProcessCandlesWithIndicators(
	ctx context.Context,
	candles []CandleData,
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
		result, err = p.v2Service.ProcessCandlesOptimized(ctx, candles, indicators)
		
		// Fallback to V1 if V2 fails and fallback is enabled
		if err != nil && p.featureFlags.FallbackToV1OnError {
			log.Printf("PRODUCTION: V2 service failed for request %s: %v. Falling back to V1.", requestID, err)
			p.metrics.RecordFallback(requestID, fmt.Sprintf("V2 error: %v", err))
			
			serviceUsed = "v1"
			result, err = p.v1Service.ProcessCandles(ctx, candles, indicators)
		}
	} else {
		// Use V1 service
		serviceUsed = "v1"
		result, err = p.v1Service.ProcessCandles(ctx, candles, indicators)
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
		"feature_flags":     p.featureFlags,
		"metrics":          metricsSnapshot,
		"deployment":       deploymentStatus,
		"system_info": map[string]interface{}{
			"uptime":           metricsSnapshot.GetUptime(),
			"total_requests":   metricsSnapshot.TotalRequests,
			"current_phase":    deploymentStatus["current_phase"],
			"auto_rollback":    deploymentStatus["auto_rollback"],
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
		"healthy":            p.IsHealthy(),
		"error_rate":         metrics.GetErrorRate(),
		"fallback_rate":      metrics.GetFallbackRate(),
		"cache_hit_rate":     metrics.CacheHitRate,
		"avg_response_time":  metrics.AvgV2ResponseTime,
		"rollback_occurred":  p.deploymentManager.IsRolledBack(),
		"uptime":            metrics.GetUptime(),
		"last_errors":       metrics.LastErrors,
	}
}
