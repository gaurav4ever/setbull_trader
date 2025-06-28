<!-- frontend/src/routes/+page.svelte (Updated Section) -->
<script lang="ts">
	import { onMount } from 'svelte';
	import EnhancedStockSelector from '../lib/components/EnhancedStockSelector.svelte';
	import StockCard from '../lib/components/StockCard.svelte';
	import ExecutionControl from '../lib/components/ExecutionControl.svelte';
	import ExecutionResults from '../lib/components/ExecutionResults.svelte';
	import { selectedStocksStore } from '../lib/stores/selectedStocks';
	import { executionStatusStore, isExecuting, hasResults } from '../lib/stores/executionStatus';
	import { executeOrdersForAllSelectedStocks } from '../lib/services/executionService';
	import { createStock } from '../lib/services/stocksService';
	import { tradeApi } from '$lib/services/apiService';
	import StockGroupSummary from '../lib/components/StockGroupSummary.svelte';

	interface BacktestData {
		long: {
			last5: Array<{ stock: string; rate: number }>;
			last10: Array<{ stock: string; rate: number }>;
			last15: Array<{ stock: string; rate: number }>;
		};
		short: {
			last5: Array<{ stock: string; rate: number }>;
			last10: Array<{ stock: string; rate: number }>;
			last15: Array<{ stock: string; rate: number }>;
		};
	}

	// Backtest performance state
	let backtestData: BacktestData = {
		long: {
			last5: [],
			last10: [],
			last15: []
		},
		short: {
			last5: [],
			last10: [],
			last15: []
		}
	};
	let isLoadingBacktest = false;
	let backtestError: string | null = null;

	// Function to parse backtest file
	async function parseBacktestFile(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) return;

		isLoadingBacktest = true;
		backtestError = null;

		try {
			const file = input.files[0];
			const text = await file.text();

			// Split into long and short sections
			const sections = text.split('SHORT DIRECTION PERFORMANCE REPORT');
			const longSection = sections[0];
			const shortSection = sections[1];

			// Parse long section
			backtestData.long = {
				last5: parseTimeframeData(longSection, 'Last 5 Entries'),
				last10: parseTimeframeData(longSection, 'Last 10 Entries'),
				last15: parseTimeframeData(longSection, 'Last 15 Entries')
			};

			// Parse short section
			backtestData.short = {
				last5: parseTimeframeData(shortSection, 'Last 5 Entries'),
				last10: parseTimeframeData(shortSection, 'Last 10 Entries'),
				last15: parseTimeframeData(shortSection, 'Last 15 Entries')
			};
		} catch (err) {
			console.error('Error parsing backtest file:', err);
			backtestError = err instanceof Error ? err.message : 'Failed to parse backtest file';
		} finally {
			isLoadingBacktest = false;
		}
	}

	// Helper function to parse timeframe data
	function parseTimeframeData(
		section: string,
		timeframe: string
	): Array<{ stock: string; rate: number }> {
		const lines = section.split('\n');
		const startIndex = lines.findIndex((line: string) => line.includes(timeframe));
		if (startIndex === -1) return [];

		const data: Array<{ stock: string; rate: number }> = [];
		let i = startIndex + 3; // Skip header lines

		while (i < lines.length && lines[i].trim() && !lines[i].includes('Last')) {
			const [stock, rate] = lines[i].split(/\s+/).filter(Boolean);
			if (stock && rate) {
				data.push({
					stock,
					rate: parseFloat(rate.replace('%', ''))
				});
			}
			i++;
		}

		return data;
	}

	// Stock selection state
	let selectedStocks: any[] = [];
	let loading = true;
	let error: string | null = null;
	let activeStockId: any = null;
	let addingStock = false;
	let newlyAddedStockId: any = null;

	// Trade statistics state
	let trades: any[] = [];
	let isLoadingTrades = true;
	let tradeError: string | null = null;

	// Statistics
	type Stats = {
		totalTrades: number;
		totalBuyValue: number;
		totalSellValue: number;
		netPosition: number;
	};
	let stats: Stats = {
		totalTrades: 0,
		totalBuyValue: 0,
		totalSellValue: 0,
		netPosition: 0
	};

	// Subscribe to the selected stocks store
	const unsubscribeStocks = selectedStocksStore.subscribe((state) => {
		selectedStocks = state.stocks;
		loading = state.loading;
		error = state.error;
	});

	// Execution status
	let executing = false;
	let executionResults: any[] = [];
	let executionError: any = null;

	// Subscribe to execution status store
	const unsubscribeExecutionStatus = executionStatusStore.subscribe((state) => {
		executing = state.isExecuting;
		executionResults = state.results;
		executionError = state.error;
	});

	onMount(() => {
		// Load selected stocks
		selectedStocksStore.loadSelectedStocks();

		// Load trade data
		(async () => {
			try {
				// Fetch real trade data from the backend
				const response = await tradeApi.getAllTrades();
				trades = response.trades || [];

				// Calculate statistics
				calculateStats();
				isLoadingTrades = false;
			} catch (err) {
				console.error('Error fetching trades:', err);
				tradeError = (err as Error).message || 'Failed to load trades';
				isLoadingTrades = false;

				// If API fails, use mock data for demonstration
				useMockData();
			}
		})();

		// Clean up subscriptions when component is destroyed
		return () => {
			unsubscribeStocks();
			unsubscribeExecutionStatus();
		};
	});

	// // Handle stock selection with parameters
	// async function handleStockWithParametersSelected(event) {
	// 	const { stockSymbol, parameters } = event.detail;

	// 	// Show loading state
	// 	addingStock = true;
	// 	error = '';

	// 	try {
	// 		// Create stock with parameters
	// 		await createStockWithParameters(stockSymbol, parameters);

	// 		// Reload selected stocks
	// 		await selectedStocksStore.loadSelectedStocks();
	// 	} catch (err) {
	// 		console.error('Error adding stock with parameters:', err);
	// 		error = err.message || 'Failed to add stock';
	// 	} finally {
	// 		addingStock = false;
	// 	}
	// }

	// Fallback to mock data if API fails
	function useMockData() {
		console.log('Using mock data as fallback');
		const mockTrades = [
			{
				orderId: 'ORD001',
				transactionType: 'BUY',
				symbol: 'RELIANCE',
				exchangeSegment: 'NSE_EQ',
				productType: 'CNC',
				quantity: 10,
				price: 2500.5,
				timestamp: new Date().toISOString()
			},
			{
				orderId: 'ORD002',
				transactionType: 'SELL',
				symbol: 'INFY',
				exchangeSegment: 'NSE_EQ',
				productType: 'CNC',
				quantity: 5,
				price: 1750.25,
				timestamp: new Date().toISOString()
			}
		];

		trades = mockTrades;
		calculateStats();

		// Add a warning about using mock data
		tradeError = 'Could not connect to API. Using demo data for visualization.';
	}

	function calculateStats() {
		let buyValue = 0;
		let sellValue = 0;

		trades.forEach((trade) => {
			const value = trade.price * trade.quantity;

			if (trade.transactionType === 'BUY') {
				buyValue += value;
			} else if (trade.transactionType === 'SELL') {
				sellValue += value;
			}
		});

		stats = {
			totalTrades: trades.length,
			totalBuyValue: buyValue,
			totalSellValue: sellValue,
			netPosition: sellValue - buyValue
		};
	}

	function formatCurrency(value) {
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			minimumFractionDigits: 2
		}).format(value);
	}

	function getNetPositionClass(value) {
		return value >= 0 ? 'text-green-600' : 'text-red-600';
	}

	// Set active stock
	function setActiveStock(stockId) {
		activeStockId = stockId === activeStockId ? null : stockId;
	}

	// Handle execution
	async function handleExecute() {
		// Clear previous results
		executionStatusStore.clearResults();

		// Set executing state
		executionStatusStore.startExecution();

		try {
			// Execute orders
			const results = await executeOrdersForAllSelectedStocks();

			// Update store with results
			executionStatusStore.setResults(results);

			// Reload trade data after execution
			const response = await tradeApi.getAllTrades();
			trades = response.trades || [];
			calculateStats();
		} catch (err) {
			console.error('Error executing orders:', err);
			executionStatusStore.setError(err.message || 'Failed to execute orders');
		}
	}

	// Handle direct stock selection (new flow)
	async function handleStockSelected(event: any) {
		const stockId = event.detail;
		if (!stockId) return;
		if (selectedStocks.includes(stockId)) return;
		if (selectedStocks.length >= 5) return;
		selectedStocks = [...selectedStocks, stockId];

		// Show loading state
		addingStock = true;
		error = '';

		try {
			// Create the stock (without parameters for now)
			const stockData = {
				symbol: stockId,
				name: stockId,
				securityId: stockId,
				isSelected: true
			};

			const response = await createStock(stockData);

			// Store the ID of the newly added stock to auto-expand it
			if (response.data && response.data.id) {
				newlyAddedStockId = response.data.id;
			}

			// Reload selected stocks
			await selectedStocksStore.loadSelectedStocks();
		} catch (err) {
			console.error('Error adding stock:', err);
			error = (err as Error).message || 'Failed to add stock';
		} finally {
			addingStock = false;
		}
	}

	// Handle card expansion
	function handleToggleExpanded(event) {
		const { stockId, expanded } = event.detail;

		// If a stock was just added and this is a different one,
		// clear the newly added flag
		if (newlyAddedStockId && stockId !== newlyAddedStockId) {
			newlyAddedStockId = null;
		}
	}
