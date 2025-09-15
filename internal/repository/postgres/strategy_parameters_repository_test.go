package postgres

import (
	"testing"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createTestDB creates a test database for testing
func createTestDB(t *testing.T) *gorm.DB {
	// For now, return nil since we don't have sqlite driver
	// In a real implementation, this would create a test database
	t.Skip("Test database setup requires sqlite driver")
	return nil
}

// createTestStrategyParameters creates test strategy parameters
func createTestStrategyParameters() *domain.StrategyParameters {
	mrHigh := 100.5
	mrLow := 99.0
	mrCalculated := true
	bufferPercentage := 0.5

	return &domain.StrategyParameters{
		// Common parameters
		CanGenerateLong:  true,
		CanGenerateShort: true,

		// 1ST_ENTRY strategy parameters
		MRHigh:           &mrHigh,
		MRLow:            &mrLow,
		MRCalculated:     &mrCalculated,
		BufferPercentage: &bufferPercentage,

		// Strategy state and metadata
		StrategyState: map[string]interface{}{
			"can_generate_long":  true,
			"can_generate_short": true,
			"mr_calculated":      true,
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
			"source":  "v2_engine",
		},
	}
}

func TestNewStrategyParametersRepositoryV2(t *testing.T) {
	t.Skip("Test requires database setup")

	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	assert.NotNil(t, repo)
	assert.Implements(t, (*repository.StrategyParametersRepository)(nil), repo)
}

func TestCreateTestStrategyParameters(t *testing.T) {
	params := createTestStrategyParameters()

	assert.NotNil(t, params)
	assert.True(t, params.CanGenerateLong)
	assert.True(t, params.CanGenerateShort)
	assert.NotNil(t, params.MRHigh)
	assert.NotNil(t, params.MRLow)
	assert.NotNil(t, params.MRCalculated)
	assert.NotNil(t, params.BufferPercentage)
	assert.Equal(t, 100.5, *params.MRHigh)
	assert.Equal(t, 99.0, *params.MRLow)
	assert.True(t, *params.MRCalculated)
	assert.Equal(t, 0.5, *params.BufferPercentage)
}

func TestStrategyParametersRepositoryV2_SaveParameters(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	params := createTestStrategyParameters()
	stockID := "RELIANCE"
	strategyName := "1ST_ENTRY"
	timestamp := time.Now()

	err := repo.SaveParameters(stockID, strategyName, timestamp, params)
	assert.NoError(t, err)

	// Verify the record was saved
	var record StrategyParametersRecord
	result := db.Where("stock_id = ? AND strategy_name = ? AND timestamp = ?",
		stockID, strategyName, timestamp).First(&record)
	assert.NoError(t, result.Error)
	assert.Equal(t, stockID, record.StockID)
	assert.Equal(t, strategyName, record.StrategyName)
	assert.True(t, record.IsValid)
	assert.False(t, record.IsSynchronized)
}

func TestStrategyParametersRepositoryV2_GetParameters(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	params := createTestStrategyParameters()
	stockID := "RELIANCE"
	strategyName := "1ST_ENTRY"
	timestamp := time.Now()

	// Save parameters first
	err := repo.SaveParameters(stockID, strategyName, timestamp, params)
	require.NoError(t, err)

	// Retrieve parameters
	retrievedParams, err := repo.GetParameters(stockID, strategyName, timestamp)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedParams)

	// Verify the parameters match
	assert.Equal(t, params.CanGenerateLong, retrievedParams.CanGenerateLong)
	assert.Equal(t, params.CanGenerateShort, retrievedParams.CanGenerateShort)
	assert.Equal(t, *params.MRHigh, *retrievedParams.MRHigh)
	assert.Equal(t, *params.MRLow, *retrievedParams.MRLow)
	assert.Equal(t, *params.MRCalculated, *retrievedParams.MRCalculated)
	assert.Equal(t, *params.BufferPercentage, *retrievedParams.BufferPercentage)
}

func TestStrategyParametersRepositoryV2_GetParameters_NotFound(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	// Try to get non-existent parameters
	retrievedParams, err := repo.GetParameters("NONEXISTENT", "1ST_ENTRY", time.Now())
	assert.NoError(t, err)
	assert.Nil(t, retrievedParams)
}

func TestStrategyParametersRepositoryV2_GetLatestParameters(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	params := createTestStrategyParameters()
	stockID := "RELIANCE"
	strategyName := "1ST_ENTRY"
	timestamp1 := time.Now().Add(-5 * time.Minute)
	timestamp2 := time.Now()

	// Save parameters with different timestamps
	err := repo.SaveParameters(stockID, strategyName, timestamp1, params)
	require.NoError(t, err)

	err = repo.SaveParameters(stockID, strategyName, timestamp2, params)
	require.NoError(t, err)

	// Get latest parameters
	latestParams, err := repo.GetLatestParameters(stockID, strategyName)
	assert.NoError(t, err)
	assert.NotNil(t, latestParams)

	// Verify we got the latest parameters (timestamp2)
	assert.Equal(t, params.CanGenerateLong, latestParams.CanGenerateLong)
	assert.Equal(t, params.CanGenerateShort, latestParams.CanGenerateShort)
}

