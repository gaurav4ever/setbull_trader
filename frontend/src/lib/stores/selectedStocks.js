// lib/stores/selectedStocks.js
import { writable, derived } from 'svelte/store';
import { getSelectedStocks, toggleStockSelection } from '../services/stocksService';

// Create a writable store to hold selected stocks
const createSelectedStocksStore = () => {
    const { subscribe, set, update } = writable({
        stocks: [], // Array of selected stock objects
        loading: false, // Loading state
        error: null, // Error message if any
        maxAllowed: 3 // Maximum allowed selected stocks
    });

    return {
        subscribe,

        // Load selected stocks from the API
        async loadSelectedStocks() {
            update(state => ({ ...state, loading: true, error: null }));

            try {
                const stocks = await getSelectedStocks();
                set({ stocks, loading: false, error: null, maxAllowed: 3 });
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

        // Toggle selection status of a stock
        async toggleSelection(stockId, isSelected) {
            update(state => ({ ...state, loading: true, error: null }));

            try {
                // If trying to select and already at max, prevent
                if (isSelected) {
                    const currentSelected = await getSelectedStocks();
                    if (currentSelected.length >= 3 && !currentSelected.some(s => s.id === stockId)) {
                        throw new Error('Maximum of 3 stocks can be selected');
                    }
                }

                // Call API to toggle selection
                await toggleStockSelection(stockId, isSelected);

                // Reload stocks to get updated state
                const stocks = await getSelectedStocks();
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

// Create a derived store that indicates if more stocks can be added
export const canAddMoreStocks = derived(
    selectedStocksStore,
    $selectedStocksStore => $selectedStocksStore.stocks.length < $selectedStocksStore.maxAllowed
);

// Create a derived store with just the count
export const selectedStocksCount = derived(
    selectedStocksStore,
    $selectedStocksStore => $selectedStocksStore.stocks.length
);

export default selectedStocksStore;