package v2

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"setbull_trader/pkg/log"
)

// LogLevel represents the logging level
type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogCategory represents the logging category
type LogCategory string

const (
	SYSTEM      LogCategory = "SYSTEM"
	ENGINE      LogCategory = "ENGINE"
	STRATEGY    LogCategory = "STRATEGY"
	PERFORMANCE LogCategory = "PERFORMANCE"
	ERROR_CAT   LogCategory = "ERROR"
	PROGRESS    LogCategory = "PROGRESS"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Category      string                 `json:"category"`
	Component     string                 `json:"component"`
	Method        string                 `json:"method"`
	TraceID       string                 `json:"trace_id"`
	JobID         string                 `json:"job_id"`
	StockGroupID  string                 `json:"stock_group_id"`
	StrategyName  string                 `json:"strategy_name"`
	Message       string                 `json:"message"`
	DurationMs    int64                  `json:"duration_ms"`
	MemoryUsageMB int64                  `json:"memory_usage_mb"`
	Context       map[string]interface{} `json:"context"`
	Error         string                 `json:"error,omitempty"`
	StackTrace    string                 `json:"stack_trace,omitempty"`
}

// DebugLogger interface for structured logging
type DebugLogger interface {
	Log(level LogLevel, category LogCategory, component string, method string,
		traceID string, jobID string, stockGroupID string, strategyName string,
		message string, duration time.Duration, memoryUsage int64,
		context map[string]interface{}, err error)

	SetLogLevel(level LogLevel)
	EnableCategory(category LogCategory)
	DisableCategory(category LogCategory)
	GetLogLevel() LogLevel
	IsCategoryEnabled(category LogCategory) bool
}

// StructuredLogger implements the DebugLogger interface
type StructuredLogger struct {
	logLevel    LogLevel
	categories  map[LogCategory]bool
	mu          sync.RWMutex
	logFile     *os.File
	jsonEncoder *json.Encoder
	formatter   *LogFormatter
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(logDir, logFileName string) (*StructuredLogger, error) {
	// Create log formatter
	formatter := NewLogFormatter(logDir, logFileName, 100*1024*1024, 10) // 100MB max size, 10 backup files

	// Ensure log directory exists
	if err := formatter.EnsureLogDir(); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Check if log rotation is needed
	if formatter.ShouldRotate() {
		if err := formatter.RotateLogFile(); err != nil {
			return nil, fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	// Create log file
	file, err := os.OpenFile(formatter.GetLogFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := &StructuredLogger{
		logLevel:   INFO, // Default log level
		categories: make(map[LogCategory]bool),
		logFile:    file,
		formatter:  formatter,
	}

	// Enable all categories by default
	logger.categories[SYSTEM] = true
	logger.categories[ENGINE] = true
	logger.categories[STRATEGY] = true
	logger.categories[PERFORMANCE] = true
	logger.categories[ERROR_CAT] = true
	logger.categories[PROGRESS] = true

	logger.jsonEncoder = json.NewEncoder(file)
	logger.jsonEncoder.SetIndent("", "  ")

	// Log startup message
	logger.Log(INFO, SYSTEM, "StructuredLogger", "NewStructuredLogger",
		"", "", "", "", "Structured logger initialized", 0, 0, nil, nil)

	return logger, nil
}

// Log logs a structured log entry
func (l *StructuredLogger) Log(level LogLevel, category LogCategory, component string, method string,
	traceID string, jobID string, stockGroupID string, strategyName string,
	message string, duration time.Duration, memoryUsage int64,
	context map[string]interface{}, err error) {

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if log level is enabled
	if level < l.logLevel {
		return
	}

	// Check if category is enabled
	if !l.categories[category] {
		return
	}

	// Create log entry
	entry := LogEntry{
		Timestamp:     time.Now(),
		Level:         level.String(),
		Category:      string(category),
		Component:     component,
		Method:        method,
		TraceID:       traceID,
		JobID:         jobID,
		StockGroupID:  stockGroupID,
		StrategyName:  strategyName,
		Message:       message,
		DurationMs:    duration.Milliseconds(),
		MemoryUsageMB: memoryUsage,
		Context:       context,
	}

	// Add error information if present
	if err != nil {
		entry.Error = err.Error()
		// TODO: Add stack trace extraction
		// entry.StackTrace = getStackTrace(err)
	}

	// Write to log file
	if err := l.jsonEncoder.Encode(entry); err != nil {
		// Fallback to standard logging if JSON encoding fails
		log.Error("Failed to encode log entry: %v", err)
	}

	// Also log to standard logger for immediate visibility
	l.logToStandardLogger(entry)
}

// logToStandardLogger logs to the standard logger for immediate visibility
func (l *StructuredLogger) logToStandardLogger(entry LogEntry) {
	// Create a formatted message for standard logging
	formattedMsg := fmt.Sprintf("[%s] %s/%s %s: %s",
		entry.Level, entry.Category, entry.Component, entry.Method, entry.Message)

	// Add context information
	if len(entry.Context) > 0 {
		contextStr := ""
		for key, value := range entry.Context {
			contextStr += fmt.Sprintf(" %s=%v", key, value)
		}
		formattedMsg += contextStr
	}

	// Add trace and job information
	if entry.TraceID != "" {
		formattedMsg += fmt.Sprintf(" trace_id=%s", entry.TraceID)
	}
	if entry.JobID != "" {
		formattedMsg += fmt.Sprintf(" job_id=%s", entry.JobID)
	}

	// Add performance information
	if entry.DurationMs > 0 {
		formattedMsg += fmt.Sprintf(" duration=%dms", entry.DurationMs)
	}
	if entry.MemoryUsageMB > 0 {
		formattedMsg += fmt.Sprintf(" memory=%dMB", entry.MemoryUsageMB)
	}

	// Log based on level
	switch entry.Level {
	case "TRACE":
		log.Debug("%s", formattedMsg)
	case "DEBUG":
		log.Debug("%s", formattedMsg)
	case "INFO":
		log.Info("%s", formattedMsg)
	case "WARN":
		log.Warn("%s", formattedMsg)
	case "ERROR":
		log.Error("%s", formattedMsg)
	}
}

// SetLogLevel sets the log level
func (l *StructuredLogger) SetLogLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = level
}

// EnableCategory enables a log category
func (l *StructuredLogger) EnableCategory(category LogCategory) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.categories[category] = true
}

// DisableCategory disables a log category
func (l *StructuredLogger) DisableCategory(category LogCategory) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.categories[category] = false
}

// GetLogLevel returns the current log level
func (l *StructuredLogger) GetLogLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.logLevel
}

