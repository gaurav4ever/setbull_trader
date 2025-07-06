package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/log"
)

// AlertService handles audio alerts for pattern detection
type AlertService struct {
	config     *config.BBWidthMonitoringConfig
	enabled    bool
	alertCache map[string]time.Time // Track last alert time per symbol
	cacheMutex sync.RWMutex
}

// NewAlertService creates a new alert service
func NewAlertService(cfg *config.BBWidthMonitoringConfig) *AlertService {
	if cfg == nil {
		// Default configuration if none provided
		cfg = &config.BBWidthMonitoringConfig{
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
				SoundPath:           "/assets",
				CooldownSeconds:     180, // 3 minutes
				MaxAlertsPerHour:    100,
				SymbolPronunciation: true,
			},
		}
	}

	return &AlertService{
		config:     cfg,
		enabled:    cfg.Enabled && cfg.Alert.Enabled,
		alertCache: make(map[string]time.Time),
	}
}

// PlayAlert plays an audio alert for the given alert event
func (s *AlertService) PlayAlert(alert AlertEvent) error {
	if !s.enabled {
		log.Debug("[Alert Service] Alerts are disabled")
		return nil
	}

	// Check cooldown and rate limits
	if !s.shouldAlert(alert.Symbol) {
		log.Debug("[Alert Service] Alert suppressed for %s due to cooldown/rate limit", alert.Symbol)
		return nil
	}

	log.Info("[Alert Service] Playing alert for %s: %s", alert.Symbol, alert.Message)

	// Update alert cache
	s.updateAlertCache(alert.Symbol)

	// Play the audio alert
	return s.playAudioAlert(alert)
}

// shouldAlert checks if an alert should be played based on cooldown and rate limits
func (s *AlertService) shouldAlert(symbol string) bool {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	lastAlertTime, exists := s.alertCache[symbol]
	if !exists {
		return true // First alert for this symbol
	}

	// Check cooldown period
	cooldownDuration := time.Duration(s.config.Alert.CooldownSeconds) * time.Second
	if time.Since(lastAlertTime) < cooldownDuration {
		return false
	}

	// Check hourly rate limit
	hourlyLimit := s.config.Alert.MaxAlertsPerHour
	if hourlyLimit > 0 {
		oneHourAgo := time.Now().Add(-time.Hour)
		alertCount := 0
		for _, alertTime := range s.alertCache {
			if alertTime.After(oneHourAgo) {
				alertCount++
			}
		}
		if alertCount >= hourlyLimit {
			return false
		}
	}

	return true
}

// updateAlertCache updates the alert cache with the current time
func (s *AlertService) updateAlertCache(symbol string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.alertCache[symbol] = time.Now()
}

// playAudioAlert plays the actual audio alert with comprehensive fallbacks
func (s *AlertService) playAudioAlert(alert AlertEvent) error {
	// Try audio playback first
	if err := s.tryAudioPlayback(alert); err == nil {
		return nil
	}

	// Audio failed, try system notifications
	if err := s.trySystemNotification(alert); err == nil {
		log.Info("[Alert Service] Audio failed, but system notification sent for %s", alert.Symbol)
		return nil
	}

	// System notification failed, try console fallback
	if err := s.tryConsoleFallback(alert); err == nil {
		log.Info("[Alert Service] Audio and notifications failed, but console alert sent for %s", alert.Symbol)
		return nil
	}

	// All fallbacks failed, but don't fail the alert - log and continue
	log.Warn("[Alert Service] All alert methods failed for %s: %s", alert.Symbol, alert.Message)
	return nil
}

// getSoundFile determines which sound file to play based on alert type
// This is kept for backward compatibility, but tryAudioPlayback uses getSoundFileWithFormat
func (s *AlertService) getSoundFile(alert AlertEvent) string {
	return s.getSoundFileWithFormat(alert, "wav")
}

// playAudioFile plays an audio file using system commands
func (s *AlertService) playAudioFile(soundFile string, alert AlertEvent) error {
	// Try different audio players based on the operating system
	players := []string{"afplay", "paplay", "aplay", "mpg123", "mpg321"}

	for _, player := range players {
		if err := s.tryAudioPlayer(player, soundFile, alert); err == nil {
			return nil
		}
	}

	// If no audio player works, log a warning but don't fail
	log.Warn("[Alert Service] No audio player found. Alert message: %s", alert.Message)
	return nil
}

// tryAudioPlayer attempts to play audio using a specific player
func (s *AlertService) tryAudioPlayer(player, soundFile string, alert AlertEvent) error {
	var cmd *exec.Cmd

	switch player {
	case "afplay": // macOS
		cmd = exec.Command("afplay", soundFile)
	case "paplay": // Linux (PulseAudio)
		cmd = exec.Command("paplay", soundFile)
	case "aplay": // Linux (ALSA)
		cmd = exec.Command("aplay", soundFile)
	case "mpg123", "mpg321": // MP3 players
		cmd = exec.Command(player, soundFile)
	default:
		return fmt.Errorf("unsupported audio player: %s", player)
	}

	// Set a timeout for the audio playback
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play audio with %s: %w", player, err)
	}

	log.Debug("[Alert Service] Successfully played audio with %s for %s", player, alert.Symbol)
	return nil
}

