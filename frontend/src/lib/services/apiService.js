import { apiUrl, ENDPOINTS, testApiConnection } from '../config/api';

// Global connectivity status
let isApiConnected = null;

// Test API connectivity on service initialization
async function checkApiConnectivity() {
    if (isApiConnected === null) {
        isApiConnected = await testApiConnection();
        console.log(`API connectivity check result: ${isApiConnected ? 'Connected' : 'Not connected'}`);
    }
    return isApiConnected;
}

// Initialize connection check 
checkApiConnectivity();

// Generic request function with error handling
async function request(url, options = {}) {
    // First check if API is reachable
    const isConnected = await checkApiConnectivity();
    if (!isConnected) {
        throw new Error('API server is not reachable. Check if the backend is running.');
    }

    try {
        console.log(`Making API request to: ${url}`);
        const response = await fetch(url, {
            ...options,
            mode: 'cors', // Enable CORS
            credentials: 'omit', // Don't send cookies
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
                ...options.headers
            }
        });

        console.log(`API response status: ${response.status}`);

        // Check if the response is not OK
        if (!response.ok) {
            // Try to parse error JSON if present
            try {
                const errorData = await response.json();
                throw new Error(errorData.error || `Request failed with status ${response.status}`);
            } catch (parseError) {
                // If we can't parse JSON, use status text
                throw new Error(`Request failed with status ${response.status}: ${response.statusText}`);
            }
        }

        // For successful responses, parse and return JSON
        const data = await response.json();
        console.log('API response data:', data);
        return data;
    } catch (error) {
        console.error('API request failed:', error);
        // Update global connectivity status if network error 
        if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
            isApiConnected = false;
        }
        throw error;
    }
}

// Order API functions
export const orderApi = {
    // Place a new order
    placeOrder: async (orderData) => {
        return request(apiUrl(ENDPOINTS.ORDERS), {
            method: 'POST',
            body: JSON.stringify(orderData)
        });
    },

    // Modify an existing order
    modifyOrder: async (orderId, orderData) => {
        return request(apiUrl(ENDPOINTS.ORDER_BY_ID(orderId)), {
            method: 'PUT',
            body: JSON.stringify(orderData)
        });
    },

    // Cancel an order
    cancelOrder: async (orderId) => {
        return request(apiUrl(ENDPOINTS.ORDER_BY_ID(orderId)), {
            method: 'DELETE'
        });
    }
};

// Trade API functions
export const tradeApi = {
    // Get all trades for today
    getAllTrades: async () => {
        return request(apiUrl(ENDPOINTS.TRADES));
    },

    // Get trade history with optional filtering
    getTradeHistory: async (fromDate, toDate, page = 0) => {
        return request(apiUrl(ENDPOINTS.TRADE_HISTORY_WITH_PARAMS(fromDate, toDate, page)));
    }
};

export default {
    order: orderApi,
    trade: tradeApi,
    checkConnection: checkApiConnectivity
};