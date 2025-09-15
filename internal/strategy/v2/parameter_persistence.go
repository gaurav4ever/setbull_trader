package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"
	"setbull_trader/pkg/log"
)

// ParameterPersistenceManager manages parameter persistence for V2 strategies
type ParameterPersistenceManager struct {
	repository repository.StrategyParametersRepository
	cache      map[string]*domain.StrategyParameters
	mu         sync.RWMutex
}

// NewParameterPersistenceManager creates a new parameter persistence manager
func NewParameterPersistenceManager(repository repository.StrategyParametersRepository) *ParameterPersistenceManager {
	return &ParameterPersistenceManager{
		repository: repository,
		cache:      make(map[string]*domain.StrategyParameters),
	}
}

// SaveStrategyParameters saves strategy parameters to the database
func (m *ParameterPersistenceManager) SaveStrategyParameters(
	ctx context.Context,
	stockID, strategyName string,
	timestamp time.Time,
	params map[string]interface{},
) error {
	// Convert params to domain.StrategyParameters
	domainParams := m.convertToDomainParameters(params)

	// Save to database
	err := m.repository.SaveParameters(stockID, strategyName, timestamp, domainParams)
	if err != nil {
		return fmt.Errorf("failed to save parameters: %w", err)
	}

	// Update cache
	cacheKey := m.getCacheKey(stockID, strategyName, timestamp)
	m.mu.Lock()
	m.cache[cacheKey] = domainParams
	m.mu.Unlock()

	log.Info("Saved parameters for %s:%s at %s", stockID, strategyName, timestamp.Format("2006-01-02 15:04:05"))
	return nil
}

// GetStrategyParameters retrieves strategy parameters from cache or database
func (m *ParameterPersistenceManager) GetStrategyParameters(
	ctx context.Context,
	stockID, strategyName string,
	timestamp time.Time,
) (map[string]interface{}, error) {
	cacheKey := m.getCacheKey(stockID, strategyName, timestamp)

	// Check cache first
	m.mu.RLock()
	if cached, exists := m.cache[cacheKey]; exists {
		m.mu.RUnlock()
		return m.convertFromDomainParameters(cached), nil
	}
	m.mu.RUnlock()

	// Get from database
	domainParams, err := m.repository.GetParameters(stockID, strategyName, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get parameters: %w", err)
	}

	// Update cache
	m.mu.Lock()
	m.cache[cacheKey] = domainParams
	m.mu.Unlock()

	return m.convertFromDomainParameters(domainParams), nil
}

// GetLatestParameters gets the latest parameters for a stock and strategy
func (m *ParameterPersistenceManager) GetLatestParameters(
	ctx context.Context,
	stockID, strategyName string,
) (map[string]interface{}, error) {
	domainParams, err := m.repository.GetLatestParameters(stockID, strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest parameters: %w", err)
	}

	return m.convertFromDomainParameters(domainParams), nil
}

// convertToDomainParameters converts generic parameters to domain.StrategyParameters
func (m *ParameterPersistenceManager) convertToDomainParameters(params map[string]interface{}) *domain.StrategyParameters {
	domainParams := &domain.StrategyParameters{
		StrategyState: make(map[string]interface{}),
		Metadata:      make(map[string]interface{}),
	}

	// Convert common parameters
	if canGenerateLong, ok := params["can_generate_long"].(bool); ok {
		domainParams.CanGenerateLong = canGenerateLong
	}
	if canGenerateShort, ok := params["can_generate_short"].(bool); ok {
		domainParams.CanGenerateShort = canGenerateShort
	}

	// Convert 1ST_ENTRY parameters
	if mrHigh, ok := params["mr_high"].(float64); ok {
		domainParams.MRHigh = &mrHigh
	}
	if mrLow, ok := params["mr_low"].(float64); ok {
		domainParams.MRLow = &mrLow
	}
	if mrHighWithBuffer, ok := params["mr_high_with_buffer"].(float64); ok {
		domainParams.MRHighWithBuffer = &mrHighWithBuffer
	}
	if mrLowWithBuffer, ok := params["mr_low_with_buffer"].(float64); ok {
		domainParams.MRLowWithBuffer = &mrLowWithBuffer
	}
	if bufferPercentage, ok := params["buffer_percentage"].(float64); ok {
		domainParams.BufferPercentage = &bufferPercentage
	}
	if mrCalculated, ok := params["mr_calculated"].(bool); ok {
		domainParams.MRCalculated = &mrCalculated
	}

	// Convert 2_30_ENTRY parameters
	if entryTime, ok := params["entry_time"].(time.Time); ok {
		domainParams.EntryTime = &entryTime
	}
	if rangeHigh, ok := params["range_high"].(float64); ok {
		domainParams.RangeHigh = &rangeHigh
	}
	if rangeLow, ok := params["range_low"].(float64); ok {
		domainParams.RangeLow = &rangeLow
	}
	if rangeHighEntryPrice, ok := params["range_high_entry_price"].(float64); ok {
		domainParams.RangeHighEntryPrice = &rangeHighEntryPrice
	}
	if rangeLowEntryPrice, ok := params["range_low_entry_price"].(float64); ok {
		domainParams.RangeLowEntryPrice = &rangeLowEntryPrice
	}
	if direction, ok := params["direction"].(string); ok {
		domainParams.Direction = &direction
	}

	// Convert BB_WIDTH_ENTRY parameters
	if bbWidthThreshold, ok := params["bb_width_threshold"].(float64); ok {
		domainParams.BBWidthThreshold = &bbWidthThreshold
	}
	if bbPeriod, ok := params["bb_period"].(int); ok {
		domainParams.BBPeriod = &bbPeriod
	}
	if bbStdDev, ok := params["bb_std_dev"].(float64); ok {
		domainParams.BBStdDev = &bbStdDev
	}
	if currentBBWidth, ok := params["current_bb_width"].(float64); ok {
		domainParams.CurrentBBWidth = &currentBBWidth
	}
	if lowestBBWidth, ok := params["lowest_bb_width"].(float64); ok {
		domainParams.LowestBBWidth = &lowestBBWidth
	}
	if squeezeDetected, ok := params["squeeze_detected"].(bool); ok {
		domainParams.SqueezeDetected = squeezeDetected
	}
	if squeezeStartTime, ok := params["squeeze_start_time"].(time.Time); ok {
		domainParams.SqueezeStartTime = &squeezeStartTime
	}
	if squeezeCandleCount, ok := params["squeeze_candle_count"].(int); ok {
		domainParams.SqueezeCandleCount = &squeezeCandleCount
	}
	if bbUpper, ok := params["bb_upper"].(float64); ok {
		domainParams.BBUpper = &bbUpper
	}
	if bbLower, ok := params["bb_lower"].(float64); ok {
		domainParams.BBLower = &bbLower
	}
	if bbMiddle, ok := params["bb_middle"].(float64); ok {
		domainParams.BBMiddle = &bbMiddle
	}

	// Convert strategy state and metadata
	if strategyState, ok := params["strategy_state"].(map[string]interface{}); ok {
		domainParams.StrategyState = strategyState
	}
	if metadata, ok := params["metadata"].(map[string]interface{}); ok {
		domainParams.Metadata = metadata
	}

	return domainParams
}

