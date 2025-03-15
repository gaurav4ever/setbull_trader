<!-- frontend/src/routes/order/+page.svelte -->
<script>
	import { onMount } from 'svelte';
	import Autocomplete from '$lib/components/Autocomplete.svelte';
	import { getStocksList } from '$lib/services/stocksService';

	// Order form data
	let formData = {
		transactionType: 'BUY',
		exchangeSegment: 'NSE_EQ',
		productType: 'CNC',
		orderType: 'LIMIT',
		securityId: '',
		quantity: 1,
		disclosedQty: 0,
		price: 0,
		triggerPrice: 0,
		validity: 'DAY',
		isAMO: false,
		targetPrice: 0,
		stopLossPrice: 0
	};

	// Stock list for autocomplete
	let stocksList = [];
	let isLoadingStocks = true;

	// Form state
	let isSubmitting = false;
	let errorMessage = '';
	let successMessage = '';

	// Options for select fields
	const transactionTypes = ['BUY', 'SELL'];
	const exchangeSegments = ['NSE_EQ', 'NSE_FNO', 'BSE_EQ', 'BSE_FNO', 'MCX_COMM'];
	const productTypes = ['CNC', 'INTRADAY', 'MARGIN', 'MTF', 'CO', 'BO'];
	const orderTypes = ['LIMIT', 'MARKET', 'STOP_LOSS', 'STOP_LOSS_MARKET'];
	const validityTypes = ['DAY', 'IOC'];

	onMount(async () => {
		// Load stocks list when component mounts
		try {
			stocksList = await getStocksList();
			console.log(`Loaded ${stocksList.length} stocks for autocomplete`);
		} catch (error) {
			console.error('Error loading stocks:', error);
			errorMessage = 'Failed to load stocks list. Please refresh the page.';
		} finally {
			isLoadingStocks = false;
		}
	});

	// Handle stock selection
	function handleStockSelect(event) {
		formData.securityId = event.detail;
	}

	// Handle form submission
	async function handleSubmit() {
		isSubmitting = true;
		errorMessage = '';
		successMessage = '';

		try {
			const response = await fetch('/api/v1/orders', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(formData)
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || 'Failed to place order');
			}

			successMessage = `Order placed successfully! Order ID: ${data.orderId}`;

			// Reset form after successful submission
			resetForm();
		} catch (error) {
			errorMessage = error.message || 'Failed to place order. Please try again.';
		} finally {
			isSubmitting = false;
		}
	}

	// Reset form to initial state
	function resetForm() {
		formData = {
			transactionType: 'BUY',
			exchangeSegment: 'NSE_EQ',
			productType: 'CNC',
			orderType: 'LIMIT',
			securityId: '',
			quantity: 1,
			disclosedQty: 0,
			price: 0,
			triggerPrice: 0,
			validity: 'DAY',
			isAMO: false,
			targetPrice: 0,
			stopLossPrice: 0
		};
	}

	// Function to check if stop loss fields should be shown
	function showStopLoss() {
		return formData.productType === 'BO' || formData.productType === 'CO';
	}

	// Function to check if target price field should be shown
	function showTargetPrice() {
		return formData.productType === 'BO';
	}

	// Function to check if trigger price field should be shown
	function showTriggerPrice() {
		return formData.orderType === 'STOP_LOSS' || formData.orderType === 'STOP_LOSS_MARKET';
	}

	// Function to check if price field should be shown
	function showPrice() {
		return formData.orderType === 'LIMIT' || formData.orderType === 'STOP_LOSS';
	}
</script>

