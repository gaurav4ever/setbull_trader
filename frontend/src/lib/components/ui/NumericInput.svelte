<script>
	import { createEventDispatcher } from 'svelte';

	// Props
	export let id = '';
	export let name = '';
	export let label = '';
	export let value = '';
	export let placeholder = '';
	export let required = false;
	export let min = null;
	export let max = null;
	export let step = 'any';
	export let disabled = false;
	export let readonly = false;
	export let error = '';
	export let helpText = '';
	export let inputClass = '';
	export let labelClass = '';
	export let errorClass = '';
	export let precision = 2; // Number of decimal places to show

	// State
	let inputElement;
	let isFocused = false;
	let internalValue = formatValueForDisplay(value);

	// Create event dispatcher
	const dispatch = createEventDispatcher();

	// Format the value for display based on precision
	function formatValueForDisplay(val) {
		if (val === '' || val === null || val === undefined || isNaN(val)) {
			return '';
		}
		return Number(val).toFixed(precision);
	}

	// Parse input value to number
	function parseInputValue(val) {
		if (val === '' || val === null || val === undefined) {
			return '';
		}
		const parsedValue = parseFloat(val);
		return isNaN(parsedValue) ? '' : parsedValue;
	}

	// Update internal value when prop changes
	$: {
		if (value !== parseInputValue(internalValue)) {
			internalValue = formatValueForDisplay(value);
		}
	}

	// Handle input change
	function handleInput(event) {
		// Allow empty string or valid numbers
		const input = event.target.value;
		const numericRegex = /^-?\d*\.?\d*$/;

		if (input === '' || numericRegex.test(input)) {
			internalValue = input;
			const parsedValue = parseInputValue(input);

			// Check min/max constraints
			if (parsedValue !== '' && min !== null && parsedValue < min) {
				internalValue = formatValueForDisplay(min);
				dispatch('change', min);
				dispatch('input', min);
			} else if (parsedValue !== '' && max !== null && parsedValue > max) {
				internalValue = formatValueForDisplay(max);
				dispatch('change', max);
				dispatch('input', max);
			} else {
				dispatch('input', parsedValue);
			}
		}
	}

	// Handle focus
	function handleFocus() {
		isFocused = true;
		dispatch('focus');
	}

	// Handle blur - format the value
	function handleBlur() {
		isFocused = false;

		// Parse and format the value
		const parsedValue = parseInputValue(internalValue);

		// If it's a valid number, format it
		if (parsedValue !== '') {
			internalValue = formatValueForDisplay(parsedValue);
		}

		dispatch('blur');
		dispatch('change', parsedValue);
	}

	// Handle keyboard input
	function handleKeydown(event) {
		// Allow: backspace, delete, tab, escape, enter
		if (
			[46, 8, 9, 27, 13].indexOf(event.keyCode) !== -1 ||
			// Allow: Ctrl+A, Command+A
			(event.keyCode === 65 && (event.ctrlKey === true || event.metaKey === true)) ||
			// Allow: home, end, left, right, down, up
			(event.keyCode >= 35 && event.keyCode <= 40) ||
			// Allow: - (minus) at the beginning for negative numbers
			(event.keyCode === 189 && internalValue === '')
		) {
			return;
		}

		// Ensure that it is a number or decimal point and stop the keypress if not
		if (
			(event.shiftKey || event.keyCode < 48 || event.keyCode > 57) &&
			(event.keyCode < 96 || event.keyCode > 105) &&
			event.keyCode !== 190 &&
			event.keyCode !== 110
		) {
			event.preventDefault();
		}

		// Prevent multiple decimal points
		if ((event.key === '.' || event.key === ',') && internalValue.includes('.')) {
			event.preventDefault();
		}
	}
</script>

<div class="numeric-input">
	{#if label}
		<label for={id} class={`block text-sm font-medium text-gray-700 mb-1 ${labelClass}`}>
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
			bind:value={internalValue}
			{placeholder}
			{required}
			{disabled}
			{readonly}
			{step}
			min={min !== null ? min : undefined}
			max={max !== null ? max : undefined}
			on:input={handleInput}
			on:focus={handleFocus}
			on:blur={handleBlur}
			on:keydown={handleKeydown}
			class={`block w-full px-3 py-2 border ${error ? 'border-red-300' : 'border-gray-300'} rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm ${inputClass}`}
			aria-invalid={!!error}
			aria-describedby={error ? `${id}-error` : undefined}
		/>
	</div>

	{#if error}
		<p id="{id}-error" class={`mt-1 text-sm text-red-600 ${errorClass}`}>
			{error}
		</p>
	{:else if helpText}
		<p class="mt-1 text-sm text-gray-500">{helpText}</p>
	{/if}
</div>
