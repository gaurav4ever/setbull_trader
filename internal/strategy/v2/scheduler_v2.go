package v2

import (
	"context"
	"fmt"
	"time"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository/postgres"
	"setbull_trader/internal/service"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
)

// GroupExecutionSchedulerV2 enhances the existing scheduler with V2 strategy processing
type GroupExecutionSchedulerV2 struct {
	// Existing V1 components
	groupExecutionService *service.GroupExecutionService
	stockGroupService     *service.StockGroupService
	universeService       *service.StockUniverseService
	bbWidthMonitorService *service.BBWidthMonitorService

	// New V2 components
	strategyEngine   *StrategyEngineV2
	candleRepository *postgres.CandleRepository
	config           *config.StrategyEngineV2Config
	metrics          *SchedulerMetrics
}

// SchedulerMetrics contains metrics for the V2 scheduler
type SchedulerMetrics struct {
	TotalV1Executions       int64         `json:"total_v1_executions"`
	TotalV2Executions       int64         `json:"total_v2_executions"`
	TotalV1Errors           int64         `json:"total_v1_errors"`
	TotalV2Errors           int64         `json:"total_v2_errors"`
	AverageV1ProcessingTime time.Duration `json:"average_v1_processing_time"`
	AverageV2ProcessingTime time.Duration `json:"average_v2_processing_time"`
	LastV1ExecutionTime     time.Time     `json:"last_v1_execution_time"`
	LastV2ExecutionTime     time.Time     `json:"last_v2_execution_time"`
}

// NewGroupExecutionSchedulerV2 creates a new enhanced scheduler
func NewGroupExecutionSchedulerV2(
	groupExecutionService *service.GroupExecutionService,
	stockGroupService *service.StockGroupService,
	universeService *service.StockUniverseService,
	bbWidthMonitorService *service.BBWidthMonitorService,
	strategyEngine *StrategyEngineV2,
	candleRepository *postgres.CandleRepository,
	config *config.StrategyEngineV2Config,
) *GroupExecutionSchedulerV2 {
	return &GroupExecutionSchedulerV2{
		groupExecutionService: groupExecutionService,
		stockGroupService:     stockGroupService,
		universeService:       universeService,
		bbWidthMonitorService: bbWidthMonitorService,
		strategyEngine:        strategyEngine,
		candleRepository:      candleRepository,
		config:                config,
		metrics:               &SchedulerMetrics{},
	}
}

// OnFiveMinClose handles 5-minute candle close events with both V1 and V2 processing
func (s *GroupExecutionSchedulerV2) OnFiveMinClose(start, end time.Time) {
	log.Info("[SchedulerV2] Received 5-min candle close event from %s to %s",
		start.Format(time.RFC3339), end.Format(time.RFC3339))

	// V1: Time-based triggers (maintain backward compatibility)
	s.handleV1Triggers(start, end)

	// V2: Strategy engine processing
	if s.config.Enabled {
		s.handleV2Processing(start, end)
	}
}

// handleV1Triggers processes V1 time-based triggers
func (s *GroupExecutionSchedulerV2) handleV1Triggers(start, end time.Time) {
	v1StartTime := time.Now()
	s.metrics.LastV1ExecutionTime = v1StartTime
	s.metrics.TotalV1Executions++

	log.Info("[SchedulerV2] Processing V1 triggers")

	candleHHMM := start.Format("15:04")

	// Process time-based entry types
	for entryType, triggerTime := range service.EntryTypeTriggerTimes {
		if triggerTime != "" && candleHHMM == triggerTime {
			log.Info("[SchedulerV2] Triggering V1 group execution for entry type %s at %s", entryType, triggerTime)

			groups, err := s.stockGroupService.GetGroupsByEntryType(context.Background(), entryType, s.universeService)
			if err != nil {
				log.Error("[SchedulerV2] V1: Failed to fetch groups for entryType=%s: %v", entryType, err)
				s.metrics.TotalV1Errors++
				continue
			}

			for _, group := range groups {
				log.Info("[SchedulerV2] V1: Executing group %s for entryType=%s", group.ID, entryType)
				err = s.groupExecutionService.ExecuteDetailedGroup(context.Background(), group, start, end)
				if err != nil {
					log.Error("[SchedulerV2] V1: Group execution failed for group %s: %v", group.ID, err)
					s.metrics.TotalV1Errors++
				}
			}
		}
	}

	// Process BB width monitoring
	if s.bbWidthMonitorService != nil {
		log.Info("[SchedulerV2] V1: Triggering BB width monitoring")
		err := s.bbWidthMonitorService.MonitorBBRangeGroups(context.Background(), start, end)
		if err != nil {
			log.Error("[SchedulerV2] V1: BB width monitoring failed: %v", err)
			s.metrics.TotalV1Errors++
		}
	}

	// Update V1 metrics
	v1ProcessingTime := time.Since(v1StartTime)
	s.metrics.AverageV1ProcessingTime = (s.metrics.AverageV1ProcessingTime + v1ProcessingTime) / 2

	log.Info("[SchedulerV2] V1 processing completed in %v", v1ProcessingTime)
}

