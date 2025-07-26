<!-- BBW Alert History Component -->
<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import apiService from '$lib/services/apiService.js';
    
    const dispatch = createEventDispatcher();
    
    // Props
    export let show = false;
    export let alertHistory: any[] = [];
    
    // Local state
    let loading = false;
    let error = '';
    let filteredAlerts: any[] = [];
    let filterType = '';
    let filterSymbol = '';
    let limit = 50;
    
    // Watch for prop changes
    $: if (show && alertHistory) {
        applyFilters();
    }
    
    // Apply filters to alert history
    function applyFilters() {
        // Ensure alertHistory is an array
        if (!Array.isArray(alertHistory)) {
            filteredAlerts = [];
            return;
        }
        
        filteredAlerts = alertHistory.filter(alert => {
            // Filter by alert type
            if (filterType && alert.AlertType !== filterType) {
                return false;
            }
            
            // Filter by symbol
            if (filterSymbol && !alert.Symbol.toLowerCase().includes(filterSymbol.toLowerCase())) {
                return false;
            }
            
            return true;
        }).slice(0, limit);
    }
    
    // Handle filter changes
    function handleFilterChange() {
        applyFilters();
    }
    
    // Clear filters
    function clearFilters() {
        filterType = '';
        filterSymbol = '';
        limit = 50;
        applyFilters();
    }
    
    // Clear alert history
    async function clearHistory() {
        if (!confirm('Are you sure you want to clear all alert history?')) {
            return;
        }
        
        loading = true;
        error = '';
        
        try {
            const response = await apiService.bbw.clearAlertHistory();
            
            if (response.status === 'success') {
                alertHistory = [];
                filteredAlerts = [];
                dispatch('historyCleared');
            } else {
                error = response.message || 'Failed to clear history';
            }
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to clear history';
            error = errorMessage;
        } finally {
            loading = false;
        }
    }
    
    // Get alert type display
    function getAlertTypeDisplay(alertType: string) {
        switch (alertType) {
            case 'threshold':
                return { label: 'Threshold', color: 'bg-blue-100 text-blue-800' };
            case 'pattern':
                return { label: 'Pattern', color: 'bg-green-100 text-green-800' };
            case 'squeeze':
                return { label: 'Squeeze', color: 'bg-red-100 text-red-800' };
            default:
                return { label: alertType, color: 'bg-gray-100 text-gray-800' };
        }
    }
    
    // Format timestamp
    function formatTimestamp(timestamp: string) {
        return new Date(timestamp).toLocaleString('en-IN', {
            timeZone: 'Asia/Kolkata',
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }
</script>

<!-- Modal Overlay -->
{#if show}
    <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] flex flex-col">
            <!-- Header -->
            <div class="flex items-center justify-between p-6 border-b">
                <h3 class="text-lg font-semibold text-gray-900">
                    Alert History ({Array.isArray(alertHistory) ? alertHistory.length : 0} alerts)
                </h3>
                <button 
                    type="button" 
                    class="text-gray-400 hover:text-gray-600"
                    on:click={() => dispatch('close')}
                    aria-label="Close alert history"
                >
                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                    </svg>
                </button>
            </div>
            
            <!-- Filters -->
            <div class="p-4 border-b bg-gray-50">
                <div class="flex flex-wrap gap-4 items-center">
                    <!-- Alert Type Filter -->
                    <div>
                        <label for="filterType" class="block text-sm font-medium text-gray-700 mb-1">
                            Alert Type
                        </label>
                        <select
                            id="filterType"
                            bind:value={filterType}
                            on:change={handleFilterChange}
                            class="px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                            <option value="">All Types</option>
                            <option value="threshold">Threshold</option>
                            <option value="pattern">Pattern</option>
                            <option value="squeeze">Squeeze</option>
                        </select>
                    </div>
                    
                    <!-- Symbol Filter -->
                    <div>
                        <label for="filterSymbol" class="block text-sm font-medium text-gray-700 mb-1">
                            Symbol
                        </label>
                        <input
                            id="filterSymbol"
                            type="text"
                            bind:value={filterSymbol}
                            on:input={handleFilterChange}
                            placeholder="Filter by symbol..."
                            class="px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>
                    
                    <!-- Limit -->
                    <div>
                        <label for="limit" class="block text-sm font-medium text-gray-700 mb-1">
                            Limit
                        </label>
                        <select
                            id="limit"
                            bind:value={limit}
                            on:change={handleFilterChange}
                            class="px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                            <option value={25}>25</option>
                            <option value={50}>50</option>
                            <option value={100}>100</option>
                        </select>
                    </div>
                    
                    <!-- Clear Filters -->
                    <div class="flex items-end">
                        <button
                            type="button"
                            on:click={clearFilters}
                            class="px-3 py-1 text-sm text-gray-600 hover:text-gray-800"
                        >
                            Clear Filters
                        </button>
                    </div>
                    
                    <!-- Clear History -->
                    <div class="flex items-end ml-auto">
                        <button
                            type="button"
                            on:click={clearHistory}
                            disabled={loading}
                            class="px-3 py-1 text-sm text-red-600 hover:text-red-800 disabled:opacity-50"
                        >
                            {loading ? 'Clearing...' : 'Clear History'}
                        </button>
                    </div>
                </div>
            </div>
            
            <!-- Content -->
            <div class="flex-1 overflow-auto p-6">
                <!-- Error Message -->
                {#if error}
                    <div class="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
                        <p class="text-sm text-red-600">{error}</p>
                    </div>
                {/if}
                
                <!-- Alert List -->
                {#if filteredAlerts.length > 0}
                    <div class="space-y-3">
                        {#each filteredAlerts as alert}
                            <div class="border border-gray-200 rounded-lg p-4 hover:bg-gray-50">
                                <div class="flex items-start justify-between">
                                    <div class="flex-1">
                                        <div class="flex items-center space-x-3 mb-2">
                                            <span class="text-lg font-semibold text-gray-900">
                                                {alert.Symbol}
                                            </span>
                                            <span class="px-2 py-1 text-xs font-medium rounded-full {getAlertTypeDisplay(alert.AlertType).color}">
                                                {getAlertTypeDisplay(alert.AlertType).label}
                                            </span>
                                            {#if alert.PatternLength}
                                                <span class="text-sm text-gray-600">
                                                    {alert.PatternLength} candles
                                                </span>
                                            {/if}
                                        </div>
                                        
                                        <p class="text-sm text-gray-700 mb-2">
                                            {alert.Message}
                                        </p>
                                        
                                        <div class="flex items-center space-x-4 text-xs text-gray-500">
                                            <span>BB Width: {alert.BBWidth?.toFixed(4) || 'N/A'}</span>
                                            <span>Min BB Width: {alert.LowestMinBBWidth?.toFixed(4) || 'N/A'}</span>
                                            <span>Time: {formatTimestamp(alert.Timestamp)}</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {/each}
                    </div>
                {:else}
                    <div class="text-center py-8">
                        <p class="text-gray-500">No alerts found</p>
                    </div>
                {/if}
            </div>
            
            <!-- Footer -->
            <div class="p-4 border-t bg-gray-50">
                <div class="flex items-center justify-between text-sm text-gray-600">
                    <span>Showing {filteredAlerts.length} of {Array.isArray(alertHistory) ? alertHistory.length : 0} alerts</span>
                    <button
                        type="button"
                        on:click={() => dispatch('close')}
                        class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                    >
                        Close
                    </button>
                </div>
            </div>
        </div>
    </div>
{/if} 