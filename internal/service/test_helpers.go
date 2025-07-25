package service

import (
	"context"
	"time"

	"setbull_trader/internal/domain"

	"github.com/stretchr/testify/mock"
)

// MockCandleRepository is a complete mock implementation of repository.CandleRepository
type MockCandleRepository struct {
	mock.Mock
}

func (m *MockCandleRepository) Store(ctx context.Context, candle *domain.Candle) error {
	args := m.Called(ctx, candle)
	return args.Error(0)
}

func (m *MockCandleRepository) StoreBatch(ctx context.Context, candles []domain.Candle) (int, error) {
	args := m.Called(ctx, candles)
	return args.Int(0), args.Error(1)
}

func (m *MockCandleRepository) FindByInstrumentKey(ctx context.Context, instrumentKey string) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *MockCandleRepository) FindByInstrumentAndInterval(ctx context.Context, instrumentKey, interval string) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *MockCandleRepository) FindByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, interval string, fromTime, toTime time.Time) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval, fromTime, toTime)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *MockCandleRepository) DeleteByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, interval string, fromTime, toTime time.Time) (int, error) {
	args := m.Called(ctx, instrumentKey, interval, fromTime, toTime)
	return args.Int(0), args.Error(1)
}

func (m *MockCandleRepository) CountByInstrumentAndTimeRange(ctx context.Context, instrumentKey string, interval string, fromTime, toTime time.Time) (int, error) {
	args := m.Called(ctx, instrumentKey, interval, fromTime, toTime)
	return args.Int(0), args.Error(1)
}

func (m *MockCandleRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) (int, error) {
	args := m.Called(ctx, olderThan)
	return args.Int(0), args.Error(1)
}

func (m *MockCandleRepository) GetLatestCandle(ctx context.Context, instrumentKey, interval string) (*domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Candle), args.Error(1)
}

func (m *MockCandleRepository) GetEarliestCandle(ctx context.Context, instrumentKey string, interval string) (*domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Candle), args.Error(1)
}

func (m *MockCandleRepository) GetCandleDateRange(ctx context.Context, instrumentKey string, interval string) (earliest, latest time.Time, exists bool, err error) {
	args := m.Called(ctx, instrumentKey, interval)
	return args.Get(0).(time.Time), args.Get(1).(time.Time), args.Bool(2), args.Error(3)
}

func (m *MockCandleRepository) GetNDailyCandlesByTimeframe(ctx context.Context, instrumentKey string, interval string, n int) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, interval, n)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *MockCandleRepository) GetAggregated5MinCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	args := m.Called(ctx, instrumentKey, start, end)
	return args.Get(0).([]domain.AggregatedCandle), args.Error(1)
}

func (m *MockCandleRepository) GetAggregatedDailyCandles(ctx context.Context, instrumentKey string, start, end time.Time) ([]domain.AggregatedCandle, error) {
	args := m.Called(ctx, instrumentKey, start, end)
	return args.Get(0).([]domain.AggregatedCandle), args.Error(1)
}

func (m *MockCandleRepository) GetDailyCandlesByTimeframe(ctx context.Context, instrumentKey string, startTime time.Time) ([]domain.Candle, error) {
	args := m.Called(ctx, instrumentKey, startTime)
	return args.Get(0).([]domain.Candle), args.Error(1)
}

func (m *MockCandleRepository) StoreAggregatedCandles(ctx context.Context, candles []domain.CandleData) error {
	args := m.Called(ctx, candles)
	return args.Error(0)
}

func (m *MockCandleRepository) GetStocksWithExistingDailyCandles(ctx context.Context, startDate, endDate time.Time) ([]string, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).([]string), args.Error(1)
}

// MockMasterDataProcessRepository is a complete mock implementation of repository.MasterDataProcessRepository
type MockMasterDataProcessRepository struct {
	mock.Mock
}

func (m *MockMasterDataProcessRepository) Create(ctx context.Context, processDate time.Time, numberOfPastDays int) (*domain.MasterDataProcess, error) {
	args := m.Called(ctx, processDate, numberOfPastDays)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcess), args.Error(1)
}

func (m *MockMasterDataProcessRepository) GetByDate(ctx context.Context, processDate time.Time) (*domain.MasterDataProcess, error) {
	args := m.Called(ctx, processDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcess), args.Error(1)
}

func (m *MockMasterDataProcessRepository) GetByID(ctx context.Context, processID int64) (*domain.MasterDataProcess, error) {
	args := m.Called(ctx, processID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcess), args.Error(1)
}

func (m *MockMasterDataProcessRepository) UpdateStatus(ctx context.Context, processID int64, status string) error {
	args := m.Called(ctx, processID, status)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) CompleteProcess(ctx context.Context, processID int64) error {
	args := m.Called(ctx, processID)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) CreateStep(ctx context.Context, processID int64, stepNumber int, stepName string) error {
	args := m.Called(ctx, processID, stepNumber, stepName)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) GetStep(ctx context.Context, processID int64, stepNumber int) (*domain.MasterDataProcessStep, error) {
	args := m.Called(ctx, processID, stepNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MasterDataProcessStep), args.Error(1)
}

func (m *MockMasterDataProcessRepository) UpdateStepStatus(ctx context.Context, processID int64, stepNumber int, status string, errorMessage ...string) error {
	args := m.Called(ctx, processID, stepNumber, status, errorMessage)
	return args.Error(0)
}

func (m *MockMasterDataProcessRepository) GetFilteredStocks(ctx context.Context, processDate time.Time) ([]domain.FilteredStockRecord, error) {
	args := m.Called(ctx, processDate)
	return args.Get(0).([]domain.FilteredStockRecord), args.Error(1)
}

func (m *MockMasterDataProcessRepository) GetProcessHistory(ctx context.Context, limit int) ([]domain.MasterDataProcess, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]domain.MasterDataProcess), args.Error(1)
}
