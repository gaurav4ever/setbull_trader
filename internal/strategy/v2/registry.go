package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"setbull_trader/pkg/log"

	"github.com/go-gota/gota/dataframe"
)

// StrategyRegistryV2 manages the registration and execution of V2 strategies
type StrategyRegistryV2 struct {
	strategies map[string]StrategyV2
	config     *RegistryConfig
	mu         sync.RWMutex
	metrics    *RegistryMetrics
}

// RegistryConfig contains configuration for the strategy registry
type RegistryConfig struct {
	AutoDiscovery   bool          `yaml:"auto_discovery" json:"auto_discovery"`
	DiscoveryPath   string        `yaml:"discovery_path" json:"discovery_path"`
	ParallelExec    bool          `yaml:"parallel_execution" json:"parallel_execution"`
	StrategyTimeout time.Duration `yaml:"strategy_timeout" json:"strategy_timeout"`
	ErrorHandling   string        `yaml:"error_handling" json:"error_handling"`
}

// RegistryMetrics contains metrics for the strategy registry
type RegistryMetrics struct {
	TotalStrategies     int           `json:"total_strategies"`
	ActiveStrategies    int           `json:"active_strategies"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	TotalErrors         int           `json:"total_errors"`
	LastExecutionTime   time.Time     `json:"last_execution_time"`
}

// NewStrategyRegistryV2 creates a new strategy registry
func NewStrategyRegistryV2(config *RegistryConfig) *StrategyRegistryV2 {
	return &StrategyRegistryV2{
		strategies: make(map[string]StrategyV2),
		config:     config,
		metrics:    &RegistryMetrics{},
	}
}

// Register adds a strategy to the registry
func (r *StrategyRegistryV2) Register(strategy StrategyV2) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strategy.GetName()
	if name == "" {
		return fmt.Errorf("strategy name cannot be empty")
	}

	if _, exists := r.strategies[name]; exists {
		return fmt.Errorf("strategy %s already registered", name)
	}

	// Validate strategy configuration
	if err := strategy.ValidateConfiguration(); err != nil {
		return fmt.Errorf("invalid configuration for strategy %s: %w", name, err)
	}

	r.strategies[name] = strategy
	r.metrics.TotalStrategies++

	if strategy.IsEnabled() {
		r.metrics.ActiveStrategies++
	}

	log.Info("Strategy %s (v%s) registered successfully", name, strategy.GetVersion())
	return nil
}

// Unregister removes a strategy from the registry
func (r *StrategyRegistryV2) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.strategies[name]; !exists {
		return fmt.Errorf("strategy %s not found", name)
	}

	strategy := r.strategies[name]
	if strategy.IsEnabled() {
		r.metrics.ActiveStrategies--
	}

	delete(r.strategies, name)
	r.metrics.TotalStrategies--

	log.Info("Strategy %s unregistered successfully", name)
	return nil
}

// GetStrategy retrieves a strategy by name
func (r *StrategyRegistryV2) GetStrategy(name string) (StrategyV2, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	strategy, exists := r.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", name)
	}

	return strategy, nil
}

// ListStrategies returns all registered strategies
func (r *StrategyRegistryV2) ListStrategies() []StrategyV2 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	strategies := make([]StrategyV2, 0, len(r.strategies))
	for _, strategy := range r.strategies {
		strategies = append(strategies, strategy)
	}

	return strategies
}

// ListActiveStrategies returns only enabled strategies
func (r *StrategyRegistryV2) ListActiveStrategies() []StrategyV2 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	strategies := make([]StrategyV2, 0)
	for _, strategy := range r.strategies {
		if strategy.IsEnabled() {
			strategies = append(strategies, strategy)
		}
	}

	return strategies
}

// ProcessAll executes all active strategies on the given DataFrame
func (r *StrategyRegistryV2) ProcessAll(ctx context.Context, df *dataframe.DataFrame) (map[string]*StrategyResult, error) {
	startTime := time.Now()
	r.metrics.LastExecutionTime = startTime

	activeStrategies := r.ListActiveStrategies()
	if len(activeStrategies) == 0 {
		log.Info("No active strategies found")
		return make(map[string]*StrategyResult), nil
	}

	log.Info("Processing %d active strategies", len(activeStrategies))

	var results map[string]*StrategyResult
	var err error

	if r.config.ParallelExec {
		results, err = r.processParallel(ctx, activeStrategies, df)
	} else {
		results, err = r.processSequential(ctx, activeStrategies, df)
	}

	if err != nil {
		r.metrics.TotalErrors++
		return results, err
	}

	r.metrics.TotalProcessingTime += time.Since(startTime)
	return results, nil
}

// processParallel executes strategies in parallel
func (r *StrategyRegistryV2) processParallel(ctx context.Context, strategies []StrategyV2, df *dataframe.DataFrame) (map[string]*StrategyResult, error) {
	results := make(map[string]*StrategyResult)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, strategy := range strategies {
		wg.Add(1)
		go func(s StrategyV2) {
			defer wg.Done()

			result := r.executeStrategy(ctx, s, df)

			mu.Lock()
			results[s.GetName()] = result
			if result.Error != nil {
				errors = append(errors, result.Error)
			}
			mu.Unlock()
		}(strategy)
	}

	wg.Wait()

	// Handle errors based on configuration
	if len(errors) > 0 {
		if r.config.ErrorHandling == "fail_fast" {
			return results, fmt.Errorf("strategy execution failed: %v", errors[0])
		}
		// For "continue_on_error", we log errors but don't fail
		for _, err := range errors {
			log.Error("Strategy execution error: %v", err)
		}
	}

	return results, nil
}

// processSequential executes strategies sequentially
func (r *StrategyRegistryV2) processSequential(ctx context.Context, strategies []StrategyV2, df *dataframe.DataFrame) (map[string]*StrategyResult, error) {
	results := make(map[string]*StrategyResult)

	for _, strategy := range strategies {
		result := r.executeStrategy(ctx, strategy, df)
		results[strategy.GetName()] = result

		if result.Error != nil {
			if r.config.ErrorHandling == "fail_fast" {
				return results, result.Error
			}
			log.Error("Strategy %s execution error: %v", strategy.GetName(), result.Error)
		}
	}

	return results, nil
}

// executeStrategy executes a single strategy with timeout
func (r *StrategyRegistryV2) executeStrategy(ctx context.Context, strategy StrategyV2, df *dataframe.DataFrame) *StrategyResult {
	startTime := time.Now()
	result := &StrategyResult{
		StrategyName:  strategy.GetName(),
		RowsProcessed: df.Nrow(),
		ColumnsAdded:  make([]string, 0),
	}

	// Create a context with timeout
	timeout := r.config.StrategyTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute strategy in a goroutine to handle timeout
	done := make(chan error, 1)
	go func() {
		processedDF, err := strategy.Process(df)
		if err != nil {
			done <- err
			return
		}

		// Calculate columns added
		originalCols := df.Names()
		newCols := processedDF.Names()
		for _, col := range newCols {
			found := false
			for _, origCol := range originalCols {
				if col == origCol {
					found = true
					break
				}
			}
			if !found {
				result.ColumnsAdded = append(result.ColumnsAdded, col)
			}
		}

		done <- nil
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		if err != nil {
			result.Error = err
		}
	case <-execCtx.Done():
		result.Error = fmt.Errorf("strategy execution timeout after %v", timeout)
	}

	result.ProcessingTime = time.Since(startTime)

	// Estimate memory usage (rough calculation)
	result.MemoryUsed = strategy.GetMemoryRequirements()

	return result
}

// GetMetrics returns the registry metrics
func (r *StrategyRegistryV2) GetMetrics() *RegistryMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *r.metrics
	return &metrics
}

// ResetMetrics resets the registry metrics
func (r *StrategyRegistryV2) ResetMetrics() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metrics = &RegistryMetrics{
		TotalStrategies:  r.metrics.TotalStrategies,
		ActiveStrategies: r.metrics.ActiveStrategies,
	}
}
