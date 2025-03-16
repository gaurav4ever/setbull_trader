// lib/services/executionService.js

/**
 * Execute orders for a stock
 * @param {string} stockId - Stock ID
 * @returns {Promise<Object>} - Execution result
 */
export const executeOrdersForStock = async (stockId) => {
    try {
        const response = await fetch(`/api/v1/execute/stock/${stockId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to execute orders');
        }

        const data = await response.json();
        return data.data; // Assuming the response has a data property
    } catch (error) {
        console.error('Error executing orders:', error);
        throw error;
    }
};

/**
 * Execute orders for all selected stocks
 * @returns {Promise<Object[]>} - Array of execution results
 */
export const executeOrdersForAllSelectedStocks = async () => {
    try {
        const response = await fetch('/api/v1/execute/all', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to execute orders');
        }

        const data = await response.json();
        return data.data; // Assuming the response has a data property
    } catch (error) {
        console.error('Error executing orders for all stocks:', error);
        throw error;
    }
};

/**
 * Get order execution by ID
 * @param {string} executionId - Execution ID
 * @returns {Promise<Object>} - Execution details
 */
export const getOrderExecution = async (executionId) => {
    try {
        const response = await fetch(`/api/v1/executions/${executionId}`);

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to get execution details');
        }

        const data = await response.json();
        return data.data; // Assuming the response has a data property
    } catch (error) {
        console.error('Error getting execution details:', error);
        throw error;
    }
};

/**
 * Get order executions for a plan
 * @param {string} planId - Execution plan ID
 * @returns {Promise<Object[]>} - Array of execution details
 */
export const getOrderExecutionsForPlan = async (planId) => {
    try {
        const response = await fetch(`/api/v1/executions/plan/${planId}`);

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to get executions for plan');
        }

        const data = await response.json();
        return data.data; // Assuming the response has a data property
    } catch (error) {
        console.error('Error getting executions for plan:', error);
        return [];
    }
};

export default {
    executeOrdersForStock,
    executeOrdersForAllSelectedStocks,
    getOrderExecution,
    getOrderExecutionsForPlan
};