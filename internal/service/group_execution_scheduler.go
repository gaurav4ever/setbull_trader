package service

import (
	"context"
	dto "setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
	"time"
)

// V2StrategyEngine interface to break import cycle
type V2StrategyEngine interface {
	ProcessStockGroups(ctx context.Context, stockGroups []domain.StockGroup, currentTime time.Time) (map[string]map[string]interface{}, error)
	GetMetrics() interface{}
}

// EntryTypeTriggerTimes maps entry types to their trigger times (in HH:MM, 24h format)
var EntryTypeTriggerTimes = map[string]string{
	"1ST_ENTRY":  "09:15",
	"2_30_ENTRY": "13:00",
	// BB_RANGE entry type: Monitor for contracting pattern within lowest_min_bb_width_range
	// No specific trigger time - monitored continuously during market hours
	"BB_RANGE": "", // Empty string indicates continuous monitoring, not time-based triggers
}

// GroupExecutionScheduler listens for candle close events and triggers group execution
// at the correct times for each entry type.
type GroupExecutionScheduler struct {
	groupExecutionService *GroupExecutionService
	stockGroupService     *StockGroupService
	universeService       *StockUniverseService
	// NEW: Add BB width monitoring service
	bbWidthMonitorService *BBWidthMonitorService
	// V2 Strategy Engine
	v2EngineEnabled bool
	v2Engine        V2StrategyEngine
}

// NewGroupExecutionScheduler creates and registers the scheduler
func NewGroupExecutionScheduler(
	groupExecutionService *GroupExecutionService,
	stockGroupService *StockGroupService,
	universeService *StockUniverseService,
	bbWidthMonitorService *BBWidthMonitorService, // NEW: Add BB width monitoring service
) *GroupExecutionScheduler {
	s := &GroupExecutionScheduler{
		groupExecutionService: groupExecutionService,
		stockGroupService:     stockGroupService,
		universeService:       universeService,
		bbWidthMonitorService: bbWidthMonitorService, // NEW: Add BB width monitoring service
	}
	// Register as a listener for 5-min candle close events
	stockGroupService.RegisterFiveMinCloseListener(s.OnFiveMinClose)
	return s
}

// SetV2Engine sets the V2 strategy engine
func (s *GroupExecutionScheduler) SetV2Engine(v2EngineEnabled bool) {
	s.v2EngineEnabled = v2EngineEnabled
	log.Info("[Scheduler] V2 Strategy Engine enabled: %t", s.v2EngineEnabled)
}

// SetV2EngineInstance sets the V2 strategy engine instance
func (s *GroupExecutionScheduler) SetV2EngineInstance(engine V2StrategyEngine) {
	s.v2Engine = engine
	log.Info("[Scheduler] V2 Strategy Engine instance set")
}

