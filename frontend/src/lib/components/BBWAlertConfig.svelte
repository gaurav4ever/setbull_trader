<!-- BBW Alert Configuration Component -->
<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import apiService from '$lib/services/apiService.js';
    
    const dispatch = createEventDispatcher();
    
    // Props
    export let show = false;
    export let currentConfig = {
        alertThreshold: 0.1,
        contractingLookback: 5,
        enableAlerts: true
    };
    
    // Local state
    let config: { [key: string]: any } = { ...currentConfig };
    let loading = false;
    let error = '';
    let success = '';
    
    // Watch for prop changes
    $: if (show && currentConfig) {
        config = { ...currentConfig };
    }
    
    // Handle form submission
    async function handleSubmit() {
        loading = true;
        error = '';
        success = '';
        
        try {
            const response = await apiService.bbw.configureAlerts(config);
            
            if (response.status === 'success') {
                success = 'Alert configuration updated successfully';
                dispatch('configUpdated', config);
                
                // Clear success message after 3 seconds
                setTimeout(() => {
                    success = '';
                }, 3000);
            } else {
                error = response.message || 'Failed to update configuration';
            }
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to update configuration';
            error = errorMessage;
        } finally {
            loading = false;
        }
    }
    
    // Handle cancel
    function handleCancel() {
        config = { ...currentConfig };
        error = '';
        success = '';
        dispatch('close');
    }
    
    // Handle input changes
    function handleInputChange(field: string, value: any) {
        config[field] = value;
    }
</script>

<!-- Modal Overlay -->
{#if show}
    <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
            <!-- Header -->
            <div class="flex items-center justify-between p-6 border-b">
                <h3 class="text-lg font-semibold text-gray-900">
                    Alert Configuration
                </h3>
                <button 
                    type="button" 
                    class="text-gray-400 hover:text-gray-600"
                    on:click={handleCancel}
                >
                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                    </svg>
                </button>
            </div>
            
            <!-- Content -->
            <div class="p-6">
                <!-- Alert Threshold -->
                <div class="mb-6">
                    <label for="alertThreshold" class="block text-sm font-medium text-gray-700 mb-2">
                        Alert Threshold (%)
                    </label>
                    <div class="relative">
                        <input
                            id="alertThreshold"
                            type="number"
                            step="0.01"
                            min="0"
                            max="100"
                            bind:value={config.alertThreshold}
                            on:input={(e) => handleInputChange('alertThreshold', parseFloat((e.target as HTMLInputElement).value))}
                            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="0.1"
                        />
                        <div class="absolute inset-y-0 right-0 flex items-center pr-3">
                            <span class="text-gray-500 text-sm">%</span>
                        </div>
                    </div>
                    <p class="mt-1 text-sm text-gray-500">
                        Percentage distance from historical minimum BB width to trigger alerts
                    </p>
                </div>
                
                <!-- Contracting Lookback -->
                <div class="mb-6">
                    <label for="contractingLookback" class="block text-sm font-medium text-gray-700 mb-2">
                        Contracting Lookback (Candles)
                    </label>
                    <input
                        id="contractingLookback"
                        type="number"
                        min="1"
                        max="20"
                        bind:value={config.contractingLookback}
                                                    on:input={(e) => handleInputChange('contractingLookback', parseInt((e.target as HTMLInputElement).value))}
                        class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                        placeholder="5"
                    />
                    <p class="mt-1 text-sm text-gray-500">
                        Number of consecutive candles to analyze for contracting patterns
                    </p>
                </div>
                
                <!-- Enable Alerts -->
                <div class="mb-6">
                    <div class="flex items-center">
                        <input
                            id="enableAlerts"
                            type="checkbox"
                            bind:checked={config.enableAlerts}
                            on:change={(e) => handleInputChange('enableAlerts', (e.target as HTMLInputElement).checked)}
                            class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                        />
                        <label for="enableAlerts" class="ml-2 block text-sm text-gray-700">
                            Enable Audio Alerts
                        </label>
                    </div>
                    <p class="mt-1 text-sm text-gray-500">
                        Play audio alerts when patterns are detected
                    </p>
                </div>
                
                <!-- Error Message -->
                {#if error}
                    <div class="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
                        <p class="text-sm text-red-600">{error}</p>
                    </div>
                {/if}
                
                <!-- Success Message -->
                {#if success}
                    <div class="mb-4 p-3 bg-green-50 border border-green-200 rounded-md">
                        <p class="text-sm text-green-600">{success}</p>
                    </div>
                {/if}
            </div>
            
            <!-- Footer -->
            <div class="flex items-center justify-end space-x-3 p-6 border-t bg-gray-50">
                <button
                    type="button"
                    on:click={handleCancel}
                    class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                    Cancel
                </button>
                <button
                    type="button"
                    on:click={handleSubmit}
                    disabled={loading}
                    class="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {loading ? 'Saving...' : 'Save Configuration'}
                </button>
            </div>
        </div>
    </div>
{/if} 