// IsCategoryEnabled checks if a category is enabled
func (l *StructuredLogger) IsCategoryEnabled(category LogCategory) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.categories[category]
}

// Close closes the logger and its underlying file
func (l *StructuredLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// TraceIDGenerator generates unique trace IDs
type TraceIDGenerator struct {
	counter int64
	mu      sync.Mutex
}

// NewTraceIDGenerator creates a new trace ID generator
func NewTraceIDGenerator() *TraceIDGenerator {
	return &TraceIDGenerator{
		counter: 0,
	}
}

// GenerateTraceID generates a unique trace ID
func (g *TraceIDGenerator) GenerateTraceID() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.counter++
	return fmt.Sprintf("trace_%d_%d", time.Now().Unix(), g.counter)
}

// Global logger instance
var (
	globalLogger      DebugLogger
	globalTraceGen    *TraceIDGenerator
	loggerInitialized bool
	loggerMu          sync.RWMutex
)

// InitializeGlobalLogger initializes the global logger
func InitializeGlobalLogger(logDir, logFileName string) error {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if loggerInitialized {
		return fmt.Errorf("global logger already initialized")
	}

	logger, err := NewStructuredLogger(logDir, logFileName)
	if err != nil {
		return err
	}

	globalLogger = logger
	globalTraceGen = NewTraceIDGenerator()
	loggerInitialized = true

	return nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() DebugLogger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return globalLogger
}

// GenerateTraceID generates a trace ID using the global generator
func GenerateTraceID() string {
	loggerMu.RLock()
	defer loggerMu.RUnlock()

	if globalTraceGen == nil {
		// Fallback if not initialized
		return fmt.Sprintf("trace_%d", time.Now().UnixNano())
	}

	return globalTraceGen.GenerateTraceID()
}

// LogHelper provides convenience methods for logging
type LogHelper struct {
	logger  DebugLogger
	traceID string
	jobID   string
}

// NewLogHelper creates a new log helper
func NewLogHelper(logger DebugLogger, traceID string, jobID string) *LogHelper {
	return &LogHelper{
		logger:  logger,
		traceID: traceID,
		jobID:   jobID,
	}
}

// Log logs a message with the helper's context
func (h *LogHelper) Log(level LogLevel, category LogCategory, component string, method string,
	stockGroupID string, strategyName string, message string, duration time.Duration,
	memoryUsage int64, context map[string]interface{}, err error) {

	if h.logger != nil {
		h.logger.Log(level, category, component, method, h.traceID, h.jobID,
			stockGroupID, strategyName, message, duration, memoryUsage, context, err)
	}
}

// LogInfo logs an info message
func (h *LogHelper) LogInfo(category LogCategory, component string, method string,
	stockGroupID string, strategyName string, message string, context map[string]interface{}) {
	h.Log(INFO, category, component, method, stockGroupID, strategyName, message, 0, 0, context, nil)
}

// LogDebug logs a debug message
func (h *LogHelper) LogDebug(category LogCategory, component string, method string,
	stockGroupID string, strategyName string, message string, context map[string]interface{}) {
	h.Log(DEBUG, category, component, method, stockGroupID, strategyName, message, 0, 0, context, nil)
}

// LogWarn logs a warning message
func (h *LogHelper) LogWarn(category LogCategory, component string, method string,
	stockGroupID string, strategyName string, message string, context map[string]interface{}) {
	h.Log(WARN, category, component, method, stockGroupID, strategyName, message, 0, 0, context, nil)
}

// LogError logs an error message
func (h *LogHelper) LogError(category LogCategory, component string, method string,
	stockGroupID string, strategyName string, message string, context map[string]interface{}, err error) {
	h.Log(ERROR, category, component, method, stockGroupID, strategyName, message, 0, 0, context, err)
}

// LogPerformance logs a performance message
func (h *LogHelper) LogPerformance(component string, method string, stockGroupID string,
	strategyName string, message string, duration time.Duration, memoryUsage int64, context map[string]interface{}) {
	h.Log(INFO, PERFORMANCE, component, method, stockGroupID, strategyName,
		message, duration, memoryUsage, context, nil)
}
