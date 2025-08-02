// @ts-nocheck
import { writable, derived, get } from 'svelte/store';
import bbwWebSocketService from '../services/bbwWebSocketService.js';
import { bbwApi } from '../services/apiService.js';

// BBW Dashboard Store
function createBBWDashboardStore() {
    // Main state
    const { subscribe, set, update } = writable({
        // Dashboard data
        stocks: [],
        statistics: null,
        alerts: [],
        
        // UI state
        loading: false,
        error: null,
        searchTerm: '',
        sortBy: 'distance_from_min_percent',
        sortOrder: 'asc',
        filterBy: 'all', // all, alerted, contracting, expanding
        
        // Connection state
        websocketConnected: false,
        lastUpdate: null,
        
        // Market status
        marketHours: false,
        currentTime: new Date(),
        
        // NEW: Market status and latest data info
        marketStatus: null,
        lastDataTimestamp: null,
        lastDataAgeMinutes: -1
    });

    // Derived stores for filtered and sorted data
    const filteredStocks = derived(
        subscribe,
        ($store) => {
            console.log('Derived store function called with store:', $store);
            let stocks = [...$store.stocks];
            console.log('Stocks:', stocks);
            // Apply search filter
            console.log('Search term:', $store.searchTerm);
            if ($store.searchTerm) {
                const term = $store.searchTerm.toLowerCase();
                stocks = stocks.filter(stock => 
                    stock.symbol.toLowerCase().includes(term) ||
                    stock.instrument_key.toLowerCase().includes(term)
                );
            }

            // Apply category filter
            console.log('Filter by:', $store.filterBy);
            switch ($store.filterBy) {
                case 'alerted':
                    stocks = stocks.filter(stock => stock.alert_triggered);
                    break;
                case 'contracting':
                    stocks = stocks.filter(stock => stock.bb_width_trend === 'contracting');
                    break;
                case 'expanding':
                    stocks = stocks.filter(stock => stock.bb_width_trend === 'expanding');
                    break;
                case 'stable':
                    stocks = stocks.filter(stock => stock.bb_width_trend === 'stable');
                    break;
            }
            
            // Apply sorting
            stocks.sort((a, b) => {
                let aVal = a[$store.sortBy];
                let bVal = b[$store.sortBy];
                
                // Handle numeric values
                if (typeof aVal === 'number' && typeof bVal === 'number') {
                    return $store.sortOrder === 'asc' ? aVal - bVal : bVal - aVal;
                }
                
                // Handle string values
                if (typeof aVal === 'string' && typeof bVal === 'string') {
                    return $store.sortOrder === 'asc' 
                        ? aVal.localeCompare(bVal) 
                        : bVal.localeCompare(aVal);
                }
                
                return 0;
            });

            console.log('Filtered stocks:', stocks);
            
            return stocks;
        }
    );

    // Dashboard statistics
    const dashboardStats = derived(
        subscribe,
        ($store) => {
            console.log('Dashboard stats derived store called with stocks:', $store.stocks?.length || 0);
            const stocks = $store.stocks;
            const totalStocks = stocks.length;
            const alertedStocks = stocks.filter(s => s.alert_triggered).length;
            const contractingStocks = stocks.filter(s => s.bb_width_trend === 'contracting').length;
            const expandingStocks = stocks.filter(s => s.bb_width_trend === 'expanding').length;
            const stableStocks = stocks.filter(s => s.bb_width_trend === 'stable').length;
            
            // Calculate average BBW
            const avgBBW = stocks.length > 0 
                ? stocks.reduce((sum, stock) => sum + stock.current_bb_width, 0) / stocks.length 
                : 0;
            
            return {
                totalStocks,
                alertedStocks,
                contractingStocks,
                expandingStocks,
                stableStocks,
                avgBBW: avgBBW.toFixed(4)
            };
        }
    );

    // Actions
    const actions = {
        // Initialize dashboard
        async initialize() {
            console.log('Initializing BBW Dashboard store...');
            update(state => ({ ...state, loading: true, error: null }));
            
            try {
                // Load market status first
                console.log('Loading market status...');
                await actions.loadMarketStatus();
                
                // Load initial data
                console.log('Loading dashboard data...');
                await actions.loadDashboardData();
                console.log('Loading statistics...');
                await actions.loadStatistics();
                console.log('Loading active alerts...');
                await actions.loadActiveAlerts();
                
                // Connect WebSocket only during market hours
                const currentState = get({ subscribe });
                if (currentState.marketHours) {
                    bbwWebSocketService.connect();
                    
                    // Setup WebSocket listeners
                    bbwWebSocketService.addEventListener('connected', actions.handleWebSocketConnected);
                    bbwWebSocketService.addEventListener('disconnected', actions.handleWebSocketDisconnected);
                    bbwWebSocketService.addEventListener('bbw_update', actions.handleBBWUpdate);
                    bbwWebSocketService.addEventListener('alert_triggered', actions.handleAlertTriggered);
                    bbwWebSocketService.addEventListener('market_status', actions.handleMarketStatus);
                }
                
                update(state => ({ ...state, loading: false }));
            } catch (error) {
                console.error('Failed to initialize BBW dashboard:', error);
                console.error('Initialization error details:', {
                    message: error.message,
                    stack: error.stack,
                    name: error.name
                });
                update(state => ({ 
                    ...state, 
                    loading: false, 
                    error: error.message || 'Failed to initialize dashboard' 
                }));
            }
        },

        // Load dashboard data
        async loadDashboardData() {
            try {
                console.log('Loading dashboard data...');
                // Try to get real-time data first (during market hours)
                let data;
                try {
                    console.log('Trying dashboard data...');
                    data = await bbwApi.getDashboardData();
                    if (data.data == null || data.data.length === 0) {
                        throw new Error('No data returned from API');
                    }
                    console.log('Dashboard data loaded:', data);
                } catch (error) {
                    console.log('Real-time data not available, trying latest available day data...');
                    // If real-time data fails, try to get latest available day data
                    data = await bbwApi.getLatestAvailableDayData();
                    console.log('Latest day data loaded:', data);
                }
                
                console.log('About to update store with stocks:', data.data?.length || 0);
                update(state => {
                    console.log('Current state stocks length:', state.stocks.length);
                    const newState = { 
                        ...state, 
                        stocks: data.data || [],
                        loading: false,
                        lastUpdate: new Date()
                    };
                    console.log('New state stocks length:', newState.stocks.length);
                    return newState;
                });
                console.log('Store updated successfully');
            } catch (error) {
                console.error('Failed to load dashboard data:', error);
                console.error('Error details:', {
                    message: error.message,
                    stack: error.stack,
                    name: error.name
                });
                throw error;
            }
        },

        // Load statistics
        async loadStatistics() {
            try {
                const stats = await bbwApi.getStatistics();
                update(state => ({ ...state, statistics: stats }));
            } catch (error) {
                console.error('Failed to load statistics:', error);
            }
        },

        // Load active alerts
        async loadActiveAlerts() {
            try {
                const response = await bbwApi.getActiveAlerts();
                const alerts = Array.isArray(response.data) ? response.data : [];
                update(state => ({ ...state, alerts }));
            } catch (error) {
                console.error('Failed to load active alerts:', error);
                // Ensure alerts is always an array even on error
                update(state => ({ ...state, alerts: [] }));
            }
        },

        // NEW: Load market status
        async loadMarketStatus() {
            try {
                const response = await bbwApi.getMarketStatus();
                if (response.success) {
                    update(state => ({
                        ...state,
                        marketStatus: response.data,
                        marketHours: response.data.market_open,
                        lastDataTimestamp: response.data.last_data_timestamp,
                        lastDataAgeMinutes: response.data.last_data_age_minutes
                    }));
                }
            } catch (error) {
                console.error('Failed to load market status:', error);
            }
        },

        // Update search term
        setSearchTerm(term) {
            update(state => ({ ...state, searchTerm: term }));
        },

        // Update sort settings
        setSort(sortBy, sortOrder = 'asc') {
            console.log('Store: Setting sort to', sortBy, sortOrder);
            update(state => ({ ...state, sortBy, sortOrder }));
        },

        // Update filter
        setFilter(filterBy) {
            update(state => ({ ...state, filterBy }));
        },

        // WebSocket event handlers
        handleWebSocketConnected(data) {
            update(state => ({ ...state, websocketConnected: true }));
        },

        handleWebSocketDisconnected(data) {
            update(state => ({ ...state, websocketConnected: false }));
        },

        handleBBWUpdate(data) {
            update(state => {
                // Handle bulk update (array of stocks)
                if (Array.isArray(data)) {
                    return { 
                        ...state, 
                        stocks: data,
                        lastUpdate: new Date()
                    };
                }
                
                // Handle individual stock update (legacy format)
                const updatedStocks = state.stocks.map(stock => {
                    if (stock.instrument_key === data.instrument_key) {
                        return { ...stock, ...data };
                    }
                    return stock;
                });
                
                return { 
                    ...state, 
                    stocks: updatedStocks,
                    lastUpdate: new Date()
                };
            });
        },

        handleAlertTriggered(data) {
            update(state => {
                // Update stock alert status
                const updatedStocks = state.stocks.map(stock => {
                    if (stock.instrument_key === data.instrument_key) {
                        return { 
                            ...stock, 
                            alert_triggered: true,
                            alert_triggered_at: new Date().toISOString()
                        };
                    }
                    return stock;
                });
                
                // Add to alerts list
                const newAlert = {
                    id: Date.now(),
                    Symbol: data.symbol,
                    BBWidth: data.current_bb_width,
                    LowestMinBBWidth: data.historical_min_bb_width || 0,
                    PatternLength: data.contracting_sequence_count || 0,
                    AlertType: data.alert_type || 'threshold',
                    Timestamp: new Date(),
                    GroupID: data.instrument_key,
                    Message: `BB Width alert triggered for ${data.symbol} at ${data.current_bb_width?.toFixed(4)}`
                };
                
                return { 
                    ...state, 
                    stocks: updatedStocks,
                    alerts: [newAlert, ...state.alerts.slice(0, 9)] // Keep last 10 alerts
                };
            });
        },

        handleMarketStatus(data) {
            update(state => ({ 
                ...state, 
                marketHours: data.market_hours || false,
                currentTime: new Date()
            }));
        },

        // Configure alerts for a stock
        async configureAlerts(instrumentKey, config) {
            try {
                await bbwApi.configureAlerts({
                    instrument_key: instrumentKey,
                    ...config
                });
                
                // Update local state
                update(state => {
                    const updatedStocks = state.stocks.map(stock => {
                        if (stock.instrument_key === instrumentKey) {
                            return { ...stock, alert_config: config };
                        }
                        return stock;
                    });
                    
                    return { ...state, stocks: updatedStocks };
                });
            } catch (error) {
                console.error('Failed to configure alerts:', error);
                throw error;
            }
        },

        // Refresh alerts
        async refreshAlerts() {
            try {
                const response = await bbwApi.getAlertHistory();
                const alerts = Array.isArray(response.data) ? response.data : [];
                update(state => ({ ...state, alerts }));
            } catch (error) {
                console.error('Failed to refresh alerts:', error);
                // Ensure alerts is always an array even on error
                update(state => ({ ...state, alerts: [] }));
            }
        },

        // Cleanup
        cleanup() {
            bbwWebSocketService.removeEventListener('connected', actions.handleWebSocketConnected);
            bbwWebSocketService.removeEventListener('disconnected', actions.handleWebSocketDisconnected);
            bbwWebSocketService.removeEventListener('bbw_update', actions.handleBBWUpdate);
            bbwWebSocketService.removeEventListener('alert_triggered', actions.handleAlertTriggered);
            bbwWebSocketService.removeEventListener('market_status', actions.handleMarketStatus);
            
            bbwWebSocketService.disconnect();
        }
    };

    return {
        subscribe,
        ...actions,
        filteredStocks,
        dashboardStats
    };
}

// Create and export the store
export const bbwDashboardStore = createBBWDashboardStore(); 