// frontend/src/lib/services/stocksService.js

// Global variable to hold our stocks list
let stocksList = [];

/**
 * Parses a line from the NSE stocks file
 * @param {string} line - A line from the stocks file
 * @returns {Object|null} - Parsed stock object or null if invalid
 */
const parseStockLine = (line) => {
    if (!line || line.trim() === '') return null;

    const parts = line.trim().split(',');
    if (parts.length >= 2) {
        // New format: SYMBOL,SECURITY_ID
        return {
            symbol: parts[0].trim(),
            securityId: parts[1].trim(),
            // Use symbol as display name
            name: parts[0].trim()
        };
    } else {
        // Old format: just symbol (fallback)
        const symbol = line.trim();
        return {
            symbol: symbol,
            securityId: symbol, // Use symbol as security ID
            name: symbol
        };
    }
};

/**
 * Loads stocks from the nse_stocks.txt file
 * @returns {Promise<Array>} Array of stock objects with symbol and securityId
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

        // Parse the file content - each stock can be on a new line
        return text
            .split('\n')
            .map(parseStockLine)
            .filter(stock => stock !== null); // Remove invalid entries
    } catch (error) {
        console.error('Error loading stocks file:', error);
        return [];
    }
};

/**
 * Gets the list of stocks, loading from file if needed
 * @returns {Promise<Array>} Array of stock objects
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
 * Gets the list of stock symbols only (for backward compatibility)
 * @returns {Promise<string[]>} Array of stock symbols
 */
export const getStocksSymbolsList = async () => {
    const stocks = await getStocksList();
    return stocks.map(stock => stock.symbol);
};

/**
 * Function to search stocks based on a query
 * @param {string} query The search query
 * @returns {Object[]} Array of matching stock objects (limited to 10)
 */
export const searchStocks = (query) => {
    if (!query || query.trim() === '') return [];

    const normalizedQuery = query.toLowerCase().trim();

    return stocksList
        .filter(stock =>
            stock.symbol.toLowerCase().includes(normalizedQuery) ||
            stock.name.toLowerCase().includes(normalizedQuery)
        )
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
 * Creates a stock and saves its parameters in one operation
 * @param {string} symbol - Stock symbol
 * @param {Object} parameters - Trading parameters
 * @returns {Promise<Object>} Object containing stock and parameters
 */
export const createStockWithParameters = async (symbol, parameters) => {
    try {
        // Find the stock in our list to get the security ID
        const stockInfo = stocksList.find(s => s.symbol === symbol);
        if (!stockInfo) {
            throw new Error(`Stock ${symbol} not found in stock list`);
        }

        // First create the stock
        const stockResponse = await createStock({
            symbol: stockInfo.symbol,
            name: stockInfo.name || stockInfo.symbol,
            securityId: stockInfo.securityId,
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

// Other methods remain unchanged...

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
    getStocksSymbolsList,
    searchStocks,
    // Other exports...
};