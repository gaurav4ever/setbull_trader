# BBW Dashboard API - Postman Collection

## Overview
This Postman collection provides comprehensive testing capabilities for the BBW Dashboard API, including real-time monitoring, alert management, and historical data retrieval.

## Files Included
- `BBW_Dashboard_API_Collection.json` - Complete Postman collection
- `BBW_Dashboard_API_Documentation.md` - Detailed API documentation
- `README.md` - This setup guide

## Quick Start

### 1. Import the Collection
1. Open Postman
2. Click "Import" button
3. Select the `BBW_Dashboard_API_Collection.json` file
4. The collection will be imported with all folders and requests

### 2. Configure Environment Variables
The collection uses the following variables that you can configure:

| Variable | Default Value | Description |
|----------|---------------|-------------|
| `base_url` | `http://localhost:8083` | Base URL for the API |
| `instrument_key` | `NSE_EQ\|INE002A01018` | Example instrument key for testing |
| `symbol` | `RELIANCE` | Example stock symbol for testing |
| `limit` | `50` | Default limit for pagination |
| `alert_type` | `threshold` | Example alert type for filtering |
| `timeframe` | `1d` | Default timeframe for historical data |

### 3. Set Up Environment
1. In Postman, click on the "Environments" tab
2. Create a new environment called "BBW Dashboard Local"
3. Add the variables listed above
4. Select this environment for testing

## Collection Structure

### 📊 Dashboard Data
Core endpoints for retrieving BBW dashboard information:
- **Get Dashboard Data** - All monitored stocks with BBW data
- **Get Stock BBW Data** - Specific stock BBW information
- **Get Dashboard Statistics** - Comprehensive dashboard metrics
- **Get Market Statistics** - Market-wide BBW analysis

### 🔔 Alert Management
Alert configuration and monitoring endpoints:
- **Get Active Alerts** - Currently triggered alerts
- **Get Alert History** - Historical alert data with filtering
- **Clear Alert History** - Remove all alert history
- **Configure Alerts** - Update alert settings

### 📈 Historical Data
Historical BBW data analysis:
- **Get Stock BBW History** - Historical BBW data for specific stocks

### 🔌 WebSocket
Real-time communication:
- **WebSocket Connection** - Real-time BBW updates and alerts

## Testing Scenarios

### 1. Basic Dashboard Testing
1. Start with "Get Dashboard Data" to verify the API is working
2. Check "Get Dashboard Statistics" for overview metrics
3. Test "Get Market Statistics" for market-wide analysis

### 2. Stock-Specific Testing
1. Use "Get Stock BBW Data" with a valid instrument key
2. Test "Get Stock BBW History" for historical data
3. Verify data consistency across endpoints

### 3. Alert System Testing
1. Check "Get Active Alerts" for current alerts
2. Use "Get Alert History" with different filters
3. Test "Configure Alerts" with different settings
4. Use "Clear Alert History" to reset (use with caution)

### 4. WebSocket Testing
1. Use Postman's WebSocket support or external tools
2. Connect to the WebSocket endpoint
3. Monitor real-time updates during market hours

## Example Test Cases

### Test Case 1: Dashboard Data Retrieval
```javascript
// Pre-request Script
pm.environment.set("test_symbol", "RELIANCE");

// Test Script
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Response has success field", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('success');
    pm.expect(jsonData.success).to.be.true;
});

pm.test("Data array is present", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('data');
    pm.expect(jsonData.data).to.be.an('array');
});
```

### Test Case 2: Alert Configuration
```javascript
// Pre-request Script
pm.environment.set("test_threshold", "0.15");

// Test Script
pm.test("Configuration updated successfully", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData.success).to.be.true;
    pm.expect(jsonData.message).to.include("updated successfully");
});
```

## WebSocket Testing

### Using Postman WebSocket
1. Create a new WebSocket request in Postman
2. Set URL to: `ws://localhost:8083/api/v1/bbw/live`
3. Connect and monitor messages

### Using Browser Console
```javascript
const ws = new WebSocket('ws://localhost:8083/api/v1/bbw/live');

ws.onopen = function() {
    console.log('Connected to BBW Dashboard WebSocket');
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('BBW Update:', data);
    
    if (data.type === 'bbw_dashboard_update') {
        console.log('Dashboard updated with', data.data.length, 'stocks');
    }
};

ws.onclose = function() {
    console.log('WebSocket connection closed');
};
```

### Using wscat (Command Line)
```bash
# Install wscat
npm install -g wscat

# Connect to WebSocket
wscat -c ws://localhost:8083/api/v1/bbw/live
```

## Error Handling

### Common Error Responses
- **400 Bad Request**: Missing required parameters
- **404 Not Found**: Stock or data not found
- **500 Internal Server Error**: Server-side issues

### Testing Error Scenarios
1. Test with invalid instrument keys
2. Test with missing required parameters
3. Test with invalid alert configurations
4. Test during non-market hours

## Performance Testing

### Load Testing
Use the collection with tools like:
- **Newman** (Postman CLI) for automated testing
- **Artillery** for load testing
- **JMeter** for performance testing

### Example Newman Command
```bash
# Install Newman
npm install -g newman

# Run collection
newman run BBW_Dashboard_API_Collection.json -e environment.json

# Run with reporting
newman run BBW_Dashboard_API_Collection.json -e environment.json --reporters cli,json --reporter-json-export results.json
```

## Monitoring and Logging

### API Logs
All API requests are logged to:
- `logs/setbull_trader_YYYY-MM-DD.log`
- JSON format for easy parsing
- Includes request details and performance metrics

### Log Analysis
```bash
# Monitor real-time logs
tail -f logs/setbull_trader_$(date +%Y-%m-%d).log | jq '.'

# Filter API requests
jq -r 'select(.subcomponent == "api_handler") | .action + " - " + .remote_addr' logs/setbull_trader_*.log

# Check for errors
jq -r 'select(.level == "error") | .message' logs/setbull_trader_*.log
```

## Troubleshooting

### Common Issues

#### 1. Connection Refused
- Verify the server is running on the correct port
- Check firewall settings
- Ensure the correct base URL is set

#### 2. No Data Returned
- Check if BBW monitoring is enabled
- Verify stock groups are configured
- Check market hours (9:15 AM - 3:30 PM IST)

#### 3. WebSocket Connection Issues
- Verify WebSocket endpoint is accessible
- Check for CORS issues in browser
- Ensure server supports WebSocket connections

#### 4. Alert Not Triggering
- Verify alert configuration
- Check alert thresholds
- Ensure audio alerts are enabled

### Debug Steps
1. Check server logs for errors
2. Verify configuration in `application.yaml`
3. Test individual endpoints
4. Monitor WebSocket connections
5. Check database connectivity

## Best Practices

### 1. Testing Strategy
- Test during market hours for real data
- Use realistic test data
- Test error scenarios
- Monitor performance

### 2. Environment Management
- Use separate environments for dev/staging/prod
- Keep sensitive data in environment variables
- Document environment configurations

### 3. Automation
- Automate regression testing
- Use CI/CD integration
- Monitor API health
- Set up alerting for failures

### 4. Documentation
- Keep collection updated
- Document test scenarios
- Maintain troubleshooting guides
- Update API documentation

## Support

For issues or questions:
1. Check the API documentation
2. Review server logs
3. Test with provided examples
4. Contact the development team

## Version History

- **v1.0.0** - Initial collection with all BBW Dashboard endpoints
- **v1.1.0** - Added WebSocket testing support
- **v1.2.0** - Enhanced error handling and documentation 