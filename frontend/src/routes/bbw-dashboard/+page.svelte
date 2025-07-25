<!-- BBW Dashboard Page -->
<script>
    import { onMount, onDestroy } from 'svelte';
    import { bbwDashboardStore } from '$lib/stores/bbwDashboardStore.js';
    import { formatNumber, formatCurrency } from '$lib/utils/formatting.js';
    import BBWAlertConfig from '$lib/components/BBWAlertConfig.svelte';
    import BBWAlertHistory from '$lib/components/BBWAlertHistory.svelte';
    
    // Subscribe to main store
    $: store = $bbwDashboardStore;
    $: ({ 
        stocks, 
        loading, 
        error, 
        searchTerm, 
        sortBy, 
        sortOrder, 
        filterBy,
        websocketConnected,
        lastUpdate,
        marketHours,
        currentTime
    } = store);
    
    // Subscribe to derived stores
    $: filteredStocks = $bbwDashboardStore.filteredStocks;
    $: stats = $bbwDashboardStore.dashboardStats;
    
    // Format current time
    $: formattedTime = currentTime ? currentTime.toLocaleTimeString('en-IN', { 
        hour12: false,
        timeZone: 'Asia/Kolkata'
    }) : '';
    
    // Format last update
    $: formattedLastUpdate = lastUpdate ? lastUpdate.toLocaleTimeString('en-IN', { 
        hour12: false,
        timeZone: 'Asia/Kolkata'
    }) : '';
    
    // Alert configuration state
    let showAlertConfig = false;
    let showAlertHistory = false;
    let currentAlertConfig = {
        alertThreshold: 0.1,
        contractingLookback: 5,
        enableAlerts: true
    };
    
    // Initialize dashboard on mount
    onMount(async () => {
        await bbwDashboardStore.initialize();
    });
    
    // Cleanup on destroy
    onDestroy(() => {
        bbwDashboardStore.cleanup();
    });
    
    // Handle search input
    function handleSearch(event) {
        bbwDashboardStore.setSearchTerm(event.target.value);
    }
    
    // Handle sort change
    function handleSortChange(event) {
        const [field, order] = event.target.value.split('-');
        bbwDashboardStore.setSort(field, order);
    }
    
    // Handle filter change
    function handleFilterChange(event) {
        bbwDashboardStore.setFilter(event.target.value);
    }
    
    // Get trend icon and color
    function getTrendDisplay(trend) {
        switch (trend) {
            case 'contracting':
                return { icon: '↓↓', color: 'text-red-600', bgColor: 'bg-red-50' };
            case 'expanding':
                return { icon: '↑↑', color: 'text-green-600', bgColor: 'bg-green-50' };
            case 'stable':
                return { icon: '→', color: 'text-blue-600', bgColor: 'bg-blue-50' };
            default:
                return { icon: '→', color: 'text-gray-600', bgColor: 'bg-gray-50' };
        }
    }
    
    // Get alert status display
    function getAlertDisplay(alertTriggered) {
        return alertTriggered 
            ? { icon: '🔔', color: 'text-orange-600', bgColor: 'bg-orange-50' }
            : { icon: '', color: 'text-gray-400', bgColor: 'bg-transparent' };
    }
    
    // Get distance color
    function getDistanceColor(distance) {
        if (distance <= 0.1) return 'text-red-600 font-semibold';
        if (distance <= 1.0) return 'text-orange-600';
        if (distance <= 5.0) return 'text-yellow-600';
        return 'text-gray-600';
    }
</script>

<svelte:head>
    <title>BBW Dashboard - SetBull Trader</title>
</svelte:head>

