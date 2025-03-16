// frontend/src/lib/services/stocksService.js

// Global variable to hold our stocks list
let stocksList = [];

/**
 * Loads stocks from the nse_stocks.txt file
 * @returns {Promise<string[]>} Array of stock symbols
 */
const loadStocksFromFile = async () => {
    try {
        // Use fetch to load the stocks file
        const response = await fetch('/nse_stocks.txt');
        if (!response.ok) {
            console.error('Failed to load stocks file:', response.statusText);
            return [];
        }

        const text = await response.text();

        // Parse the file content - assuming each stock is on a new line
        return text
            .split('\n')
            .map(line => line.trim())
            .filter(line => line && line.length > 0); // Remove empty lines
    } catch (error) {
        console.error('Error loading stocks file:', error);
        return [];
    }
};

/**
 * Gets the list of stocks, loading from file if needed
 * @returns {Promise<string[]>} Array of stock symbols
 */
export const getStocksList = async () => {
    // If we've already loaded the stocks, return the cached list
    if (stocksList.length > 0) {
        return stocksList;
    }

    // Otherwise load from file
    stocksList = await loadStocksFromFile();
    return stocksList;
};

/**
 * Function to search stocks based on a query
 * @param {string} query The search query
 * @returns {string[]} Array of matching stock symbols (limited to 10)
 */
export const searchStocks = (query) => {
    if (!query || query.trim() === '') return [];

    const normalizedQuery = query.toLowerCase().trim();

    return stocksList
        .filter(stock => stock.toLowerCase().includes(normalizedQuery))
        .slice(0, 10); // Limit to 10 results for performance
};

/**
 * Creates a new stock
 * @param {Object} stockData - Stock data to create
 * @returns {Promise<Object>} Created stock object
 */
export const createStock = async (stockData) => {
    try {
        const response = await fetch('/api/v1/stocks', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(stockData)
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to create stock');
        }

        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error creating stock:', error);
        throw error;
    }
};

/**
 * Saves trade parameters for a stock
 * @param {Object} paramsData - Parameters data
 * @returns {Promise<Object>} Created parameters object
 */
export const saveTradeParameters = async (paramsData) => {
    try {
        const response = await fetch('/api/v1/parameters', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(paramsData)
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to save parameters');
        }

        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error saving parameters:', error);
        throw error;
    }
};

/**
 * Creates a stock and saves its parameters in one operation
 * @param {string} symbol - Stock symbol
 * @param {Object} parameters - Trading parameters
 * @returns {Promise<Object>} Object containing stock and parameters
 */
export const createStockWithParameters = async (symbol, parameters) => {
    try {
        // First create the stock
        const stockResponse = await createStock({
            symbol,
            name: symbol, // Use symbol as name for simplicity
            isSelected: true
        });

        if (!stockResponse.data || !stockResponse.data.id) {
            throw new Error('Failed to get stock ID from response');
        }

        // Then save the parameters
        const stockId = stockResponse.data.id;
        const paramData = {
            stockId,
            ...parameters
        };

        const paramResponse = await saveTradeParameters(paramData);

        return {
            stock: stockResponse.data,
            parameters: paramResponse.data
        };
    } catch (error) {
        console.error('Error creating stock with parameters:', error);
        throw error;
    }
};

/**
 * Fetches stock details by symbol
 * @param {string} symbol Stock symbol
 * @returns {Promise<Object>} Stock details
 */
export const getStockBySymbol = async (symbol) => {
    try {
        const response = await fetch(`/api/v1/stocks/symbol/${symbol}`);
        if (!response.ok) {
            throw new Error(`Failed to fetch stock: ${response.statusText}`);
        }
        const data = await response.json();
        return data.data; // Assuming response has a data property
    } catch (error) {
        console.error(`Error fetching stock ${symbol}:`, error);
        return null;
    }
};

/**
 * Fetches all selected stocks
 * @returns {Promise<Object[]>} Array of selected stock objects
 */
export const getSelectedStocks = async () => {
    try {
        const response = await fetch('/api/v1/stocks/selected');
        if (!response.ok) {
            throw new Error(`Failed to fetch selected stocks: ${response.statusText}`);
        }
        const data = await response.json();
        return data.data || []; // Assuming response has a data property
    } catch (error) {
        console.error('Error fetching selected stocks:', error);
        return [];
    }
};

/**
 * Toggles stock selection status
 * @param {string} stockId Stock ID
 * @param {boolean} isSelected Whether the stock should be selected
 * @returns {Promise<boolean>} Success status
 */
export const toggleStockSelection = async (stockId, isSelected) => {
    try {
        const response = await fetch(`/api/v1/stocks/${stockId}/toggle-selection`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ isSelected })
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || `Failed to toggle stock selection: ${response.statusText}`);
        }

        return true;
    } catch (error) {
        console.error(`Error toggling selection for stock ${stockId}:`, error);
        throw error; // Re-throw to let the caller handle it
    }
};

// Initialize stocks list on module load
loadStocksFromFile().then(stocks => {
    stocksList = stocks;
    console.log(`Loaded ${stocksList.length} stocks from file`);
}).catch(error => {
    console.error('Failed to preload stocks:', error);
});

// Export default for convenience
export default {
    getStocksList,
    searchStocks,
    getStockBySymbol,
    getSelectedStocks,
    toggleStockSelection,
    createStock,
    saveTradeParameters,
    createStockWithParameters
};