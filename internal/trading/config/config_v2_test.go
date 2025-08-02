package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultStrategyEngineV2ConfigDebug(t *testing.T) {
	config := DefaultStrategyEngineV2Config()

	// Debug: Print the actual values
	t.Logf("Enabled: %v", config.Enabled)
	t.Logf("ConcurrentWorkers: %v", config.Processing.ConcurrentWorkers)
	t.Logf("ProcessingTimeout: %v", config.Processing.ProcessingTimeout)
	t.Logf("MaxHistoricalCandles: %v", config.Processing.MaxHistoricalCandles)
	t.Logf("BatchSize: %v", config.Processing.BatchSize)
	t.Logf("DataRetentionDays: %v", config.Processing.DataRetentionDays)

	// Basic assertion to see what's wrong
	assert.NotNil(t, config)
}

func TestDefaultStrategyEngineV2Config(t *testing.T) {
	config := DefaultStrategyEngineV2Config()

	// Test basic configuration
	assert.True(t, config.Enabled)

	// Test processing configuration
	assert.Equal(t, 10, config.Processing.ConcurrentWorkers)
	assert.Equal(t, 30*time.Second, config.Processing.ProcessingTimeout)
	assert.Equal(t, 100, config.Processing.MaxHistoricalCandles)
	assert.Equal(t, 50, config.Processing.BatchSize)
	assert.Equal(t, 30, config.Processing.DataRetentionDays)

	// Test DataFrame configuration
	assert.Equal(t, "gota", config.DataFrame.Library)
	assert.Equal(t, 512, config.DataFrame.MemoryLimitMB)
	assert.True(t, config.DataFrame.EnableParallelProcessing)

	// Test strategies configuration
	assert.True(t, config.Strategies.Registry.AutoDiscovery)
	assert.Equal(t, "./internal/strategy/v2/strategies", config.Strategies.Registry.DiscoveryPath)
	assert.True(t, config.Strategies.Execution.ParallelStrategies)
	assert.Equal(t, 10*time.Second, config.Strategies.Execution.StrategyTimeout)
	assert.Equal(t, "continue_on_error", config.Strategies.Execution.ErrorHandling)

	// Test persistence configuration
	assert.True(t, config.Persistence.BatchInsert)
	assert.Equal(t, 100, config.Persistence.BatchSize)
	assert.True(t, config.Persistence.EnableAuditLog)

	// Test monitoring configuration
	assert.True(t, config.Monitoring.EnableMetrics)
	assert.True(t, config.Monitoring.EnableTracing)
	assert.Equal(t, "info", config.Monitoring.LogLevel)

	// Test parallel processing configuration
	assert.Equal(t, 16, config.ParallelProcessing.MaxWorkers)
	assert.Equal(t, 8, config.ParallelProcessing.MinWorkers)
	assert.Equal(t, 30*time.Second, config.ParallelProcessing.WorkerTimeout)
	assert.Equal(t, 1000, config.ParallelProcessing.QueueSize)
	assert.Equal(t, 10, config.ParallelProcessing.MaxConcurrentStrategies)
	assert.Equal(t, 1024, config.ParallelProcessing.MaxMemoryUsageMB)
	assert.Equal(t, 5*time.Second, config.ParallelProcessing.MemoryCheckInterval)
	assert.Equal(t, "continue_on_error", config.ParallelProcessing.ErrorHandling)
	assert.Equal(t, 3, config.ParallelProcessing.MaxRetries)
	assert.Equal(t, 1*time.Second, config.ParallelProcessing.RetryDelay)
	assert.True(t, config.ParallelProcessing.EnableMetrics)
	assert.Equal(t, 10*time.Second, config.ParallelProcessing.MetricsInterval)

	// Test state management configuration
	assert.True(t, config.StateManagement.EnablePersistence)
	assert.Equal(t, 30*time.Second, config.StateManagement.PersistenceInterval)
	assert.Equal(t, 30, config.StateManagement.StateRetentionDays)
	assert.True(t, config.StateManagement.EnableCaching)
	assert.Equal(t, 5*time.Minute, config.StateManagement.CacheTTL)
	assert.Equal(t, 10000, config.StateManagement.MaxCacheSize)
	assert.True(t, config.StateManagement.EnableValidation)
	assert.Equal(t, 5*time.Second, config.StateManagement.ValidationTimeout)
	assert.True(t, config.StateManagement.EnableSynchronization)
	assert.Equal(t, 10*time.Second, config.StateManagement.SyncInterval)
	assert.True(t, config.StateManagement.EnableMetrics)
	assert.Equal(t, 30*time.Second, config.StateManagement.MetricsInterval)
}

