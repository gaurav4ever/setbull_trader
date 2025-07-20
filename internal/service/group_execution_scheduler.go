package service

import (
	"context"
	"setbull_trader/pkg/log"
	"time"
)

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

// OnFiveMinClose listener is called when a new 5-min candle closes
func (s *GroupExecutionScheduler) OnFiveMinClose(start, end time.Time) {
	log.Info("[Scheduler] Received 5-min candle close event from %s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	candleHHMM := start.Format("15:04")

	// EXISTING: Group execution logic for time-based entry types
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