func TestStrategyParametersRepositoryV2_GetParametersRange(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	params := createTestStrategyParameters()
	stockID := "RELIANCE"
	strategyName := "1ST_ENTRY"

	// Create timestamps for range testing
	startTime := time.Now().Add(-10 * time.Minute)
	middleTime := time.Now().Add(-5 * time.Minute)
	endTime := time.Now()

	// Save parameters at different times
	err := repo.SaveParameters(stockID, strategyName, startTime, params)
	require.NoError(t, err)

	err = repo.SaveParameters(stockID, strategyName, middleTime, params)
	require.NoError(t, err)

	err = repo.SaveParameters(stockID, strategyName, endTime, params)
	require.NoError(t, err)

	// Get parameters in range
	rangeParams, err := repo.GetParametersRange(stockID, strategyName, startTime, endTime)
	assert.NoError(t, err)
	assert.Len(t, rangeParams, 3)

	// Verify the parameters are ordered by timestamp
	assert.True(t, rangeParams[0].CandleTimestamp.Before(rangeParams[1].CandleTimestamp))
	assert.True(t, rangeParams[1].CandleTimestamp.Before(rangeParams[2].CandleTimestamp))
}

func TestStrategyParametersRepositoryV2_UpdateParameters(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	params := createTestStrategyParameters()
	stockID := "RELIANCE"
	strategyName := "1ST_ENTRY"
	timestamp := time.Now()

	// Save initial parameters
	err := repo.SaveParameters(stockID, strategyName, timestamp, params)
	require.NoError(t, err)

	// Update parameters
	newMRHigh := 105.0
	params.MRHigh = &newMRHigh
	params.Metadata["updated"] = true

	// Get the record ID for update
	var record StrategyParametersRecord
	result := db.Where("stock_id = ? AND strategy_name = ? AND timestamp = ?",
		stockID, strategyName, timestamp).First(&record)
	require.NoError(t, result.Error)

	// Update parameters (note: this would need a proper ID field in the domain record)
	// For now, we'll test the save functionality which handles upserts
	err = repo.SaveParameters(stockID, strategyName, timestamp, params)
	assert.NoError(t, err)

	// Verify the update
	retrievedParams, err := repo.GetParameters(stockID, strategyName, timestamp)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedParams)
	assert.Equal(t, newMRHigh, *retrievedParams.MRHigh)
}

func TestStrategyParametersRepositoryV2_DeleteParameters(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	params := createTestStrategyParameters()
	stockID := "RELIANCE"
	strategyName := "1ST_ENTRY"
	timestamp := time.Now()

	// Save parameters
	err := repo.SaveParameters(stockID, strategyName, timestamp, params)
	require.NoError(t, err)

	// Verify parameters exist
	retrievedParams, err := repo.GetParameters(stockID, strategyName, timestamp)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedParams)

	// Delete parameters
	err = repo.DeleteParameters(stockID, strategyName, timestamp)
	assert.NoError(t, err)

	// Verify parameters are deleted
	retrievedParams, err = repo.GetParameters(stockID, strategyName, timestamp)
	assert.NoError(t, err)
	assert.Nil(t, retrievedParams)
}

func TestStrategyParametersRepositoryV2_CleanupOldParameters(t *testing.T) {
	db := createTestDB(t)
	repo := NewStrategyParametersRepositoryV2(db)

	params := createTestStrategyParameters()
	stockID := "RELIANCE"
	strategyName := "1ST_ENTRY"

	// Create old and new timestamps
	oldTime := time.Now().Add(-24 * time.Hour)
	newTime := time.Now()

	// Save old parameters
	err := repo.SaveParameters(stockID, strategyName, oldTime, params)
	require.NoError(t, err)

	// Save new parameters
	err = repo.SaveParameters(stockID, strategyName, newTime, params)
	require.NoError(t, err)

	// Cleanup old parameters
	cutoffTime := time.Now().Add(-12 * time.Hour)
	err = repo.CleanupOldParameters(cutoffTime)
	assert.NoError(t, err)

	// Verify old parameters are deleted
	oldParams, err := repo.GetParameters(stockID, strategyName, oldTime)
	assert.NoError(t, err)
	assert.Nil(t, oldParams)

	// Verify new parameters still exist
	newParams, err := repo.GetParameters(stockID, strategyName, newTime)
	assert.NoError(t, err)
	assert.NotNil(t, newParams)
}

func BenchmarkStrategyParametersRepositoryV2_SaveParameters(b *testing.B) {
	b.Skip("Benchmark requires test database setup")
}

func BenchmarkStrategyParametersRepositoryV2_GetParameters(b *testing.B) {
	b.Skip("Benchmark requires test database setup")
}
