<!-- frontend/src/lib/components/StockParameterForm.svelte -->
<script>
	import { createEventDispatcher } from 'svelte';
	import NumericInput from './ui/NumericInput.svelte';
	import { validateTradingParameters, hasErrors } from '../utils/validation';

	// Props
	export let stockSymbol = '';
	export let onCancel = () => {};

	// State
	let isLoading = false;
	let error = '';
	/** @type {Record<string, any>} */
	let formErrors = {};

	// Form data with defaults
	/** @type {Record<string, string>} */
	let formData = {
		startingPrice: '',
		stopLossPercentage: '',
		riskAmount: '30',
		tradeSide: 'BUY',
		psType: 'FIXED',
		entryType: '1ST_ENTRY'
	};

	// Event dispatcher
	const dispatch = createEventDispatcher();

	// Save parameters and dispatch event
	async function handleSubmit() {
		// Validate form data
		formErrors = validateTradingParameters(formData);
		if (hasErrors(formErrors)) {
			return;
		}

		isLoading = true;
		error = '';

		try {
			// Dispatch the submit event with form data
			dispatch('submit', {
				stockSymbol,
				parameters: formData
			});
		} catch (err) {
			console.error('Error in form submission:', err);
			const message =
				typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
					? err.message
					: String(err);
			error = message || 'An error occurred during submission';
		} finally {
			isLoading = false;
		}
	}

	// Handle form cancellation
	function handleCancel() {
		if (typeof onCancel === 'function') {
			onCancel();
		}
		dispatch('cancel');
	}

	// Handle input changes
	/**
	 * @param {string} field
	 * @param {CustomEvent<any>} event
	 */
	function handleInputChange(field, event) {
		formData[field] = event.detail;
		if (formErrors[field]) {
			formErrors[field] = null;
		}
		// Clear general error when form is changed
		error = '';
	}

	// Handle trade side change
	/**
	 * @param {Event} event
	 */
	function handleTradeSideChange(event) {
		const target = event.target;
		if (target && typeof target.value === 'string') {
			formData.tradeSide = target.value;
		}
		if (formErrors.tradeSide) {
			formErrors.tradeSide = null;
		}
		// Clear general error
		error = '';
	}
</script>

<div class="stock-parameter-form bg-white rounded-lg shadow-md p-6">
	<div class="mb-4">
		<h3 class="text-lg font-medium text-gray-900">{stockSymbol} Parameters</h3>
		<p class="text-sm text-gray-500">Set trading parameters before adding this stock</p>
	</div>

	{#if error}
		<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md">
			<p>{error}</p>
		</div>
	{/if}

	<form on:submit|preventDefault={handleSubmit} class="space-y-4">
		<!-- Starting Price -->
		<div>
			<NumericInput
				id="startingPrice"
				name="startingPrice"
				label="Starting Price"
				value={String(formData.startingPrice ?? '')}
				on:input={(e) => {
					formData.startingPrice = e.detail;
					formErrors.startingPrice = null;
				}}
				error={String(formErrors.startingPrice ?? '')}
				required={true}
				min={0.01}
				precision={2}
				placeholder="Enter starting price"
				helpText="The price at which you want to start trading"
			/>
		</div>

		<!-- Stop Loss Percentage -->
		<div>
			<NumericInput
				id="stopLossPercentage"
				name="stopLossPercentage"
				label="Stop Loss Percentage"
				value={String(formData.stopLossPercentage ?? '')}
				on:input={(e) => {
					formData.stopLossPercentage = e.detail;
					formErrors.stopLossPercentage = null;
				}}
				error={String(formErrors.stopLossPercentage ?? '')}
				required={true}
				min={0.1}
				max={5}
				precision={2}
				placeholder="Enter stop loss percentage"
				helpText="Recommended: 0.5% - 0.8%"
			/>
		</div>

		<!-- Risk Amount -->
		<div>
			<NumericInput
				id="riskAmount"
				name="riskAmount"
				label="Risk Amount (₹)"
				value={String(formData.riskAmount ?? '')}
				on:change={(e) => handleInputChange('riskAmount', e)}
				error={String(formErrors.riskAmount ?? '')}
				required={true}
				min={1}
				precision={2}
				placeholder="Enter risk amount"
				helpText="Amount you're willing to risk on this trade"
			/>
		</div>

		<!-- Trade Side -->
		<div>
			<label for="tradeSide" class="block text-sm font-medium text-gray-700 mb-1">
				Trade Side
				<span class="text-red-500">*</span>
			</label>
			<div class="mt-1">
				<select
					id="tradeSide"
					name="tradeSide"
					bind:value={formData.tradeSide}
					on:change={handleTradeSideChange}
					class={`block w-full px-3 py-2 border ${formErrors.tradeSide ? 'border-red-300' : 'border-gray-300'} rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm`}
				>
					<option value="BUY">BUY</option>
					<option value="SELL">SELL</option>
				</select>
			</div>
			{#if formErrors.tradeSide}
				<p class="mt-1 text-sm text-red-600">{formErrors.tradeSide}</p>
			{/if}
		</div>

		<!-- Position Sizing Type -->
		<div>
			<label for="psType" class="block text-sm font-medium text-gray-700 mb-1">
				Position Sizing Type
				<span class="text-red-500">*</span>
			</label>
			<div class="mt-1">
				<select
					id="psType"
					name="psType"
					bind:value={formData.psType}
					class="block w-full px-3 py-2 border rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
				>
					<option value="FIXED">FIXED</option>
					<option value="DYNAMIC">DYNAMIC</option>
				</select>
			</div>
			{#if formErrors.psType}
				<p class="mt-1 text-sm text-red-600">{formErrors.psType}</p>
			{/if}
		</div>

		<!-- Entry Type -->
		<div>
			<label for="entryType" class="block text-sm font-medium text-gray-700 mb-1">
				Entry Type
				<span class="text-red-500">*</span>
			</label>
			<div class="mt-1">
				<select
					id="entryType"
					name="entryType"
					bind:value={formData.entryType}
					class="block w-full px-3 py-2 border rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
				>
					<option value="1ST_ENTRY">1ST_ENTRY</option>
					<option value="2_30_ENTRY">2_30_ENTRY</option>
				</select>
			</div>
			{#if formErrors.entryType}
				<p class="mt-1 text-sm text-red-600">{formErrors.entryType}</p>
			{/if}
		</div>

		<!-- Action Buttons -->
		<div class="flex justify-end space-x-3 pt-4">
			<button
				type="button"
				on:click={handleCancel}
				class="px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
			>
				Cancel
			</button>
			<button
				type="submit"
				class="px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
				disabled={isLoading}
			>
				{#if isLoading}
					<svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
					Saving...
				{:else}
					Add Stock
				{/if}
			</button>
		</div>
	</form>
</div>
