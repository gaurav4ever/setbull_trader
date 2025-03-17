// frontend/src/lib/utils/stockFormatting.js

/**
 * Formats a stock for display in autocomplete or other UI components
 * @param {Object} stock - Stock object with symbol and securityId
 * @returns {string} - Formatted display string
 */
export const formatStockForDisplay = (stock) => {
    if (!stock) return '';

    // If security ID is the same as symbol (old format), just show symbol
    if (stock.securityId === stock.symbol) {
        return stock.symbol;
    }

    // Otherwise show "SYMBOL - SECURITY_ID"
    return `${stock.symbol} - ${stock.securityId}`;
};

/**
 * Extracts symbol from a formatted stock display string
 * @param {string} displayString - Formatted stock display string
 * @returns {string} - Stock symbol
 */
export const extractSymbolFromDisplay = (displayString) => {
    if (!displayString) return '';

    // If it has the format "SYMBOL - SECURITY_ID", extract the symbol
    const parts = displayString.split(' - ');
    return parts[0].trim();
};

/**
 * Extracts security ID from a formatted stock display string or stock object
 * @param {string|Object} stockOrDisplayString - Stock object or formatted display string
 * @returns {string} - Security ID
 */
export const extractSecurityId = (stockOrDisplayString) => {
    if (!stockOrDisplayString) return '';

    // If it's an object, return the securityId property
    if (typeof stockOrDisplayString === 'object') {
        return stockOrDisplayString.securityId || '';
    }

    // If it's a string with the format "SYMBOL - SECURITY_ID", extract the security ID
    const parts = stockOrDisplayString.split(' - ');
    if (parts.length >= 2) {
        return parts[1].trim();
    }

    // Fallback to the string itself
    return stockOrDisplayString;
};

/**
 * Finds a stock by symbol in the stocks list
 * @param {Array} stocksList - List of stock objects
 * @param {string} symbol - Stock symbol to find
 * @returns {Object|null} - Found stock or null
 */
export const findStockBySymbol = (stocksList, symbol) => {
    if (!stocksList || !symbol) return null;
    return stocksList.find(stock => stock.symbol === symbol) || null;
};

export default {
    formatStockForDisplay,
    extractSymbolFromDisplay,
    extractSecurityId,
    findStockBySymbol
};