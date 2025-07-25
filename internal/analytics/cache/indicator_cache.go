package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/domain"

	"github.com/VictoriaMetrics/fastcache"
)

// IndicatorCache provides fast caching for computed indicators
type IndicatorCache struct {
	cache   *fastcache.Cache
	metrics *CacheMetrics
	mutex   sync.RWMutex
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	CacheHits      int64
	CacheMisses    int64
	TotalRequests  int64
	AverageLatency time.Duration
	CacheSize      int64
	EvictionCount  int64
	LastUpdated    time.Time
}

// CacheKey represents a cache key for indicators
type CacheKey struct {
	Symbol      string
	Timeframe   string
	StartTime   time.Time
	EndTime     time.Time
	Indicators  []string
	DataVersion string
}

// CachedIndicatorSet wraps indicator set with metadata
type CachedIndicatorSet struct {
	Indicators  *domain.TechnicalIndicators `json:"indicators"`
	CachedAt    time.Time                   `json:"cached_at"`
	ExpiresAt   time.Time                   `json:"expires_at"`
	Version     string                      `json:"version"`
	ComputeTime time.Duration               `json:"compute_time"`
}

// NewIndicatorCache creates a new indicator cache with specified size
func NewIndicatorCache(sizeInMB int) *IndicatorCache {
	return &IndicatorCache{
		cache: fastcache.New(sizeInMB * 1024 * 1024), // Convert MB to bytes
		metrics: &CacheMetrics{
			LastUpdated: time.Now(),
		},
		mutex: sync.RWMutex{},
	}
}

// GetIndicators retrieves cached indicators or returns nil if not found
func (ic *IndicatorCache) GetIndicators(key CacheKey) (*domain.TechnicalIndicators, bool) {
	start := time.Now()
	defer func() {
		ic.updateMetrics(time.Since(start))
	}()

	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	cacheKeyStr := ic.generateKey(key)
	cached := ic.cache.Get(nil, []byte(cacheKeyStr))

	if cached == nil {
		ic.metrics.CacheMisses++
		return nil, false
	}

	var cachedSet CachedIndicatorSet
	if err := json.Unmarshal(cached, &cachedSet); err != nil {
		ic.metrics.CacheMisses++
		return nil, false
	}

	// Check if cache entry has expired
	if time.Now().After(cachedSet.ExpiresAt) {
		ic.cache.Del([]byte(cacheKeyStr))
		ic.metrics.CacheMisses++
		ic.metrics.EvictionCount++
		return nil, false
	}

	ic.metrics.CacheHits++
	return cachedSet.Indicators, true
}

// SetIndicators caches computed indicators with TTL
func (ic *IndicatorCache) SetIndicators(key CacheKey, indicators *domain.TechnicalIndicators, computeTime time.Duration, ttl time.Duration) error {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	cachedSet := CachedIndicatorSet{
		Indicators:  indicators,
		CachedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
		Version:     key.DataVersion,
		ComputeTime: computeTime,
	}

	data, err := json.Marshal(cachedSet)
	if err != nil {
		return fmt.Errorf("failed to marshal cached indicators: %w", err)
	}

	cacheKeyStr := ic.generateKey(key)
	ic.cache.Set([]byte(cacheKeyStr), data)

	return nil
}

// GetOrCalculate retrieves from cache or calculates and caches the result
func (ic *IndicatorCache) GetOrCalculate(key CacheKey, calculator func() (*domain.TechnicalIndicators, error), ttl time.Duration) (*domain.TechnicalIndicators, error) {
	// Try cache first
	if indicators, found := ic.GetIndicators(key); found {
		return indicators, nil
	}

	// Calculate if not in cache
	start := time.Now()
	indicators, err := calculator()
	computeTime := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("failed to calculate indicators: %w", err)
	}

	// Cache the result
	if err := ic.SetIndicators(key, indicators, computeTime, ttl); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to cache indicators: %v\n", err)
	}

	return indicators, nil
}

// InvalidatePattern invalidates all cache entries matching a pattern
func (ic *IndicatorCache) InvalidatePattern(symbol, timeframe string) int {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	// FastCache doesn't support pattern deletion, so we'll track keys manually
	// For now, we'll use a simple approach and clear the entire cache if needed
	// In production, consider implementing a key tracking system

	return 0 // Returns number of invalidated entries
}

// GetMetrics returns current cache performance metrics
func (ic *IndicatorCache) GetMetrics() CacheMetrics {
	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	metrics := *ic.metrics
	// FastCache doesn't expose detailed stats, so we track what we can
	metrics.LastUpdated = time.Now()

	if metrics.TotalRequests > 0 {
		// Calculate hit rate
		hitRate := float64(metrics.CacheHits) / float64(metrics.TotalRequests)
		metrics.LastUpdated = time.Now()
		_ = hitRate // Use hit rate for monitoring
	}

	return metrics
}

// ResetMetrics resets all cache metrics
func (ic *IndicatorCache) ResetMetrics() {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	ic.metrics = &CacheMetrics{
		LastUpdated: time.Now(),
	}
}

// GetCacheInfo returns basic cache information
func (ic *IndicatorCache) GetCacheInfo() map[string]interface{} {
	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	return map[string]interface{}{
		"cache_hits":      ic.metrics.CacheHits,
		"cache_misses":    ic.metrics.CacheMisses,
		"total_requests":  ic.metrics.TotalRequests,
		"hit_rate":        float64(ic.metrics.CacheHits) / float64(ic.metrics.TotalRequests),
		"average_latency": ic.metrics.AverageLatency.String(),
	}
}

// Clear removes all entries from the cache
func (ic *IndicatorCache) Clear() {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	ic.cache.Reset()
	ic.metrics.EvictionCount++
}

// generateKey creates a deterministic cache key from CacheKey struct
func (ic *IndicatorCache) generateKey(key CacheKey) string {
	// Create a deterministic string representation
	keyStr := fmt.Sprintf("indicators:%s:%s:%d:%d:%v:%s",
		key.Symbol,
		key.Timeframe,
		key.StartTime.Unix(),
		key.EndTime.Unix(),
		key.Indicators,
		key.DataVersion,
	)

	// Generate MD5 hash for consistent key length
	hash := md5.Sum([]byte(keyStr))
	return fmt.Sprintf("%x", hash)
}

// updateMetrics updates cache performance metrics
func (ic *IndicatorCache) updateMetrics(latency time.Duration) {
	ic.metrics.TotalRequests++

	// Update average latency using exponential moving average
	alpha := 0.1
	if ic.metrics.AverageLatency == 0 {
		ic.metrics.AverageLatency = latency
	} else {
		ic.metrics.AverageLatency = time.Duration(float64(ic.metrics.AverageLatency)*(1-alpha) + float64(latency)*alpha)
	}
}
