import { writable, derived } from 'svelte/store';
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
        currentTime: new Date()
    });

    // Derived stores for filtered and sorted data
    const filteredStocks = derived(
        [subscribe],
        ([$store]) => {
            let stocks = [...$store.stocks];
            
            // Apply search filter
            if ($store.searchTerm) {
                const term = $store.searchTerm.toLowerCase();
                stocks = stocks.filter(stock => 
                    stock.symbol.toLowerCase().includes(term) ||
                    stock.instrument_key.toLowerCase().includes(term)
                );
            }
            
            // Apply category filter
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
            
            return stocks;
        }
    );

    // Dashboard statistics
    const dashboardStats = derived(
        [subscribe],
        ([$store]) => {
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
            update(state => ({ ...state, loading: true, error: null }));
            
            try {
                // Connect WebSocket
                bbwWebSocketService.connect();
                
                // Load initial data
                await actions.loadDashboardData();
                await actions.loadStatistics();
                await actions.loadActiveAlerts();
                
                // Setup WebSocket listeners
                bbwWebSocketService.addEventListener('connected', actions.handleWebSocketConnected);
                bbwWebSocketService.addEventListener('disconnected', actions.handleWebSocketDisconnected);
                bbwWebSocketService.addEventListener('bbw_update', actions.handleBBWUpdate);
                bbwWebSocketService.addEventListener('alert_triggered', actions.handleAlertTriggered);
                bbwWebSocketService.addEventListener('market_status', actions.handleMarketStatus);
                
                update(state => ({ ...state, loading: false }));
            } catch (error) {
                console.error('Failed to initialize BBW dashboard:', error);
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
                const data = await bbwApi.getDashboardData();
                update(state => ({ 
                    ...state, 
                    stocks: data.stocks || [],
                    lastUpdate: new Date()
                }));
            } catch (error) {
                console.error('Failed to load dashboard data:', error);
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
                const alerts = await bbwApi.getActiveAlerts();
                update(state => ({ ...state, alerts: alerts || [] }));
            } catch (error) {
                console.error('Failed to load active alerts:', error);
            }
        },

        // Update search term
        setSearchTerm(term) {
            update(state => ({ ...state, searchTerm: term }));
        },

        // Update sort settings
        setSort(sortBy, sortOrder = 'asc') {
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
                    instrument_key: data.instrument_key,
                    symbol: data.symbol,
                    bb_width: data.current_bb_width,
                    triggered_at: new Date().toISOString(),
                    type: data.alert_type || 'threshold'
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
        filteredStocks: { subscribe: filteredStocks.subscribe },
        dashboardStats: { subscribe: dashboardStats.subscribe }
    };
}

// Create and export the store
export const bbwDashboardStore = createBBWDashboardStore(); 