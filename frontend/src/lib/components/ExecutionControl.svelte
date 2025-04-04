<script>
	import { onMount, createEventDispatcher } from 'svelte';
	import { executeOrdersForAllSelectedStocks } from '../services/executionService';
	import { selectedStocksStore, selectedStocksCount } from '../stores/selectedStocks';
	import { formatCurrency } from '../utils/formatting';

	// Props
	export let disabled = false; // Whether the execution button is disabled
	export let requireConfirmation = true; // Whether to require confirmation before executing

	// State
	let isExecuting = false;
	let error = '';
	let confirmationVisible = false;
	let selectedStocks = [];
	let count = 0;
	let executionInProgress = false; // Flag to prevent duplicate executions

	// Subscribe to selected stocks store
	const unsubscribeStocks = selectedStocksStore.subscribe((state) => {
		selectedStocks = state.stocks;
	});

	const unsubscribeCount = selectedStocksCount.subscribe((value) => {
		count = value;
	});

	// Event dispatcher
	const dispatch = createEventDispatcher();

	// Clean up subscriptions when component is destroyed
	onMount(() => {
		return () => {
			unsubscribeStocks();
			unsubscribeCount();
		};
	});

	// Calculate total risk amount
	$: totalRiskAmount = selectedStocks.reduce((sum, stock) => {
		// Get risk amount from parameters if available
		const riskAmount = stock.parameters?.riskAmount || 30;
		return sum + riskAmount;
	}, 0);

	// Check if any stock is missing parameters or plan
	$: hasMissingPrerequisites = selectedStocks.some(
		(stock) =>
			(!stock.parameters && !stock.executionPlan) ||
			(!stock.executionPlan && !stock.executionPlan?.levelEntries?.length)
	);

	// Handle execute button click
	function handleExecuteClick() {
		// Prevent multiple clicks
		if (executionInProgress) {
			console.log('Execution already in progress, ignoring click');
			return;
		}

		if (requireConfirmation) {
			// Show confirmation dialog
			confirmationVisible = true;
		} else {
			// Execute directly
			executeOrders();
		}
	}

	// Handle confirmation
	function confirmExecution() {
		confirmationVisible = false;
		executeOrders();
	}

	// Cancel confirmation
	function cancelExecution() {
		confirmationVisible = false;
	}

	// Execute orders for all selected stocks
	async function executeOrders() {
		// Prevent duplicate executions
		if (executionInProgress) {
			console.log('Execution already in progress, preventing duplicate call');
			return;
		}

		executionInProgress = true;
		isExecuting = true;
		error = '';

		try {
			console.log('Starting order execution for selected stocks');
			// Execute orders
			const executions = await executeOrdersForAllSelectedStocks();
			console.log('Order execution completed successfully', executions);

			// Dispatch success event
			dispatch('executed', { executions });
		} catch (err) {
			console.error('Error executing orders:', err);
			error = err.message || 'Failed to execute orders';

			// Dispatch error event
			dispatch('error', { error });
		} finally {
			isExecuting = false;
		}
	}
</script>

<div class="execution-control p-4 bg-white rounded-lg shadow-sm border border-gray-200">
	<div class="mb-4">
		<h2 class="text-lg font-medium text-gray-900">Execution Control</h2>
		<p class="text-sm text-gray-500">Execute trades for all selected stocks</p>
	</div>

	{#if error}
		<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md">
			<p>{error}</p>
		</div>
	{/if}

	<!-- Selected Stocks Summary -->
	<div class="mb-4 space-y-2">
		<div class="flex justify-between items-center">
			<span class="text-sm text-gray-700">Selected Stocks:</span>
			<span class="font-medium">{count}/3</span>
		</div>

		<div class="flex justify-between items-center">
			<span class="text-sm text-gray-700">Total Risk:</span>
			<span class="font-medium">{formatCurrency(totalRiskAmount)}</span>
		</div>
	</div>

	<!-- Warning for missing prerequisites -->
	{#if count > 0 && hasMissingPrerequisites}
		<div class="mb-4 p-3 bg-yellow-50 text-yellow-700 rounded-md text-sm">
			<p>
				Some selected stocks don't have trading parameters set. Please configure all stocks before
				executing.
			</p>
		</div>
	{/if}

	<!-- Execute Button -->
	<div class="flex justify-center">
		<button
			type="button"
			on:click={handleExecuteClick}
			disabled={disabled || isExecuting || count === 0 || hasMissingPrerequisites}
			class="w-full py-2 px-4 border border-transparent text-sm font-medium rounded-md shadow-sm text-white {count ===
				0 || hasMissingPrerequisites
				? 'bg-gray-400'
				: 'bg-blue-600 hover:bg-blue-700'} focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
		>
			{#if isExecuting}
				<div class="flex items-center justify-center">
					<svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
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
					Executing Trades...
				</div>
			{:else if count === 0}
				No Stocks Selected
			{:else}
				Execute Trades
			{/if}
		</button>
	</div>
</div>

<!-- Confirmation Modal -->
{#if confirmationVisible}
	<div class="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center z-50">
		<div class="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
			<h3 class="text-lg font-medium text-gray-900 mb-4">Confirm Execution</h3>

			<p class="mb-4 text-gray-700">
				You are about to execute trades for {count} selected stock{count !== 1 ? 's' : ''} with a total
				risk of {formatCurrency(totalRiskAmount)}.
			</p>

			<p class="mb-6 text-sm text-gray-500">
				This will place orders according to the Fibonacci levels in the execution plan.
			</p>

			<div class="flex justify-end space-x-3">
				<button
					type="button"
					on:click={cancelExecution}
					class="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
				>
					Cancel
				</button>

				<button
					type="button"
					on:click={confirmExecution}
					class="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
				>
					Confirm & Execute
				</button>
			</div>
		</div>
	</div>
{/if}
