<script>
	import { onMount } from 'svelte';
	import { tradeApi } from '$lib/services/apiService';

	let trades = [];
	let isLoading = true;
	let error = null;

	// Statistics
	let stats = {
		totalTrades: 0,
		totalBuyValue: 0,
		totalSellValue: 0,
		netPosition: 0
	};

	onMount(async () => {
		try {
			// Fetch real trade data from the backend
			const response = await tradeApi.getAllTrades();
			trades = response.trades || [];

			// Calculate statistics
			calculateStats();
			isLoading = false;
		} catch (err) {
			console.error('Error fetching trades:', err);
			error = err.message || 'Failed to load trades';
			isLoading = false;

			// If API fails, use mock data for demonstration
			useMockData();
		}
	});

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
		error = 'Could not connect to API. Using demo data for visualization.';
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
</script>

<svelte:head>
	<title>Dashboard | SetBull Trader</title>
</svelte:head>

<div class="py-6">
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
						{isLoading ? '...' : stats.totalTrades}
					</dd>
				</div>
			</div>

			<!-- Buy Value -->
			<div class="bg-white overflow-hidden shadow rounded-lg">
				<div class="px-4 py-5 sm:p-6">
					<dt class="text-sm font-medium text-gray-500 truncate">Total Buy Value</dt>
					<dd class="mt-1 text-3xl font-semibold text-gray-900">
						{isLoading ? '...' : formatCurrency(stats.totalBuyValue)}
					</dd>
				</div>
			</div>

			<!-- Sell Value -->
			<div class="bg-white overflow-hidden shadow rounded-lg">
				<div class="px-4 py-5 sm:p-6">
					<dt class="text-sm font-medium text-gray-500 truncate">Total Sell Value</dt>
					<dd class="mt-1 text-3xl font-semibold text-gray-900">
						{isLoading ? '...' : formatCurrency(stats.totalSellValue)}
					</dd>
				</div>
			</div>

			<!-- Net Position -->
			<div class="bg-white overflow-hidden shadow rounded-lg">
				<div class="px-4 py-5 sm:p-6">
					<dt class="text-sm font-medium text-gray-500 truncate">Net Position</dt>
					<dd
						class="mt-1 text-3xl font-semibold {isLoading
							? ''
							: getNetPositionClass(stats.netPosition)}"
					>
						{isLoading ? '...' : formatCurrency(stats.netPosition)}
					</dd>
				</div>
			</div>
		</dl>
	</div>

	<!-- Connection Status -->
	{#if error}
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
					<p class="text-sm text-yellow-700">{error}</p>
					<p class="text-sm text-yellow-600 mt-2">
						Make sure your backend API is running at http://localhost:8080/api/v1
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Quick Actions -->
	<div class="bg-white shadow rounded-lg mb-8">
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
	</div>

	<!-- Recent Trades Section -->
	<div class="bg-white shadow rounded-lg">
		<div class="px-4 py-5 sm:p-6">
			<h2 class="text-lg font-medium text-gray-900 mb-4">Recent Trades</h2>

			{#if isLoading}
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
