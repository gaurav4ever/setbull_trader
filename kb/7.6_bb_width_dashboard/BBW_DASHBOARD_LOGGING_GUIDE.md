# BBW Dashboard Logging Guide

## Overview
This guide explains the comprehensive logging system implemented for the BBW Dashboard to ensure proper debugging, monitoring, and troubleshooting capabilities.

## Logging Architecture

### 1. Enhanced Logging System (`pkg/log/log.go`)

The logging system has been enhanced with the following features:

#### Daily Log Files
- **Location**: `logs/setbull_trader_YYYY-MM-DD.log`
- **Format**: JSON structured logging with timestamps
- **Rotation**: Automatic daily rotation
- **Retention**: Configurable retention (default: 30 days)

#### Log Levels
- **DEBUG**: Detailed debugging information
- **INFO**: General operational information
- **WARN**: Warning messages for potential issues
- **ERROR**: Error conditions that need attention
- **FATAL**: Critical errors that cause application termination

#### Structured Logging
All logs include structured fields for easy filtering and analysis:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "bbw_dashboard",
  "subcomponent": "candle_processing",
  "action": "start",
  "message": "Processing 5-minute candle close",
  "start_time": "10:25",
  "end_time": "10:30",
  "market_hours": true
}
```

### 2. BBW Dashboard Specific Logging Functions

#### Core Logging Functions
- `BBWInfo(component, action, message, fields)` - General information
- `BBWError(component, action, message, err, fields)` - Error conditions
- `BBWDebug(component, action, message, fields)` - Debug information
- `BBWWarn(component, action, message, fields)` - Warning messages

#### Specialized Logging Functions
- `AlertInfo(alertType, symbol, message, fields)` - Alert-specific logging
- `AlertError(alertType, symbol, message, err, fields)` - Alert error logging
- `PatternDetectionInfo(symbol, patternType, message, fields)` - Pattern detection logging
- `WebSocketInfo(action, message, fields)` - WebSocket operations
- `WebSocketError(action, message, err, fields)` - WebSocket errors

## Log Categories and Components

### 1. Candle Processing (`candle_processing`)
**Purpose**: Logs 5-minute candle processing operations

**Key Events**:
- Start/end of candle processing
- Market hours validation
- Stock data retrieval
- Processing completion

**Example Log**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "bbw_dashboard",
  "subcomponent": "candle_processing",
  "action": "process_start",
  "message": "Processing BBW data for stocks",
  "stock_count": 25,
  "start_time": "10:25",
  "end_time": "10:30"
}
```

### 2. Stock Processing (`stock_processing`)
**Purpose**: Logs individual stock BBW data processing

**Key Events**:
- Stock data processing start/completion
- BBW value calculations
- Data validation
- Alert condition checking

**Example Log**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "debug",
  "component": "bbw_dashboard",
  "subcomponent": "stock_processing",
  "action": "complete",
  "message": "Completed processing stock BBW data",
  "symbol": "RELIANCE",
  "instrument_key": "NSE_EQ|INE002A01018",
  "current_bbw": 0.0187,
  "historical_min_bbw": 0.0172,
  "distance_percent": 8.7,
  "contracting_count": 3,
  "trend": "contracting",
  "alert_triggered": true,
  "alert_type": "threshold",
  "pattern_strength": "moderate"
}
```

### 3. Alert System (`alert_system`)
**Purpose**: Logs alert detection and processing

**Key Events**:
- Alert condition detection
- Audio alert playback
- Alert history management
- Alert configuration changes

**Example Log**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "bbw_dashboard",
  "subcomponent": "alert_system",
  "alert_type": "threshold",
  "symbol": "RELIANCE",
  "message": "Alert triggered",
  "current_bbw": 0.0187,
  "historical_min_bbw": 0.0172,
  "pattern_length": 3,
  "pattern_strength": "moderate",
  "alert_message": "BB Width entered optimal range (0.0187)"
}
```

### 4. Pattern Detection (`pattern_detection`)
**Purpose**: Logs pattern recognition and analysis

**Key Events**:
- Pattern strength calculation
- Threshold alert detection
- Contracting pattern detection
- Squeeze condition detection

