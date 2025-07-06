<!-- frontend/src/lib/components/TradingParameters.svelte -->
<script>
	import { createEventDispatcher, onMount } from 'svelte';
	import NumericInput from './ui/NumericInput.svelte';
	import { validateTradingParameters, hasErrors } from '../utils/validation';

	// Props
	export let stockId = '';
	export let stockSymbol = '';
	export let stockSecurityId = ''; // Security ID for order placement
	export let initialParameters = null;
	export let readOnly = false;
	export let isNewStock = false; // New prop to indicate if this is a newly added stock

	// State
	let isLoading = false;
	let isSaving = false;
	let error = '';
	let successMessage = '';

	// Type definitions
	/** @typedef {Object} TradingParametersFormData
	 *  @property {string} stockId
	 *  @property {string} stockSymbol
	 *  @property {string} stockSecurityId
	 *  @property {string|number} startingPrice
	 *  @property {string|number} stopLossPercentage
	 *  @property {string|number} riskAmount
	 *  @property {string} tradeSide
	 *  @property {string} psType
	 *  @property {string} entryType
	 */

	/** @type {Record<string, any>} */
	let formErrors = {};

	/** @type {Record<string, string | number>} */
	let formData = {
		stockId: stockId,
		stockSymbol: stockSymbol,
		stockSecurityId: stockSecurityId,
		startingPrice: '',
		stopLossPercentage: '',
		riskAmount: '30',
		tradeSide: 'BUY',
		psType: 'FIXED',
		entryType: '1ST_ENTRY'
	};

	// Event dispatcher
	const dispatch = createEventDispatcher();

	// Handle form initialization and loading of existing parameters
	onMount(async () => {
		if (initialParameters && typeof initialParameters === 'object' && initialParameters !== null) {
			formData = {
				...formData,
				...initialParameters,
				stockSymbol: stockSymbol,
				stockSecurityId: stockSecurityId
			};
		} else if (stockId) {
			await loadParameters();
		}

		// If this is a new stock and we don't have parameters, auto-focus the first field
		if (isNewStock && !initialParameters) {
			// Use setTimeout to ensure DOM is ready
			setTimeout(() => {
				const startingPriceInput = document.getElementById('startingPrice');
				if (startingPriceInput) {
					startingPriceInput.focus();
				}
			}, 100);
		}
	});

	// Load parameters from API
	async function loadParameters() {
		if (!stockId) return;

		isLoading = true;
		error = '';

		try {
			const response = await fetch(`/api/v1/parameters/stock/${stockId}`);
			const result = await response.json();

			if (!response.ok) {
				throw new Error(result.error || 'Failed to load parameters');
			}

			if (result.data) {
				formData = {
					stockId: stockId,
					stockSymbol: stockSymbol,
					stockSecurityId: stockSecurityId,
					startingPrice:
						result.data.startingPrice !== undefined ? String(result.data.startingPrice) : '',
					stopLossPercentage:
						result.data.stopLossPercentage !== undefined
							? String(result.data.stopLossPercentage)
							: '',
					riskAmount: result.data.riskAmount !== undefined ? String(result.data.riskAmount) : '30',
					tradeSide: result.data.tradeSide ?? 'BUY',
					psType: result.data.psType ?? 'FIXED',
					entryType: result.data.entryType ?? '1ST_ENTRY'
				};
			}
		} catch (err) {
			console.error('Error loading parameters:', err);
			const message =
				typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
					? err.message
					: String(err);
			if (!(typeof message === 'string' && message.includes('not found'))) {
				error = message || 'Failed to load parameters';
			}
		} finally {
			isLoading = false;
		}
	}

	// Save parameters to API
	async function saveParameters() {
		formErrors = validateTradingParameters(formData);
		if (hasErrors(formErrors)) {
			return false;
		}

		isSaving = true;
		error = '';
		successMessage = '';

		try {
			const payload = {
				...formData,
				stockSecurityId: stockSecurityId || formData.stockSecurityId || ''
			};

			const response = await fetch('/api/v1/parameters', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(payload)
			});

			const result = await response.json();

			if (!response.ok) {
				throw new Error(result.error || 'Failed to save parameters');
			}

			successMessage = 'Parameters saved successfully!';

			const responseData = {
				...result.data,
				stockSymbol: stockSymbol || formData.stockSymbol,
				stockSecurityId: stockSecurityId || formData.stockSecurityId
			};

			dispatch('saved', responseData);
			return true;
		} catch (err) {
			console.error('Error saving parameters:', err);
			const message =
				typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
					? err.message
					: String(err);
			error = message || 'Failed to save parameters';
			return false;
		} finally {
			isSaving = false;
		}
	}

	// Handle form submission
	async function handleSubmit() {
		const success = await saveParameters();
		if (success) {
			// Notify parent component
			dispatch('submit', formData);
		}
	}

	/**
	 * @param {string} field
	 * @param {CustomEvent<any>} event
	 */
	function handleInputChange(field, event) {
		formData[field] = event.detail;
		if (formErrors[field]) {
			formErrors[field] = null;
		}
		successMessage = '';
		error = '';
	}

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
		successMessage = '';
		error = '';
	}
