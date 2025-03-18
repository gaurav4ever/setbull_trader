<!-- frontend/src/lib/components/Autocomplete.svelte -->
<script>
	import { createEventDispatcher, onMount } from 'svelte';
	import { clickOutside } from '../actions/clickOutside';
	import { formatStockForDisplay } from '../utils/stockFormatting';

	// Props
	export let items = []; // Array of items to search through
	export let placeholder = 'Search...';
	export let value = ''; // Current selected value
	export let minChars = 1; // Minimum characters to start showing suggestions
	export let maxItems = 10; // Maximum number of items to display at once
	export let inputId = 'autocomplete'; // ID for the input field
	export let inputClass = ''; // Additional class for input styling
	export let name = ''; // Name attribute for the input
	export let displayFormat = null; // Function to format display (optional)

	// State
	let inputElement;
	let isOpen = false;
	let filteredItems = [];
	let highlightedIndex = -1;
	let searchTerm = value;
	let touchedByUser = false;

	const dispatch = createEventDispatcher();

	// Format item for display
	const formatItem = (item) => {
		if (displayFormat && typeof displayFormat === 'function') {
			return displayFormat(item);
		}

		// If it's a stock object, format it specially
		if (typeof item === 'object' && item.symbol && item.securityId) {
			return formatStockForDisplay(item);
		}

		// Otherwise, return as is
		return item;
	};

	// Filter items based on search term
	$: {
		if (searchTerm && searchTerm.length >= minChars) {
			// If items are objects, search in their fields
			if (items.length > 0 && typeof items[0] === 'object') {
				filteredItems = items
					.filter((item) => {
						const symbol = item.symbol || '';
						const name = item.name || item.symbol || '';

						return (
							symbol.toLowerCase().includes(searchTerm.toLowerCase()) ||
							name.toLowerCase().includes(searchTerm.toLowerCase())
						);
					})
					.slice(0, maxItems);
			} else {
				// For string items
				filteredItems = items
					.filter((item) => {
						if (typeof item === 'string') {
							return item.toLowerCase().includes(searchTerm.toLowerCase());
						}
						return false;
					})
					.slice(0, maxItems);
			}

			if (filteredItems.length > 0 && touchedByUser) {
				isOpen = true;
			} else {
				isOpen = false;
			}
		} else {
			filteredItems = [];
			isOpen = false;
		}
	}

	// When value changes from the parent component
	$: if (value !== searchTerm && !touchedByUser) {
		searchTerm = value;
	}

	function handleInput() {
		touchedByUser = true;
		highlightedIndex = -1;
		dispatch('input', searchTerm);
	}

	function selectItem(item) {
		// If the item is an object, we want to extract the value
		const displayValue = formatItem(item);
		const actualValue = typeof item === 'object' ? item : displayValue;

		searchTerm = displayValue;

		isOpen = false;
		highlightedIndex = -1;

		// Dispatch the whole item for objects, otherwise just the string
		dispatch('select', actualValue);
		dispatch('change', actualValue);
		inputElement.blur();
	}

	function handleKeydown(event) {
		switch (event.key) {
			case 'ArrowDown':
				event.preventDefault();
				if (isOpen) {
					highlightedIndex = (highlightedIndex + 1) % filteredItems.length;
				} else if (searchTerm.length >= minChars) {
					isOpen = true;
				}
				break;
			case 'ArrowUp':
				event.preventDefault();
				if (isOpen) {
					highlightedIndex =
						highlightedIndex <= 0 ? filteredItems.length - 1 : highlightedIndex - 1;
				}
				break;
			case 'Enter':
				event.preventDefault();
				if (isOpen && highlightedIndex !== -1) {
					selectItem(filteredItems[highlightedIndex]);
				} else if (filteredItems.length > 0) {
					selectItem(filteredItems[0]);
				}
				break;
			case 'Escape':
				event.preventDefault();
				isOpen = false;
				highlightedIndex = -1;
				break;
			case 'Tab':
				if (isOpen && highlightedIndex !== -1) {
					selectItem(filteredItems[highlightedIndex]);
				} else if (isOpen && filteredItems.length > 0) {
					selectItem(filteredItems[0]);
				}
				isOpen = false;
				break;
		}
	}

	function handleFocus() {
		if (searchTerm.length >= minChars && filteredItems.length > 0) {
			isOpen = true;
		}
	}

	function handleBlur() {
		// Give time for item selection click to register
		setTimeout(() => {
			isOpen = false;
		}, 150);
	}

	function handleClickOutside() {
		isOpen = false;
		highlightedIndex = -1;

		// Reset the search term to display value if we had a selection
		if (value) {
			if (typeof value === 'object') {
				searchTerm = formatItem(value);
			} else if (value !== searchTerm) {
				searchTerm = value;
			}
		}
	}
</script>

<div class="relative w-full" use:clickOutside on:click_outside={handleClickOutside}>
	<input
		bind:this={inputElement}
		id={inputId}
		{name}
		type="text"
		{placeholder}
		bind:value={searchTerm}
		on:input={handleInput}
		on:keydown={handleKeydown}
		on:focus={handleFocus}
		on:blur={handleBlur}
		class="w-full {inputClass}"
		autocomplete="off"
	/>

	{#if isOpen && filteredItems.length > 0}
		<div
			class="absolute z-50 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base overflow-auto focus:outline-none sm:text-sm"
		>
			{#each filteredItems as item, index}
				<div
					on:mousedown={() => selectItem(item)}
					class="cursor-pointer select-none relative py-2 pl-3 pr-9 {index === highlightedIndex
						? 'bg-blue-100 text-blue-900'
						: 'text-gray-900 hover:bg-gray-100'}"
				>
					{formatItem(item)}
				</div>
			{/each}
		</div>
	{/if}
</div>
