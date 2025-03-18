// frontend/src/lib/stores/selectedStocks.js
// Update the store to handle stocks with security IDs

import { writable, derived } from 'svelte/store';
import { getSelectedStocks, toggleStockSelection, createStock } from '../services/stocksService';

// Create a writable store to hold selected stocks
const createSelectedStocksStore = () => {
    const { subscribe, set, update } = writable({
        stocks: [], // Array of selected stock objects
        loading: false, // Loading state
        error: null, // Error message if any
        maxAllowed: 3, // Maximum allowed selected stocks
        recentlyAddedId: null // ID of the most recently added stock
    });

    return {
        subscribe,

        // Load selected stocks from the API
        async loadSelectedStocks() {
            update(state => ({ ...state, loading: true, error: null }));

            try {
                const stocks = await getSelectedStocks(true);
                update(state => ({
                    ...state,
                    stocks,
                    loading: false,
                    error: null
                }));
                // set({ stocks, loading: false, error: null, maxAllowed: 3, recentlyAddedId: state.recentlyAddedId });
                return stocks;
            } catch (error) {
                console.error('Failed to load selected stocks:', error);
                update(state => ({
                    ...state,
                    loading: false,
                    error: error.message || 'Failed to load selected stocks'
                }));
                return [];
            }
        },

        // Add a stock directly without parameters
        async addStock(stockData) {
            update(state => ({ ...state, loading: true, error: null }));

            try {
                // Check if we've reached the maximum
                const currentSelected = await getSelectedStocks(false);
                if (currentSelected.length >= 3) {
                    throw new Error('Maximum of 3 stocks can be selected');
                }

                // Create the stock with isSelected=true
                const stockPayload = {
                    ...stockData,
                    isSelected: true
                };

                const response = await createStock(stockPayload);

                // Store the ID of the newly created stock
                const newStockId = response.data?.id;

                // Reload stocks to get updated state
                const stocks = await getSelectedStocks(true);

                set({
                    stocks,
                    loading: false,
                    error: null,
                    maxAllowed: 3,
                    recentlyAddedId: newStockId // Store the new stock ID
                });

                return newStockId;
            } catch (error) {
                console.error(`Failed to add stock:`, error);
                update(state => ({
                    ...state,
                    loading: false,
                    error: error.message || 'Failed to add stock'
                }));
                return null;
            }
        },


        // Create a new stock with parameters in one step
        async addStockWithParameters(stockSymbol, parameters) {
            update(state => ({ ...state, loading: true, error: null }));

            try {
                // Check if we've reached the maximum
                const currentSelected = await getSelectedStocks(false);
                if (currentSelected.length >= 3) {
                    throw new Error('Maximum of 3 stocks can be selected');
                }

                // Create stock with parameters
                await createStockWithParameters(stockSymbol, parameters);

                // Reload stocks to get updated state
                const stocks = await getSelectedStocks(true);
                set({ stocks, loading: false, error: null, maxAllowed: 3 });

                return true;
            } catch (error) {
                console.error(`Failed to add stock with parameters:`, error);
                update(state => ({
                    ...state,
                    loading: false,
                    error: error.message || 'Failed to add stock with parameters'
                }));
                return false;
            }
        },

        // Toggle selection status of a stock
        async toggleSelection(stockId, isSelected) {
            update(state => ({ ...state, loading: true, error: null }));

            try {
                // If trying to select and already at max, prevent
                if (isSelected) {
                    const currentSelected = await getSelectedStocks(false);
                    if (currentSelected.length >= 3 && !currentSelected.some(s => s.id === stockId)) {
                        throw new Error('Maximum of 3 stocks can be selected');
                    }
                }

                // Call API to toggle selection
                await toggleStockSelection(stockId, isSelected);

                // Reload stocks to get updated state
                const stocks = await getSelectedStocks(true);
                set({ stocks, loading: false, error: null, maxAllowed: 3 });

                return true;
            } catch (error) {
                console.error(`Failed to toggle stock selection for ${stockId}:`, error);
                update(state => ({
                    ...state,
                    loading: false,
                    error: error.message || 'Failed to update stock selection'
                }));
                return false;
            }
        },

        // Clear the recently added stock ID
        clearRecentlyAdded() {
            update(state => ({ ...state, recentlyAddedId: null }));
        },

        // Update a specific stock in the store without an API call
        updateStockLocally(stockId, updatedData) {
            update(state => {
                const updatedStocks = state.stocks.map(stock =>
                    stock.id === stockId ? { ...stock, ...updatedData } : stock
                );

                return { ...state, stocks: updatedStocks };
            });
        },

        // Clear any error message
        clearError() {
            update(state => ({ ...state, error: null }));
        }
    };
};

// Create the store
export const selectedStocksStore = createSelectedStocksStore();

// Create derived stores
export const canAddMoreStocks = derived(
    selectedStocksStore,
    $selectedStocksStore => $selectedStocksStore.stocks.length < $selectedStocksStore.maxAllowed
);

export const selectedStocksCount = derived(
    selectedStocksStore,
    $selectedStocksStore => $selectedStocksStore.stocks.length
);

export const recentlyAddedStockId = derived(
    selectedStocksStore,
    $selectedStocksStore => $selectedStocksStore.recentlyAddedId
);

export default selectedStocksStore;