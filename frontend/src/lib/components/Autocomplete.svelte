<script>
	import { createEventDispatcher, onMount } from 'svelte';
	import { clickOutside } from '../actions/clickOutside';

	// Props
	export let items = []; // Array of items to search through
	export let placeholder = 'Search...';
	export let value = ''; // Current selected value
	export let minChars = 1; // Minimum characters to start showing suggestions
	export let maxItems = 10; // Maximum number of items to display at once
	export let inputId = 'autocomplete'; // ID for the input field
	export let inputClass = ''; // Additional class for input styling
	export let name = ''; // Name attribute for the input

	// State
	let inputElement;
	let isOpen = false;
	let filteredItems = [];
	let highlightedIndex = -1;
	let searchTerm = value;
	let touchedByUser = false;

	const dispatch = createEventDispatcher();

	// Filter items based on search term
	$: {
		if (searchTerm && searchTerm.length >= minChars) {
			filteredItems = items
				.filter((item) => item.toLowerCase().includes(searchTerm.toLowerCase()))
				.slice(0, maxItems);

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
		searchTerm = item;
		value = item;
		isOpen = false;
		highlightedIndex = -1;
		dispatch('select', item);
		dispatch('change', item);
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

		// If the search term doesn't match any item and we're losing focus,
		// reset to the original value if there was one
		if (value && searchTerm !== value && !items.includes(searchTerm)) {
			searchTerm = value;
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
					{item}
				</div>
			{/each}
		</div>
	{/if}
</div>
