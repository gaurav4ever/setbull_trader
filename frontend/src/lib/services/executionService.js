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

        // Transform API response to the format expected by the UI
        if (data.success && data.data) {
            // Combine execution metadata with results for easier rendering
            const transformedResults = transformExecutionResults(data.data);
            return transformedResults;
        } else {
            throw new Error(data.error || 'Unknown error during execution');
        }
    } catch (error) {
        console.error('Error executing orders for all stocks:', error);
        throw error;
    }
};

/**
 * Transforms the API response into a format compatible with the UI components
 * @param {Object} apiData - The API response data
 * @returns {Array} - Transformed execution results
 */
function transformExecutionResults(apiData) {
    const { execution, results } = apiData;

    // Map results to include execution metadata
    return results.map((result, index) => {
        const executionData = execution[index] || {};

        // Create a combined result object with all needed data
        return {
            id: executionData.id || result.ExecutionID,
            executionPlanId: executionData.executionPlanId,
            status: executionData.status,
            executedAt: executionData.executedAt,
            errorMessage: executionData.errorMessage,
            stock: {
                symbol: result.StockSymbol
            },
            success: result.Success,
            // Transform order results to match the expected format
            orders: result.Results.map(orderResult => ({
                description: orderResult.LevelDescription,
                orderId: orderResult.OrderID,
                status: orderResult.Success ? 'COMPLETED' : 'FAILED',
                price: 0, // Not provided in API response
                quantity: 0, // Not provided in API response
                error: orderResult.Error
            }))
        };
    });
}

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