<!-- Main Dashboard Container -->
<div class="min-h-screen bg-gray-50">
    <!-- Header -->
    <div class="bg-white shadow-sm border-b">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-2xl font-bold text-gray-900">BBW Dashboard</h1>
                    <p class="text-sm text-gray-600">Bollinger Band Width Monitoring</p>
                </div>
                
                <!-- Market Status and Connection -->
                <div class="flex items-center space-x-4">
                    <div class="flex items-center space-x-2">
                        <div class="w-3 h-3 rounded-full {marketHours ? 'bg-green-500' : 'bg-red-500'}"></div>
                        <span class="text-sm font-medium">
                            Market: {marketHours ? 'OPEN' : 'CLOSED'}
                        </span>
                    </div>
                    
                    <div class="flex items-center space-x-2">
                        <div class="w-3 h-3 rounded-full {websocketConnected ? 'bg-green-500' : 'bg-red-500'}"></div>
                        <span class="text-sm font-medium">
                            {websocketConnected ? 'Connected' : 'Disconnected'}
                        </span>
                    </div>
                    
                    <div class="text-sm text-gray-500">
                        {formattedTime}
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Quick Stats -->
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="bg-white rounded-lg shadow p-4">
                <div class="text-sm font-medium text-gray-500">Monitored</div>
                <div class="text-2xl font-bold text-gray-900">{stats.totalStocks}</div>
            </div>
            
            <div class="bg-white rounded-lg shadow p-4">
                <div class="text-sm font-medium text-gray-500">In Range</div>
                <div class="text-2xl font-bold text-orange-600">{stats.alertedStocks}</div>
            </div>
            
            <div class="bg-white rounded-lg shadow p-4">
                <div class="text-sm font-medium text-gray-500">Contracting</div>
                <div class="text-2xl font-bold text-red-600">{stats.contractingStocks}</div>
            </div>
            
            <div class="bg-white rounded-lg shadow p-4">
                <div class="text-sm font-medium text-gray-500">Avg BBW</div>
                <div class="text-2xl font-bold text-gray-900">{stats.avgBBW}</div>
            </div>
        </div>
    </div>

    <!-- Controls -->
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="bg-white rounded-lg shadow p-4">
            <div class="flex flex-col sm:flex-row gap-4 items-center justify-between">
                <!-- Search -->
                <div class="flex-1 max-w-md">
                    <input
                        type="text"
                        placeholder="Search stocks..."
                        value={searchTerm}
                        on:input={handleSearch}
                        class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>
                
                <!-- Filters -->
                <div class="flex gap-2">
                    <select
                        value={filterBy}
                        on:change={handleFilterChange}
                        class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                        <option value="all">All Stocks</option>
                        <option value="alerted">Alerted</option>
                        <option value="contracting">Contracting</option>
                        <option value="expanding">Expanding</option>
                        <option value="stable">Stable</option>
                    </select>
                    
                    <select
                        value="{sortBy}-{sortOrder}"
                        on:change={handleSortChange}
                        class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                        <option value="distance_from_min_percent-asc">Distance ↑</option>
                        <option value="distance_from_min_percent-desc">Distance ↓</option>
                        <option value="current_bb_width-asc">BBW ↑</option>
                        <option value="current_bb_width-desc">BBW ↓</option>
                        <option value="symbol-asc">Symbol A-Z</option>
                        <option value="symbol-desc">Symbol Z-A</option>
                    </select>
                    
                    <!-- Alert Controls -->
                    <button
                        type="button"
                        on:click={() => showAlertConfig = true}
                        class="px-3 py-2 text-sm font-medium text-blue-600 bg-blue-50 border border-blue-200 rounded-md hover:bg-blue-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                        Alert Settings
                    </button>
                    
                    <button
                        type="button"
                        on:click={() => showAlertHistory = true}
                        class="px-3 py-2 text-sm font-medium text-orange-600 bg-orange-50 border border-orange-200 rounded-md hover:bg-orange-100 focus:outline-none focus:ring-2 focus:ring-orange-500"
                    >
                        Alert History
                    </button>
                </div>
            </div>
        </div>
    </div>

    <!-- Loading State -->
    {#if loading}
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <div class="text-center">
                <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
                <p class="mt-4 text-gray-600">Loading BBW data...</p>
            </div>
        </div>
    {:else if error}
        <!-- Error State -->
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <div class="bg-red-50 border border-red-200 rounded-lg p-4">
                <div class="flex">
                    <div class="flex-shrink-0">
                        <svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                        </svg>
                    </div>
                    <div class="ml-3">
                        <h3 class="text-sm font-medium text-red-800">Error loading dashboard</h3>
                        <p class="mt-1 text-sm text-red-700">{error}</p>
                    </div>
                </div>
            </div>
        </div>
    {:else}
        <!-- Stock List -->
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
            <div class="bg-white rounded-lg shadow overflow-hidden">
                <div class="px-6 py-4 border-b border-gray-200">
                    <div class="flex items-center justify-between">
                        <h2 class="text-lg font-medium text-gray-900">Stock List</h2>
                        <div class="text-sm text-gray-500">
                            Last update: {formattedLastUpdate}
                        </div>
                    </div>
                </div>
                
                <div class="overflow-x-auto">
                    <table class="min-w-full divide-y divide-gray-200">
                        <thead class="bg-gray-50">
                            <tr>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Symbol
                                </th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Current BBW
                                </th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Trend
                                </th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Min BBW
                                </th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Distance
                                </th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Alert
                                </th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                    Time
                                </th>
                            </tr>
                        </thead>
                        <tbody class="bg-white divide-y divide-gray-200">
                            {#each filteredStocks as stock}
                                {@const trendDisplay = getTrendDisplay(stock.bb_width_trend)}
                                {@const alertDisplay = getAlertDisplay(stock.alert_triggered)}
                                {@const distanceColor = getDistanceColor(stock.distance_from_min_percent)}
                                
                                <tr class="hover:bg-gray-50">
                                    <td class="px-6 py-4 whitespace-nowrap">
                                        <div class="text-sm font-medium text-gray-900">{stock.symbol}</div>
                                        <div class="text-sm text-gray-500">{stock.instrument_key}</div>
                                    </td>
                                    
                                    <td class="px-6 py-4 whitespace-nowrap">
                                        <div class="text-sm font-medium text-gray-900">
                                            {formatNumber(stock.current_bb_width, 4)}
                                        </div>
                                    </td>
                                    
                                    <td class="px-6 py-4 whitespace-nowrap">
                                        <div class="flex items-center">
                                            <span class="text-lg {trendDisplay.color}">
                                                {trendDisplay.icon}
                                            </span>
                                            <span class="ml-2 text-sm text-gray-600">
                                                {stock.contracting_sequence_count || 0}
                                            </span>
                                        </div>
                                    </td>
                                    
                                    <td class="px-6 py-4 whitespace-nowrap">
                                        <div class="text-sm text-gray-900">
                                            {formatNumber(stock.historical_min_bb_width, 4)}
                                        </div>
                                    </td>
                                    
                                    <td class="px-6 py-4 whitespace-nowrap">
                                        <div class="text-sm {distanceColor}">
                                            {formatNumber(stock.distance_from_min_percent, 1)}%
                                        </div>
                                    </td>
                                    
                                    <td class="px-6 py-4 whitespace-nowrap">
                                        <div class="text-lg {alertDisplay.color}">
                                            {alertDisplay.icon}
                                        </div>
                                    </td>
                                    
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {stock.timestamp ? new Date(stock.timestamp).toLocaleTimeString('en-IN', { 
                                            hour12: false,
                                            timeZone: 'Asia/Kolkata'
                                        }) : ''}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
                
                {#if filteredStocks.length === 0}
                    <div class="text-center py-8">
                        <p class="text-gray-500">No stocks found matching your criteria.</p>
                    </div>
                {/if}
            </div>
        </div>
    {/if}
    
    <!-- Alert Configuration Modal -->
    <BBWAlertConfig 
        show={showAlertConfig}
        currentConfig={currentAlertConfig}
        on:close={() => showAlertConfig = false}
        on:configUpdated={(event) => {
            currentAlertConfig = event.detail;
            showAlertConfig = false;
        }}
    />
    
    <!-- Alert History Modal -->
    <BBWAlertHistory 
        show={showAlertHistory}
        alertHistory={store.alerts}
        on:close={() => showAlertHistory = false}
        on:historyCleared={() => {
            // Refresh alert history
            bbwDashboardStore.refreshAlerts();
        }}
    />
</div> 