// convertFromDomainParameters converts domain.StrategyParameters to generic parameters
func (m *ParameterPersistenceManager) convertFromDomainParameters(domainParams *domain.StrategyParameters) map[string]interface{} {
	params := make(map[string]interface{})

	// Convert common parameters
	params["can_generate_long"] = domainParams.CanGenerateLong
	params["can_generate_short"] = domainParams.CanGenerateShort

	// Convert 1ST_ENTRY parameters
	if domainParams.MRHigh != nil {
		params["mr_high"] = *domainParams.MRHigh
	}
	if domainParams.MRLow != nil {
		params["mr_low"] = *domainParams.MRLow
	}
	if domainParams.MRHighWithBuffer != nil {
		params["mr_high_with_buffer"] = *domainParams.MRHighWithBuffer
	}
	if domainParams.MRLowWithBuffer != nil {
		params["mr_low_with_buffer"] = *domainParams.MRLowWithBuffer
	}
	if domainParams.BufferPercentage != nil {
		params["buffer_percentage"] = *domainParams.BufferPercentage
	}
	if domainParams.MRCalculated != nil {
		params["mr_calculated"] = *domainParams.MRCalculated
	}

	// Convert 2_30_ENTRY parameters
	if domainParams.EntryTime != nil {
		params["entry_time"] = *domainParams.EntryTime
	}
	if domainParams.RangeHigh != nil {
		params["range_high"] = *domainParams.RangeHigh
	}
	if domainParams.RangeLow != nil {
		params["range_low"] = *domainParams.RangeLow
	}
	if domainParams.RangeHighEntryPrice != nil {
		params["range_high_entry_price"] = *domainParams.RangeHighEntryPrice
	}
	if domainParams.RangeLowEntryPrice != nil {
		params["range_low_entry_price"] = *domainParams.RangeLowEntryPrice
	}
	if domainParams.Direction != nil {
		params["direction"] = *domainParams.Direction
	}

	// Convert BB_WIDTH_ENTRY parameters
	if domainParams.BBWidthThreshold != nil {
		params["bb_width_threshold"] = *domainParams.BBWidthThreshold
	}
	if domainParams.BBPeriod != nil {
		params["bb_period"] = *domainParams.BBPeriod
	}
	if domainParams.BBStdDev != nil {
		params["bb_std_dev"] = *domainParams.BBStdDev
	}
	if domainParams.CurrentBBWidth != nil {
		params["current_bb_width"] = *domainParams.CurrentBBWidth
	}
	if domainParams.LowestBBWidth != nil {
		params["lowest_bb_width"] = *domainParams.LowestBBWidth
	}
	params["squeeze_detected"] = domainParams.SqueezeDetected
	if domainParams.SqueezeStartTime != nil {
		params["squeeze_start_time"] = *domainParams.SqueezeStartTime
	}
	if domainParams.SqueezeCandleCount != nil {
		params["squeeze_candle_count"] = *domainParams.SqueezeCandleCount
	}
	if domainParams.BBUpper != nil {
		params["bb_upper"] = *domainParams.BBUpper
	}
	if domainParams.BBLower != nil {
		params["bb_lower"] = *domainParams.BBLower
	}
	if domainParams.BBMiddle != nil {
		params["bb_middle"] = *domainParams.BBMiddle
	}

	// Convert strategy state and metadata
	params["strategy_state"] = domainParams.StrategyState
	params["metadata"] = domainParams.Metadata

	return params
}

// getCacheKey generates a cache key for strategy parameters
func (m *ParameterPersistenceManager) getCacheKey(stockID, strategyName string, timestamp time.Time) string {
	return fmt.Sprintf("%s:%s:%d", stockID, strategyName, timestamp.Unix())
}

// CleanupCache cleans up old cache entries
func (m *ParameterPersistenceManager) CleanupCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple cleanup - clear all cache (in production, implement LRU or TTL)
	m.cache = make(map[string]*domain.StrategyParameters)
}