<svelte:head>
	<title>Place Order | SetBull Trader</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Place New Order</h1>

	<!-- Alert Messages -->
	{#if errorMessage}
		<div class="bg-red-50 border-l-4 border-red-400 p-4 mb-6">
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
					<p class="text-sm text-red-700">{errorMessage}</p>
				</div>
			</div>
		</div>
	{/if}

	{#if successMessage}
		<div class="bg-green-50 border-l-4 border-green-400 p-4 mb-6">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-green-400" viewBox="0 0 20 20" fill="currentColor">
						<path
							fill-rule="evenodd"
							d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
							clip-rule="evenodd"
						/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm text-green-700">{successMessage}</p>
				</div>
			</div>
		</div>
	{/if}

	<form on:submit|preventDefault={handleSubmit} class="bg-white shadow-md rounded-lg p-6">
		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<!-- Transaction Type -->
			<div>
				<label for="transactionType" class="block text-sm font-medium text-gray-700"
					>Transaction Type</label
				>
				<select
					id="transactionType"
					bind:value={formData.transactionType}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				>
					{#each transactionTypes as type}
						<option value={type}>{type}</option>
					{/each}
				</select>
			</div>

			<!-- Exchange Segment -->
			<div>
				<label for="exchangeSegment" class="block text-sm font-medium text-gray-700"
					>Exchange Segment</label
				>
				<select
					id="exchangeSegment"
					bind:value={formData.exchangeSegment}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				>
					{#each exchangeSegments as segment}
						<option value={segment}>{segment}</option>
					{/each}
				</select>
			</div>

			<!-- Product Type -->
			<div>
				<label for="productType" class="block text-sm font-medium text-gray-700">Product Type</label
				>
				<select
					id="productType"
					bind:value={formData.productType}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				>
					{#each productTypes as type}
						<option value={type}>{type}</option>
					{/each}
				</select>
			</div>

			<!-- Order Type -->
			<div>
				<label for="orderType" class="block text-sm font-medium text-gray-700">Order Type</label>
				<select
					id="orderType"
					bind:value={formData.orderType}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				>
					{#each orderTypes as type}
						<option value={type}>{type}</option>
					{/each}
				</select>
			</div>

			<!-- Security ID with Autocomplete -->
			<div>
				<label for="securityId" class="block text-sm font-medium text-gray-700">Security ID</label>
				{#if isLoadingStocks}
					<div class="mt-1 flex items-center">
						<div class="animate-pulse h-9 bg-gray-200 rounded w-full"></div>
						<span class="ml-2 text-sm text-gray-500">Loading stocks...</span>
					</div>
				{:else}
					<Autocomplete
						items={stocksList}
						bind:value={formData.securityId}
						on:select={handleStockSelect}
						placeholder="Type to search stocks..."
						inputId="securityId"
						name="securityId"
						inputClass="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
						maxItems={10}
						minChars={1}
					/>
				{/if}
			</div>

			<!-- Quantity -->
			<div>
				<label for="quantity" class="block text-sm font-medium text-gray-700">Quantity</label>
				<input
					type="number"
					id="quantity"
					bind:value={formData.quantity}
					min="1"
					required
					class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
				/>
			</div>

			<!-- Validity -->
			<div>
				<label for="validity" class="block text-sm font-medium text-gray-700">Validity</label>
				<select
					id="validity"
					bind:value={formData.validity}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				>
					{#each validityTypes as type}
						<option value={type}>{type}</option>
					{/each}
				</select>
			</div>

			<!-- Price (conditionally shown) -->
			{#if showPrice()}
				<div>
					<label for="price" class="block text-sm font-medium text-gray-700">Price</label>
					<input
						type="number"
						id="price"
						bind:value={formData.price}
						min="0"
						step="0.05"
						class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
					/>
				</div>
			{/if}

			<!-- Trigger Price (conditionally shown) -->
			{#if showTriggerPrice()}
				<div>
					<label for="triggerPrice" class="block text-sm font-medium text-gray-700"
						>Trigger Price</label
					>
					<input
						type="number"
						id="triggerPrice"
						bind:value={formData.triggerPrice}
						min="0"
						step="0.05"
						class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
					/>
				</div>
			{/if}

			<!-- Disclosed Quantity -->
			<div>
				<label for="disclosedQty" class="block text-sm font-medium text-gray-700"
					>Disclosed Quantity</label
				>
				<input
					type="number"
					id="disclosedQty"
					bind:value={formData.disclosedQty}
					min="0"
					class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
				/>
			</div>

			<!-- After Market Order Checkbox -->
			<div class="flex items-center h-12 mt-6">
				<input
					id="isAMO"
					type="checkbox"
					bind:checked={formData.isAMO}
					class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
				/>
				<label for="isAMO" class="ml-2 block text-sm text-gray-700">After Market Order</label>
			</div>

			<!-- Stop Loss Price (conditionally shown) -->
			{#if showStopLoss()}
				<div>
					<label for="stopLossPrice" class="block text-sm font-medium text-gray-700"
						>Stop Loss Price</label
					>
					<input
						type="number"
						id="stopLossPrice"
						bind:value={formData.stopLossPrice}
						min="0"
						step="0.05"
						class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
					/>
				</div>
			{/if}

			<!-- Target Price (conditionally shown) -->
			{#if showTargetPrice()}
				<div>
					<label for="targetPrice" class="block text-sm font-medium text-gray-700"
						>Target Price</label
					>
					<input
						type="number"
						id="targetPrice"
						bind:value={formData.targetPrice}
						min="0"
						step="0.05"
						class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
					/>
				</div>
			{/if}
		</div>

		<div class="mt-8 flex justify-end">
			<button
				type="button"
				on:click={resetForm}
				class="mr-3 inline-flex justify-center py-2 px-4 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
			>
				Reset
			</button>
			<button
				type="submit"
				disabled={isSubmitting}
				class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
			>
				{isSubmitting ? 'Placing Order...' : 'Place Order'}
			</button>
		</div>
	</form>
</div>
