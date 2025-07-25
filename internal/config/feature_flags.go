package config

import (
	"fmt"
	"os"
	"strconv"
)

// FeatureFlags holds configuration for feature toggles in production
type FeatureFlags struct {
	// UseOptimizedAnalytics enables the new DataFrame + GoNum analytics engine
	UseOptimizedAnalytics bool `json:"use_optimized_analytics"`

	// CacheEnabled enables the FastCache system for indicators
	CacheEnabled bool `json:"cache_enabled"`

	// ConcurrencyEnabled enables worker pool for concurrent processing
	ConcurrencyEnabled bool `json:"concurrency_enabled"`

	// RolloutPercentage controls what percentage of requests use the optimized system (0-100)
	RolloutPercentage float64 `json:"rollout_percentage"`

	// EnableDetailedMetrics enables comprehensive performance monitoring
	EnableDetailedMetrics bool `json:"enable_detailed_metrics"`

	// FallbackToV1OnError automatically falls back to V1 service on errors
	FallbackToV1OnError bool `json:"fallback_to_v1_on_error"`

	// MaxCacheSize sets the maximum cache size in MB
	MaxCacheSize int64 `json:"max_cache_size"`

	// WorkerPoolSize sets the number of concurrent workers
	WorkerPoolSize int `json:"worker_pool_size"`
}

// DefaultFeatureFlags returns safe default values for production
func DefaultFeatureFlags() *FeatureFlags {
	return &FeatureFlags{
		UseOptimizedAnalytics: false, // Start disabled for safety
		CacheEnabled:          true,  // Cache is stable
		ConcurrencyEnabled:    true,  // Concurrency is stable
		RolloutPercentage:     0.0,   // Start with 0% rollout
		EnableDetailedMetrics: true,  // Always monitor in production
		FallbackToV1OnError:   true,  // Always have fallback
		MaxCacheSize:          512,   // 512 MB default cache size
		WorkerPoolSize:        4,     // Conservative worker pool size
	}
}

// LoadFeatureFlagsFromEnv loads feature flags from environment variables
func LoadFeatureFlagsFromEnv() *FeatureFlags {
	flags := DefaultFeatureFlags()

	// Load from environment variables with fallbacks to defaults
	if val := os.Getenv("USE_OPTIMIZED_ANALYTICS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			flags.UseOptimizedAnalytics = enabled
		}
	}

	if val := os.Getenv("CACHE_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			flags.CacheEnabled = enabled
		}
	}

	if val := os.Getenv("CONCURRENCY_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			flags.ConcurrencyEnabled = enabled
		}
	}

	if val := os.Getenv("ROLLOUT_PERCENTAGE"); val != "" {
		if percentage, err := strconv.ParseFloat(val, 64); err == nil {
			if percentage >= 0 && percentage <= 100 {
				flags.RolloutPercentage = percentage
			}
		}
	}

	if val := os.Getenv("ENABLE_DETAILED_METRICS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			flags.EnableDetailedMetrics = enabled
		}
	}

	if val := os.Getenv("FALLBACK_TO_V1_ON_ERROR"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			flags.FallbackToV1OnError = enabled
		}
	}

	if val := os.Getenv("MAX_CACHE_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil && size > 0 {
			flags.MaxCacheSize = size
		}
	}

	if val := os.Getenv("WORKER_POOL_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			flags.WorkerPoolSize = size
		}
	}

	return flags
}

// Validate ensures feature flag values are valid for production
func (f *FeatureFlags) Validate() error {
	if f.RolloutPercentage < 0 || f.RolloutPercentage > 100 {
		return fmt.Errorf("rollout percentage must be between 0 and 100, got %f", f.RolloutPercentage)
	}

	if f.MaxCacheSize <= 0 {
		return fmt.Errorf("max cache size must be positive, got %d", f.MaxCacheSize)
	}

	if f.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker pool size must be positive, got %d", f.WorkerPoolSize)
	}

	// Safety check: don't allow 100% rollout without explicit confirmation
	if f.RolloutPercentage == 100 && !f.FallbackToV1OnError {
		return fmt.Errorf("100%% rollout requires fallback_to_v1_on_error to be enabled")
	}

	return nil
}

// ShouldUseOptimizedAnalytics determines if a request should use the optimized analytics
func (f *FeatureFlags) ShouldUseOptimizedAnalytics(requestID string) bool {
	if !f.UseOptimizedAnalytics {
		return false
	}

	if f.RolloutPercentage == 0 {
		return false
	}

	if f.RolloutPercentage == 100 {
		return true
	}

	// Use hash of request ID to determine if this request should use optimized analytics
	hash := simpleHash(requestID)
	return (hash % 100) < int(f.RolloutPercentage)
}

// GetCacheSizeBytes returns cache size in bytes
func (f *FeatureFlags) GetCacheSizeBytes() int64 {
	return f.MaxCacheSize * 1024 * 1024 // Convert MB to bytes
}

// LogConfiguration logs the current feature flag configuration
func (f *FeatureFlags) LogConfiguration() string {
	return fmt.Sprintf(
		"FeatureFlags{OptimizedAnalytics=%t, Cache=%t, Concurrency=%t, Rollout=%.1f%%, DetailedMetrics=%t, FallbackV1=%t, CacheSize=%dMB, Workers=%d}",
		f.UseOptimizedAnalytics,
		f.CacheEnabled,
		f.ConcurrencyEnabled,
		f.RolloutPercentage,
		f.EnableDetailedMetrics,
		f.FallbackToV1OnError,
		f.MaxCacheSize,
		f.WorkerPoolSize,
	)
}

// Production deployment phases
const (
	PhaseDisabled    = 0.0   // 0% - All traffic uses V1
	PhaseTesting     = 10.0  // 10% - Initial production testing
	PhaseValidation  = 50.0  // 50% - Broader validation
	PhaseFullRollout = 100.0 // 100% - Full deployment
)

// GetDeploymentPhase returns the current deployment phase
func (f *FeatureFlags) GetDeploymentPhase() string {
	switch f.RolloutPercentage {
	case PhaseDisabled:
		return "DISABLED"
	case PhaseTesting:
		return "TESTING"
	case PhaseValidation:
		return "VALIDATION"
	case PhaseFullRollout:
		return "FULL_ROLLOUT"
	default:
		return fmt.Sprintf("CUSTOM_%.1f%%", f.RolloutPercentage)
	}
}

// SetDeploymentPhase updates the rollout percentage to a predefined phase
func (f *FeatureFlags) SetDeploymentPhase(phase float64) error {
	switch phase {
	case PhaseDisabled, PhaseTesting, PhaseValidation, PhaseFullRollout:
		f.RolloutPercentage = phase
		return nil
	default:
		return fmt.Errorf("invalid deployment phase: %f. Use PhaseDisabled, PhaseTesting, PhaseValidation, or PhaseFullRollout", phase)
	}
}

// Simple hash function for consistent request routing
func simpleHash(s string) int {
	hash := 0
	for _, char := range s {
		hash = (hash * 31) + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}
