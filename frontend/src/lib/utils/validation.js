// lib/utils/validation.js

/**
 * Check if a value is empty (null, undefined, empty string, or just whitespace)
 * @param {*} value - The value to check
 * @returns {boolean} True if the value is empty
 */
export const isEmpty = (value) => {
    return value === null || value === undefined || (typeof value === 'string' && value.trim() === '');
};

/**
 * Validates that a value is not empty
 * @param {*} value - The value to validate
 * @param {string} fieldName - Name of the field for error message
 * @returns {string|null} Error message or null if valid
 */
export const validateRequired = (value, fieldName = 'This field') => {
    return isEmpty(value) ? `${fieldName} is required` : null;
};

/**
 * Validates that a value is a number
 * @param {*} value - The value to validate
 * @param {string} fieldName - Name of the field for error message
 * @returns {string|null} Error message or null if valid
 */
export const validateNumber = (value, fieldName = 'This field') => {
    if (isEmpty(value)) return null; // Skip if empty (use validateRequired for required fields)

    const num = Number(value);
    return isNaN(num) ? `${fieldName} must be a valid number` : null;
};

/**
 * Validates that a number is within a range
 * @param {number} value - The value to validate
 * @param {number} min - Minimum value (inclusive)
 * @param {number} max - Maximum value (inclusive)
 * @param {string} fieldName - Name of the field for error message
 * @returns {string|null} Error message or null if valid
 */
export const validateRange = (value, min, max, fieldName = 'This value') => {
    if (isEmpty(value)) return null; // Skip if empty

    const num = Number(value);
    if (isNaN(num)) return null; // Skip if not a number (use validateNumber first)

    if (min !== null && num < min) {
        return `${fieldName} must be at least ${min}`;
    }

    if (max !== null && num > max) {
        return `${fieldName} must be no more than ${max}`;
    }

    return null;
};

/**
 * Validates a stop loss percentage (typically between 0-5%)
 * @param {number} value - The value to validate
 * @returns {string|null} Error message or null if valid
 */
export const validateStopLossPercentage = (value) => {
    if (isEmpty(value)) return 'Stop loss percentage is required';

    const num = Number(value);
    if (isNaN(num)) return 'Stop loss percentage must be a valid number';

    if (num <= 0) {
        return 'Stop loss percentage must be greater than 0';
    }

    if (num > 5) {
        return 'Stop loss percentage should not exceed 5%';
    }

    return null;
};

/**
 * Validates the starting price
 * @param {number} value - The value to validate
 * @returns {string|null} Error message or null if valid
 */
export const validateStartingPrice = (value) => {
    if (isEmpty(value)) return 'Starting price is required';

    const num = Number(value);
    if (isNaN(num)) return 'Starting price must be a valid number';

    if (num <= 0) {
        return 'Starting price must be greater than 0';
    }

    return null;
};

/**
 * Validates the risk amount
 * @param {number} value - The value to validate
 * @returns {string|null} Error message or null if valid
 */
export const validateRiskAmount = (value) => {
    if (isEmpty(value)) return 'Risk amount is required';

    const num = Number(value);
    if (isNaN(num)) return 'Risk amount must be a valid number';

    if (num <= 0) {
        return 'Risk amount must be greater than 0';
    }

    return null;
};

/**
 * Validates all trading parameters
 * @param {Object} params - The parameters to validate
 * @returns {Object} Object with field errors
 */
export const validateTradingParameters = (params) => {
    const errors = {};

    const startingPriceError = validateStartingPrice(params.startingPrice);
    if (startingPriceError) errors.startingPrice = startingPriceError;

    const stopLossError = validateStopLossPercentage(params.stopLossPercentage);
    if (stopLossError) errors.stopLossPercentage = stopLossError;

    const riskAmountError = validateRiskAmount(params.riskAmount);
    if (riskAmountError) errors.riskAmount = riskAmountError;

    if (isEmpty(params.tradeSide)) {
        errors.tradeSide = 'Trade side is required';
    } else if (!['BUY', 'SELL'].includes(params.tradeSide)) {
        errors.tradeSide = 'Trade side must be either BUY or SELL';
    }

    return errors;
};

/**
 * Checks if an object has any properties (used to check for validation errors)
 * @param {Object} obj - The object to check
 * @returns {boolean} True if the object has properties
 */
export const hasErrors = (obj) => {
    return Object.keys(obj).length > 0;
};