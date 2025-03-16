// lib/stores/executionStatus.js
import { writable, derived } from 'svelte/store';

// Create a writable store for execution status
const createExecutionStatusStore = () => {
    const { subscribe, set, update } = writable({
        isExecuting: false,           // Whether execution is in progress
        lastExecutionTime: null,      // Timestamp of last execution
        results: [],                  // Array of execution results
        error: null,                  // Error message if execution failed
        activeExecutionId: null       // ID of active execution if any
    });

    return {
        subscribe,

        // Start execution
        startExecution() {
            update(state => ({
                ...state,
                isExecuting: true,
                error: null
            }));
        },

        // Set execution results
        setResults(results) {
            update(state => ({
                ...state,
                isExecuting: false,
                lastExecutionTime: new Date(),
                results: results || [],
                activeExecutionId: results && results.length > 0 ? results[0].id : null
            }));
        },

        // Set execution error
        setError(error) {
            update(state => ({
                ...state,
                isExecuting: false,
                error: error
            }));
        },

        // Add a single result
        addResult(result) {
            update(state => ({
                ...state,
                results: [...state.results, result]
            }));
        },

        // Update a result
        updateResult(id, updatedData) {
            update(state => {
                const updatedResults = state.results.map(result =>
                    result.id === id ? { ...result, ...updatedData } : result
                );

                return {
                    ...state,
                    results: updatedResults
                };
            });
        },

        // Clear all results
        clearResults() {
            update(state => ({
                ...state,
                results: [],
                activeExecutionId: null
            }));
        },

        // Clear error
        clearError() {
            update(state => ({
                ...state,
                error: null
            }));
        },

        // Reset the store to initial state
        reset() {
            set({
                isExecuting: false,
                lastExecutionTime: null,
                results: [],
                error: null,
                activeExecutionId: null
            });
        }
    };
};

// Create the store
export const executionStatusStore = createExecutionStatusStore();

// Create derived stores for common queries
export const isExecuting = derived(
    executionStatusStore,
    $status => $status.isExecuting
);

export const hasResults = derived(
    executionStatusStore,
    $status => $status.results && $status.results.length > 0
);

export const executionError = derived(
    executionStatusStore,
    $status => $status.error
);

export default executionStatusStore;