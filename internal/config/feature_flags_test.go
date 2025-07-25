package config

import (
	"os"
	"testing"
)

func TestDefaultFeatureFlags(t *testing.T) {
	flags := DefaultFeatureFlags()
	
	// Test safe defaults
	if flags.UseOptimizedAnalytics {
		t.Error("UseOptimizedAnalytics should default to false for safety")
	}
	
	if !flags.CacheEnabled {
		t.Error("CacheEnabled should default to true")
	}
	
	if !flags.ConcurrencyEnabled {
		t.Error("ConcurrencyEnabled should default to true")
	}
	
	if flags.RolloutPercentage != 0.0 {
		t.Errorf("RolloutPercentage should default to 0.0, got %f", flags.RolloutPercentage)
	}
	
	if !flags.EnableDetailedMetrics {
		t.Error("EnableDetailedMetrics should default to true")
	}
	
	if !flags.FallbackToV1OnError {
		t.Error("FallbackToV1OnError should default to true")
	}
	
	if flags.MaxCacheSize != 512 {
		t.Errorf("MaxCacheSize should default to 512, got %d", flags.MaxCacheSize)
	}
	
	if flags.WorkerPoolSize != 4 {
		t.Errorf("WorkerPoolSize should default to 4, got %d", flags.WorkerPoolSize)
	}
}

func TestLoadFeatureFlagsFromEnv(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"USE_OPTIMIZED_ANALYTICS":  os.Getenv("USE_OPTIMIZED_ANALYTICS"),
		"CACHE_ENABLED":           os.Getenv("CACHE_ENABLED"),
		"CONCURRENCY_ENABLED":     os.Getenv("CONCURRENCY_ENABLED"),
		"ROLLOUT_PERCENTAGE":      os.Getenv("ROLLOUT_PERCENTAGE"),
		"ENABLE_DETAILED_METRICS": os.Getenv("ENABLE_DETAILED_METRICS"),
		"FALLBACK_TO_V1_ON_ERROR": os.Getenv("FALLBACK_TO_V1_ON_ERROR"),
		"MAX_CACHE_SIZE":          os.Getenv("MAX_CACHE_SIZE"),
		"WORKER_POOL_SIZE":        os.Getenv("WORKER_POOL_SIZE"),
	}
	
	// Cleanup function
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()
	
	// Test loading from environment
	os.Setenv("USE_OPTIMIZED_ANALYTICS", "true")
	os.Setenv("CACHE_ENABLED", "false")
	os.Setenv("CONCURRENCY_ENABLED", "false")
	os.Setenv("ROLLOUT_PERCENTAGE", "25.5")
	os.Setenv("ENABLE_DETAILED_METRICS", "false")
	os.Setenv("FALLBACK_TO_V1_ON_ERROR", "false")
	os.Setenv("MAX_CACHE_SIZE", "1024")
	os.Setenv("WORKER_POOL_SIZE", "8")
	
	flags := LoadFeatureFlagsFromEnv()
	
	if !flags.UseOptimizedAnalytics {
		t.Error("UseOptimizedAnalytics should be true from env")
	}
	
	if flags.CacheEnabled {
		t.Error("CacheEnabled should be false from env")
	}
	
	if flags.ConcurrencyEnabled {
		t.Error("ConcurrencyEnabled should be false from env")
	}
	
	if flags.RolloutPercentage != 25.5 {
		t.Errorf("RolloutPercentage should be 25.5 from env, got %f", flags.RolloutPercentage)
	}
	
	if flags.EnableDetailedMetrics {
		t.Error("EnableDetailedMetrics should be false from env")
	}
	
	if flags.FallbackToV1OnError {
		t.Error("FallbackToV1OnError should be false from env")
	}
	
	if flags.MaxCacheSize != 1024 {
		t.Errorf("MaxCacheSize should be 1024 from env, got %d", flags.MaxCacheSize)
	}
	
	if flags.WorkerPoolSize != 8 {
		t.Errorf("WorkerPoolSize should be 8 from env, got %d", flags.WorkerPoolSize)
	}
}