**Example Log**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "bbw_dashboard",
  "subcomponent": "pattern_detection",
  "symbol": "RELIANCE",
  "pattern_type": "threshold_alert",
  "message": "Threshold alert triggered",
  "current_bbw": 0.0187,
  "historical_min": 0.0172,
  "threshold_range": 0.1,
  "min_range": 0.0171,
  "max_range": 0.0173,
  "pattern_strength": "moderate"
}
```

### 5. WebSocket Operations (`websocket`)
**Purpose**: Logs real-time communication

**Key Events**:
- Client connections/disconnections
- Message broadcasting
- Connection health monitoring
- Error handling

**Example Log**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "bbw_dashboard",
  "subcomponent": "websocket",
  "action": "broadcast",
  "message": "Dashboard update broadcasted",
  "data_count": 25,
  "message_size": 2048
}
```

### 6. API Handlers (`api_handler`)
**Purpose**: Logs HTTP API requests and responses

**Key Events**:
- Request reception
- Response generation
- Error handling
- Performance metrics

**Example Log**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "bbw_dashboard",
  "subcomponent": "api_handler",
  "action": "dashboard_data_sent",
  "message": "Dashboard data sent successfully",
  "remote_addr": "192.168.1.100:54321",
  "data_count": 25
}
```

## Configuration

### Logging Configuration
```yaml
bbw_dashboard:
  logging:
    level: "info"           # Log level (debug, info, warn, error, fatal)
    log_dir: "logs"         # Log directory
    max_size: 100           # Maximum log file size in MB
    max_backups: 30         # Number of backup files to keep
    max_age: 30             # Maximum age of log files in days
    compress: true          # Compress old log files
```

### Alerting Configuration
```yaml
bbw_dashboard:
  alerting:
    default_threshold: 0.1      # Default alert threshold percentage
    default_lookback: 5         # Default contracting lookback period
    enable_audio_alerts: true   # Enable audio alerts
    alert_cooldown_minutes: 3   # Alert cooldown period
    max_alert_history: 100      # Maximum alert history entries
```

### Processing Configuration
```yaml
bbw_dashboard:
  processing:
    concurrent_workers: 10      # Number of concurrent processing workers
    processing_timeout: 30      # Processing timeout in seconds
    data_retention_days: 180    # Data retention period in days
```

## Log Analysis and Monitoring

### 1. Real-time Monitoring
Monitor logs in real-time using:
```bash
# Follow current day's log
tail -f logs/setbull_trader_$(date +%Y-%m-%d).log

# Follow with JSON formatting
tail -f logs/setbull_trader_$(date +%Y-%m-%d).log | jq '.'

# Filter by component
tail -f logs/setbull_trader_$(date +%Y-%m-%d).log | jq 'select(.component == "bbw_dashboard")'
```

### 2. Log Analysis Commands
```bash
# Count alerts by type
jq -r '.subcomponent + " " + .alert_type' logs/setbull_trader_*.log | grep "alert_system" | sort | uniq -c

# Find error patterns
jq -r 'select(.level == "error") | .message + " - " + .error' logs/setbull_trader_*.log

# Analyze performance
jq -r 'select(.action == "process_complete") | .processed_count + "/" + .total_stocks + " - " + .start_time + "-" + .end_time' logs/setbull_trader_*.log

# Monitor WebSocket connections
jq -r 'select(.subcomponent == "websocket") | .action + " - " + .total_clients' logs/setbull_trader_*.log
```

### 3. Alert Analysis
```bash
# Count alerts by symbol
jq -r 'select(.subcomponent == "alert_system") | .symbol + " - " + .alert_type' logs/setbull_trader_*.log | sort | uniq -c

# Analyze pattern detection
jq -r 'select(.subcomponent == "pattern_detection") | .symbol + " - " + .pattern_type + " - " + .pattern_strength' logs/setbull_trader_*.log | sort | uniq -c
```

## Troubleshooting Guide

### 1. No Logs Generated
**Symptoms**: No log files in the logs directory
**Causes**:
- Log directory permissions
- Logger not initialized
- Invalid log level

**Solutions**:
```bash
# Check log directory permissions
ls -la logs/

# Check if logger is initialized
grep "Logger initialized" logs/setbull_trader_*.log

