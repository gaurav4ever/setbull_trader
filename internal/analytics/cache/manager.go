package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"setbull_trader/internal/analytics"

	"github.com/VictoriaMetrics/fastcache"
)

// CacheManager handles caching of analytics results
type CacheManager struct {
	cache   *fastcache.Cache
	config  *CacheConfig
	metrics *CacheMetrics
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled            bool          `json:"enabled"`
	SizeInMB           int           `json:"size_in_mb"`
	TTL                time.Duration `json:"ttl"`
	KeyPrefix          string        `json:"key_prefix"`
	MaxKeyLength       int           `json:"max_key_length"`
	CompressionEnabled bool          `json:"compression_enabled"`
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	TotalHits      int64     `json:"total_hits"`
	TotalMisses    int64     `json:"total_misses"`
	TotalSets      int64     `json:"total_sets"`
	TotalDeletes   int64     `json:"total_deletes"`
	HitRate        float64   `json:"hit_rate"`
	BytesStored    int64     `json:"bytes_stored"`
	KeyCount       int64     `json:"key_count"`
	LastAccessTime time.Time `json:"last_access_time"`
	LastResetTime  time.Time `json:"last_reset_time"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled:            true,
		SizeInMB:           512, // 512MB default
		TTL:                30 * time.Minute,
		KeyPrefix:          "analytics:",
		MaxKeyLength:       250,
		CompressionEnabled: false,
	}
}

// NewCacheManager creates a new cache manager
func NewCacheManager(config *CacheConfig) *CacheManager {
	if config == nil {
		config = DefaultCacheConfig()
	}

	var cache *fastcache.Cache
	if config.Enabled {
		cache = fastcache.New(config.SizeInMB * 1024 * 1024) // Convert MB to bytes
	}

	return &CacheManager{
		cache:  cache,
		config: config,
		metrics: &CacheMetrics{
			LastResetTime: time.Now(),
		},
	}
}

// GetProcessingResult retrieves cached processing result
func (c *CacheManager) GetProcessingResult(ctx context.Context, key string) (*analytics.ProcessingResult, bool) {
	if !c.config.Enabled || c.cache == nil {
		return nil, false
	}

	normalizedKey := c.normalizeKey(key)
	data := c.cache.Get(nil, []byte(normalizedKey))

	if data == nil {
		c.metrics.TotalMisses++
		c.updateHitRate()
		return nil, false
	}

	var result analytics.ProcessingResult
	if err := json.Unmarshal(data, &result); err != nil {
		// Invalid data in cache, remove it
		c.cache.Del([]byte(normalizedKey))
		c.metrics.TotalMisses++
		c.updateHitRate()
		return nil, false
	}

	c.metrics.TotalHits++
	c.metrics.LastAccessTime = time.Now()
	c.updateHitRate()

	return &result, true
}

// SetProcessingResult stores processing result in cache
func (c *CacheManager) SetProcessingResult(ctx context.Context, key string, result *analytics.ProcessingResult) error {
	if !c.config.Enabled || c.cache == nil || result == nil {
		return nil
	}

	normalizedKey := c.normalizeKey(key)

	// Serialize result (excluding DataFrame for now as it's complex to serialize)
	cacheData := struct {
		Indicators  *analytics.IndicatorSet `json:"indicators"`
		CacheHits   int                     `json:"cache_hits"`
		ProcessTime time.Duration           `json:"process_time"`
		MemoryUsage int64                   `json:"memory_usage"`
		CachedAt    time.Time               `json:"cached_at"`
	}{
		Indicators:  result.Indicators,
		CacheHits:   result.CacheHits,
		ProcessTime: result.ProcessTime,
		MemoryUsage: result.MemoryUsage,
		CachedAt:    time.Now(),
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %v", err)
	}

	c.cache.Set([]byte(normalizedKey), data)
	c.metrics.TotalSets++
	c.metrics.BytesStored += int64(len(data))
	c.metrics.KeyCount++

	return nil
}

// GetIndicators retrieves cached indicators
func (c *CacheManager) GetIndicators(ctx context.Context, key string) (*analytics.IndicatorSet, bool) {
	if !c.config.Enabled || c.cache == nil {
		return nil, false
	}

	normalizedKey := c.normalizeKey("indicators:" + key)
	data := c.cache.Get(nil, []byte(normalizedKey))

	if data == nil {
		c.metrics.TotalMisses++
		c.updateHitRate()
		return nil, false
	}

	var indicators analytics.IndicatorSet
	if err := json.Unmarshal(data, &indicators); err != nil {
		c.cache.Del([]byte(normalizedKey))
		c.metrics.TotalMisses++
		c.updateHitRate()
		return nil, false
	}

	c.metrics.TotalHits++
	c.metrics.LastAccessTime = time.Now()
	c.updateHitRate()

	return &indicators, true
}

// SetIndicators stores indicators in cache
func (c *CacheManager) SetIndicators(ctx context.Context, key string, indicators *analytics.IndicatorSet) error {
	if !c.config.Enabled || c.cache == nil || indicators == nil {
		return nil
	}

	normalizedKey := c.normalizeKey("indicators:" + key)

	data, err := json.Marshal(indicators)
	if err != nil {
		return fmt.Errorf("failed to marshal indicators: %v", err)
	}

	c.cache.Set([]byte(normalizedKey), data)
	c.metrics.TotalSets++
	c.metrics.BytesStored += int64(len(data))

	return nil
}

// Delete removes item from cache
func (c *CacheManager) Delete(ctx context.Context, key string) error {
	if !c.config.Enabled || c.cache == nil {
		return nil
	}

	normalizedKey := c.normalizeKey(key)
	c.cache.Del([]byte(normalizedKey))
	c.metrics.TotalDeletes++
	c.metrics.KeyCount--

	return nil
}

// Clear clears all cache entries
func (c *CacheManager) Clear(ctx context.Context) error {
	if !c.config.Enabled || c.cache == nil {
		return nil
	}

	c.cache.Reset()
	c.resetMetrics()

	return nil
}

// GetMetrics returns cache metrics
func (c *CacheManager) GetMetrics() *CacheMetrics {
	return c.metrics
}

// GetStats returns cache statistics
func (c *CacheManager) GetStats() map[string]interface{} {
	if c.cache == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	return map[string]interface{}{
		"enabled":      c.config.Enabled,
		"size_mb":      c.config.SizeInMB,
		"total_hits":   c.metrics.TotalHits,
		"total_misses": c.metrics.TotalMisses,
		"hit_rate":     c.metrics.HitRate,
		"key_count":    c.metrics.KeyCount,
		"bytes_stored": c.metrics.BytesStored,
	}
}

// normalizeKey ensures key is valid and within length limits
func (c *CacheManager) normalizeKey(key string) string {
	normalized := c.config.KeyPrefix + key

	if len(normalized) > c.config.MaxKeyLength {
		// Truncate key if too long
		normalized = normalized[:c.config.MaxKeyLength]
	}

	return normalized
}

// updateHitRate calculates current hit rate
func (c *CacheManager) updateHitRate() {
	total := c.metrics.TotalHits + c.metrics.TotalMisses
	if total > 0 {
		c.metrics.HitRate = float64(c.metrics.TotalHits) / float64(total)
	}
}

// resetMetrics resets all metrics
func (c *CacheManager) resetMetrics() {
	c.metrics = &CacheMetrics{
		LastResetTime: time.Now(),
	}
}

// IsEnabled returns whether caching is enabled
func (c *CacheManager) IsEnabled() bool {
	return c.config.Enabled && c.cache != nil
}

// GetConfig returns current cache configuration
func (c *CacheManager) GetConfig() *CacheConfig {
	return c.config
}

// SetConfig updates cache configuration
func (c *CacheManager) SetConfig(config *CacheConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	c.config = config

	// Recreate cache if enabled state changed or cache doesn't exist
	if config.Enabled && c.cache == nil {
		c.cache = fastcache.New(config.SizeInMB * 1024 * 1024)
		c.resetMetrics()
	} else if !config.Enabled {
		c.cache = nil
		c.resetMetrics()
	}

	return nil
}

// Close gracefully shuts down the cache manager
func (c *CacheManager) Close() error {
	// fastcache doesn't require explicit closing
	// This method is here for interface compatibility
	return nil
}

// GenerateCandleKey generates a cache key for candle data
func (c *CacheManager) GenerateCandleKey(instrumentKey string, startTime, endTime time.Time, candleCount int) string {
	return fmt.Sprintf("candles:%s:%s:%s:%d",
		instrumentKey,
		startTime.Format("2006-01-02T15:04:05"),
		endTime.Format("2006-01-02T15:04:05"),
		candleCount,
	)
}

// GenerateIndicatorKey generates a cache key for indicator data
func (c *CacheManager) GenerateIndicatorKey(instrumentKey string, timeframe string, candleCount int) string {
	return fmt.Sprintf("indicators:%s:%s:%d",
		instrumentKey,
		timeframe,
		candleCount,
	)
}

// GenerateAggregationKey generates a cache key for aggregated data
func (c *CacheManager) GenerateAggregationKey(instrumentKey string, timeframe string, interval string) string {
	return fmt.Sprintf("aggregation:%s:%s:%s",
		instrumentKey,
		timeframe,
		interval,
	)
}
