// lib/utils/formatting.js

/**
 * Formats a number as currency (INR)
 * @param {number} value - The number to format
 * @param {boolean} showSymbol - Whether to include the currency symbol
 * @returns {string} Formatted currency string
 */
export function formatCurrency(value, showSymbol = true) {
    if (value === null || value === undefined || isNaN(value)) {
        return '—';
    }

    return new Intl.NumberFormat('en-IN', {
        style: showSymbol ? 'currency' : 'decimal',
        currency: 'INR',
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
    }).format(value);
}

/**
 * Formats a number with specified precision
 * @param {number} value - The number to format
 * @param {number} precision - Number of decimal places
 * @returns {string} Formatted number string
 */
export function formatNumber(value, precision = 2) {
    if (value === null || value === undefined || isNaN(value)) {
        return '—';
    }

    return new Intl.NumberFormat('en-IN', {
        minimumFractionDigits: precision,
        maximumFractionDigits: precision
    }).format(value);
}

/**
 * Formats a price to round to nearest 0.05 or 0.00
 * @param {number} price - The price to format
 * @returns {number} Price rounded to nearest 0.05 or 0.00
 */
export function roundToNearestFiveOrZero(price) {
    // Multiply by 100 to work with integers
    const scaled = price * 100;

    // Round to nearest integer
    const rounded = Math.round(scaled);

    // Get the last digit
    const lastDigit = rounded % 10;

    // Adjust to nearest 0 or 5
    let adjusted;
    if (lastDigit < 3) {
        adjusted = rounded - lastDigit;
    } else if (lastDigit < 8) {
        adjusted = rounded - lastDigit + 5;
    } else {
        adjusted = rounded - lastDigit + 10;
    }

    // Convert back to original scale
    return adjusted / 100;
}

/**
 * Returns the ordinal suffix for a number (1st, 2nd, 3rd, etc.)
 * @param {number} n - The number
 * @returns {string} Number with ordinal suffix
 */
export function getOrdinal(n) {
    if (n <= 0) return n.toString();

    const suffixes = ['th', 'st', 'nd', 'rd'];
    const v = n % 100;
    return n + (suffixes[(v - 20) % 10] || suffixes[v] || suffixes[0]);
}