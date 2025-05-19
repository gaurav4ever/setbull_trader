package service

import (
	"context"
	"setbull_trader/internal/domain"
	"setbull_trader/pkg/log"
	"time"
)

// EntryTypeTriggerTimes maps entry types to their trigger times (in HH:MM, 24h format)
var EntryTypeTriggerTimes = map[string]string{
	"1ST_ENTRY":  "09:20",
	"2_30_ENTRY": "13:05",
	// Add more entry types and times as needed
}

// GroupExecutionScheduler listens for candle close events and triggers group execution
// at the correct times for each entry type.
type GroupExecutionScheduler struct {
	candleAggService      *CandleAggregationService
	groupExecutionService *GroupExecutionService
	stockGroupService     *StockGroupService
	universeService       *StockUniverseService
}

// NewGroupExecutionScheduler creates and registers the scheduler
func NewGroupExecutionScheduler(
	candleAggService *CandleAggregationService,
	groupExecutionService *GroupExecutionService,
	stockGroupService *StockGroupService,
	universeService *StockUniverseService,
) *GroupExecutionScheduler {
	s := &GroupExecutionScheduler{
		candleAggService:      candleAggService,
		groupExecutionService: groupExecutionService,
		stockGroupService:     stockGroupService,
		universeService:       universeService,
	}
	// Register as a listener for 5-min candle close events
	candleAggService.RegisterCandleCloseListener(s.OnCandleClose)
	return s
}

// OnCandleClose listener is called when a new 5-min candle closes
func (s *GroupExecutionScheduler) OnCandleClose(candles []domain.AggregatedCandle) {
	for _, candle := range candles {
		candleTime := candle.Timestamp.In(time.FixedZone("IST", 5*3600+1800))
		candleHHMM := candleTime.Format("15:04")
		// This is still 1min candle close. Not the 5min candle close.
		for entryType, triggerTime := range EntryTypeTriggerTimes {
			log.Info("[Scheduler] Checking if %s matches %s for candle %+v", candleHHMM, triggerTime, candle)
			if candleHHMM == triggerTime {
				log.Info("[Scheduler] Triggering group execution for entry type %s at %s (candle: %+v)", entryType, triggerTime, candle)
				s.TriggerGroupExecution(context.Background(), entryType, candle, candleHHMM)
			}
		}
	}
}

// TriggerGroupExecution triggers group execution for all groups with the given entry type and candle
func (s *GroupExecutionScheduler) TriggerGroupExecution(
	ctx context.Context,
	entryType string,
	candle domain.AggregatedCandle,
	candleTime string,
) {
	groups, err := s.stockGroupService.GetGroupsByEntryType(ctx, entryType, s.universeService)
	if err != nil {
		log.Error("[Scheduler] Failed to fetch groups for entryType=%s: %v", entryType, err)
		return
	}
	if len(groups) == 0 {
		log.Info("[Scheduler] No groups found for entryType=%s at candle %v", entryType, candle.Timestamp)
		return
	}
	for _, group := range groups {
		log.Info("[Scheduler] Executing group %s for entryType=%s at candle %v", group.ID, entryType, candle.Timestamp)
		err := s.groupExecutionService.ExecuteGroupWithCandle(ctx, group, candle, candleTime)
		if err != nil {
			log.Error("[Scheduler] Group execution failed for group %s: %v", group.ID, err)
		} else {
			log.Info("[Scheduler] Group execution succeeded for group %s", group.ID)
		}
	}
}
