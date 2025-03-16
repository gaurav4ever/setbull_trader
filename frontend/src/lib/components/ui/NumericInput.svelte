<script>
	import { createEventDispatcher, onMount } from 'svelte';

	// Props
	export let id = '';
	export let name = '';
	export let label = '';
	export let value = '';
	export let placeholder = '';
	export let required = false;
	export let min = null;
	export let max = null;
	export let precision = 2;
	export let error = '';
	export let helpText = '';
	export let inputClass = '';

	// Event dispatcher
	const dispatch = createEventDispatcher();

	// Local state
	let inputElement;
	let currentValue = value !== '' ? value : '';

	onMount(() => {
		// Set initial value
		if (inputElement && value !== '') {
			inputElement.value = value;
		}
	});

	// Handle input changes
	function handleInput(event) {
		const input = event.target.value;

		// Allow only valid numeric inputs
		if (input === '' || /^-?\d*\.?\d*$/.test(input)) {
			currentValue = input;

			// Convert to number if possible
			const numValue = input === '' ? '' : parseFloat(input);

			// Don't dispatch NaN values
			dispatch('input', isNaN(numValue) ? '' : numValue);
		}
	}

	// Handle blur events - format the number
	function handleBlur() {
		if (currentValue !== '' && !isNaN(parseFloat(currentValue))) {
			const numValue = parseFloat(currentValue);

			// Apply min/max constraints
			if (min !== null && numValue < min) {
				currentValue = min.toString();
			} else if (max !== null && numValue > max) {
				currentValue = max.toString();
			}

			// Format to fixed decimal places
			if (currentValue !== '') {
				const formatted = parseFloat(currentValue).toFixed(precision);
				currentValue = formatted;
				if (inputElement) {
					inputElement.value = formatted;
				}
			}

			dispatch('change', parseFloat(currentValue));
		} else {
			dispatch('change', '');
		}
	}
</script>

<div>
	{#if label}
		<label for={id} class="block text-sm font-medium text-gray-700 mb-1">
			{label}
			{#if required}
				<span class="text-red-500">*</span>
			{/if}
		</label>
	{/if}

	<div class="relative rounded-md shadow-sm">
		<input
			bind:this={inputElement}
			{id}
			{name}
			type="text"
			value={currentValue}
			{placeholder}
			{required}
			on:input={handleInput}
			on:blur={handleBlur}
			class={`block w-full px-3 py-2 border ${error ? 'border-red-300' : 'border-gray-300'} rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm ${inputClass}`}
			aria-invalid={!!error}
			aria-describedby={error ? `${id}-error` : undefined}
		/>
	</div>

	{#if error}
		<p id="{id}-error" class="mt-1 text-sm text-red-600">
			{error}
		</p>
	{:else if helpText}
		<p class="mt-1 text-sm text-gray-500">{helpText}</p>
	{/if}
</div>