</script>

<svelte:head>
	<title>Dashboard | SetBull Trader</title>
</svelte:head>

<div class="py-6">
	<!-- Backtest Performance Section -->
	<div class="mb-8">
		<div class="bg-white shadow rounded-lg p-6">
			<h2 class="text-lg font-medium text-gray-900 mb-4">Backtest Performance Analysis</h2>

			<!-- File Input -->
			<div class="mb-6">
				<label class="block text-sm font-medium text-gray-700 mb-2">Backtest Report File</label>
				<input
					type="file"
					accept=".txt"
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
					on:change={parseBacktestFile}
				/>
			</div>

			{#if isLoadingBacktest}
				<div class="flex justify-center py-4">
					<div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
					<span class="ml-2 text-sm text-gray-500">Loading backtest data...</span>
				</div>
			{:else if backtestError}
				<div class="bg-red-50 border-l-4 border-red-400 p-4">
					<p class="text-sm text-red-700">{backtestError}</p>
				</div>
			{:else if backtestData.long.last5.length > 0}
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<!-- Long Trades Section -->
					<div>
						<h3 class="text-md font-medium text-gray-900 mb-3">Long Trades Performance</h3>
						<div class="space-y-6">
							<!-- Last 5 Entries -->
							<div class="bg-white rounded-lg shadow">
								<div class="px-4 py-3 border-b border-gray-200">
									<h4 class="text-sm font-medium text-gray-700">Last 5 Entries</h4>
								</div>
								<div class="overflow-x-auto max-h-60">
									<table class="min-w-full divide-y divide-gray-200">
										<thead class="bg-gray-50 sticky top-0">
											<tr>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Stock</th
												>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Success Rate</th
												>
											</tr>
										</thead>
										<tbody class="bg-white divide-y divide-gray-200">
											{#each backtestData.long.last5 as item}
												<tr class="hover:bg-gray-50">
													<td class="px-4 py-3 text-sm font-medium text-gray-900">{item.stock}</td>
													<td class="px-4 py-3 text-sm">
														<span
															class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
																item.rate >= 70
																	? 'bg-green-100 text-green-800'
																	: item.rate >= 50
																		? 'bg-yellow-100 text-yellow-800'
																		: 'bg-red-100 text-red-800'
															}`}
														>
															{item.rate}%
														</span>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							</div>

							<!-- Last 10 Entries -->
							<div class="bg-white rounded-lg shadow">
								<div class="px-4 py-3 border-b border-gray-200">
									<h4 class="text-sm font-medium text-gray-700">Last 10 Entries</h4>
								</div>
								<div class="overflow-x-auto max-h-60">
									<table class="min-w-full divide-y divide-gray-200">
										<thead class="bg-gray-50 sticky top-0">
											<tr>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Stock</th
												>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Success Rate</th
												>
											</tr>
										</thead>
										<tbody class="bg-white divide-y divide-gray-200">
											{#each backtestData.long.last10 as item}
												<tr class="hover:bg-gray-50">
													<td class="px-4 py-3 text-sm font-medium text-gray-900">{item.stock}</td>
													<td class="px-4 py-3 text-sm">
														<span
															class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
																item.rate >= 70
																	? 'bg-green-100 text-green-800'
																	: item.rate >= 50
																		? 'bg-yellow-100 text-yellow-800'
																		: 'bg-red-100 text-red-800'
															}`}
														>
															{item.rate}%
														</span>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							</div>

							<!-- Last 15 Entries -->
							<div class="bg-white rounded-lg shadow">
								<div class="px-4 py-3 border-b border-gray-200">
									<h4 class="text-sm font-medium text-gray-700">Last 15 Entries</h4>
								</div>
								<div class="overflow-x-auto max-h-60">
									<table class="min-w-full divide-y divide-gray-200">
										<thead class="bg-gray-50 sticky top-0">
											<tr>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Stock</th
												>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Success Rate</th
												>
											</tr>
										</thead>
										<tbody class="bg-white divide-y divide-gray-200">
											{#each backtestData.long.last15 as item}
												<tr class="hover:bg-gray-50">
													<td class="px-4 py-3 text-sm font-medium text-gray-900">{item.stock}</td>
													<td class="px-4 py-3 text-sm">
														<span
															class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
																item.rate >= 70
																	? 'bg-green-100 text-green-800'
																	: item.rate >= 50
																		? 'bg-yellow-100 text-yellow-800'
																		: 'bg-red-100 text-red-800'
															}`}
														>
															{item.rate}%
														</span>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							</div>
						</div>
					</div>

					<!-- Short Trades Section -->
					<div>
						<h3 class="text-md font-medium text-gray-900 mb-3">Short Trades Performance</h3>
						<div class="space-y-6">
							<!-- Last 5 Entries -->
							<div class="bg-white rounded-lg shadow">
								<div class="px-4 py-3 border-b border-gray-200">
									<h4 class="text-sm font-medium text-gray-700">Last 5 Entries</h4>
								</div>
								<div class="overflow-x-auto max-h-60">
									<table class="min-w-full divide-y divide-gray-200">
										<thead class="bg-gray-50 sticky top-0">
											<tr>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Stock</th
												>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Success Rate</th
												>
											</tr>
										</thead>
										<tbody class="bg-white divide-y divide-gray-200">
											{#each backtestData.short.last5 as item}
												<tr class="hover:bg-gray-50">
													<td class="px-4 py-3 text-sm font-medium text-gray-900">{item.stock}</td>
													<td class="px-4 py-3 text-sm">
														<span
															class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
																item.rate >= 70
																	? 'bg-green-100 text-green-800'
																	: item.rate >= 50
																		? 'bg-yellow-100 text-yellow-800'
																		: 'bg-red-100 text-red-800'
															}`}
														>
															{item.rate}%
														</span>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							</div>

							<!-- Last 10 Entries -->
							<div class="bg-white rounded-lg shadow">
								<div class="px-4 py-3 border-b border-gray-200">
									<h4 class="text-sm font-medium text-gray-700">Last 10 Entries</h4>
								</div>
								<div class="overflow-x-auto max-h-60">
									<table class="min-w-full divide-y divide-gray-200">
										<thead class="bg-gray-50 sticky top-0">
											<tr>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Stock</th
												>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Success Rate</th
												>
											</tr>
										</thead>
										<tbody class="bg-white divide-y divide-gray-200">
											{#each backtestData.short.last10 as item}
												<tr class="hover:bg-gray-50">
													<td class="px-4 py-3 text-sm font-medium text-gray-900">{item.stock}</td>
													<td class="px-4 py-3 text-sm">
														<span
															class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
																item.rate >= 70
																	? 'bg-green-100 text-green-800'
																	: item.rate >= 50
																		? 'bg-yellow-100 text-yellow-800'
																		: 'bg-red-100 text-red-800'
															}`}
														>
															{item.rate}%
														</span>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							</div>

							<!-- Last 15 Entries -->
							<div class="bg-white rounded-lg shadow">
								<div class="px-4 py-3 border-b border-gray-200">
									<h4 class="text-sm font-medium text-gray-700">Last 15 Entries</h4>
								</div>
								<div class="overflow-x-auto max-h-60">
									<table class="min-w-full divide-y divide-gray-200">
										<thead class="bg-gray-50 sticky top-0">
											<tr>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Stock</th
												>
												<th
													class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
													>Success Rate</th
												>
											</tr>
										</thead>
										<tbody class="bg-white divide-y divide-gray-200">
											{#each backtestData.short.last15 as item}
												<tr class="hover:bg-gray-50">
													<td class="px-4 py-3 text-sm font-medium text-gray-900">{item.stock}</td>
													<td class="px-4 py-3 text-sm">
														<span
															class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
																item.rate >= 70
																	? 'bg-green-100 text-green-800'
																	: item.rate >= 50
																		? 'bg-yellow-100 text-yellow-800'
																		: 'bg-red-100 text-red-800'
															}`}
														>
															{item.rate}%
														</span>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Existing Stats Cards Section -->
	<div class="flex justify-between items-center mb-8">
		<h1 class="text-2xl font-bold text-gray-900">Trading Dashboard</h1>
		<a
			href="/order"
			class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
		>
			Place New Order
		</a>
	</div>

	<!-- Stats Cards -->
	<div class="mb-8">
		<dl class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
			<!-- Total Trades -->
			<div class="bg-white overflow-hidden shadow rounded-lg">
				<div class="px-4 py-5 sm:p-6">
					<dt class="text-sm font-medium text-gray-500 truncate">Total Trades Today</dt>
					<dd class="mt-1 text-3xl font-semibold text-gray-900">
						{isLoadingTrades ? '...' : stats.totalTrades}
					</dd>
				</div>
			</div>

			<!-- Buy Value -->
			<div class="bg-white overflow-hidden shadow rounded-lg">
				<div class="px-4 py-5 sm:p-6">
					<dt class="text-sm font-medium text-gray-500 truncate">Total Buy Value</dt>
					<dd class="mt-1 text-3xl font-semibold text-gray-900">
						{isLoadingTrades ? '...' : formatCurrency(stats.totalBuyValue)}
					</dd>
				</div>
			</div>

			<!-- Sell Value -->
			<div class="bg-white overflow-hidden shadow rounded-lg">
				<div class="px-4 py-5 sm:p-6">
					<dt class="text-sm font-medium text-gray-500 truncate">Total Sell Value</dt>
					<dd class="mt-1 text-3xl font-semibold text-gray-900">
						{isLoadingTrades ? '...' : formatCurrency(stats.totalSellValue)}
					</dd>
				</div>
			</div>

			<!-- Net Position -->
			<div class="bg-white overflow-hidden shadow rounded-lg">
				<div class="px-4 py-5 sm:p-6">
					<dt class="text-sm font-medium text-gray-500 truncate">Net Position</dt>
					<dd
						class="mt-1 text-3xl font-semibold {isLoadingTrades
							? ''
							: getNetPositionClass(stats.netPosition)}"
					>
						{isLoadingTrades ? '...' : formatCurrency(stats.netPosition)}
					</dd>
				</div>
			</div>
		</dl>
	</div>

	<!-- Add Stock Groups summary below stats cards -->
	<StockGroupSummary />

	<!-- API Connection Status -->
	{#if tradeError}
		<div class="bg-yellow-50 border-l-4 border-yellow-400 p-4 mb-8">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
						<path
							fill-rule="evenodd"
							d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
							clip-rule="evenodd"
						/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm text-yellow-700">{tradeError}</p>
					<p class="text-sm text-yellow-600 mt-2">
						Make sure your backend API is running at http://localhost:8083/api/v1
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Stock Selection Error (if any) -->
	{#if error}
		<div class="mb-6 p-4 bg-red-50 border-l-4 border-red-400 text-red-700">
			<p>{error}</p>
			<button
				class="mt-2 text-sm text-red-500 hover:text-red-700"
				on:click={() => selectedStocksStore.clearError()}
			>
				Dismiss
			</button>
		</div>
	{/if}

	<!-- Quick Actions -->
	<!-- <div class="bg-white shadow rounded-lg mb-8">
		<div class="px-4 py-5 sm:p-6">
			<h2 class="text-lg font-medium text-gray-900 mb-4">Quick Actions</h2>
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<a
					href="/order"
					class="inline-flex items-center justify-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
				>
					Place Order
				</a>
				<a
					href="/order/modify"
					class="inline-flex items-center justify-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
				>
					Modify Order
				</a>
				<a
					href="/order/cancel"
					class="inline-flex items-center justify-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
				>
					Cancel Order
				</a>
				<a
					href="/trades"
					class="inline-flex items-center justify-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
				>
					View Trades
				</a>
			</div>
		</div>
	</div> -->

	<!-- Stock Selection Section (Updated) -->
	<div class="mb-8">
		<div class="bg-white shadow rounded-lg p-6">
			<!-- <h2 class="text-lg font-medium text-gray-900 mb-4">Select Stocks</h2> -->

			{#if selectedStocks.length < 3}
				<!-- Enhanced Stock Selector Component -->
				<EnhancedStockSelector on:stockSelected={handleStockSelected} maxSelectedStocks={3} />

				{#if addingStock}
					<div class="mt-4 flex items-center justify-center py-4">
						<div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
						<span class="ml-2 text-sm text-gray-500">Adding stock...</span>
					</div>
				{/if}
			{:else}
				<p class="text-sm text-gray-500">
					You've selected 3 stocks (maximum allowed). Remove a stock to add a different one.
				</p>
			{/if}
		</div>
	</div>

	<!-- Selected Stocks Section with auto-expansion -->
	<div class="mb-8">
		<h2 class="text-lg font-medium text-gray-900 mb-4">Selected Stocks</h2>

		{#if loading}
			<div class="flex justify-center py-8">
				<div
					class="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-blue-500"
				></div>
			</div>
		{:else if selectedStocks.length === 0}
			<div class="bg-white shadow rounded-lg p-6 text-center">
				<p class="text-gray-500">No stocks selected. Use the stock selector above to add stocks.</p>
			</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
				{#each selectedStocks as stock (stock.id)}
					<StockCard
						{stock}
						expanded={stock.id === activeStockId || stock.id === newlyAddedStockId}
						active={stock.id === activeStockId}
						on:click={() => setActiveStock(stock.id === activeStockId ? null : stock.id)}
						on:toggle-expanded={handleToggleExpanded}
					/>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Execution Section -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
		<!-- Execution Control -->
		<ExecutionControl disabled={executing} on:executed={handleExecute} />

		<!-- Execution Results -->
		<ExecutionResults results={executionResults} visible={true} />
	</div>

	<!-- Recent Trades Section -->
	<div class="bg-white shadow rounded-lg">
		<div class="px-4 py-5 sm:p-6">
			<h2 class="text-lg font-medium text-gray-900 mb-4">Recent Trades</h2>

			{#if isLoadingTrades}
				<div class="flex justify-center py-10">
					<div
						class="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-blue-500"
					></div>
				</div>
			{:else if trades.length === 0}
				<div class="text-center py-10">
					<svg
						class="mx-auto h-12 w-12 text-gray-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
						/>
					</svg>
					<h3 class="mt-2 text-sm font-medium text-gray-900">No trades</h3>
					<p class="mt-1 text-sm text-gray-500">You haven't made any trades today.</p>
					<div class="mt-6">
						<a
							href="/order"
							class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
						>
							Place Your First Order
						</a>
					</div>
				</div>
			{:else}
				<div class="overflow-x-auto">
					<table class="min-w-full divide-y divide-gray-200">
						<thead class="bg-gray-50">
							<tr>
								<th
									scope="col"
									class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
									>Symbol</th
								>
								<th
									scope="col"
									class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
									>Type</th
								>
								<th
									scope="col"
									class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
									>Quantity</th
								>
								<th
									scope="col"
									class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
									>Price</th
								>
								<th
									scope="col"
									class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
									>Time</th
								>
							</tr>
						</thead>
						<tbody class="bg-white divide-y divide-gray-200">
							{#each trades.slice(0, 5) as trade}
								<tr>
									<td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900"
										>{trade.symbol}</td
									>
									<td class="px-6 py-4 whitespace-nowrap">
										<span
											class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full {trade.transactionType ===
											'BUY'
												? 'bg-green-100 text-green-800'
												: 'bg-red-100 text-red-800'}"
										>
											{trade.transactionType}
										</span>
									</td>
									<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{trade.quantity}</td
									>
									<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
										>{formatCurrency(trade.price)}</td
									>
									<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
										{new Date(trade.timestamp).toLocaleTimeString()}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>

					{#if trades.length > 5}
						<div class="mt-4 text-center">
							<a
								href="/trades"
								class="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
							>
								View All Trades
							</a>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>
