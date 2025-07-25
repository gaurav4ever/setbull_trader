# BBW Dashboard Phase 2 Integration Guide

## Overview
This guide covers the implementation of **Phase 2: Dashboard Core** for the BBW Dashboard, which provides a basic dashboard interface with real-time updates via WebSocket.

## Files Created/Modified

### Frontend Files
1. **Navigation**: `frontend/src/routes/+layout.svelte` - Added BBW Dashboard link
2. **API Configuration**: `frontend/src/lib/config/api.js` - Added BBW API endpoints
3. **API Service**: `frontend/src/lib/services/apiService.js` - Added BBW API functions
4. **WebSocket Service**: `frontend/src/lib/services/bbwWebSocketService.js` - Real-time communication
5. **Dashboard Store**: `frontend/src/lib/stores/bbwDashboardStore.js` - State management
6. **Dashboard Page**: `frontend/src/routes/bbw-dashboard/+page.svelte` - Main dashboard UI
7. **Stock Card Component**: `frontend/src/lib/components/BBWStockCard.svelte` - Reusable stock display

### Backend Files (Already Created)
1. **BBW Dashboard Service**: `internal/service/bbw_dashboard_service.go`
2. **WebSocket Hub**: `internal/service/websocket_hub.go`
3. **REST Handlers**: `cmd/trading/transport/rest/bbw_dashboard_handlers.go`

## Integration Steps

### Step 1: Backend Integration

#### 1.1 Add WebSocket Support to HTTP Server
Add the following to your main HTTP server setup in `cmd/trading/transport/http.go`:

```go
import (
    "github.com/gorilla/websocket"
    "setbull_trader/internal/service"
)

// Add WebSocket upgrader
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins for development
    },
}

// Add WebSocket handler
func (s *Server) handleBBWWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }
    
    client := &service.WebSocketClient{
        Hub:  s.bbwDashboardService.GetWebSocketHub(),
        Conn: conn,
        Send: make(chan []byte, 256),
    }
    
    client.Hub.Register <- client
    
    go client.WritePump()
    go client.ReadPump()
}
```

#### 1.2 Add WebSocket Route
Add the WebSocket route to your HTTP server routes:

```go
// In your route setup
http.HandleFunc("/api/v1/bbw/live", s.handleBBWWebSocket)
```

#### 1.3 Initialize BBW Dashboard Service
Add the BBW dashboard service to your service initialization:

```go
// In your main service setup
bbwDashboardService := service.NewBBWDashboardService(
    candleAggService,
    technicalIndicatorSvc,
    stockGroupService,
    universeService,
    websocketHub,
)

// Start the WebSocket hub
go websocketHub.Run()

// Start the BBW dashboard service
go bbwDashboardService.Start()
```

#### 1.4 Add BBW Handlers to Router
Add the BBW dashboard handlers to your HTTP router:

```go
// In your route setup
bbwHandler := rest.NewBBWDashboardHandler(bbwDashboardService)

// BBW Dashboard routes
router.HandleFunc("/api/v1/bbw/dashboard-data", bbwHandler.GetDashboardData).Methods("GET")
router.HandleFunc("/api/v1/bbw/stocks", bbwHandler.GetStockBBWData).Methods("GET")
router.HandleFunc("/api/v1/bbw/stocks/{symbol}/history", bbwHandler.GetStockHistory).Methods("GET")
router.HandleFunc("/api/v1/bbw/alerts/active", bbwHandler.GetActiveAlerts).Methods("GET")
router.HandleFunc("/api/v1/bbw/alerts/configure", bbwHandler.ConfigureAlerts).Methods("POST")
router.HandleFunc("/api/v1/bbw/statistics", bbwHandler.GetStatistics).Methods("GET")
```

### Step 2: Frontend Integration

#### 2.1 Install Dependencies
Add the required dependencies to your `frontend/package.json`:

```json
{
  "dependencies": {
    "svelte": "^4.0.0"
  }
}
```

#### 2.2 Configure Vite Proxy
Add WebSocket proxy configuration to your `frontend/vite.config.ts`:

```typescript
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true // Enable WebSocket proxy
      }
    }
  }
});
```

#### 2.3 Test the Integration

1. **Start the Backend**:
   ```bash
   cd /path/to/your/project
   go run main.go
   ```

2. **Start the Frontend**:
   ```bash
   cd frontend
   npm run dev
   ```

3. **Access the Dashboard**:
   Navigate to `http://localhost:5173/bbw-dashboard`