</script>

<div
	class="trading-parameters p-4 bg-white rounded-lg shadow {isNewStock
		? 'ring-2 ring-green-200'
		: ''}"
>
	{#if stockSymbol}
		<div class="mb-4">
			<h3 class="text-lg font-medium text-gray-900">
				{stockSymbol} Trading Parameters
				{#if isNewStock}
					<span class="ml-2 text-xs font-medium px-2 py-1 bg-green-100 text-green-700 rounded-full">
						New
					</span>
				{/if}
			</h3>
			{#if stockSecurityId && stockSecurityId !== stockSymbol}
				<p class="text-sm text-gray-500">Security ID: {stockSecurityId}</p>
			{/if}
			<p class="text-sm text-gray-500">Set trading parameters for this stock</p>
		</div>
	{/if}

	{#if error}
		<div class="mb-4 p-3 bg-red-50 text-red-700 rounded-md">
			<p>{error}</p>
		</div>
	{/if}

	{#if successMessage}
		<div class="mb-4 p-3 bg-green-50 text-green-700 rounded-md">
			<p>{successMessage}</p>
		</div>
	{/if}

	{#if isLoading}
		<div class="flex justify-center py-4">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
		</div>
	{:else}
		<form on:submit|preventDefault={handleSubmit} class="space-y-4">
			<!-- Hidden fields for stock info -->
			<input type="hidden" name="stockId" bind:value={formData.stockId} />
			<input type="hidden" name="stockSymbol" bind:value={formData.stockSymbol} />
			<input type="hidden" name="stockSecurityId" bind:value={formData.stockSecurityId} />

			<!-- Starting Price -->
			<div>
				<NumericInput
					id="startingPrice"
					name="startingPrice"
					label="Starting Price"
					bind:value={formData.startingPrice}
					on:change={(e) => handleInputChange('startingPrice', e)}
					error={formErrors.startingPrice || ''}
					required={true}
					min={0.01}
					precision={2}
					placeholder="Enter starting price"
					disabled={readOnly}
					helpText="The price at which you want to start trading"
					inputClass={isNewStock
						? 'border-green-300 focus:border-green-500 focus:ring-green-500'
						: ''}
				/>
			</div>

			<!-- Stop Loss Percentage -->
			<div>
				<NumericInput
					id="stopLossPercentage"
					name="stopLossPercentage"
					label="Stop Loss Percentage"
					bind:value={formData.stopLossPercentage}
					on:change={(e) => handleInputChange('stopLossPercentage', e)}
					error={formErrors.stopLossPercentage || ''}
					required={true}
					min={0.1}
					max={5}
					precision={2}
					placeholder="Enter stop loss percentage"
					disabled={readOnly}
					helpText="Recommended: 0.5% - 0.8%"
				/>
			</div>

			<!-- Risk Amount -->
			<div>
				<NumericInput
					id="riskAmount"
					name="riskAmount"
					label="Risk Amount (₹)"
					bind:value={formData.riskAmount}
					on:change={(e) => handleInputChange('riskAmount', e)}
					error={formErrors.riskAmount || ''}
					required={true}
					min={1}
					precision={2}
					placeholder="Enter risk amount"
					disabled={readOnly}
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
						class={`block w-full px-3 py-2 border ${
							formErrors.tradeSide
								? 'border-red-300'
								: isNewStock
									? 'border-green-300'
									: 'border-gray-300'
						} rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm`}
						disabled={readOnly}
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
						class="block w-full px-3 py-2 border {formErrors.psType
							? 'border-red-300'
							: 'border-gray-300'} rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
						disabled={readOnly}
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
						class="block w-full px-3 py-2 border {formErrors.entryType
							? 'border-red-300'
							: 'border-gray-300'} rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
						disabled={readOnly}
					>
						<option value="1ST_ENTRY">1ST_ENTRY</option>
						<option value="2_30_ENTRY">2_30_ENTRY</option>
						<option value="BB_RANGE">BB_RANGE</option>
					</select>
				</div>
				{#if formErrors.entryType}
					<p class="mt-1 text-sm text-red-600">{formErrors.entryType}</p>
				{/if}
			</div>

			<!-- Submit Button -->
			{#if !readOnly}
				<div class="flex justify-end pt-4">
					<button
						type="submit"
						class="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white {isNewStock
							? 'bg-green-600 hover:bg-green-700 focus:ring-green-500'
							: 'bg-blue-600 hover:bg-blue-700 focus:ring-blue-500'} focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50"
						disabled={isSaving}
					>
						{#if isSaving}
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
							Saving...
						{:else}
							{isNewStock ? 'Save New Stock Parameters' : 'Save Parameters'}
						{/if}
					</button>
				</div>
			{/if}
		</form>
	{/if}
</div>
