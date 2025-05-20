package service

import (
	"testing"
	"time"
)

func TestIsFiveMinBoundarySinceMarketOpen(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		expected bool
	}{
		{"Before market open", "2025-05-19T09:14:00+05:30", false},
		{"At market open", "2025-05-19T09:15:00+05:30", true},
		{"First 5-min boundary", "2025-05-19T09:20:00+05:30", true},
		{"Not a 5-min boundary", "2025-05-19T09:18:00+05:30", false},
		{"Second 5-min boundary", "2025-05-19T09:25:00+05:30", true},
		{"After market open, not boundary", "2025-05-19T09:22:00+05:30", false},
		{"Far after open, boundary", "2025-05-19T15:15:00+05:30", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := time.Parse(time.RFC3339, tt.timeStr)
			if err != nil {
				t.Fatalf("failed to parse time: %v", err)
			}
			got := isFiveMinBoundarySinceMarketOpen(tm)
			if got != tt.expected {
				t.Errorf("isFiveMinBoundarySinceMarketOpen(%s) = %v, want %v", tt.timeStr, got, tt.expected)
			}
		})
	}
}