# Verify log level
grep "level" logs/setbull_trader_*.log | head -1
```

### 2. High Log Volume
**Symptoms**: Large log files, performance impact
**Causes**:
- Debug level logging in production
- Excessive debug statements
- Log rotation issues

**Solutions**:
```bash
# Check log file sizes
ls -lh logs/setbull_trader_*.log

# Analyze log levels
jq -r '.level' logs/setbull_trader_*.log | sort | uniq -c

# Check for debug logs
jq -r 'select(.level == "debug") | .message' logs/setbull_trader_*.log | wc -l
```

### 3. Missing Alert Logs
**Symptoms**: No alert-related logs despite alerts being triggered
**Causes**:
- Alert service not available
- Log level too high
- Alert processing errors

**Solutions**:
```bash
# Check alert service availability
grep "service_unavailable" logs/setbull_trader_*.log

# Check alert processing
jq -r 'select(.subcomponent == "alert_system") | .message' logs/setbull_trader_*.log

# Check for alert errors
jq -r 'select(.subcomponent == "alert_system" and .level == "error") | .message + " - " + .error' logs/setbull_trader_*.log
```

### 4. WebSocket Connection Issues
**Symptoms**: No real-time updates, connection errors
**Causes**:
- WebSocket hub not running
- Connection timeouts
- Client disconnections

**Solutions**:
```bash
# Check WebSocket hub status
grep "hub started" logs/setbull_trader_*.log

# Monitor client connections
jq -r 'select(.subcomponent == "websocket") | .action + " - " + .total_clients' logs/setbull_trader_*.log

# Check for connection errors
jq -r 'select(.subcomponent == "websocket" and .level == "error") | .message + " - " + .error' logs/setbull_trader_*.log
```

## Performance Monitoring

### 1. Processing Performance
Monitor processing performance using:
```bash
# Average processing time
jq -r 'select(.action == "process_complete") | .processed_count + "/" + .total_stocks + " stocks in " + .start_time + "-" + .end_time' logs/setbull_trader_*.log

# Processing errors
jq -r 'select(.action == "process_stock" and .level == "error") | .symbol + " - " + .error' logs/setbull_trader_*.log
```

### 2. Alert Performance
Monitor alert system performance:
```bash
# Alert frequency
jq -r 'select(.subcomponent == "alert_system") | .alert_type + " - " + .symbol' logs/setbull_trader_*.log | sort | uniq -c

# Alert processing time
jq -r 'select(.subcomponent == "alert_system") | .timestamp + " - " + .alert_type + " - " + .symbol' logs/setbull_trader_*.log
```

### 3. API Performance
Monitor API performance:
```bash
# API request frequency
jq -r 'select(.subcomponent == "api_handler") | .action' logs/setbull_trader_*.log | sort | uniq -c

# API response times (if timing is added)
jq -r 'select(.subcomponent == "api_handler") | .action + " - " + .response_time' logs/setbull_trader_*.log
```

## Best Practices

### 1. Log Level Management
- Use `DEBUG` for development and troubleshooting
- Use `INFO` for production monitoring
- Use `WARN` for potential issues
- Use `ERROR` for actual problems
- Use `FATAL` sparingly

### 2. Structured Logging
- Always include relevant context fields
- Use consistent field names
- Include timing information where relevant
- Add error details for error logs

### 3. Log Rotation
- Monitor log file sizes
- Set appropriate retention periods
- Compress old logs to save space
- Archive important logs

### 4. Monitoring and Alerting
- Set up log monitoring for critical errors
- Monitor log volume and growth
- Set up alerts for unusual patterns
- Regular log analysis and cleanup

## Integration with Monitoring Systems

### 1. Prometheus Metrics
The logging system can be integrated with Prometheus for metrics collection:
- Alert frequency metrics
- Processing performance metrics
- Error rate metrics
- WebSocket connection metrics

### 2. ELK Stack Integration
Logs can be sent to Elasticsearch for advanced analysis:
- Real-time log aggregation
- Advanced search and filtering
- Custom dashboards
- Alert rules

### 3. Grafana Dashboards
Create dashboards for:
- BBW Dashboard performance
- Alert patterns and trends
- Error rates and types
- System health metrics

This comprehensive logging system ensures that the BBW Dashboard can be properly monitored, debugged, and optimized for production use. 