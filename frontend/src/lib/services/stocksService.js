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
    searchStocks
};