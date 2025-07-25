package deployment

import (
	"context"
	"fmt"
	"log"
	"time"

	"setbull_trader/internal/config"
	"setbull_trader/internal/monitoring"
)

// DeploymentManager manages the gradual rollout of the optimized analytics system
type DeploymentManager struct {
	featureFlags *config.FeatureFlags
	metrics      *monitoring.MetricsCollector
	rolloutPlan  *RolloutPlan

	// Rollback configuration
	rollbackThresholds *RollbackThresholds
	autoRollback       bool

	// State tracking
	currentPhase     string
	lastPhaseChange  time.Time
	rollbackOccurred bool
}

// RolloutPlan defines the phases and timing for deployment
type RolloutPlan struct {
	Phases []RolloutPhase `json:"phases"`
}

// RolloutPhase represents a single phase in the rollout plan
type RolloutPhase struct {
	Name                 string        `json:"name"`
	Percentage           float64       `json:"percentage"`
	MinDuration          time.Duration `json:"min_duration"`            // Minimum time in this phase
	RequiredSuccessRate  float64       `json:"required_success_rate"`   // Required success rate to proceed
	RequiredCacheHitRate float64       `json:"required_cache_hit_rate"` // Required cache performance
	MaxErrorRate         float64       `json:"max_error_rate"`          // Maximum allowable error rate
}

// RollbackThresholds defines when automatic rollback should occur
type RollbackThresholds struct {
	ErrorRatePercent       float64       `json:"error_rate_percent"`
	FallbackRatePercent    float64       `json:"fallback_rate_percent"`
	ResponseTimeDelta      time.Duration `json:"response_time_delta"`       // V2 should not be slower than V1 by this much
	MinRequestsForDecision int64         `json:"min_requests_for_decision"` // Minimum requests before making rollback decision
}

// NewDeploymentManager creates a new deployment manager
func NewDeploymentManager(featureFlags *config.FeatureFlags, metrics *monitoring.MetricsCollector) *DeploymentManager {
	return &DeploymentManager{
		featureFlags: featureFlags,
		metrics:      metrics,
		rolloutPlan:  createDefaultRolloutPlan(),
		rollbackThresholds: &RollbackThresholds{
			ErrorRatePercent:       10.0,            // Rollback if error rate > 10%
			FallbackRatePercent:    25.0,            // Rollback if fallback rate > 25%
			ResponseTimeDelta:      2 * time.Second, // Rollback if V2 is 2s+ slower than V1
			MinRequestsForDecision: 100,             // Need at least 100 requests to make decision
		},
		autoRollback:    true,
		currentPhase:    "DISABLED",
		lastPhaseChange: time.Now(),
	}
}

// createDefaultRolloutPlan creates a conservative rollout plan
func createDefaultRolloutPlan() *RolloutPlan {
	return &RolloutPlan{
		Phases: []RolloutPhase{
			{
				Name:                 "CANARY",
				Percentage:           1.0,
				MinDuration:          10 * time.Minute,
				RequiredSuccessRate:  95.0,
				RequiredCacheHitRate: 70.0,
				MaxErrorRate:         2.0,
			},
			{
				Name:                 "TESTING",
				Percentage:           10.0,
				MinDuration:          30 * time.Minute,
				RequiredSuccessRate:  95.0,
				RequiredCacheHitRate: 75.0,
				MaxErrorRate:         3.0,
			},
			{
				Name:                 "VALIDATION",
				Percentage:           50.0,
				MinDuration:          1 * time.Hour,
				RequiredSuccessRate:  97.0,
				RequiredCacheHitRate: 80.0,
				MaxErrorRate:         2.0,
			},
			{
				Name:                 "FULL_ROLLOUT",
				Percentage:           100.0,
				MinDuration:          0, // No minimum for full rollout
				RequiredSuccessRate:  98.0,
				RequiredCacheHitRate: 85.0,
				MaxErrorRate:         1.0,
			},
		},
	}
}

// StartDeployment begins the deployment process
func (dm *DeploymentManager) StartDeployment(ctx context.Context) error {
	log.Printf("DEPLOYMENT: Starting deployment with plan: %+v", dm.rolloutPlan)

	// Start with the first phase
	if len(dm.rolloutPlan.Phases) == 0 {
		return fmt.Errorf("no rollout phases defined")
	}

	// Enable the optimized analytics feature
	dm.featureFlags.UseOptimizedAnalytics = true

	// Start with the first phase
	firstPhase := dm.rolloutPlan.Phases[0]
	return dm.moveToPhase(firstPhase)
}

