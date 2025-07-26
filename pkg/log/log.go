package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string `json:"level" yaml:"level"`
	LogDir     string `json:"log_dir" yaml:"log_dir"`
	MaxSize    int    `json:"max_size" yaml:"max_size"`       // MB
	MaxBackups int    `json:"max_backups" yaml:"max_backups"` // Number of backup files
	MaxAge     int    `json:"max_age" yaml:"max_age"`         // Days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      "info",
		LogDir:     "logs",
		MaxSize:    100, // 100MB
		MaxBackups: 30,  // 30 backup files
		MaxAge:     30,  // 30 days
		Compress:   true,
	}
}

// InitLogger initializes the logger with default configuration
func InitLogger() {
	InitLoggerWithConfig(DefaultLogConfig())
}

// InitLoggerWithConfig initializes the logger with custom configuration
func InitLoggerWithConfig(config *LogConfig) {
	logger = logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		// Fallback to stdout
		logger.SetOutput(os.Stdout)
	} else {
		// Set up daily log file
		logFile := getDailyLogFile(config.LogDir)
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
			// Fallback to stdout
			logger.SetOutput(os.Stdout)
		} else {
			// Use both file and stdout for development
			logger.SetOutput(file)
		}
	}

	// Set JSON formatter with timestamp
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Log initialization
	logger.WithFields(logrus.Fields{
		"component": "logger",
		"log_dir":   config.LogDir,
		"level":     config.Level,
	}).Info("Logger initialized successfully")
}

// getDailyLogFile returns the log file path for the current day
func getDailyLogFile(logDir string) string {
	today := time.Now().Format("2006-01-02")
	return filepath.Join(logDir, fmt.Sprintf("setbull_trader_%s.log", today))
}

// Info logs an info message
func Info(msg string, args ...interface{}) {
	if logger != nil {
		logger.Infof(msg, args...)
	}
}

// Error logs an error message
func Error(msg string, args ...interface{}) {
	if logger != nil {
		logger.Errorf(msg, args...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(msg string, args ...interface{}) {
	if logger != nil {
		logger.Fatalf(msg, args...)
	}
}

// Fatalf logs a fatal message with format and exits
func Fatalf(format string, args ...interface{}) {
	if logger != nil {
		logger.Fatalf(format, args...)
	}
}

// Warn logs a warning message
func Warn(msg string, args ...interface{}) {
	if logger != nil {
		logger.Warnf(msg, args...)
	}
}

// Debug logs a debug message
func Debug(msg string, args ...interface{}) {
	if logger != nil {
		logger.Debugf(msg, args...)
	}
}

// BBW Dashboard specific logging functions

// BBWInfo logs BBW Dashboard info messages with structured fields
func BBWInfo(component, action, message string, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": component,
			"action":       action,
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Info(message)
	}
}

// BBWError logs BBW Dashboard error messages with structured fields
func BBWError(component, action, message string, err error, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": component,
			"action":       action,
		}

		if err != nil {
			logFields["error"] = err.Error()
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Error(message)
	}
}

// BBWDebug logs BBW Dashboard debug messages with structured fields
func BBWDebug(component, action, message string, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": component,
			"action":       action,
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Debug(message)
	}
}

// BBWWarn logs BBW Dashboard warning messages with structured fields
func BBWWarn(component, action, message string, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": component,
			"action":       action,
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Warn(message)
	}
}

// AlertInfo logs alert-specific info messages
func AlertInfo(alertType, symbol, message string, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": "alert_system",
			"alert_type":   alertType,
			"symbol":       symbol,
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Info(message)
	}
}

// AlertError logs alert-specific error messages
func AlertError(alertType, symbol, message string, err error, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": "alert_system",
			"alert_type":   alertType,
			"symbol":       symbol,
		}

		if err != nil {
			logFields["error"] = err.Error()
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Error(message)
	}
}

// PatternDetectionInfo logs pattern detection info messages
func PatternDetectionInfo(symbol, patternType, message string, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": "pattern_detection",
			"symbol":       symbol,
			"pattern_type": patternType,
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Info(message)
	}
}

// WebSocketInfo logs WebSocket-related info messages
func WebSocketInfo(action, message string, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": "websocket",
			"action":       action,
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Info(message)
	}
}

// WebSocketError logs WebSocket-related error messages
func WebSocketError(action, message string, err error, fields map[string]interface{}) {
	if logger != nil {
		logFields := logrus.Fields{
			"component":    "bbw_dashboard",
			"subcomponent": "websocket",
			"action":       action,
		}

		if err != nil {
			logFields["error"] = err.Error()
		}

		// Add custom fields
		for key, value := range fields {
			logFields[key] = value
		}

		logger.WithFields(logFields).Error(message)
	}
}
