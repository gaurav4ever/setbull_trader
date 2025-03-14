package log

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// InitLogger initializes the logger
func InitLogger() {
	logger = logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})
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
