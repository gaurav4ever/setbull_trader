package service

import (
	"testing"
	"time"

	"setbull_trader/internal/trading/config"
)

func TestNewAlertService(t *testing.T) {
	// Test with nil config (should use defaults)
	service := NewAlertService(nil)
	if service == nil {
		t.Fatal("Expected AlertService to be created")
	}
	if !service.enabled {
		t.Error("Expected service to be enabled by default")
	}

	// Test with custom config
	cfg := &config.BBWidthMonitoringConfig{
		Enabled: true,
		Alert: struct {
			Enabled             bool    `yaml:"enabled" json:"enabled"`
			Volume              float64 `yaml:"volume" json:"volume"`
			SoundPath           string  `yaml:"sound_path" json:"sound_path"`
			CooldownSeconds     int     `yaml:"cooldown_seconds" json:"cooldown_seconds"`
			MaxAlertsPerHour    int     `yaml:"max_alerts_per_hour" json:"max_alerts_per_hour"`
			SymbolPronunciation bool    `yaml:"symbol_pronunciation" json:"symbol_pronunciation"`
		}{
			Enabled:             true,
			Volume:              0.8,
			SoundPath:           "/test/path",
			CooldownSeconds:     300,
			MaxAlertsPerHour:    50,
			SymbolPronunciation: true,
		},
	}

	service = NewAlertService(cfg)
	if service == nil {
		t.Fatal("Expected AlertService to be created")
	}
	if !service.enabled {
		t.Error("Expected service to be enabled")
	}
}

func TestAlertService_ShouldAlert(t *testing.T) {
	cfg := &config.BBWidthMonitoringConfig{
		Enabled: true,
		Alert: struct {
			Enabled             bool    `yaml:"enabled" json:"enabled"`
			Volume              float64 `yaml:"volume" json:"volume"`
			SoundPath           string  `yaml:"sound_path" json:"sound_path"`
			CooldownSeconds     int     `yaml:"cooldown_seconds" json:"cooldown_seconds"`
			MaxAlertsPerHour    int     `yaml:"max_alerts_per_hour" json:"max_alerts_per_hour"`
			SymbolPronunciation bool    `yaml:"symbol_pronunciation" json:"symbol_pronunciation"`
		}{
			Enabled:             true,
			Volume:              0.8,
			SoundPath:           "/test/path",
			CooldownSeconds:     60, // 1 minute cooldown
			MaxAlertsPerHour:    10,
			SymbolPronunciation: true,
		},
	}

	service := NewAlertService(cfg)

	// Test first alert (should be allowed)
	if !service.shouldAlert("TEST") {
		t.Error("First alert should be allowed")
	}

	// Test immediate second alert (should be blocked by cooldown)
	if service.shouldAlert("TEST") {
		t.Error("Second alert should be blocked by cooldown")
	}

	// Test alert for different symbol (should be allowed)
	if !service.shouldAlert("TEST2") {
		t.Error("Alert for different symbol should be allowed")
	}
}

func TestAlertService_GetSoundFile(t *testing.T) {
	service := NewAlertService(nil)

	tests := []struct {
		name      string
		alertType string
		expected  string
	}{
		{
			name:      "BB Range Contracting",
			alertType: "bb_range_contracting",
			expected:  "/assets/bb_range_alert.wav",
		},
		{
			name:      "BB Squeeze",
			alertType: "bb_squeeze",
			expected:  "/assets/bb_squeeze_alert.wav",
		},
		{
			name:      "BB Expansion",
			alertType: "bb_expansion",
			expected:  "/assets/bb_expansion_alert.wav",
		},
		{
			name:      "Default",
			alertType: "unknown",
			expected:  "/assets/alert.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := AlertEvent{
				AlertType: tt.alertType,
			}
			result := service.getSoundFile(alert)
			if result != tt.expected {
				t.Errorf("getSoundFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAlertService_GetAlertStats(t *testing.T) {
	service := NewAlertService(nil)

	// Test initial stats
	stats := service.GetAlertStats()
	if stats["enabled"] != true {
		t.Error("Expected enabled to be true")
	}
	if stats["total_symbols"] != 0 {
		t.Error("Expected total_symbols to be 0 initially")
	}

	// Add some alerts
	service.updateAlertCache("TEST1")
	service.updateAlertCache("TEST2")

	// Test stats after adding alerts
	stats = service.GetAlertStats()
	if stats["total_symbols"] != 2 {
		t.Errorf("Expected total_symbols to be 2, got %v", stats["total_symbols"])
	}
}

func TestAlertService_ClearAlertCache(t *testing.T) {
	service := NewAlertService(nil)

	// Add some alerts
	service.updateAlertCache("TEST1")
	service.updateAlertCache("TEST2")

	// Verify alerts were added
	stats := service.GetAlertStats()
	if stats["total_symbols"] != 2 {
		t.Error("Expected 2 symbols before clearing")
	}

	// Clear cache
	service.ClearAlertCache()

	// Verify cache was cleared
	stats = service.GetAlertStats()
	if stats["total_symbols"] != 0 {
		t.Error("Expected 0 symbols after clearing")
	}
}

func TestAlertService_PlayAlert_Disabled(t *testing.T) {
	// Test with disabled service
	cfg := &config.BBWidthMonitoringConfig{
		Enabled: false,
		Alert: struct {
			Enabled             bool    `yaml:"enabled" json:"enabled"`
			Volume              float64 `yaml:"volume" json:"volume"`
			SoundPath           string  `yaml:"sound_path" json:"sound_path"`
			CooldownSeconds     int     `yaml:"cooldown_seconds" json:"cooldown_seconds"`
			MaxAlertsPerHour    int     `yaml:"max_alerts_per_hour" json:"max_alerts_per_hour"`
			SymbolPronunciation bool    `yaml:"symbol_pronunciation" json:"symbol_pronunciation"`
		}{
			Enabled: false,
		},
	}

	service := NewAlertService(cfg)
	alert := AlertEvent{
		Symbol:    "TEST",
		AlertType: "bb_range_contracting",
		Message:   "Test alert",
		Timestamp: time.Now(),
	}

	// Should not error when disabled
	err := service.PlayAlert(alert)
	if err != nil {
		t.Errorf("Expected no error when disabled, got: %v", err)
	}
}
