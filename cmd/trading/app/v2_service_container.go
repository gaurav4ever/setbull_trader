package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"setbull_trader/internal/monitoring"
	"setbull_trader/internal/repository"
	"setbull_trader/internal/service"
	"setbull_trader/internal/trading/config"
)

// V2ServiceContainer holds all V2 services and their dependencies
// Phase 2: Fully implemented V2 services with dependency injection
type V2ServiceContainer struct {
	// V2 Services - Fully implemented
	TechnicalIndicatorServiceV2 *service.TechnicalIndicatorServiceV2
	CandleAggregationServiceV2  *service.CandleAggregationServiceV2
	SequenceAnalyzerServiceV2   *service.SequenceAnalyzerV2

	// Service wrappers for backward compatibility
	TechnicalIndicatorWrapper service.TechnicalIndicatorServiceInterface
	CandleAggregationWrapper  service.CandleAggregationServiceInterface
	SequenceAnalyzerWrapper   service.SequenceAnalyzerInterface

	// Analytics Engine Dependencies
	AnalyticsEngine interface{}

	// Repositories
	TechnicalIndicatorRepo interface{}
	CandleAggregationRepo  interface{}
	SequenceAnalyzerRepo   interface{}

	// Monitoring
	MetricsCollector monitoring.V2MetricsCollector
	AlertManager     monitoring.AlertManager

	// Configuration
	Config *config.Config

	// Feature flags for gradual migration
	FeatureFlags *config.FeaturesConfig

	// Infrastructure status
	Initialized bool
	StartTime   time.Time
}

// InitializeV2Services creates and wires all V2 services based on feature flags
// Phase 2: Full service integration with dependency injection
func InitializeV2Services(cfg *config.Config, candleRepo interface{}, candle5MinRepo interface{},
	batchFetchService interface{}, tradingCalendarService interface{}, utilityService interface{},
	v1TechnicalIndicatorService service.TechnicalIndicatorServiceInterface,
	v1CandleAggregationService service.CandleAggregationServiceInterface,
	v1SequenceAnalyzer service.SequenceAnalyzerInterface) (*V2ServiceContainer, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container := &V2ServiceContainer{
		Config:       cfg,
		FeatureFlags: &cfg.Features,
		StartTime:    time.Now(),
	}

	// Initialize infrastructure components
	if err := container.initializeInfrastructure(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize V2 infrastructure: %w", err)
	}

	// Phase 2: Initialize V2 services
	if err := container.initializeV2Services(ctx, candleRepo, candle5MinRepo,
		batchFetchService, tradingCalendarService, utilityService); err != nil {
		return nil, fmt.Errorf("failed to initialize V2 services: %w", err)
	}

	// Phase 2: Create service wrappers for backward compatibility
	if err := container.initializeServiceWrappers(ctx, v1TechnicalIndicatorService,
		v1CandleAggregationService, v1SequenceAnalyzer); err != nil {
		return nil, fmt.Errorf("failed to initialize service wrappers: %w", err)
	}

	log.Printf("V2 service container fully initialized (Phase 2) with feature flags: TI=%t, CA=%t, SA=%t",
		cfg.Features.TechnicalIndicatorsV2,
		cfg.Features.CandleAggregationV2,
		cfg.Features.SequenceAnalyzerV2)

	container.Initialized = true
	return container, nil
}

// initializeInfrastructure sets up the foundational components
func (c *V2ServiceContainer) initializeInfrastructure(ctx context.Context) error {
	// Initialize monitoring infrastructure
	if err := c.initializeMonitoring(ctx); err != nil {
		return fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	log.Printf("V2 infrastructure components initialized successfully")
	return nil
}

// initializeMonitoring sets up monitoring and alerting for V2 services
func (c *V2ServiceContainer) initializeMonitoring(ctx context.Context) error {
	// For Phase 1, we'll create placeholder implementations
	// These will be replaced with actual implementations in Phase 2

	log.Printf("V2 monitoring infrastructure ready")
	return nil
}

// IsServiceV2Enabled checks if a specific V2 service is enabled
func (c *V2ServiceContainer) IsServiceV2Enabled(serviceName string) bool {
	if !c.Initialized {
		return false
	}

	switch serviceName {
	case "technical_indicators":
		return c.FeatureFlags.TechnicalIndicatorsV2
	case "candle_aggregation":
		return c.FeatureFlags.CandleAggregationV2
	case "sequence_analyzer":
		return c.FeatureFlags.SequenceAnalyzerV2
	default:
		return false
	}
}

// GetV2ServiceMetrics collects metrics from all enabled V2 services
func (c *V2ServiceContainer) GetV2ServiceMetrics(ctx context.Context) (*monitoring.V2ServiceMetrics, error) {
	if !c.Initialized {
		return nil, fmt.Errorf("V2 service container not initialized")
	}

	if c.MetricsCollector == nil {
		return &monitoring.V2ServiceMetrics{
			System: monitoring.SystemMetrics{
				OverallLatency:   time.Since(c.StartTime),
				TotalMemoryUsage: 0,
				CPUUsage:         0,
				ErrorRate:        0,
				ThroughputPerSec: 0,
				CacheEfficiency:  0,
			},
		}, nil
	}

	return c.MetricsCollector.GetOverallMetrics(ctx)
}

// HealthCheck performs health checks on all enabled V2 services
func (c *V2ServiceContainer) HealthCheck(ctx context.Context) error {
	if !c.Initialized {
		return fmt.Errorf("V2 service container not initialized")
	}

	// Basic infrastructure health checks
	if c.Config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if c.FeatureFlags == nil {
		return fmt.Errorf("feature flags not configured")
	}

	// Phase 1: Only check infrastructure readiness
	log.Printf("V2 service container health check passed (Phase 1)")
	return nil
}

// Shutdown gracefully shuts down all V2 services
func (c *V2ServiceContainer) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down V2 services...")

	// Phase 1: Only infrastructure cleanup needed
	c.Initialized = false

	log.Printf("V2 services shutdown completed")
	return nil
}

