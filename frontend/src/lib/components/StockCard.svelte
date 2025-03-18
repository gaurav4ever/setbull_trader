<!-- frontend/src/lib/components/StockCard.svelte -->
<script>
	import { onMount, createEventDispatcher } from 'svelte';
	import { formatCurrency, formatNumber } from '../utils/formatting';
	import TradingParameters from './TradingParameters.svelte';
	import ExecutionLevelDisplay from './ExecutionLevelDisplay.svelte';
	import {
		calculateFibonacciLevels,
		getExecutionPlan,
		createExecutionPlan
	} from '../services/calculationService';
	import { selectedStocksStore } from '../stores/selectedStocks';

	// Props
	export let stock = null; // Stock object with id, symbol, name, securityId, etc.
	export let expanded = false; // Whether the card is expanded to show full details
	export let active = false; // Whether the card is the active/focused card
	export let isNewlyAdded = false; // Whether this stock was just added

	// State
	let isLoading = false;
	let isCalculating = false;
	let isCreatingPlan = false;
	let error = '';
	let parameters = null;
	let executionPlan = null;
	let calculatedLevels = null;
	let hasSavedParameters = false;
	let showCalculationResults = false;

	// Event dispatcher
	const dispatch = createEventDispatcher();

	// Initialize when component mounts or stock changes
	$: if (stock) {
		loadData();
	}

	// Automatically expand when the component is newly added
	$: if (isNewlyAdded && !expanded) {
		expanded = true;
		dispatch('toggle-expanded', { stockId: stock.id, expanded });
	}

	// Load initial data
	async function loadData() {
		if (!stock) return;

		isLoading = true;
		error = '';

		try {
			// Try to load existing execution plan
			executionPlan = await getExecutionPlan(stock.id);

			if (executionPlan) {
				// We have an existing plan, use its parameters
				parameters = executionPlan.parameters;
				hasSavedParameters = true;
				showCalculationResults = true;
			} else {
				// No plan yet, but check if we have parameters
				const response = await fetch(`/api/v1/parameters/stock/${stock.id}`);

				if (response.ok) {
					const result = await response.json();
					if (result.data) {
						parameters = result.data;
						hasSavedParameters = true;
					}
				}
			}
		} catch (err) {
			console.error('Error loading data:', err);
			error = err.message || 'Failed to load data';
		} finally {
			isLoading = false;
		}
	}

	// Handle parameters saved event
	function handleParametersSaved(event) {
		parameters = event.detail;
		hasSavedParameters = true;
		// Clear any previous calculation or plan
		calculatedLevels = null;
		showCalculationResults = false;

		// Notify parent component
		dispatch('parameters-updated', parameters);
	}

	// Handle toggle selection
	async function toggleSelection() {
		if (!stock) return;

		try {
			// Toggle selection in the store
			await selectedStocksStore.toggleSelection(stock.id, !stock.isSelected);

			// Update local stock object
			stock.isSelected = !stock.isSelected;

			// Notify parent component
			dispatch('selection-toggled', {
				stockId: stock.id,
				isSelected: stock.isSelected
			});
		} catch (err) {
			error = err.message || 'Failed to toggle selection';
		}
	}

	// Calculate Fibonacci levels
	async function calculateLevels() {
		if (!parameters) return;

		isCalculating = true;
		error = '';

		try {
			// Call the API to calculate levels
			calculatedLevels = await calculateFibonacciLevels(parameters);
			showCalculationResults = true;
		} catch (err) {
			console.error('Error calculating levels:', err);
			error = err.message || 'Failed to calculate levels';
		} finally {
			isCalculating = false;
		}
	}

	// Create execution plan
	async function createPlan() {
		if (!stock?.id) return;

		isCreatingPlan = true;
		error = '';

		try {
			// Create execution plan
			executionPlan = await createExecutionPlan(stock.id);
			showCalculationResults = true;

			// Notify parent component
			dispatch('plan-created', executionPlan);
		} catch (err) {
			console.error('Error creating execution plan:', err);
			error = err.message || 'Failed to create execution plan';
		} finally {
			isCreatingPlan = false;
		}
	}

	// Toggle expanded state
	function toggleExpanded() {
		expanded = !expanded;
		dispatch('toggle-expanded', { stockId: stock.id, expanded });
	}
</script>