// app.go event comes here as a listener.
// OnFiveMinClose listener is called when a new 5-min candle closes
func (s *GroupExecutionScheduler) OnFiveMinClose(start, end time.Time) {
	log.Info("[Scheduler] Received 5-min candle close event from %s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	candleHHMM := start.Format("15:04")
	// V2 Strategy Engine processing
	if s.v2EngineEnabled {
		log.Info("[Scheduler] Triggering V2 Strategy Engine processing (candle: %+v)", start)
		s.processV2Strategies(start, end)
	} else {
		// V1 Strategy Engine processing
		for entryType, triggerTime := range EntryTypeTriggerTimes {
			if triggerTime != "" && candleHHMM == triggerTime {
				log.Info("[Scheduler] Triggering group execution for entry type %s at %s (candle: %+v)", entryType, triggerTime, start)
				s.TriggerGroupExecution(context.Background(), entryType, start, end)
			}
		}

		// NEW: BB width monitoring for BB_RANGE groups (continuous monitoring during market hours)
		if s.bbWidthMonitorService != nil {
			log.Info("[Scheduler] Triggering BB width monitoring for BB_RANGE groups (candle: %+v)", start)
			err := s.bbWidthMonitorService.MonitorBBRangeGroups(context.Background(), start, end)
			if err != nil {
				log.Error("[Scheduler] BB width monitoring failed: %v", err)
			}
		}
	}
}

// Helper function to check if a given time is a 5-min boundary since market open (9:15)
func isFiveMinBoundarySinceMarketOpen(t time.Time) bool {
	marketOpenHour := 9
	marketOpenMinute := 15
	if t.Hour() < marketOpenHour || (t.Hour() == marketOpenHour && t.Minute() < marketOpenMinute) {
		return false
	}
	minutesSinceOpen := (t.Hour()-marketOpenHour)*60 + (t.Minute() - marketOpenMinute)
	return minutesSinceOpen >= 0 && minutesSinceOpen%5 == 0
}

// TriggerGroupExecution triggers group execution for all groups with the given entry type and candle
func (s *GroupExecutionScheduler) TriggerGroupExecution(
	ctx context.Context,
	entryType string,
	start, end time.Time,
) {
	groups, err := s.stockGroupService.GetGroupsByEntryType(ctx, entryType, s.universeService)
	if err != nil {
		log.Error("[Scheduler] Failed to fetch group for entryType=%s: %v", entryType, err)
		return
	}
	if len(groups) == 0 {
		log.Info("[Scheduler] No groups found for entryType=%s", entryType)
		return
	}
	for _, group := range groups {
		log.Info("[Scheduler] Executing group %s for entryType=%s", group.ID, entryType)
		err = s.groupExecutionService.ExecuteDetailedGroup(ctx, group, start, end)
		if err != nil {
			log.Error("[Scheduler] Group execution failed for group %s: %v", group.ID, err)
		}
	}
}

// processV2Strategies processes stock groups using the V2 strategy engine
func (s *GroupExecutionScheduler) processV2Strategies(start, end time.Time) {
	ctx := context.Background()

	// Get active stock groups for all entry types
	var allStockGroups []domain.StockGroup

	// Get groups for each entry type
	for entryType := range EntryTypeTriggerTimes {
		stockGroupResponses, err := s.stockGroupService.GetGroupsByEntryType(ctx, entryType, s.universeService)
		if err != nil {
			log.Error("[Scheduler] Failed to get stock groups for entry type %s: %v", entryType, err)
			continue
		}

		// Convert DTO responses to domain models
		for _, response := range stockGroupResponses {
			domainGroup := s.convertResponseToDomainGroup(response)
			allStockGroups = append(allStockGroups, domainGroup)
		}
	}

	if len(allStockGroups) == 0 {
		log.Debug("[Scheduler] No active stock groups for V2 processing")
		return
	}

	log.Info("[Scheduler] V2 Strategy Engine processing %d stock groups (candle: %+v)",
		len(allStockGroups), start)

	// Call the V2 engine's ProcessStockGroups method
	if s.v2Engine != nil {
		results, err := s.v2Engine.ProcessStockGroups(ctx, allStockGroups, start)
		if err != nil {
			log.Error("[Scheduler] V2 Strategy Engine processing failed: %v", err)
			return
		}

		log.Info("[Scheduler] V2 Strategy Engine completed processing with %d result groups", len(results))

		// Log metrics if available
		metrics := s.v2Engine.GetMetrics()
		if metrics != nil {
			log.Info("[Scheduler] V2 Engine Metrics available: %+v", metrics)
		}
	} else {
		log.Error("[Scheduler] V2 Strategy Engine is not initialized")
	}
}

// convertResponseToDomainGroup converts a StockGroupResponse DTO to a domain.StockGroup
func (s *GroupExecutionScheduler) convertResponseToDomainGroup(response dto.StockGroupResponse) domain.StockGroup {
	// Parse timestamps
	createdAt, _ := time.Parse(time.RFC3339, response.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, response.UpdatedAt)

	// Parse status
	status := domain.StockGroupStatus(response.Status)

	// Convert stocks
	var stocks []domain.StockGroupStock
	for _, stockDTO := range response.Stocks {
		stock := domain.StockGroupStock{
			ID:      "", // This will be set by the database
			GroupID: response.ID,
			StockID: stockDTO.StockID,
		}
		stocks = append(stocks, stock)
	}

	return domain.StockGroup{
		ID:        response.ID,
		EntryType: response.EntryType,
		Status:    status,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Stocks:    stocks,
	}
}
