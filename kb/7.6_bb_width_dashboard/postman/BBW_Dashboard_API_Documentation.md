# BBW Dashboard API Documentation

## Overview
The BBW Dashboard API provides comprehensive endpoints for monitoring Bollinger Band Width patterns, managing alerts, and retrieving real-time and historical data for trading analysis.

## Base URL
- **Development**: `http://localhost:8083`
- **Production**: `http://your-production-domain:8080`

## Authentication
Currently, the API does not require authentication. All endpoints are publicly accessible.

## Response Format
All API responses follow a consistent JSON format:

```json
{
  "success": true,
  "data": {...},
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## API Endpoints

### 1. Dashboard Data Endpoints

#### 1.1 Get Dashboard Data
**GET** `/api/v1/bbw/dashboard-data`

Retrieves all BBW dashboard data for monitored stocks with real-time BBW values, trends, and alert status.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "symbol": "RELIANCE",
      "instrument_key": "NSE_EQ|INE002A01018",
      "current_bb_width": 0.0187,
      "historical_min_bb_width": 0.0172,
      "distance_from_min_percent": 8.7,
      "contracting_sequence_count": 3,
      "bb_width_trend": "contracting",
      "alert_triggered": true,
      "alert_type": "threshold",
      "alert_message": "BB Width entered optimal range (0.0187)",
      "pattern_strength": "moderate",
      "timestamp": "2024-01-15T10:30:00Z",
      "last_updated": "2024-01-15T10:30:00Z",
      "alert_triggered_at": "2024-01-15T10:30:00Z"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 1.2 Get Stock BBW Data
**GET** `/api/v1/bbw/stocks?instrument_key={instrument_key}`

Retrieves BBW data for a specific stock by instrument key.

**Parameters:**
- `instrument_key` (required): Instrument key of the stock (e.g., `NSE_EQ|INE002A01018`)

**Response:**
```json
{
  "success": true,
  "data": {
    "symbol": "RELIANCE",
    "instrument_key": "NSE_EQ|INE002A01018",
    "current_bb_width": 0.0187,
    "historical_min_bb_width": 0.0172,
    "distance_from_min_percent": 8.7,
    "contracting_sequence_count": 3,
    "bb_width_trend": "contracting",
    "alert_triggered": true,
    "alert_type": "threshold",
    "alert_message": "BB Width entered optimal range (0.0187)",
    "pattern_strength": "moderate",
    "timestamp": "2024-01-15T10:30:00Z",
    "last_updated": "2024-01-15T10:30:00Z",
    "alert_triggered_at": "2024-01-15T10:30:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 1.3 Get Dashboard Statistics
**GET** `/api/v1/bbw/stats`

Retrieves comprehensive dashboard statistics including stock counts, BBW ranges, and alert distributions.

**Response:**
```json
{
  "success": true,
  "data": {
    "total_stocks": 25,
    "alerted_stocks": 3,
    "contracting_count": 8,
    "expanding_count": 12,
    "stable_count": 5,
    "min_bb_width": 0.0156,
    "max_bb_width": 0.0234,
    "avg_bb_width": 0.0198,
    "recent_alerts": [
      {
        "symbol": "RELIANCE",
        "alert_type": "threshold",
        "timestamp": "2024-01-15T10:30:00Z",
        "message": "BB Width entered optimal range"
      }
    ],
    "total_alerts": 15
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 1.4 Get Market Statistics
**GET** `/api/v1/bbw/statistics`

Retrieves market-wide BBW statistics and distribution analysis.

**Response:**
```json
{
  "success": true,
  "data": {
    "total_stocks": 25,
    "avg_bb_width": 0.0198,
    "min_bb_width": 0.0156,
    "max_bb_width": 0.0234,
    "alert_distribution": {
      "threshold": 2,
      "pattern": 1,
      "squeeze": 0
    },
    "trend_distribution": {
      "contracting": 8,
      "expanding": 12,
      "stable": 5
    },
    "market_volatility": "medium",
    "squeeze_opportunities": 3
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 2. Alert Management Endpoints

#### 2.1 Get Active Alerts
**GET** `/api/v1/bbw/alerts/active`

Retrieves currently active BBW alerts across all monitored stocks.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "symbol": "RELIANCE",
      "instrument_key": "NSE_EQ|INE002A01018",
      "current_bb_width": 0.0187,
      "historical_min_bb_width": 0.0172,
      "distance_from_min_percent": 8.7,
      "contracting_sequence_count": 3,
      "bb_width_trend": "contracting",
      "alert_triggered": true,
      "alert_type": "threshold",
      "alert_message": "BB Width entered optimal range (0.0187)",
      "pattern_strength": "moderate",
      "timestamp": "2024-01-15T10:30:00Z",
      "last_updated": "2024-01-15T10:30:00Z",
      "alert_triggered_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 2.2 Get Alert History
**GET** `/api/v1/bbw/alerts/history?limit={limit}&alert_type={alert_type}&symbol={symbol}`

Retrieves alert history with optional filtering by type, symbol, and limit.

**Parameters:**
- `limit` (optional): Maximum number of alerts to return (default: 50)
- `alert_type` (optional): Filter by alert type (`threshold`, `pattern`, `squeeze`)
- `symbol` (optional): Filter by stock symbol

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "symbol": "RELIANCE",
      "bb_width": 0.0187,
      "lowest_min_bb_width": 0.0172,
      "pattern_length": 3,
      "alert_type": "threshold",
      "timestamp": "2024-01-15T10:30:00Z",
      "group_id": "BBW_DASHBOARD",
      "message": "BB Width entered optimal range (0.0187)"
    }
  ],
  "count": 1,
  "total": 15,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 2.3 Clear Alert History
**DELETE** `/api/v1/bbw/alerts/history`

Clears all alert history from memory.

**Response:**
```json
{
  "success": true,
  "message": "Alert history cleared successfully",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### 2.4 Configure Alerts
**POST** `/api/v1/bbw/alerts/configure`

Updates alert configuration including threshold, lookback period, and enable/disable settings.

**Request Body:**
```json
{
  "alert_threshold": 0.1,
  "contracting_lookback": 5,
  "enable_alerts": true
}
```

**Response:**
```json
{
  "success": true,
  "message": "Alert configuration updated successfully",
  "config": {
    "alert_threshold": 0.1,
    "contracting_lookback": 5,
    "enable_alerts": true
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 3. Historical Data Endpoints

#### 3.1 Get Stock BBW History
**GET** `/api/v1/bbw/stocks/{symbol}/history?timeframe={timeframe}`

Retrieves historical BBW data for a specific stock with configurable timeframe.

**Parameters:**
- `symbol` (path): Stock symbol (e.g., `RELIANCE`)
- `timeframe` (query): Timeframe for historical data (`1d`, `1w`, `1m`)

**Response:**
```json
{
  "success": true,
  "data": {
    "instrument_key": "NSE_EQ|INE002A01018",
    "timeframe": "1d",
    "history": [
      {
        "timestamp": "2024-01-15T10:25:00Z",
        "bb_width": 0.0234,
        "close": 150.25
      },
      {
        "timestamp": "2024-01-15T10:20:00Z",
        "bb_width": 0.0241,
        "close": 150.10
      },
      {
        "timestamp": "2024-01-15T10:15:00Z",
        "bb_width": 0.0256,
        "close": 149.95
      }
    ]
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 4. WebSocket Endpoints

#### 4.1 WebSocket Connection
**WebSocket** `ws://{base_url}/api/v1/bbw/live`

WebSocket endpoint for real-time BBW updates and alert notifications.

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8083/api/v1/bbw/live');

ws.onopen = function() {
    console.log('WebSocket connected');
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Received:', data);
};

ws.onclose = function() {
    console.log('WebSocket disconnected');
};
```

**Message Format:**
```json
{
  "type": "bbw_dashboard_update",
  "data": [
    {
      "symbol": "RELIANCE",
      "instrument_key": "NSE_EQ|INE002A01018",
      "current_bb_width": 0.0187,
      "historical_min_bb_width": 0.0172,
      "distance_from_min_percent": 8.7,
      "contracting_sequence_count": 3,
      "bb_width_trend": "contracting",
      "alert_triggered": true,
      "alert_type": "threshold",
      "alert_message": "BB Width entered optimal range (0.0187)",
      "pattern_strength": "moderate",
      "timestamp": "2024-01-15T10:30:00Z",
      "last_updated": "2024-01-15T10:30:00Z",
      "alert_triggered_at": "2024-01-15T10:30:00Z"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Data Models

### BBWDashboardData
```json
{
  "symbol": "string",
  "instrument_key": "string",
  "current_bb_width": "number",
  "historical_min_bb_width": "number",
  "distance_from_min_percent": "number",
  "contracting_sequence_count": "number",
  "bb_width_trend": "string",
  "alert_triggered": "boolean",
  "alert_type": "string",
  "alert_message": "string",
  "pattern_strength": "string",
  "timestamp": "string",
  "last_updated": "string",
  "alert_triggered_at": "string"
}
```

### AlertEvent
```json
{
  "symbol": "string",
  "bb_width": "number",
  "lowest_min_bb_width": "number",
  "pattern_length": "number",
  "alert_type": "string",
  "timestamp": "string",
  "group_id": "string",
  "message": "string"
}
```

## Error Responses

### 400 Bad Request
```json
{
  "success": false,
  "error": "instrument_key parameter is required",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 404 Not Found
```json
{
  "success": false,
  "error": "Stock not found",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 500 Internal Server Error
```json
{
  "success": false,
  "error": "Internal server error",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Rate Limiting
Currently, there are no rate limits implemented. However, it's recommended to:
- Limit requests to reasonable frequencies
- Use WebSocket connections for real-time data
- Implement appropriate caching on the client side

## Testing
Use the provided Postman collection (`BBW_Dashboard_API_Collection.json`) for testing all endpoints. The collection includes:
- Pre-configured variables for easy testing
- Example request bodies
- Organized folder structure
- Comprehensive endpoint coverage

## WebSocket Testing
For WebSocket testing, you can use:
- Postman's WebSocket support
- Browser developer tools
- Dedicated WebSocket testing tools like `wscat`

## Configuration
The API behavior can be configured through the `application.yaml` file:
- Logging levels and file management
- Alert thresholds and cooldowns
- Processing timeouts and worker counts
- Data retention policies

## Monitoring
The API includes comprehensive logging for:
- Request/response tracking
- Performance metrics
- Error handling
- Alert processing
- WebSocket connections

Logs are stored in daily files at `logs/setbull_trader_YYYY-MM-DD.log` in JSON format for easy analysis.