// handleV2Processing processes V2 strategy engine
func (s *GroupExecutionSchedulerV2) handleV2Processing(start, end time.Time) {
	v2StartTime := time.Now()
	s.metrics.LastV2ExecutionTime = v2StartTime
	s.metrics.TotalV2Executions++

	log.Info("[SchedulerV2] Processing V2 strategy engine")

	ctx := context.Background()

	// Get all stock groups by entry type (we'll get all entry types)
	allGroups := []domain.StockGroup{}
	entryTypes := []string{"1ST_ENTRY", "2_30_ENTRY", "BB_RANGE"} // Add all entry types as needed

	for _, entryType := range entryTypes {
		groupResponses, err := s.stockGroupService.GetGroupsByEntryType(ctx, entryType, s.universeService)
		if err != nil {
			log.Error("[SchedulerV2] V2: Failed to fetch groups for entry type %s: %v", entryType, err)
			continue
		}

		// Convert dto.StockGroupResponse to domain.StockGroup
		for _, groupResp := range groupResponses {
			domainGroup := domain.StockGroup{
				ID:        groupResp.ID,
				EntryType: groupResp.EntryType,
			}

			// Convert stocks
			for _, stockResp := range groupResp.Stocks {
				domainStock := domain.StockGroupStock{
					StockID: stockResp.StockID,
				}
				domainGroup.Stocks = append(domainGroup.Stocks, domainStock)
			}

			allGroups = append(allGroups, domainGroup)
		}
	}

	if len(allGroups) == 0 {
		log.Info("[SchedulerV2] V2: No stock groups found")
		return
	}

	log.Info("[SchedulerV2] V2: Processing %d stock groups", len(allGroups))

	// Process with V2 strategy engine
	results, err := s.strategyEngine.ProcessStockGroups(ctx, allGroups, end)
	if err != nil {
		log.Error("[SchedulerV2] V2: Strategy processing failed: %v", err)
		s.metrics.TotalV2Errors++
		return
	}

	// Process results (persist to database, trigger alerts, etc.)
	err = s.processV2Results(ctx, results, end)
	if err != nil {
		log.Error("[SchedulerV2] V2: Failed to process results: %v", err)
		s.metrics.TotalV2Errors++
	}

	// Update V2 metrics
	v2ProcessingTime := time.Since(v2StartTime)
	s.metrics.AverageV2ProcessingTime = (s.metrics.AverageV2ProcessingTime + v2ProcessingTime) / 2

	log.Info("[SchedulerV2] V2 processing completed in %v", v2ProcessingTime)
}

// processV2Results processes the results from V2 strategy execution
func (s *GroupExecutionSchedulerV2) processV2Results(
	ctx context.Context,
	results map[string]map[string]*StrategyResult,
	currentTime time.Time,
) error {
	log.Info("[SchedulerV2] Processing V2 results for %d stock groups", len(results))

	totalResults := 0
	totalErrors := 0

	for groupID, groupResults := range results {
		log.Info("[SchedulerV2] Processing results for group %s: %d strategy results", groupID, len(groupResults))

		for resultKey, result := range groupResults {
			totalResults++

			if result.Error != nil {
				totalErrors++
				log.Error("[SchedulerV2] Strategy result error for %s: %v", resultKey, result.Error)
				continue
			}

			// Log successful results
			log.Info("[SchedulerV2] Strategy %s processed %d rows, added %d columns in %v",
				result.StrategyName, result.RowsProcessed, len(result.ColumnsAdded), result.ProcessingTime)

			// TODO: Implement result persistence
			// - Store strategy results to database
			// - Trigger alerts based on strategy outputs
			// - Update real-time dashboards
			// - Generate trading signals
		}
	}

	log.Info("[SchedulerV2] V2 results processing completed: %d total results, %d errors", totalResults, totalErrors)
	return nil
}

// RegisterV2Strategy registers a strategy with the V2 engine
func (s *GroupExecutionSchedulerV2) RegisterV2Strategy(strategy StrategyV2) error {
	if s.strategyEngine == nil {
		return fmt.Errorf("V2 strategy engine not initialized")
	}
	return s.strategyEngine.RegisterStrategy(strategy)
}

// GetMetrics returns the scheduler metrics
func (s *GroupExecutionSchedulerV2) GetMetrics() *SchedulerMetrics {
	return s.metrics
}

// GetV2EngineMetrics returns the V2 strategy engine metrics
func (s *GroupExecutionSchedulerV2) GetV2EngineMetrics() *EngineMetrics {
	if s.strategyEngine == nil {
		return nil
	}
	return s.strategyEngine.GetMetrics()
}

// GetV2RegistryMetrics returns the V2 strategy registry metrics
func (s *GroupExecutionSchedulerV2) GetV2RegistryMetrics() *RegistryMetrics {
	if s.strategyEngine == nil {
		return nil
	}
	return s.strategyEngine.GetRegistry().GetMetrics()
}

// IsV2Enabled returns whether V2 processing is enabled
func (s *GroupExecutionSchedulerV2) IsV2Enabled() bool {
	return s.config != nil && s.config.Enabled
}

// GetV2Config returns the V2 configuration
func (s *GroupExecutionSchedulerV2) GetV2Config() *config.StrategyEngineV2Config {
	return s.config
}