// MonitorAndProgress monitors the current phase and progresses to the next phase when ready
func (dm *DeploymentManager) MonitorAndProgress(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if dm.shouldRollback() {
				log.Printf("DEPLOYMENT: Automatic rollback triggered")
				dm.Rollback("Automatic rollback due to poor performance metrics")
				return
			}

			if dm.canProgressToNextPhase() {
				dm.progressToNextPhase()
			}
		}
	}
}

// moveToPhase transitions to a specific rollout phase
func (dm *DeploymentManager) moveToPhase(phase RolloutPhase) error {
	log.Printf("DEPLOYMENT: Moving to phase %s (%.1f%% rollout)", phase.Name, phase.Percentage)

	dm.featureFlags.RolloutPercentage = phase.Percentage
	dm.currentPhase = phase.Name
	dm.lastPhaseChange = time.Now()

	// Validate the configuration
	if err := dm.featureFlags.Validate(); err != nil {
		return fmt.Errorf("invalid feature flag configuration: %w", err)
	}

	log.Printf("DEPLOYMENT: Successfully moved to phase %s. Feature flags: %s", phase.Name, dm.featureFlags.LogConfiguration())
	return nil
}

// shouldRollback checks if we should perform an automatic rollback
func (dm *DeploymentManager) shouldRollback() bool {
	if !dm.autoRollback || dm.rollbackOccurred {
		return false
	}

	metrics := dm.metrics.GetSnapshot()

	// Need minimum requests to make a decision
	if metrics.TotalRequests < dm.rollbackThresholds.MinRequestsForDecision {
		return false
	}

	// Check error rate
	errorRate := metrics.GetErrorRate()
	if errorRate > dm.rollbackThresholds.ErrorRatePercent {
		log.Printf("DEPLOYMENT: Error rate %.2f%% exceeds threshold %.2f%%", errorRate, dm.rollbackThresholds.ErrorRatePercent)
		return true
	}

	// Check fallback rate
	fallbackRate := metrics.GetFallbackRate()
	if fallbackRate > dm.rollbackThresholds.FallbackRatePercent {
		log.Printf("DEPLOYMENT: Fallback rate %.2f%% exceeds threshold %.2f%%", fallbackRate, dm.rollbackThresholds.FallbackRatePercent)
		return true
	}

	// Check response time delta (V2 should not be significantly slower than V1)
	if metrics.V2Requests > 10 && metrics.V1Requests > 10 {
		timeDelta := metrics.AvgV2ResponseTime - metrics.AvgV1ResponseTime
		if timeDelta > dm.rollbackThresholds.ResponseTimeDelta {
			log.Printf("DEPLOYMENT: V2 response time delta %v exceeds threshold %v", timeDelta, dm.rollbackThresholds.ResponseTimeDelta)
			return true
		}
	}

	return false
}

// canProgressToNextPhase checks if we can move to the next phase
func (dm *DeploymentManager) canProgressToNextPhase() bool {
	currentPhaseIndex := dm.getCurrentPhaseIndex()
	if currentPhaseIndex == -1 || currentPhaseIndex >= len(dm.rolloutPlan.Phases)-1 {
		return false // Already at the last phase or phase not found
	}

	currentPhase := dm.rolloutPlan.Phases[currentPhaseIndex]

	// Check minimum duration
	if time.Since(dm.lastPhaseChange) < currentPhase.MinDuration {
		return false
	}

	metrics := dm.metrics.GetSnapshot()

	// Need minimum requests to make a decision
	if metrics.V2Requests < 10 {
		return false
	}

	// Check success rate
	successRate := 100.0 - metrics.GetErrorRate()
	if successRate < currentPhase.RequiredSuccessRate {
		log.Printf("DEPLOYMENT: Success rate %.2f%% below required %.2f%% for phase %s", successRate, currentPhase.RequiredSuccessRate, currentPhase.Name)
		return false
	}

	// Check error rate
	errorRate := metrics.GetErrorRate()
	if errorRate > currentPhase.MaxErrorRate {
		log.Printf("DEPLOYMENT: Error rate %.2f%% exceeds max %.2f%% for phase %s", errorRate, currentPhase.MaxErrorRate, currentPhase.Name)
		return false
	}

	// Check cache hit rate
	if metrics.CacheHitRate < currentPhase.RequiredCacheHitRate && metrics.CacheHits+metrics.CacheMisses > 10 {
		log.Printf("DEPLOYMENT: Cache hit rate %.2f%% below required %.2f%% for phase %s", metrics.CacheHitRate, currentPhase.RequiredCacheHitRate, currentPhase.Name)
		return false
	}

	return true
}