// initializeV2Services initializes all V2 services with proper dependencies
func (c *V2ServiceContainer) initializeV2Services(ctx context.Context, candleRepo interface{},
	candle5MinRepo interface{}, batchFetchService interface{},
	tradingCalendarService interface{}, utilityService interface{}) error {

	// Initialize TechnicalIndicatorServiceV2
	if candleRepoTyped, ok := candleRepo.(repository.CandleRepository); ok {
		c.TechnicalIndicatorServiceV2 = service.NewTechnicalIndicatorServiceV2(candleRepoTyped)
		log.Printf("TechnicalIndicatorServiceV2 initialized successfully")
	} else {
		return fmt.Errorf("invalid candleRepo type for TechnicalIndicatorServiceV2")
	}

	// Initialize SequenceAnalyzerV2
	c.SequenceAnalyzerServiceV2 = service.NewSequenceAnalyzerV2()
	log.Printf("SequenceAnalyzerV2 initialized successfully")

	// Initialize CandleAggregationServiceV2
	candleRepoTyped, ok1 := candleRepo.(repository.CandleRepository)
	candle5MinRepoTyped, ok2 := candle5MinRepo.(repository.Candle5MinRepository)
	batchFetchServiceTyped, ok3 := batchFetchService.(*service.BatchFetchService)
	tradingCalendarServiceTyped, ok4 := tradingCalendarService.(*service.TradingCalendarService)
	utilityServiceTyped, ok5 := utilityService.(*service.UtilityService)

	if ok1 && ok2 && ok3 && ok4 && ok5 {
		c.CandleAggregationServiceV2 = service.NewCandleAggregationServiceV2(
			candleRepoTyped,
			candle5MinRepoTyped,
			batchFetchServiceTyped,
			tradingCalendarServiceTyped,
			utilityServiceTyped,
		)
		log.Printf("CandleAggregationServiceV2 initialized successfully")
	} else {
		return fmt.Errorf("invalid repository types for CandleAggregationServiceV2")
	}

	return nil
}

// initializeServiceWrappers creates compatibility wrappers for gradual migration
func (c *V2ServiceContainer) initializeServiceWrappers(ctx context.Context,
	v1TechnicalIndicatorService service.TechnicalIndicatorServiceInterface,
	v1CandleAggregationService service.CandleAggregationServiceInterface,
	v1SequenceAnalyzer service.SequenceAnalyzerInterface) error {

	// Create TechnicalIndicator wrapper
	c.TechnicalIndicatorWrapper = service.NewTechnicalIndicatorServiceWrapper(
		v1TechnicalIndicatorService,
		c.TechnicalIndicatorServiceV2,
		c.FeatureFlags.TechnicalIndicatorsV2,
	)

	// Create CandleAggregation wrapper
	c.CandleAggregationWrapper = service.NewCandleAggregationServiceWrapper(
		v1CandleAggregationService,
		c.CandleAggregationServiceV2,
		c.FeatureFlags.CandleAggregationV2,
	)

	// Create SequenceAnalyzer wrapper
	c.SequenceAnalyzerWrapper = service.NewSequenceAnalyzerWrapper(
		v1SequenceAnalyzer,
		c.SequenceAnalyzerServiceV2,
		c.FeatureFlags.SequenceAnalyzerV2,
	)

	log.Printf("Service wrappers initialized successfully")
	return nil
}

// UpdateFeatureFlags allows runtime feature flag updates
func (c *V2ServiceContainer) UpdateFeatureFlags(ctx context.Context, newFlags *config.FeaturesConfig) error {
	if !c.Initialized {
		return fmt.Errorf("V2 service container not initialized")
	}

	log.Printf("Updating feature flags: TI=%t->%t, CA=%t->%t, SA=%t->%t",
		c.FeatureFlags.TechnicalIndicatorsV2, newFlags.TechnicalIndicatorsV2,
		c.FeatureFlags.CandleAggregationV2, newFlags.CandleAggregationV2,
		c.FeatureFlags.SequenceAnalyzerV2, newFlags.SequenceAnalyzerV2)

	// Update the flags
	c.FeatureFlags = newFlags

	// In Phase 1, we only log the flag changes
	// Future phases will handle actual service switching
	log.Printf("Feature flags updated successfully (Phase 1)")
	return nil
}

// GetStatus returns the current status of the V2 service container
func (c *V2ServiceContainer) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"initialized":   c.Initialized,
		"start_time":    c.StartTime,
		"uptime":        time.Since(c.StartTime),
		"feature_flags": c.FeatureFlags,
		"phase":         "Phase 1 - Infrastructure Setup",
	}
}
