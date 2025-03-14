<!-- src/routes/order/modify/+page.svelte -->
<script>
	import { onMount } from 'svelte';

	let orderId = '';
	let formData = {
		orderType: '',
		quantity: '',
		price: '',
		disclosedQty: '',
		triggerPrice: '',
		validity: ''
	};

	// Form state
	let isSubmitting = false;
	let errorMessage = '';
	let successMessage = '';

	// Options for select fields
	const orderTypes = ['LIMIT', 'MARKET', 'STOP_LOSS', 'STOP_LOSS_MARKET'];
	const validityTypes = ['DAY', 'IOC'];

	// Handle form submission
	async function handleSubmit() {
		if (!orderId) {
			errorMessage = 'Order ID is required';
			return;
		}

		isSubmitting = true;
		errorMessage = '';
		successMessage = '';

		// Filter out empty fields
		const payload = Object.entries(formData)
			.filter(([_, value]) => value !== '')
			.reduce((acc, [key, value]) => {
				acc[key] = value;
				return acc;
			}, {});

		try {
			const response = await fetch(`/api/v1/orders/${orderId}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(payload)
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || 'Failed to modify order');
			}

			successMessage = `Order modified successfully! Order ID: ${data.orderId}`;
			resetForm();
		} catch (error) {
			errorMessage = error.message || 'Failed to modify order. Please try again.';
		} finally {
			isSubmitting = false;
		}
	}

	// Reset form to initial state
	function resetForm() {
		formData = {
			orderType: '',
			quantity: '',
			price: '',
			disclosedQty: '',
			triggerPrice: '',
			validity: ''
		};
	}
</script>

<svelte:head>
	<title>Modify Order | SetBull Trader</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Modify Existing Order</h1>

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
		<!-- Order ID -->
		<div class="mb-6">
			<label for="orderId" class="block text-sm font-medium text-gray-700">Order ID</label>
			<input
				type="text"
				id="orderId"
				bind:value={orderId}
				required
				placeholder="Enter the order ID you want to modify"
				class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
			/>
		</div>

		<p class="text-sm text-gray-500 mb-6">Fill only the fields you want to modify:</p>

		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<!-- Order Type -->
			<div>
				<label for="orderType" class="block text-sm font-medium text-gray-700">Order Type</label>
				<select
					id="orderType"
					bind:value={formData.orderType}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				>
					<option value="" selected>No change</option>
					{#each orderTypes as type}
						<option value={type}>{type}</option>
					{/each}
				</select>
			</div>

			<!-- Validity -->
			<div>
				<label for="validity" class="block text-sm font-medium text-gray-700">Validity</label>
				<select
					id="validity"
					bind:value={formData.validity}
					class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
				>
					<option value="" selected>No change</option>
					{#each validityTypes as type}
						<option value={type}>{type}</option>
					{/each}
				</select>
			</div>

			<!-- Quantity -->
			<div>
				<label for="quantity" class="block text-sm font-medium text-gray-700">Quantity</label>
				<input
					type="number"
					id="quantity"
					bind:value={formData.quantity}
					min="1"
					placeholder="No change"
					class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
				/>
			</div>

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
					placeholder="No change"
					class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
				/>
			</div>

			<!-- Price -->
			<div>
				<label for="price" class="block text-sm font-medium text-gray-700">Price</label>
				<input
					type="number"
					id="price"
					bind:value={formData.price}
					min="0"
					step="0.05"
					placeholder="No change"
					class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
				/>
			</div>

			<!-- Trigger Price -->
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
					placeholder="No change"
					class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
				/>
			</div>
		</div>

		<div class="mt-8 flex justify-end">
			<button
				type="button"
				on:click={resetForm}
				class="mr-3 inline-flex justify-center py-2 px-4 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
			>
				Reset
			</button>
			<button
				type="submit"
				disabled={isSubmitting}
				class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
			>
				{isSubmitting ? 'Modifying Order...' : 'Modify Order'}
			</button>
		</div>
	</form>
</div>
