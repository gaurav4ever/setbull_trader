// lib/services/calculationService.js
import { roundToNearestFiveOrZero } from '../utils/formatting';

/**
 * Calculates Fibonacci levels for a trade
 * @param {Object} params - Trade parameters
 * @param {number} params.startingPrice - Trade starting price
 * @param {number} params.stopLossPercentage - Stop loss percentage
 * @param {string} params.tradeSide - Trade side (BUY or SELL)
 * @param {number} params.riskAmount - Risk amount
 * @returns {Promise<Object>} - Calculated levels and quantities
 */
export const calculateFibonacciLevels = async (params) => {
    try {
        // Construct the query parameters
        const queryParams = new URLSearchParams({
            startingPrice: params.startingPrice,
            slPercentage: params.stopLossPercentage,
            tradeSide: params.tradeSide,
            riskAmount: params.riskAmount || 30
        });

        // Make the API call
        const response = await fetch(`/api/v1/fibonacci/calculate?${queryParams.toString()}`);

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to calculate Fibonacci levels');
        }

        const data = await response.json();
        return data.data; // Assuming the response has a data property
    } catch (error) {
        console.error('Error calculating Fibonacci levels:', error);
        throw error;
    }
};

/**
 * Calculates Fibonacci levels locally without API
 * @param {Object} params - Trade parameters
 * @param {number} params.startingPrice - Trade starting price
 * @param {number} params.stopLossPercentage - Stop loss percentage
 * @param {string} params.tradeSide - Trade side (BUY or SELL)
 * @param {number} params.riskAmount - Risk amount
 * @returns {Object} - Calculated levels and quantities
 */
export const calculateFibonacciLevelsLocally = (params) => {
    const { startingPrice, stopLossPercentage, tradeSide, riskAmount = 30 } = params;

    // Fibonacci levels
    const fibLevels = [0, 1, 1.25, 1.5, 1.75, 2];

    // Calculate the stop loss price
    let slPrice;
    if (tradeSide === 'BUY') {
        slPrice = startingPrice * (1 - stopLossPercentage / 100);
    } else {
        slPrice = startingPrice * (1 + stopLossPercentage / 100);
    }

    // Round the stop loss price
    slPrice = roundToNearestFiveOrZero(slPrice);

    // Calculate the range for Fibonacci calculations
    const priceRange = Math.abs(startingPrice - slPrice);

    // Calculate SL points for position sizing
    const slPoints = tradeSide === 'BUY' ? (startingPrice - slPrice) : (slPrice - startingPrice);

    // Calculate total quantity based on risk
    const totalQuantity = Math.floor(riskAmount / slPoints);

    // Calculate quantity per leg (distribute across 5 entry legs)
    const legCount = 5;
    const baseQtyPerLeg = Math.floor(totalQuantity / legCount);
    const remainder = totalQuantity % legCount;

    // Initialize the result
    const levels = [];

    // Calculate execution levels
    for (let i = 0; i < fibLevels.length; i++) {
        const level = fibLevels[i];
        let price;
        let description;
        let quantity = 0;

        if (i === 0) {
            // Stop Loss level
            price = slPrice;
            description = "Stop Loss";
        } else if (i === 1) {
            // First entry is the trade price
            price = startingPrice;
            description = "1st Entry";
            quantity = baseQtyPerLeg + (0 < remainder ? 1 : 0);
        } else {
            // Calculate the price for additional entries
            if (tradeSide === 'BUY') {
                // For Buy, additional entries are above the trade price
                price = startingPrice + priceRange * (level - 1);
            } else {
                // For Sell, additional entries are below the trade price
                price = startingPrice - priceRange * (level - 1);
            }

            // Round the price
            price = roundToNearestFiveOrZero(price);

            // Set the description and quantity
            const entryNumber = i;
            description = `${getOrdinal(entryNumber)} Entry`;
            quantity = baseQtyPerLeg + ((i - 1) < remainder ? 1 : 0);
        }

        levels.push({
            level,
            price,
            description,
            quantity
        });
    }

    return {
        totalQuantity,
        levels
    };
};

/**
 * Creates an execution plan for a stock
 * @param {string} stockId - Stock ID
 * @returns {Promise<Object>} - Created execution plan
 */
export const createExecutionPlan = async (stockId) => {
    try {
        const response = await fetch(`/api/v1/plans/stock/${stockId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to create execution plan');
        }

        const data = await response.json();
        return data.data; // Assuming the response has a data property
    } catch (error) {
        console.error('Error creating execution plan:', error);
        throw error;
    }
};

/**
 * Get the execution plan for a stock
 * @param {string} stockId - Stock ID
 * @returns {Promise<Object>} - Execution plan
 */
export const getExecutionPlan = async (stockId) => {
    try {
        const response = await fetch(`/api/v1/plans/stock/${stockId}`);

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to get execution plan');
        }

        const data = await response.json();
        return data.data; // Assuming the response has a data property
    } catch (error) {
        console.error('Error getting execution plan:', error);
        return null;
    }
};

/**
 * Returns the ordinal suffix for a number (1st, 2nd, 3rd, etc.)
 * @param {number} n - The number
 * @returns {string} Number with ordinal suffix
 */
function getOrdinal(n) {
    if (n <= 0) return n.toString();

    const suffixes = ['th', 'st', 'nd', 'rd'];
    const mod100 = n % 100;
    const mod10 = n % 10;

    if (mod100 >= 11 && mod100 <= 13) {
        return `${n}th`;
    }

    switch (mod10) {
        case 1: return `${n}st`;
        case 2: return `${n}nd`;
        case 3: return `${n}rd`;
        default: return `${n}th`;
    }
}