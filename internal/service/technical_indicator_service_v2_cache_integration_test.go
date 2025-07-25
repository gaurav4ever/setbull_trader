package service

import (
	"testing"

	"setbull_trader/internal/repository"

	"github.com/stretchr/testify/assert"
)

// Simple test to verify the cache-enabled service can be instantiated
func TestTechnicalIndicatorServiceV2_CacheInitialization(t *testing.T) {
	// Create service with nil repository for initialization test only
	service := NewTechnicalIndicatorServiceV2(nil)

	// Verify service is created with cache components
	assert.NotNil(t, service)

	// Verify cache metrics are accessible
	metrics := service.GetServiceMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "service_type")
	assert.Contains(t, metrics, "cache_hits")
	assert.Contains(t, metrics, "cache_misses")
	assert.Contains(t, metrics, "cache_hit_rate")
	assert.Contains(t, metrics, "pool_stats")

	// Initial state should have zero cache activity
	assert.Equal(t, int64(0), metrics["cache_hits"])
	assert.Equal(t, int64(0), metrics["cache_misses"])
	assert.Equal(t, float64(0), metrics["cache_hit_rate"])

	// Verify cache clearing works
	service.ClearCache()

	// Metrics should still be accessible after clearing
	metricsAfterClear := service.GetServiceMetrics()
	assert.Equal(t, int64(0), metricsAfterClear["cache_hits"])
}

// Test that verifies the cache functionality exists and is properly integrated
func TestTechnicalIndicatorServiceV2_CacheComponents(t *testing.T) {
	service := NewTechnicalIndicatorServiceV2(nil)

	// Verify the service has the expected cache-related components
	assert.NotNil(t, service)

	// Test cache metrics structure
	metrics := service.GetServiceMetrics()
	poolStats, ok := metrics["pool_stats"].(map[string]interface{})
	assert.True(t, ok, "pool_stats should be a map")
	assert.Contains(t, poolStats, "pool_type")
	assert.Equal(t, "memory_pool", poolStats["pool_type"])

	// Test that service type indicates GoNum optimization
	assert.Equal(t, "GoNum-optimized", metrics["service_type"])
	assert.Equal(t, 6, metrics["calculators_loaded"])
}

// Benchmark test for service creation (with cache initialization)
func BenchmarkTechnicalIndicatorServiceV2_Creation(b *testing.B) {
	var candleRepo repository.CandleRepository // nil for this benchmark

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := NewTechnicalIndicatorServiceV2(candleRepo)
		_ = service.GetServiceMetrics() // Ensure initialization is complete
	}
}