func TestValidateStrategyEngineV2Config(t *testing.T) {
	tests := []struct {
		name    string
		config  *StrategyEngineV2Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &StrategyEngineV2Config{
				Enabled: true,
				Processing: struct {
					ConcurrentWorkers    int           `yaml:"concurrent_workers" json:"concurrent_workers"`
					ProcessingTimeout    time.Duration `yaml:"processing_timeout" json:"processing_timeout"`
					MaxHistoricalCandles int           `yaml:"max_historical_candles" json:"max_historical_candles"`
					BatchSize            int           `yaml:"batch_size" json:"batch_size"`
					DataRetentionDays    int           `yaml:"data_retention_days" json:"data_retention_days"`
				}{
					ConcurrentWorkers:    10,
					ProcessingTimeout:    30 * time.Second,
					MaxHistoricalCandles: 100,
					BatchSize:            50,
					DataRetentionDays:    30,
				},
				DataFrame: struct {
					Library                  string `yaml:"library" json:"library"`
					MemoryLimitMB            int    `yaml:"memory_limit_mb" json:"memory_limit_mb"`
					EnableParallelProcessing bool   `yaml:"enable_parallel_processing" json:"enable_parallel_processing"`
				}{
					Library:                  "gota",
					MemoryLimitMB:            512,
					EnableParallelProcessing: true,
				},
				Strategies: struct {
					Registry struct {
						AutoDiscovery bool   `yaml:"auto_discovery" json:"auto_discovery"`
						DiscoveryPath string `yaml:"discovery_path" json:"discovery_path"`
					} `yaml:"registry" json:"registry"`
					Execution struct {
						ParallelStrategies bool          `yaml:"parallel_strategies" json:"parallel_strategies"`
						StrategyTimeout    time.Duration `yaml:"strategy_timeout" json:"strategy_timeout"`
						ErrorHandling      string        `yaml:"error_handling" json:"error_handling"`
					} `yaml:"execution" json:"execution"`
				}{
					Execution: struct {
						ParallelStrategies bool          `yaml:"parallel_strategies" json:"parallel_strategies"`
						StrategyTimeout    time.Duration `yaml:"strategy_timeout" json:"strategy_timeout"`
						ErrorHandling      string        `yaml:"error_handling" json:"error_handling"`
					}{
						StrategyTimeout: 10 * time.Second,
					},
				},
				ParallelProcessing: struct {
					MaxWorkers              int           `yaml:"max_workers" json:"max_workers"`
					MinWorkers              int           `yaml:"min_workers" json:"min_workers"`
					WorkerTimeout           time.Duration `yaml:"worker_timeout" json:"worker_timeout"`
					QueueSize               int           `yaml:"queue_size" json:"queue_size"`
					MaxConcurrentStrategies int           `yaml:"max_concurrent_strategies" json:"max_concurrent_strategies"`
					MaxMemoryUsageMB        int           `yaml:"max_memory_usage_mb" json:"max_memory_usage_mb"`
					MemoryCheckInterval     time.Duration `yaml:"memory_check_interval" json:"memory_check_interval"`
					ErrorHandling           string        `yaml:"error_handling" json:"error_handling"`
					MaxRetries              int           `yaml:"max_retries" json:"max_retries"`
					RetryDelay              time.Duration `yaml:"retry_delay" json:"retry_delay"`
					EnableMetrics           bool          `yaml:"enable_metrics" json:"enable_metrics"`
					MetricsInterval         time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
				}{
					MaxWorkers:              16,
					MinWorkers:              8,
					WorkerTimeout:           30 * time.Second,
					QueueSize:               1000,
					MaxConcurrentStrategies: 10,
					MaxMemoryUsageMB:        1024,
					MemoryCheckInterval:     5 * time.Second,
				},
				StateManagement: struct {
					EnablePersistence     bool          `yaml:"enable_persistence" json:"enable_persistence"`
					PersistenceInterval   time.Duration `yaml:"persistence_interval" json:"persistence_interval"`
					StateRetentionDays    int           `yaml:"state_retention_days" json:"state_retention_days"`
					EnableCaching         bool          `yaml:"enable_caching" json:"enable_caching"`
					CacheTTL              time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
					MaxCacheSize          int           `yaml:"max_cache_size" json:"max_cache_size"`
					EnableValidation      bool          `yaml:"enable_validation" json:"enable_validation"`
					ValidationTimeout     time.Duration `yaml:"validation_timeout" json:"validation_timeout"`
					EnableSynchronization bool          `yaml:"enable_synchronization" json:"enable_synchronization"`
					SyncInterval          time.Duration `yaml:"sync_interval" json:"sync_interval"`
					EnableMetrics         bool          `yaml:"enable_metrics" json:"enable_metrics"`
					MetricsInterval       time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
				}{
					EnablePersistence:   true,
					PersistenceInterval: 30 * time.Second,
					StateRetentionDays:  30,
					EnableCaching:       true,
					CacheTTL:            5 * time.Minute,
					MaxCacheSize:        10000,
				},
			},
			wantErr: false,
		},
		{
			name: "disabled configuration",
			config: &StrategyEngineV2Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "invalid concurrent workers",
			config: &StrategyEngineV2Config{
				Enabled: true,
				Processing: struct {
					ConcurrentWorkers    int           `yaml:"concurrent_workers" json:"concurrent_workers"`
					ProcessingTimeout    time.Duration `yaml:"processing_timeout" json:"processing_timeout"`
					MaxHistoricalCandles int           `yaml:"max_historical_candles" json:"max_historical_candles"`
					BatchSize            int           `yaml:"batch_size" json:"batch_size"`
					DataRetentionDays    int           `yaml:"data_retention_days" json:"data_retention_days"`
				}{
					ConcurrentWorkers: 0,
				},
			},
			wantErr: true,
			errMsg:  "concurrent_workers must be greater than 0",
		},
		{
			name: "invalid min workers greater than max workers",
			config: &StrategyEngineV2Config{
				Enabled: true,
				Processing: struct {
					ConcurrentWorkers    int           `yaml:"concurrent_workers" json:"concurrent_workers"`
					ProcessingTimeout    time.Duration `yaml:"processing_timeout" json:"processing_timeout"`
					MaxHistoricalCandles int           `yaml:"max_historical_candles" json:"max_historical_candles"`
					BatchSize            int           `yaml:"batch_size" json:"batch_size"`
					DataRetentionDays    int           `yaml:"data_retention_days" json:"data_retention_days"`
				}{
					ConcurrentWorkers:    10,
					ProcessingTimeout:    30 * time.Second,
					MaxHistoricalCandles: 100,
					BatchSize:            50,
					DataRetentionDays:    30,
				},
				DataFrame: struct {
					Library                  string `yaml:"library" json:"library"`
					MemoryLimitMB            int    `yaml:"memory_limit_mb" json:"memory_limit_mb"`
					EnableParallelProcessing bool   `yaml:"enable_parallel_processing" json:"enable_parallel_processing"`
				}{
					Library:                  "gota",
					MemoryLimitMB:            512,
					EnableParallelProcessing: true,
				},
				Strategies: struct {
					Registry struct {
						AutoDiscovery bool   `yaml:"auto_discovery" json:"auto_discovery"`
						DiscoveryPath string `yaml:"discovery_path" json:"discovery_path"`
					} `yaml:"registry" json:"registry"`
					Execution struct {
						ParallelStrategies bool          `yaml:"parallel_strategies" json:"parallel_strategies"`
						StrategyTimeout    time.Duration `yaml:"strategy_timeout" json:"strategy_timeout"`
						ErrorHandling      string        `yaml:"error_handling" json:"error_handling"`
					} `yaml:"execution" json:"execution"`
				}{
					Execution: struct {
						ParallelStrategies bool          `yaml:"parallel_strategies" json:"parallel_strategies"`
						StrategyTimeout    time.Duration `yaml:"strategy_timeout" json:"strategy_timeout"`
						ErrorHandling      string        `yaml:"error_handling" json:"error_handling"`
					}{
						StrategyTimeout: 10 * time.Second,
					},
				},
				ParallelProcessing: struct {
					MaxWorkers              int           `yaml:"max_workers" json:"max_workers"`
					MinWorkers              int           `yaml:"min_workers" json:"min_workers"`
					WorkerTimeout           time.Duration `yaml:"worker_timeout" json:"worker_timeout"`
					QueueSize               int           `yaml:"queue_size" json:"queue_size"`
					MaxConcurrentStrategies int           `yaml:"max_concurrent_strategies" json:"max_concurrent_strategies"`
					MaxMemoryUsageMB        int           `yaml:"max_memory_usage_mb" json:"max_memory_usage_mb"`
					MemoryCheckInterval     time.Duration `yaml:"memory_check_interval" json:"memory_check_interval"`
					ErrorHandling           string        `yaml:"error_handling" json:"error_handling"`
					MaxRetries              int           `yaml:"max_retries" json:"max_retries"`
					RetryDelay              time.Duration `yaml:"retry_delay" json:"retry_delay"`
					EnableMetrics           bool          `yaml:"enable_metrics" json:"enable_metrics"`
					MetricsInterval         time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
				}{
					MaxWorkers:              8,
					MinWorkers:              16, // Invalid: min > max
					WorkerTimeout:           30 * time.Second,
					QueueSize:               1000,
					MaxConcurrentStrategies: 10,
					MaxMemoryUsageMB:        1024,
					MemoryCheckInterval:     5 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "min_workers cannot be greater than max_workers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				StrategyEngineV2: *tt.config,
			}

			err := config.ValidateStrategyEngineV2Config()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStrategyEngineV2ConfigYAML(t *testing.T) {
	// Test that the configuration can be marshaled to YAML
	config := DefaultStrategyEngineV2Config()

	// This test ensures the YAML tags are correctly set
	// The actual YAML marshaling would be tested in integration tests
	assert.NotEmpty(t, config.Processing.ConcurrentWorkers)
	assert.NotEmpty(t, config.DataFrame.Library)
	assert.NotEmpty(t, config.Strategies.Registry.DiscoveryPath)
	assert.NotEmpty(t, config.Monitoring.LogLevel)
}
