package service

import (
	"time"
)

// TradingCalendarService provides methods for working with trading days
type TradingCalendarService struct {
	excludeWeekends bool
}

// NewTradingCalendarService creates a new trading calendar service
func NewTradingCalendarService(excludeWeekends bool) *TradingCalendarService {
	return &TradingCalendarService{
		excludeWeekends: excludeWeekends,
	}
}

// IsTradingDay checks if the given date is a trading day
func (s *TradingCalendarService) IsTradingDay(date time.Time) bool {
	if !s.excludeWeekends {
		return true
	}

	// Check if it's a weekend (Saturday = 6, Sunday = 0)
	weekday := date.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

// NextTradingDay returns the next trading day after the given date
func (s *TradingCalendarService) NextTradingDay(date time.Time) time.Time {
	if !s.excludeWeekends {
		return date.AddDate(0, 0, 1)
	}

	next := date.AddDate(0, 0, 1)

	// Skip to Monday if it's a weekend
	for !s.IsTradingDay(next) {
		next = next.AddDate(0, 0, 1)
	}

	return next
}

// PreviousTradingDay returns the previous trading day before the given date
func (s *TradingCalendarService) PreviousTradingDay(date time.Time) time.Time {
	if !s.excludeWeekends {
		return date.AddDate(0, 0, -1)
	}

	prev := date

	// If the current day is not a trading day, find the previous trading day
	if !s.IsTradingDay(prev) {
		for !s.IsTradingDay(prev) {
			prev = prev.AddDate(0, 0, -1)
		}
		return prev
	}

	// Otherwise, just go back one day and check again
	prev = date.AddDate(0, 0, -1)
	for !s.IsTradingDay(prev) {
		prev = prev.AddDate(0, 0, -1)
	}

	return prev
}

// AddTradingDays adds the specified number of trading days to the given date
func (s *TradingCalendarService) AddTradingDays(date time.Time, days int) time.Time {
	if !s.excludeWeekends {
		return date.AddDate(0, 0, days)
	}

	result := date
	for i := 0; i < days; i++ {
		result = s.NextTradingDay(result)
	}

	return result
}

// SubtractTradingDays subtracts the specified number of trading days from the given date
func (s *TradingCalendarService) SubtractTradingDays(date time.Time, days int) time.Time {
	if !s.excludeWeekends {
		return date.AddDate(0, 0, -days)
	}

	result := date
	for i := 0; i < days; i++ {
		result = s.PreviousTradingDay(result)
	}

	return result
}

// GetTradingDaysBetween returns the number of trading days between two dates
func (s *TradingCalendarService) GetTradingDaysBetween(start, end time.Time) int {
	if !s.excludeWeekends {
		return int(end.Sub(start).Hours()/24) + 1
	}

	count := 0
	current := start

	for !current.After(end) {
		if s.IsTradingDay(current) {
			count++
		}
		current = current.AddDate(0, 0, 1)
	}

	return count
}