// progressToNextPhase moves to the next phase in the rollout plan
func (dm *DeploymentManager) progressToNextPhase() {
	currentPhaseIndex := dm.getCurrentPhaseIndex()
	if currentPhaseIndex == -1 || currentPhaseIndex >= len(dm.rolloutPlan.Phases)-1 {
		return
	}

	nextPhase := dm.rolloutPlan.Phases[currentPhaseIndex+1]
	if err := dm.moveToPhase(nextPhase); err != nil {
		log.Printf("DEPLOYMENT: Failed to move to phase %s: %v", nextPhase.Name, err)
	}
}

// getCurrentPhaseIndex returns the index of the current phase in the rollout plan
func (dm *DeploymentManager) getCurrentPhaseIndex() int {
	for i, phase := range dm.rolloutPlan.Phases {
		if phase.Name == dm.currentPhase {
			return i
		}
	}
	return -1
}

// Rollback performs a rollback to the previous stable state
func (dm *DeploymentManager) Rollback(reason string) error {
	log.Printf("DEPLOYMENT: Performing rollback. Reason: %s", reason)

	// Disable optimized analytics
	dm.featureFlags.UseOptimizedAnalytics = false
	dm.featureFlags.RolloutPercentage = 0.0

	dm.currentPhase = "ROLLBACK"
	dm.rollbackOccurred = true

	log.Printf("DEPLOYMENT: Rollback completed. All traffic routing to V1 service")

	// Log final metrics before rollback
	metrics := dm.metrics.GetSnapshot()
	log.Printf("DEPLOYMENT: Final metrics before rollback - Total: %d, V2: %d, Errors: %d (%.2f%%), Fallbacks: %d (%.2f%%)",
		metrics.TotalRequests,
		metrics.V2Requests,
		metrics.ErrorRequests,
		metrics.GetErrorRate(),
		metrics.FallbackRequests,
		metrics.GetFallbackRate(),
	)

	return nil
}

// GetDeploymentStatus returns the current deployment status
func (dm *DeploymentManager) GetDeploymentStatus() map[string]interface{} {
	metrics := dm.metrics.GetSnapshot()

	status := map[string]interface{}{
		"current_phase":      dm.currentPhase,
		"rollout_percentage": dm.featureFlags.RolloutPercentage,
		"phase_duration":     time.Since(dm.lastPhaseChange),
		"rollback_occurred":  dm.rollbackOccurred,
		"auto_rollback":      dm.autoRollback,
		"total_requests":     metrics.TotalRequests,
		"v2_requests":        metrics.V2Requests,
		"error_rate":         metrics.GetErrorRate(),
		"fallback_rate":      metrics.GetFallbackRate(),
		"cache_hit_rate":     metrics.CacheHitRate,
		"avg_v1_response":    metrics.AvgV1ResponseTime,
		"avg_v2_response":    metrics.AvgV2ResponseTime,
		"can_progress":       dm.canProgressToNextPhase(),
		"should_rollback":    dm.shouldRollback(),
	}

	return status
}

// SetAutoRollback enables or disables automatic rollback
func (dm *DeploymentManager) SetAutoRollback(enabled bool) {
	dm.autoRollback = enabled
	log.Printf("DEPLOYMENT: Auto rollback %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// UpdateRollbackThresholds updates the rollback thresholds
func (dm *DeploymentManager) UpdateRollbackThresholds(thresholds *RollbackThresholds) {
	dm.rollbackThresholds = thresholds
	log.Printf("DEPLOYMENT: Updated rollback thresholds: %+v", thresholds)
}

// GetNextPhase returns information about the next phase
func (dm *DeploymentManager) GetNextPhase() *RolloutPhase {
	currentPhaseIndex := dm.getCurrentPhaseIndex()
	if currentPhaseIndex == -1 || currentPhaseIndex >= len(dm.rolloutPlan.Phases)-1 {
		return nil
	}

	return &dm.rolloutPlan.Phases[currentPhaseIndex+1]
}

// IsFullyDeployed returns true if the deployment is complete
func (dm *DeploymentManager) IsFullyDeployed() bool {
	return dm.currentPhase == "FULL_ROLLOUT" && dm.featureFlags.RolloutPercentage == 100.0
}

// IsRolledBack returns true if a rollback has occurred
func (dm *DeploymentManager) IsRolledBack() bool {
	return dm.rollbackOccurred
}
