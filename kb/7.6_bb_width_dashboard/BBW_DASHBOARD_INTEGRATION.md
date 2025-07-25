# BBW Dashboard Integration Guide

## Overview
This guide explains how to integrate the BBW Dashboard with your existing 5-minute candle infrastructure. The dashboard provides real-time Bollinger Band Width monitoring with WebSocket updates.

## Files Created/Modified

### Backend Files
1. `internal/service/bbw_dashboard_service.go` - Main BBW dashboard service
2. `internal/service/websocket_hub.go` - WebSocket hub for real-time updates
3. `cmd/trading/transport/rest/bbw_dashboard_handlers.go` - REST API handlers

### Frontend Files
1. `frontend/src/routes/bbw-dashboard/+page.svelte` - BBW dashboard page

## Integration Steps

### Step 1: Add Dependencies

Add the following to your `go.mod`:
```go
require (
    github.com/gorilla/websocket v1.5.0
)
```

### Step 2: Wire Up Services in app.go

Add the following to your `cmd/trading/app/app.go` in the `NewApp()` function:

```go
// Initialize WebSocket hub
websocketHub := service.NewWebSocketHub()

// Initialize BBW dashboard service
bbwDashboardService := service.NewBBWDashboardService(
    candleAggService,
    technicalIndicatorService,
    stockGroupService,
    stockUniverseService,
    websocketHub,
)

// Register BBW dashboard service as a 5-minute candle listener
stockGroupService.RegisterFiveMinCloseListener(bbwDashboardService.OnFiveMinCandleClose)

// Start WebSocket hub
go websocketHub.Run()
```

### Step 3: Add REST Routes

Add the following routes to your `cmd/trading/transport/rest/server.go` in the `setupRoutes()` function:

```go
// BBW Dashboard routes
bbwDashboardHandler := rest.NewBBWDashboardHandler(bbwDashboardService)
api.HandleFunc("/bbw/dashboard", bbwDashboardHandler.GetDashboardData).Methods(http.MethodGet)
api.HandleFunc("/bbw/stock", bbwDashboardHandler.GetStockBBWData).Methods(http.MethodGet)
api.HandleFunc("/bbw/stats", bbwDashboardHandler.GetDashboardStats).Methods(http.MethodGet)
api.HandleFunc("/bbw/threshold", bbwDashboardHandler.UpdateAlertThreshold).Methods(http.MethodPost)
api.HandleFunc("/bbw/lookback", bbwDashboardHandler.UpdateContractingLookback).Methods(http.MethodPost)
api.HandleFunc("/bbw/history", bbwDashboardHandler.GetStockBBWHistory).Methods(http.MethodGet)

// WebSocket endpoint
api.HandleFunc("/bbw/ws", websocketHub.HandleWebSocket).Methods(http.MethodGet)
```

### Step 4: Add Navigation Link

Add a link to the BBW dashboard in your main navigation. In `frontend/src/routes/+layout.svelte`:

```svelte
<a href="/bbw-dashboard" class="text-gray-300 hover:bg-gray-700 hover:text-white px-3 py-2 rounded-md text-sm font-medium">
    BBW Dashboard
</a>
```

### Step 5: Create BB_RANGE Groups

To monitor stocks for BBW patterns, create stock groups with entry type "BB_RANGE":

```sql
INSERT INTO stock_groups (name, entry_type, status, created_at, updated_at) 
VALUES ('BBW Monitor Group', 'BB_RANGE', 'ACTIVE', NOW(), NOW());
```

Then add stocks to this group through your existing stock group management interface.

## How It Works

### 1. 5-Minute Candle Trigger
- Your existing 5-minute candle aggregation process triggers `OnFiveMinCandleClose`
- This happens automatically when 5-minute candles are completed

### 2. BBW Data Processing
- The service fetches all stocks from BB_RANGE groups
- For each stock, it gets recent 5-minute candles with BBW data
- Calculates additional metrics:
  - Distance from historical minimum BBW
  - Contracting sequence count
  - BBW trend (contracting/expanding/stable)
  - Alert conditions

### 3. Real-Time Updates
- Processed data is cached in memory
- WebSocket broadcasts updates to all connected frontend clients
- Frontend receives real-time updates every 5 minutes

### 4. Frontend Display
- Dashboard shows BBW data in a sortable table
- Color-coded indicators for trends and alerts
- Search and filter functionality
- Real-time connection status

## Configuration

### Alert Threshold
Default alert threshold is 0.1% from historical minimum. You can update this via API:

```bash
curl -X POST http://localhost:8080/api/v1/bbw/threshold \
  -H "Content-Type: application/json" \
  -d '{"threshold": 0.1}'
```

### Contracting Lookback
Default lookback period is 5 candles. You can update this via API:

```bash
curl -X POST http://localhost:8080/api/v1/bbw/lookback \
  -H "Content-Type: application/json" \
  -d '{"lookback": 5}'
```

## API Endpoints

### GET /api/v1/bbw/dashboard
Returns all BBW dashboard data for monitored stocks.

### GET /api/v1/bbw/stock?instrument_key=KEY
Returns BBW data for a specific stock.

### GET /api/v1/bbw/stats
Returns dashboard statistics (total stocks, alerted stocks, etc.).

### POST /api/v1/bbw/threshold
Updates the alert threshold percentage.

### POST /api/v1/bbw/lookback
Updates the contracting lookback period.

### GET /api/v1/bbw/history?instrument_key=KEY&timeframe=1d
Returns historical BBW data for a stock (placeholder implementation).

### WebSocket: ws://localhost:8080/api/v1/bbw/ws
Real-time BBW updates during market hours.

## Data Flow

```
5-Min Candle Close → BBWDashboardService.OnFiveMinCandleClose()
    ↓
Fetch BB_RANGE Stocks → Process BBW Data → Update Cache
    ↓
WebSocket Broadcast → Frontend Update
```

## Performance Considerations

1. **Concurrent Processing**: Each stock is processed in a separate goroutine
2. **Memory Caching**: BBW data is cached in memory for fast access
3. **Market Hours Check**: Processing only occurs during market hours (9:15 AM - 3:30 PM IST)
4. **WebSocket Efficiency**: Updates are batched and sent to all connected clients

## Troubleshooting

### WebSocket Connection Issues
- Check if WebSocket hub is running
- Verify CORS settings for WebSocket endpoint
- Check browser console for connection errors

### No Data Displayed
- Ensure BB_RANGE groups exist with stocks
- Check if 5-minute candle aggregation is working
- Verify market hours (no processing outside 9:15 AM - 3:30 PM IST)

### Performance Issues
- Monitor memory usage of BBW data cache
- Consider reducing lookback period for large stock lists
- Check database performance for 5-minute candle queries

## Future Enhancements

1. **Historical Data**: Implement proper historical BBW data retrieval
2. **Advanced Alerts**: Add email/SMS notifications for alerts
3. **Charts**: Add interactive BBW charts for individual stocks
4. **Backtesting**: Integrate with existing backtesting framework
5. **Configuration UI**: Add dashboard configuration interface 