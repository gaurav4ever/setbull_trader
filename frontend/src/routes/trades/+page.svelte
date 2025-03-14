<!-- src/routes/trades/+page.svelte -->
<script>
	import { onMount } from 'svelte';
	import { tradeApi } from '$lib/services/apiService';

	let trades = [];
	let isLoading = false;
	let error = null;
	let apiConnected = true;

	// Pagination
	let currentPage = 0;

	// Date range
	let fromDate = formatDateForInput(new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)); // 7 days ago
	let toDate = formatDateForInput(new Date()); // Today

	function formatDateForInput(date) {
		return date.toISOString().split('T')[0];
	}

	async function fetchTradeHistory() {
		isLoading = true;
		error = null;

		try {
			// Fetch data from the API
			const response = await tradeApi.getTradeHistory(fromDate, toDate, currentPage);

			trades = response.trades || [];
			isLoading = false;
			apiConnected = true;
		} catch (err) {
			console.error('Error fetching trade history:', err);
			error = err.message || 'Failed to load trade history';
			isLoading = false;
			apiConnected = false;

			// Use mock data if the API fails
			useMockData();
		}
	}

	function useMockData() {
		console.log('Using mock data as fallback');
		trades = [
			{
				orderId: 'ORD001',
				transactionType: 'BUY',
				symbol: 'RELIANCE',
				exchangeSegment: 'NSE_EQ',
				productType: 'CNC',
				quantity: 10,
				price: 2500.5,
				brokerageFees: 25.75,
				taxesFees: 15.3,
				timestamp: '2025-03-10T09:30:45Z'
			},
			{
				orderId: 'ORD002',
				transactionType: 'SELL',
				symbol: 'INFY',
				exchangeSegment: 'NSE_EQ',
				productType: 'CNC',
				quantity: 5,
				price: 1750.25,
				brokerageFees: 12.8,
				taxesFees: 8.2,
				timestamp: '2025-03-12T10:15:30Z'
			},
			{
				orderId: 'ORD003',
				transactionType: 'BUY',
				symbol: 'TCS',
				exchangeSegment: 'NSE_EQ',
				productType: 'INTRADAY',
				quantity: 2,
				price: 3450.75,
				brokerageFees: 10.35,
				taxesFees: 6.9,
				timestamp: '2025-03-13T11:45:15Z'
			}
		];
	}

	function handleSearchSubmit() {
		currentPage = 0;
		fetchTradeHistory();
	}

	function nextPage() {
		currentPage += 1;
		fetchTradeHistory();
	}

	function prevPage() {
		if (currentPage > 0) {
			currentPage -= 1;
			fetchTradeHistory();
		}
	}

	function formatDate(dateString) {
		if (!dateString) return 'N/A';
		const date = new Date(dateString);
		return new Intl.DateTimeFormat('en-IN', {
			year: 'numeric',
			month: 'short',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		}).format(date);
	}

	function formatCurrency(price) {
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			minimumFractionDigits: 2
		}).format(price);
	}

	function getTransactionClass(type) {
		return type === 'BUY' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800';
	}

	function formatPrice(price) {
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			minimumFractionDigits: 2
		}).format(price);
	}
</script>

<svelte:head>
	<title>Trades | SetBull Trader</title>
</svelte:head>

<div class="py-6">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Today's Trades</h1>

	{#if isLoading}
		<div class="flex justify-center py-20">
			<div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
		</div>
	{:else if error}
		<div class="bg-red-50 border-l-4 border-red-400 p-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
						<path
							fill-rule="evenodd"
							d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
							clip-rule="evenodd"
						/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm text-red-700">{error}</p>
				</div>
			</div>
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
					class="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
				>
					Place an Order
				</a>
			</div>
		</div>
	{:else}
		<div class="bg-white shadow overflow-hidden rounded-md">
			<div class="overflow-x-auto">
				<table class="min-w-full divide-y divide-gray-200">
					<thead class="bg-gray-50">
						<tr>
							<th
								scope="col"
								class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
								>Order ID</th
							>
							<th
								scope="col"
								class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
								>Type</th
							>
							<th
								scope="col"
								class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
								>Symbol</th
							>
							<th
								scope="col"
								class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
								>Exchange</th
							>
							<th
								scope="col"
								class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
								>Product</th
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
						{#each trades as trade}
							<tr>
								<td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900"
									>{trade.orderId}</td
								>
								<td class="px-6 py-4 whitespace-nowrap">
									<span
										class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full {getTransactionClass(
											trade.transactionType
										)}"
									>
										{trade.transactionType}
									</span>
								</td>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{trade.symbol}</td>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
									>{trade.exchangeSegment}</td
								>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
									>{trade.productType}</td
								>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{trade.quantity}</td>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
									>{formatPrice(trade.price)}</td
								>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
									>{formatDate(trade.timestamp)}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>
