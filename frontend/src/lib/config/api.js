// API base URL configuration
// Use relative URL to leverage Vite proxy
export const API_BASE_URL = '/api/v1';

// Common API endpoints
export const ENDPOINTS = {
    // Order endpoints
    ORDERS: '/orders',
    ORDER_BY_ID: (id) => `/orders/${id}`,

    // Trade endpoints
    TRADES: '/trades',
    TRADE_HISTORY: '/trades/history',
    TRADE_HISTORY_WITH_PARAMS: (fromDate, toDate, page) =>
        `/trades/history?fromDate=${fromDate}&toDate=${toDate}&page=${page}`,

    // BBW Dashboard endpoints
    BBW_STOCKS: '/bbw/stocks',
    BBW_STOCK_HISTORY: (symbol) => `/bbw/stocks/${symbol}/history`,
    BBW_ALERTS_ACTIVE: '/bbw/alerts/active',
    BBW_ALERTS_CONFIGURE: '/bbw/alerts/configure',
    BBW_ALERT_HISTORY: '/bbw/alerts/history',
    BBW_STATISTICS: '/bbw/statistics',
    BBW_DASHBOARD_DATA: '/bbw/dashboard-data',

    // Health check endpoint
    HEALTH: '/health'
};

// Helper function to create a full API URL
export const apiUrl = (endpoint) => `${API_BASE_URL}${endpoint}`;

// Debug function to test API connectivity
export async function testApiConnection() {
    try {
        console.log(`Testing API connection to: ${API_BASE_URL}`);
        const response = await fetch(apiUrl(ENDPOINTS.HEALTH), {
            method: 'GET',
            cache: 'no-cache',
            headers: {
                'Accept': 'application/json'
            }
        });
        console.log(`API response status: ${response.status}`);
        return response.ok;
    } catch (error) {
        console.error('API connection test failed:', error);
        return false;
    }
}