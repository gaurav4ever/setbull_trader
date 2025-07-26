<!-- BBW Stock Card Component -->
<script>
    import { createEventDispatcher } from 'svelte';
    import { formatNumber } from '$lib/utils/formatting.js';
    
    // Props
    export let stock = {};
    export let showDetails = false;
    
    const dispatch = createEventDispatcher();
    
    // Get trend display
    function getTrendDisplay(trend) {
        switch (trend) {
            case 'contracting':
                return { 
                    icon: '↓↓', 
                    color: 'text-red-600', 
                    bgColor: 'bg-red-50',
                    label: 'Contracting'
                };
            case 'expanding':
                return { 
                    icon: '↑↑', 
                    color: 'text-green-600', 
                    bgColor: 'bg-green-50',
                    label: 'Expanding'
                };
            case 'stable':
                return { 
                    icon: '→', 
                    color: 'text-blue-600', 
                    bgColor: 'bg-blue-50',
                    label: 'Stable'
                };
            default:
                return { 
                    icon: '→', 
                    color: 'text-gray-600', 
                    bgColor: 'bg-gray-50',
                    label: 'Unknown'
                };
        }
    }
    
    // Get alert status
    function getAlertStatus(alertTriggered) {
        return alertTriggered 
            ? { 
                icon: '🔔', 
                color: 'text-orange-600', 
                bgColor: 'bg-orange-50',
                label: 'Alerted'
            }
            : { 
                icon: '', 
                color: 'text-gray-400', 
                bgColor: 'bg-transparent',
                label: 'Normal'
            };
    }
    
    // Get distance color
    function getDistanceColor(distance) {
        if (distance <= 0.1) return 'text-red-600 font-semibold';
        if (distance <= 1.0) return 'text-orange-600';
        if (distance <= 5.0) return 'text-yellow-600';
        return 'text-gray-600';
    }
    
    // Get priority level
    function getPriorityLevel(distance) {
        if (distance <= 0.1) return 'high';
        if (distance <= 1.0) return 'medium';
        if (distance <= 5.0) return 'low';
        return 'none';
    }
    
    // Handle card click
    function handleCardClick() {
        dispatch('click', { stock });
    }
    
    // Handle alert click
    function handleAlertClick(event) {
        event.stopPropagation();
        dispatch('alert', { stock });
    }
    
    $: trendDisplay = getTrendDisplay(stock.bb_width_trend);
    $: alertStatus = getAlertStatus(stock.alert_triggered);
    $: distanceColor = getDistanceColor(stock.distance_from_min_percent);
    $: priorityLevel = getPriorityLevel(stock.distance_from_min_percent);
    
    // Format timestamp
    $: formattedTime = stock.timestamp 
        ? new Date(stock.timestamp).toLocaleTimeString('en-IN', { 
            hour12: false,
            timeZone: 'Asia/Kolkata'
        }) 
        : '';
</script>

<div 
    class="bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow cursor-pointer {stock.alert_triggered ? 'ring-2 ring-orange-200' : ''}"
    on:click={handleCardClick}
>
    <!-- Header -->
    <div class="flex items-center justify-between mb-3">
        <div class="flex items-center space-x-2">
            <h3 class="text-lg font-semibold text-gray-900">{stock.symbol}</h3>
            {#if stock.alert_triggered}
                <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-orange-100 text-orange-800">
                    🔔 Alert
                </span>
            {/if}
        </div>
        
        <div class="text-sm text-gray-500">
            {formattedTime}
        </div>
    </div>
    
    <!-- Instrument Key -->
    <div class="text-sm text-gray-500 mb-3">
        {stock.instrument_key}
    </div>
    
    <!-- Main Metrics -->
    <div class="grid grid-cols-2 gap-4 mb-3">
        <div>
            <div class="text-sm font-medium text-gray-500">Current BBW</div>
            <div class="text-lg font-bold text-gray-900">
                {formatNumber(stock.current_bb_width, 4)}
            </div>
        </div>
        
        <div>
            <div class="text-sm font-medium text-gray-500">Min BBW</div>
            <div class="text-lg font-bold text-gray-900">
                {formatNumber(stock.historical_min_bb_width, 4)}
            </div>
        </div>
    </div>
    
    <!-- Distance and Trend -->
    <div class="grid grid-cols-2 gap-4 mb-3">
        <div>
            <div class="text-sm font-medium text-gray-500">Distance</div>
            <div class="text-lg font-bold {distanceColor}">
                {formatNumber(stock.distance_from_min_percent, 1)}%
            </div>
        </div>
        
        <div>
            <div class="text-sm font-medium text-gray-500">Trend</div>
            <div class="flex items-center space-x-1">
                <span class="text-lg {trendDisplay.color}">
                    {trendDisplay.icon}
                </span>
                <span class="text-sm text-gray-600">
                    {stock.contracting_sequence_count || 0}
                </span>
            </div>
        </div>
    </div>
    
    <!-- Details (if shown) -->
    {#if showDetails}
        <div class="border-t border-gray-200 pt-3 mt-3">
            <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                    <span class="text-gray-500">Trend:</span>
                    <span class="ml-1 font-medium {trendDisplay.color}">
                        {trendDisplay.label}
                    </span>
                </div>
                
                <div>
                    <span class="text-gray-500">Priority:</span>
                    <span class="ml-1 font-medium capitalize">
                        {priorityLevel}
                    </span>
                </div>
                
                {#if stock.alert_triggered_at}
                    <div class="col-span-2">
                        <span class="text-gray-500">Alerted:</span>
                        <span class="ml-1 font-medium text-orange-600">
                            {new Date(stock.alert_triggered_at).toLocaleTimeString('en-IN', { 
                                hour12: false,
                                timeZone: 'Asia/Kolkata'
                            })}
                        </span>
                    </div>
                {/if}
            </div>
        </div>
    {/if}
    
    <!-- Action Buttons -->
    <div class="flex justify-end space-x-2 mt-3 pt-3 border-t border-gray-200">
        <button
            class="px-3 py-1 text-sm bg-blue-100 text-blue-700 rounded hover:bg-blue-200 transition-colors"
            on:click|stopPropagation={() => dispatch('configure', { stock })}
        >
            Configure
        </button>
        
        {#if stock.alert_triggered}
            <button
                class="px-3 py-1 text-sm bg-orange-100 text-orange-700 rounded hover:bg-orange-200 transition-colors"
                on:click={handleAlertClick}
            >
                View Alert
            </button>
        {/if}
    </div>
</div> 