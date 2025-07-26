Summary: BBW Dashboard Access Outside Market Hours
I have successfully implemented a solution that allows the BBW Dashboard to access the latest available day data regardless of market timing. Here's what was implemented:
✅ Backend Changes
1. Enhanced BBW Dashboard Service (internal/service/bbw_dashboard_service.go)
GetLatestAvailableDayData(): New method to retrieve the most recent BBW data for all monitored stocks regardless of market hours
getLatestStockBBWData(): Helper method to get the latest 5-minute candle data for individual stocks
GetStockBBWHistory(): Enhanced method to get historical BBW data with configurable days parameter
getStockMetadata(): Helper method to retrieve stock metadata from BB_RANGE groups
2. New REST API Endpoints (cmd/trading/transport/rest/bbw_dashboard_handlers.go)
GET /api/v1/bbw/latest-day-data: Returns the most recent available BBW data for all monitored stocks
GET /api/v1/bbw/market-status: Returns current market status and last data timestamp information
Enhanced GET /api/v1/bbw/stocks/{symbol}/history: Now supports days parameter for historical data
3. Updated Route Configuration (cmd/trading/transport/rest/server.go)
Added new routes for latest day data and market status endpoints
✅ Frontend Changes
1. Enhanced API Service (frontend/src/lib/services/apiService.js)
getLatestAvailableDayData(): New method to fetch latest available day data
getMarketStatus(): New method to fetch market status and data age information
getStockBBWHistory(): Enhanced method with days parameter support
2. Updated API Configuration (frontend/src/lib/config/api.js)
Added new endpoint constants for latest day data and market status
3. Enhanced Dashboard Store (frontend/src/lib/stores/bbwDashboardStore.js)
Market Status Tracking: Added marketStatus, lastDataTimestamp, and lastDataAgeMinutes to store state
Smart Data Loading: Modified loadDashboardData() to try real-time data first, then fall back to latest available day data
loadMarketStatus(): New method to load market status and data age information
Conditional WebSocket: Only connects to WebSocket during market hours
4. Updated Dashboard UI (frontend/src/routes/bbw-dashboard/+page.svelte)
Data Age Display: Shows "Last Data: X min ago" when outside market hours
Market Status Integration: Displays market status and data freshness information
✅ Key Features
1. Smart Data Access
During Market Hours: Uses real-time data with WebSocket updates
Outside Market Hours: Automatically falls back to latest available day data
Data Age Tracking: Shows how old the displayed data is
2. Market Status Awareness
Real-time Market Status: Shows if market is currently open/closed
Last Data Timestamp: Displays when the last data was received
Data Age Calculation: Shows how many minutes ago the data was last updated
3. Graceful Degradation
WebSocket Management: Only connects during market hours to avoid unnecessary connections
Error Handling: Graceful fallback from real-time to historical data
User Feedback: Clear indication of data freshness and market status
✅ API Endpoints
New Endpoints:
Apply to +page.svelte
Run
7
Example Responses:
Market Status Response:
Apply to +page.svelte
}
Latest Day Data Response:
Apply to +page.svelte
}
✅ Benefits
24/7 Dashboard Access: Users can view BBW data even outside market hours
Data Freshness Awareness: Clear indication of how old the displayed data is
Automatic Fallback: Seamless transition between real-time and historical data
Performance Optimization: WebSocket only connects when needed
User Experience: No disruption when market closes, just clear data age indication
✅ Usage
The dashboard now automatically:
During Market Hours (9:15 AM - 3:30 PM IST): Shows real-time data with live updates
Outside Market Hours: Shows the most recent available data with age indication
Data Refresh: Users can manually refresh to get the latest available data
Status Display: Clear market status and data freshness information
This implementation ensures that traders can access BBW dashboard data at any time, with clear indication of data freshness and market status, providing a seamless experience regardless of market timing.
