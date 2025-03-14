<!-- src/routes/trades/history/+page.svelte -->
<script>
	import { onMount } from 'svelte';
	onMount(() => {
		console.log('Component is now mounted!');
	});
	let trades = [];
	let isLoading = false;
	let error = null;

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
			// For demonstration purposes, use mock data instead of making an actual API call
			// In a real app, you would uncomment the API call below when your backend is ready

			const response = await fetch(
				`/api/v1/trades/history?fromDate=${fromDate}&toDate=${toDate}&page=${currentPage}`
			);

			if (!response.ok) {
				// Check if we got HTML instead of JSON
				const contentType = response.headers.get('content-type');
				if (contentType && contentType.includes('text/html')) {
					throw new Error(
						'Received HTML response instead of JSON. The API server may not be running.'
					);
				}

				const errorData = await response.json();
				throw new Error(errorData.error || `Server error: ${response.status}`);
			}

			const data = await response.json();
			trades = data.trades || [];

			isLoading = false;
		} catch (err) {
			console.error('Error fetching trade history:', err);
			error = err.message || 'Failed to load trade history';
			isLoading = false;
		}
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

	// Initialize on mount
	onMount(() => {
		// Don't auto-fetch on mount - wait for user to click search
		// Just initialize with empty state
	});

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
</script>

<svelte:head>
	<title>Trade History | SetBull Trader</title>
</svelte:head>

<div class="py-6">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Trade History</h1>

	<!-- Search Form -->
	<div class="bg-white shadow rounded-lg p-6 mb-6">
		<form on:submit|preventDefault={handleSearchSubmit} class="flex flex-wrap gap-4 items-end">
			<div>
				<label for="fromDate" class="block text-sm font-medium text-gray-700 mb-1">From Date</label>
				<input
					type="date"
					id="fromDate"
					bind:value={fromDate}
					max={toDate}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				/>
			</div>

			<div>
				<label for="toDate" class="block text-sm font-medium text-gray-700 mb-1">To Date</label>
				<input
					type="date"
					id="toDate"
					bind:value={toDate}
					min={fromDate}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				/>
			</div>

			<div>
				<button
					type="submit"
					class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
				>
					Search
				</button>
			</div>
		</form>
	</div>

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
					<p class="text-sm text-red-600 mt-2">This might be happening because:</p>
					<ul class="list-disc pl-5 mt-1">
						<li>The backend API is not running</li>
						<li>There's a CORS issue</li>
						<li>The API URL is incorrect</li>
					</ul>

					<p class="text-sm text-red-600 mt-2">
						For now, you can use the demo data by clicking the Search button.
					</p>
				</div>
			</div>
		</div>
	{:else if !trades.length && !isLoading}
		{#if currentPage === 0}
			<div class="bg-yellow-50 border-l-4 border-yellow-400 p-4 mb-6">
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
						<p class="text-sm text-yellow-700">
							Click the Search button to view demo trade history data.
						</p>
					</div>
				</div>
			</div>
		{:else}
			<div class="text-center py-10 bg-white shadow rounded-lg">
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
				<h3 class="mt-2 text-sm font-medium text-gray-900">No trade history</h3>
				<p class="mt-1 text-sm text-gray-500">No trades found for the selected date range.</p>
			</div>
		{/if}
	{:else}
		<div class="bg-white shadow overflow-hidden rounded-lg">
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
								>Fees</th
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
									>{formatCurrency(trade.price)}</td
								>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
									>{formatCurrency(trade.brokerageFees + trade.taxesFees)}</td
								>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
									>{formatDate(trade.timestamp)}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Pagination -->
			<div class="px-4 py-3 flex items-center justify-between border-t border-gray-200 sm:px-6">
				<div class="flex-1 flex justify-between sm:hidden">
					<button
						on:click={prevPage}
						disabled={currentPage === 0}
						class="relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
					>
						Previous
					</button>
					<button
						on:click={nextPage}
						disabled={trades.length === 0}
						class="ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
					>
						Next
					</button>
				</div>
				<div class="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
					<div>
						<p class="text-sm text-gray-700">
							Showing page <span class="font-medium">{currentPage + 1}</span> of results
						</p>
					</div>
					<div>
						<nav
							class="relative z-0 inline-flex rounded-md shadow-sm -space-x-px"
							aria-label="Pagination"
						>
							<button
								on:click={prevPage}
								disabled={currentPage === 0}
								class="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								<span class="sr-only">Previous</span>
								<svg
									class="h-5 w-5"
									xmlns="http://www.w3.org/2000/svg"
									viewBox="0 0 20 20"
									fill="currentColor"
									aria-hidden="true"
								>
									<path
										fill-rule="evenodd"
										d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z"
										clip-rule="evenodd"
									/>
								</svg>
							</button>
							<span
								class="relative inline-flex items-center px-4 py-2 border border-gray-300 bg-white text-sm font-medium text-gray-700"
							>
								{currentPage + 1}
							</span>
							<button
								on:click={nextPage}
								disabled={trades.length === 0}
								class="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								<span class="sr-only">Next</span>
								<svg
									class="h-5 w-5"
									xmlns="http://www.w3.org/2000/svg"
									viewBox="0 0 20 20"
									fill="currentColor"
									aria-hidden="true"
								>
									<path
										fill-rule="evenodd"
										d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
										clip-rule="evenodd"
									/>
								</svg>
							</button>
						</nav>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
