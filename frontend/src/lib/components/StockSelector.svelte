<!-- frontend/src/lib/components/StockSelector.svelte -->
<script>
	import { onMount, createEventDispatcher } from 'svelte';
	import Autocomplete from './Autocomplete.svelte';
	import { getStocksList, searchStocks } from '../services/stocksService';
	import { formatStockForDisplay } from '../utils/stockFormatting';
	import { selectedStocksStore, canAddMoreStocks } from '../stores/selectedStocks';

	// Props
	export let onStockSelected = null; // Optional callback when a stock is selected

	// Local state
	let stocksList = [];
	let isLoading = true;
	let searchQuery = '';
	let selectedStock = null;
	let error = '';

	// Get our stores
	let canAdd;
	const unsubscribe = canAddMoreStocks.subscribe((value) => {
		canAdd = value;
	});

	// Event dispatcher
	const dispatch = createEventDispatcher();

	onMount(async () => {
		try {
			stocksList = await getStocksList();
			isLoading = false;
		} catch (err) {
			error = 'Failed to load stocks. Please refresh the page.';
			isLoading = false;
		}

		// Clean up subscription when component is destroyed
		return () => {
			unsubscribe();
		};
	});

	// Format stock for display in dropdown
	function formatStock(stock) {
		return formatStockForDisplay(stock);
	}

	// Handle stock selection
	function handleStockSelect(event) {
		selectedStock = event.detail;

		// Dispatch the event with the full stock object
		dispatch('select', selectedStock);

		// Call the callback if provided
		if (onStockSelected && typeof onStockSelected === 'function') {
			onStockSelected(selectedStock);
		}

		// Clear the selection for next use
		setTimeout(() => {
			selectedStock = null;
			searchQuery = '';
		}, 100);
	}

	// Handle search input changes
	function handleSearchInput(event) {
		searchQuery = event.detail;
	}
</script>

<div class="stock-selector">
	<h2 class="text-lg font-medium mb-2">Select a Stock</h2>

	{#if !canAdd}
		<div class="mb-2 p-2 bg-yellow-100 text-yellow-800 rounded">
			<p class="text-sm">You've already selected 3 stocks (maximum allowed).</p>
		</div>
	{/if}

	<div class={!canAdd ? 'opacity-50 pointer-events-none' : ''}>
		{#if isLoading}
			<div class="flex items-center p-2 bg-gray-100 rounded">
				<div class="animate-pulse h-8 bg-gray-300 rounded w-full"></div>
				<span class="ml-2 text-sm text-gray-500">Loading stocks...</span>
			</div>
		{:else if error}
			<div class="p-2 bg-red-100 text-red-800 rounded">
				<p>{error}</p>
			</div>
		{:else}
			<Autocomplete
				items={stocksList}
				bind:value={selectedStock}
				on:input={handleSearchInput}
				on:select={handleStockSelect}
				placeholder="Search for a stock..."
				inputClass="mt-1 block w-full px-3 py-2 bg-white border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
				displayFormat={formatStock}
			/>
			<p class="mt-1 text-sm text-gray-500">
				{#if searchQuery && searchQuery.length < 2}
					Type at least 2 characters to search
				{:else}
					Select from the list of Indian stocks
				{/if}
			</p>
		{/if}
	</div>
</div>

<style>
	.stock-selector {
		margin-bottom: 1rem;
	}
</style>