func TestFeatureFlagsValidation(t *testing.T) {
	tests := []struct {
		name    string
		flags   *FeatureFlags
		wantErr bool
	}{
		{
			name:  "valid flags",
			flags: DefaultFeatureFlags(),
			wantErr: false,
		},
		{
			name: "invalid rollout percentage - negative",
			flags: &FeatureFlags{
				RolloutPercentage: -5.0,
				MaxCacheSize:     512,
				WorkerPoolSize:   4,
			},
			wantErr: true,
		},
		{
			name: "invalid rollout percentage - over 100",
			flags: &FeatureFlags{
				RolloutPercentage: 150.0,
				MaxCacheSize:     512,
				WorkerPoolSize:   4,
			},
			wantErr: true,
		},
		{
			name: "invalid cache size",
			flags: &FeatureFlags{
				RolloutPercentage: 50.0,
				MaxCacheSize:     -1,
				WorkerPoolSize:   4,
			},
			wantErr: true,
		},
		{
			name: "invalid worker pool size",
			flags: &FeatureFlags{
				RolloutPercentage: 50.0,
				MaxCacheSize:     512,
				WorkerPoolSize:   0,
			},
			wantErr: true,
		},
		{
			name: "100% rollout without fallback",
			flags: &FeatureFlags{
				RolloutPercentage:    100.0,
				FallbackToV1OnError:  false,
				MaxCacheSize:        512,
				WorkerPoolSize:      4,
			},
			wantErr: true,
		},
		{
			name: "100% rollout with fallback",
			flags: &FeatureFlags{
				RolloutPercentage:    100.0,
				FallbackToV1OnError:  true,
				MaxCacheSize:        512,
				WorkerPoolSize:      4,
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flags.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldUseOptimizedAnalytics(t *testing.T) {
	tests := []struct {
		name              string
		useOptimized      bool
		rolloutPercentage float64
		requestID         string
		expected          bool
	}{
		{
			name:              "disabled feature",
			useOptimized:      false,
			rolloutPercentage: 50.0,
			requestID:         "test-request",
			expected:          false,
		},
		{
			name:              "zero rollout",
			useOptimized:      true,
			rolloutPercentage: 0.0,
			requestID:         "test-request",
			expected:          false,
		},
		{
			name:              "full rollout",
			useOptimized:      true,
			rolloutPercentage: 100.0,
			requestID:         "test-request",
			expected:          true,
		},
		{
			name:              "partial rollout - deterministic",
			useOptimized:      true,
			rolloutPercentage: 50.0,
			requestID:         "consistent-id",
			expected:          false, // Update based on actual hash
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := &FeatureFlags{
				UseOptimizedAnalytics: tt.useOptimized,
				RolloutPercentage:     tt.rolloutPercentage,
			}
			
			result := flags.ShouldUseOptimizedAnalytics(tt.requestID)
			if result != tt.expected {
				t.Errorf("ShouldUseOptimizedAnalytics() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetCacheSizeBytes(t *testing.T) {
	flags := &FeatureFlags{MaxCacheSize: 256}
	
	expected := int64(256 * 1024 * 1024) // 256 MB in bytes
	actual := flags.GetCacheSizeBytes()
	
	if actual != expected {
		t.Errorf("GetCacheSizeBytes() = %d, expected %d", actual, expected)
	}
}

func TestDeploymentPhases(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{"disabled", PhaseDisabled, "DISABLED"},
		{"testing", PhaseTesting, "TESTING"},
		{"validation", PhaseValidation, "VALIDATION"},
		{"full rollout", PhaseFullRollout, "FULL_ROLLOUT"},
		{"custom", 75.0, "CUSTOM_75.0%"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := &FeatureFlags{RolloutPercentage: tt.percentage}
			result := flags.GetDeploymentPhase()
			
			if result != tt.expected {
				t.Errorf("GetDeploymentPhase() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestSetDeploymentPhase(t *testing.T) {
	flags := &FeatureFlags{}
	
	// Test valid phases
	validPhases := []float64{PhaseDisabled, PhaseTesting, PhaseValidation, PhaseFullRollout}
	for _, phase := range validPhases {
		err := flags.SetDeploymentPhase(phase)
		if err != nil {
			t.Errorf("SetDeploymentPhase(%f) should not error, got: %v", phase, err)
		}
		
		if flags.RolloutPercentage != phase {
			t.Errorf("RolloutPercentage should be %f, got %f", phase, flags.RolloutPercentage)
		}
	}
	
	// Test invalid phase
	err := flags.SetDeploymentPhase(99.9)
	if err == nil {
		t.Error("SetDeploymentPhase(99.9) should return error for invalid phase")
	}
}

func TestLogConfiguration(t *testing.T) {
	flags := DefaultFeatureFlags()
	logStr := flags.LogConfiguration()
	
	// Check that log contains key information
	expectedSubstrings := []string{
		"OptimizedAnalytics=false",
		"Cache=true",
		"Concurrency=true",
		"Rollout=0.0%",
		"DetailedMetrics=true",
		"FallbackV1=true",
		"CacheSize=512MB",
		"Workers=4",
	}
	
	for _, substr := range expectedSubstrings {
		if !contains(logStr, substr) {
			t.Errorf("LogConfiguration() should contain '%s', got: %s", substr, logStr)
		}
	}
}

// Helper function since strings.Contains might not be available in test env
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      contains(s[1:], substr))))
}

// Benchmark tests
func BenchmarkShouldUseOptimizedAnalytics(b *testing.B) {
	flags := &FeatureFlags{
		UseOptimizedAnalytics: true,
		RolloutPercentage:     50.0,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		requestID := "benchmark-request-" + string(rune(i))
		flags.ShouldUseOptimizedAnalytics(requestID)
	}
}

func BenchmarkValidate(b *testing.B) {
	flags := DefaultFeatureFlags()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flags.Validate()
	}
}