{#if stock}
	<div
		class="stock-card border rounded-lg shadow-sm overflow-hidden {active
			? 'border-blue-500 ring-2 ring-blue-200'
			: 'border-gray-200'}"
	>
		<!-- Card Header -->
		<div class="bg-gray-50 px-4 py-3 border-b border-gray-200">
			<div class="flex justify-between items-center">
				<div class="flex-1">
					<h3 class="text-lg font-medium text-gray-900">{stock.symbol}</h3>
					<p class="text-sm text-gray-500">
						{stock.name || 'Stock'}
						{#if stock.securityId && stock.securityId !== stock.symbol}
							<span class="ml-1 text-xs text-gray-400">(ID: {stock.securityId})</span>
						{/if}
					</p>
				</div>

				<div class="flex items-center space-x-2">
					<!-- Selection Checkbox -->
					<div class="flex items-center">
						<input
							type="checkbox"
							id="select-{stock.id}"
							checked={stock.isSelected}
							on:change={toggleSelection}
							class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
						/>
						<label for="select-{stock.id}" class="sr-only">Select for trading</label>
					</div>

					<!-- Expand/Collapse Button -->
					<button
						type="button"
						on:click={toggleExpanded}
						class="p-1 rounded-full hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
					>
						{#if expanded}
							<svg
								xmlns="http://www.w3.org/2000/svg"
								class="h-5 w-5 text-gray-500"
								viewBox="0 0 20 20"
								fill="currentColor"
							>
								<path
									fill-rule="evenodd"
									d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
									clip-rule="evenodd"
								/>
							</svg>
						{:else}
							<svg
								xmlns="http://www.w3.org/2000/svg"
								class="h-5 w-5 text-gray-500"
								viewBox="0 0 20 20"
								fill="currentColor"
							>
								<path
									fill-rule="evenodd"
									d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
									clip-rule="evenodd"
								/>
							</svg>
						{/if}
					</button>
				</div>
			</div>

			<!-- Current Stock Price (if available) -->
			{#if stock.currentPrice}
				<div class="mt-1">
					<span class="text-sm text-gray-500">Current Price:</span>
					<span class="ml-1 font-medium text-gray-900">{formatCurrency(stock.currentPrice)}</span>
				</div>
			{/if}

			<!-- Security ID (if expanded) -->
			{#if expanded && stock.securityId && stock.securityId !== stock.symbol}
				<div class="mt-1">
					<span class="text-sm text-gray-500">Security ID:</span>
					<span class="ml-1 font-medium text-gray-900">{stock.securityId}</span>
				</div>
			{/if}
		</div>

		<!-- Error Message (if any) -->
		{#if error}
			<div class="p-3 bg-red-50 text-red-700 text-sm">
				<p>{error}</p>
			</div>
		{/if}

		<!-- Loading Indicator -->
		{#if isLoading}
			<div class="p-4 flex justify-center">
				<div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
			</div>
		{:else if expanded}
			<!-- Parameters Section -->
			<div class="p-4 border-b border-gray-200">
				<TradingParameters
					stockId={stock.id}
					stockSymbol={stock.symbol}
					initialParameters={parameters}
					on:saved={handleParametersSaved}
				/>
			</div>

			<!-- Calculation/Plan Actions -->
			{#if hasSavedParameters}
				<div
					class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between"
				>
					<span class="text-sm font-medium text-gray-700">Execution Plan</span>

					<div class="flex space-x-2">
						<!-- Calculate Button -->
						<button
							type="button"
							on:click={calculateLevels}
							disabled={isCalculating || isCreatingPlan}
							class="inline-flex items-center px-3 py-1 border border-gray-300 text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
						>
							{#if isCalculating}
								<svg
									class="animate-spin -ml-1 mr-2 h-4 w-4 text-gray-700"
									fill="none"
									viewBox="0 0 24 24"
								>
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="4"
									/>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
									/>
								</svg>
								Calculating...
							{:else}
								Calculate
							{/if}
						</button>

						<!-- Create Plan Button -->
						<button
							type="button"
							on:click={createPlan}
							disabled={isCalculating || isCreatingPlan}
							class="inline-flex items-center px-3 py-1 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
						>
							{#if isCreatingPlan}
								<svg
									class="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
									fill="none"
									viewBox="0 0 24 24"
								>
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="4"
									/>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
									/>
								</svg>
								Creating...
							{:else}
								Create Plan
							{/if}
						</button>
					</div>
				</div>
			{/if}

			<!-- Execution Levels Display -->
			{#if showCalculationResults}
				<div class="p-4">
					{#if executionPlan}
						<!-- Show plan from API -->
						<ExecutionLevelDisplay
							levels={executionPlan.levelEntries}
							totalQuantity={executionPlan.totalQuantity}
							tradeSide={parameters?.tradeSide || 'BUY'}
						/>
					{:else if calculatedLevels}
						<!-- Show calculated levels -->
						<ExecutionLevelDisplay
							levels={calculatedLevels.levels}
							totalQuantity={calculatedLevels.totalQuantity}
							tradeSide={parameters?.tradeSide || 'BUY'}
						/>
					{/if}
				</div>
			{/if}
		{:else}
			<!-- Compact View (when not expanded) -->
			<div class="p-4">
				{#if executionPlan}
					<div class="text-sm text-gray-600 mb-2">
						<span class="font-medium">Plan Ready</span> • {formatNumber(
							executionPlan.totalQuantity
						)} shares
					</div>
					{#if executionPlan.levelEntries && executionPlan.levelEntries.length > 0}
						<div class="text-sm">
							<span class="text-gray-500">Entry: </span>
							<span class="font-medium text-gray-900"
								>{formatNumber(executionPlan.levelEntries[1]?.price || 0)}</span
							>
							<span class="mx-1 text-gray-400">|</span>
							<span class="text-gray-500">SL: </span>
							<span class="font-medium text-gray-900"
								>{formatNumber(executionPlan.levelEntries[0]?.price || 0)}</span
							>
						</div>
					{/if}
				{:else if parameters}
					<p class="text-sm text-gray-600">Parameters set, click to view details</p>
				{:else}
					<p class="text-sm text-gray-500 italic">Click to set trading parameters</p>
				{/if}
			</div>
		{/if}
	</div>
{/if}