// tryAudioPlayback attempts to play audio with multiple format fallbacks
func (s *AlertService) tryAudioPlayback(alert AlertEvent) error {
	// Try multiple audio file formats for the same alert type
	audioFormats := []string{"wav", "mp3", "ogg", "aiff"}

	for _, format := range audioFormats {
		soundFile := s.getSoundFileWithFormat(alert, format)
		if soundFile == "" {
			continue
		}

		// Check if sound file exists
		if _, err := os.Stat(soundFile); os.IsNotExist(err) {
			continue
		}

		// Try to play with this format
		if err := s.playAudioFile(soundFile, alert); err == nil {
			return nil
		}
	}

	return fmt.Errorf("all audio formats failed for alert type: %s", alert.AlertType)
}

// trySystemNotification attempts to send a system notification
func (s *AlertService) trySystemNotification(alert AlertEvent) error {
	// Try different notification systems based on OS
	notifiers := []string{"osascript", "notify-send", "growlnotify"}

	for _, notifier := range notifiers {
		if err := s.tryNotificationSystem(notifier, alert); err == nil {
			return nil
		}
	}

	return fmt.Errorf("all notification systems failed")
}

// tryConsoleFallback provides a console-based fallback alert
func (s *AlertService) tryConsoleFallback(alert AlertEvent) error {
	// Print a prominent console message
	fmt.Printf("\n")
	fmt.Printf("🚨 ALERT: %s\n", alert.Message)
	fmt.Printf("   Symbol: %s\n", alert.Symbol)
	fmt.Printf("   BB Width: %.4f\n", alert.BBWidth)
	fmt.Printf("   Pattern Length: %d\n", alert.PatternLength)
	fmt.Printf("   Time: %s\n", alert.Timestamp.Format("15:04:05"))
	fmt.Printf("   Type: %s\n", alert.AlertType)
	fmt.Printf("\n")

	// Also log it prominently
	log.Info("🚨 CONSOLE ALERT: %s - %s", alert.Symbol, alert.Message)

	return nil
}

// getSoundFileWithFormat gets the sound file path with a specific format
func (s *AlertService) getSoundFileWithFormat(alert AlertEvent, format string) string {
	basePath := s.config.Alert.SoundPath
	if basePath == "" {
		basePath = "/Users/gaurav/setbull_projects/setbull_trader/assets"
	}

	var fileName string
	switch alert.AlertType {
	case "bb_range_contracting":
		fileName = fmt.Sprintf("bb_range_alert.%s", format)
	case "bb_squeeze":
		fileName = fmt.Sprintf("bb_squeeze_alert.%s", format)
	case "bb_expansion":
		fileName = fmt.Sprintf("bb_expansion_alert.%s", format)
	default:
		fileName = fmt.Sprintf("alert.%s", format)
	}

	return filepath.Join(basePath, fileName)
}

// tryNotificationSystem attempts to send a notification using a specific system
func (s *AlertService) tryNotificationSystem(notifier string, alert AlertEvent) error {
	var cmd *exec.Cmd

	switch notifier {
	case "osascript": // macOS
		script := fmt.Sprintf(`display notification "%s" with title "BB Width Alert" subtitle "%s"`,
			alert.Message, alert.Symbol)
		cmd = exec.Command("osascript", "-e", script)
	case "notify-send": // Linux
		cmd = exec.Command("notify-send",
			"--urgency=high",
			"--icon=stock_chart",
			"BB Width Alert",
			fmt.Sprintf("%s: %s", alert.Symbol, alert.Message))
	case "growlnotify": // macOS (Growl)
		cmd = exec.Command("growlnotify",
			"--title", "BB Width Alert",
			"--message", fmt.Sprintf("%s: %s", alert.Symbol, alert.Message),
			"--priority", "2")
	default:
		return fmt.Errorf("unsupported notification system: %s", notifier)
	}

	// Set a timeout for the notification
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send notification with %s: %w", notifier, err)
	}

	log.Debug("[Alert Service] Successfully sent notification with %s for %s", notifier, alert.Symbol)
	return nil
}

// GetAlertStats returns current alert statistics
func (s *AlertService) GetAlertStats() map[string]interface{} {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	stats := map[string]interface{}{
		"enabled":             s.enabled,
		"total_symbols":       len(s.alertCache),
		"cooldown_seconds":    s.config.Alert.CooldownSeconds,
		"max_alerts_per_hour": s.config.Alert.MaxAlertsPerHour,
	}

	// Count recent alerts
	oneHourAgo := time.Now().Add(-time.Hour)
	recentAlerts := 0
	for _, alertTime := range s.alertCache {
		if alertTime.After(oneHourAgo) {
			recentAlerts++
		}
	}
	stats["recent_alerts"] = recentAlerts

	return stats
}

// ClearAlertCache clears the alert cache (useful for testing or reset)
func (s *AlertService) ClearAlertCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.alertCache = make(map[string]time.Time)
	log.Info("[Alert Service] Alert cache cleared")
}

// AlertEvent represents an alert event
type AlertEvent struct {
	Symbol           string
	BBWidth          float64
	LowestMinBBWidth float64
	PatternLength    int
	AlertType        string
	Timestamp        time.Time
	GroupID          string
	Message          string
}