## Features Implemented

### ✅ Real-time Dashboard Interface
- **Stock List**: Displays all monitored stocks with BBW data
- **Real-time Updates**: WebSocket-based live updates every 5 minutes
- **Visual Indicators**: Color-coded trends and alert status
- **Responsive Design**: Works on desktop and tablet

### ✅ Data Display
- **Current BBW**: Real-time Bollinger Band Width values
- **Historical Min BBW**: Minimum BBW over the lookback period
- **Distance from Min**: Percentage distance from historical minimum
- **Trend Indicators**: Visual arrows showing BBW direction
- **Contracting Count**: Number of consecutive contracting candles

### ✅ Filtering and Sorting
- **Search**: Filter stocks by symbol or instrument key
- **Category Filter**: Filter by alerted, contracting, expanding, or stable
- **Sorting**: Sort by distance, BBW, symbol, or other metrics
- **Real-time Updates**: Filters and sorts update automatically

### ✅ Connection Management
- **WebSocket Status**: Visual indicator of connection status
- **Auto-reconnection**: Automatic reconnection on connection loss
- **Heartbeat**: Ping/pong to maintain connection health
- **Error Handling**: Graceful error handling and user feedback

### ✅ Market Status
- **Market Hours**: Real-time market open/closed status
- **Current Time**: Live clock display
- **Last Update**: Timestamp of last data update

## API Endpoints

### REST Endpoints
- `GET /api/v1/bbw/dashboard-data` - Get all dashboard data
- `GET /api/v1/bbw/stocks?instrument_key={key}` - Get specific stock data
- `GET /api/v1/bbw/stocks/{symbol}/history` - Get stock history
- `GET /api/v1/bbw/alerts/active` - Get active alerts
- `POST /api/v1/bbw/alerts/configure` - Configure alerts
- `GET /api/v1/bbw/statistics` - Get market statistics

### WebSocket Endpoint
- `ws://localhost:8080/api/v1/bbw/live` - Real-time updates

## Data Flow

### 1. 5-Minute Candle Close Trigger
```
5min Candle Close → BBWDashboardService → WebSocket Hub → Frontend
```

### 2. Real-time Updates
```
Backend BBW Data → WebSocket → Frontend Store → UI Components
```

### 3. User Interactions
```
Frontend → API Service → Backend Handlers → Database
```

## Testing

### Manual Testing Checklist

1. **Dashboard Load**:
   - [ ] Dashboard loads without errors
   - [ ] Stock list displays correctly
   - [ ] Statistics cards show proper values

2. **Real-time Updates**:
   - [ ] WebSocket connects successfully
   - [ ] Data updates every 5 minutes
   - [ ] Connection status shows correctly

3. **Filtering and Sorting**:
   - [ ] Search functionality works
   - [ ] Category filters work
   - [ ] Sorting works for all columns

4. **Responsive Design**:
   - [ ] Dashboard works on desktop
   - [ ] Dashboard works on tablet
   - [ ] Table scrolls horizontally on mobile

### API Testing

Test the endpoints using curl:

```bash
# Get dashboard data
curl http://localhost:8080/api/v1/bbw/dashboard-data

# Get specific stock
curl "http://localhost:8080/api/v1/bbw/stocks?instrument_key=NSE_EQ|INE005B01027"

# Get statistics
curl http://localhost:8080/api/v1/bbw/statistics
```

## Troubleshooting

### Common Issues

1. **WebSocket Connection Failed**:
   - Check if backend is running on port 8080
   - Verify WebSocket route is properly configured
   - Check browser console for connection errors

2. **No Data Displayed**:
   - Verify BBW dashboard service is started
   - Check if 5-minute candle data exists
   - Verify API endpoints are accessible

3. **Real-time Updates Not Working**:
   - Check WebSocket connection status
   - Verify backend is sending updates
   - Check frontend store subscription

### Debug Commands

```bash
# Check backend logs
tail -f logs/app.log

# Test WebSocket connection
wscat -c ws://localhost:8080/api/v1/bbw/live

# Check API endpoints
curl -v http://localhost:8080/api/v1/bbw/dashboard-data
```

## Next Steps

After completing Phase 2, the next phases will include:

- **Phase 3**: Alerting System (audio alerts, pattern detection)
- **Phase 4**: Advanced Features (charts, export, customization)
- **Phase 5**: Optimization and Polish (performance, monitoring)

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review the backend logs for errors
3. Check browser console for frontend errors
4. Verify all dependencies are installed correctly 