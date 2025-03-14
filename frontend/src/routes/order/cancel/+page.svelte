<!-- src/routes/order/cancel/+page.svelte -->
<script>
	let orderId = '';
	let isSubmitting = false;
	let errorMessage = '';
	let successMessage = '';

	async function handleSubmit() {
		if (!orderId) {
			errorMessage = 'Order ID is required';
			return;
		}

		isSubmitting = true;
		errorMessage = '';
		successMessage = '';

		try {
			const response = await fetch(`/api/v1/orders/${orderId}`, {
				method: 'DELETE',
				headers: {
					'Content-Type': 'application/json'
				}
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || 'Failed to cancel order');
			}

			successMessage = `Order cancelled successfully! Order ID: ${data.orderId}`;
			orderId = '';
		} catch (error) {
			errorMessage = error.message || 'Failed to cancel order. Please try again.';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>Cancel Order | SetBull Trader</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Cancel Order</h1>

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

	<div class="bg-white shadow-md rounded-lg p-6">
		<form on:submit|preventDefault={handleSubmit}>
			<div class="mb-6">
				<label for="orderId" class="block text-sm font-medium text-gray-700">Order ID</label>
				<input
					type="text"
					id="orderId"
					bind:value={orderId}
					placeholder="Enter the order ID you want to cancel"
					required
					class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
				/>
			</div>

			<div class="flex justify-end">
				<button
					type="submit"
					disabled={isSubmitting}
					class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
				>
					{isSubmitting ? 'Cancelling Order...' : 'Cancel Order'}
				</button>
			</div>
		</form>

		<div class="mt-6 bg-yellow-50 border-l-4 border-yellow-400 p-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
						<path
							fill-rule="evenodd"
							d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
							clip-rule="evenodd"
						/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm text-yellow-700">
						<strong>Warning:</strong> Cancelling an order cannot be undone. Please make sure you have
						the correct Order ID.
					</p>
				</div>
			</div>
		</div>
	</div>
</div>
