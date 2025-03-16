<!-- frontend/src/lib/components/Modal.svelte -->
<script>
	import { onMount, createEventDispatcher } from 'svelte';
	import { fade } from 'svelte/transition';
	import { clickOutside } from '../actions/clickOutside';

	// Props
	export let show = false;
	export let title = '';
	export let closeOnClickOutside = true;
	export let maxWidth = 'max-w-md';

	// Event dispatcher
	const dispatch = createEventDispatcher();

	// Handle close button click
	function handleClose() {
		dispatch('close');
	}

	// Handle click outside
	function handleClickOutside() {
		if (closeOnClickOutside) {
			dispatch('close');
		}
	}

	// Handle escape key press
	function handleKeydown(event) {
		if (event.key === 'Escape') {
			dispatch('close');
		}
	}

	// Set up event listener for escape key
	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		return () => {
			window.removeEventListener('keydown', handleKeydown);
		};
	});
</script>

<svelte:window on:keydown={handleKeydown} />

{#if show}
	<div
		class="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center z-50"
		transition:fade={{ duration: 200 }}
	>
		<div
			class="modal-container {maxWidth} w-full bg-white rounded-lg shadow-xl overflow-hidden"
			use:clickOutside
			on:click_outside={handleClickOutside}
		>
			<div
				class="modal-header px-4 py-3 bg-gray-50 border-b border-gray-200 flex justify-between items-center"
			>
				<h3 class="text-lg font-medium text-gray-900">{title}</h3>
				<button
					type="button"
					class="text-gray-400 hover:text-gray-500 focus:outline-none"
					on:click={handleClose}
				>
					<span class="sr-only">Close</span>
					<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</div>
			<div class="modal-content">
				<slot></slot>
			</div>
		</div>
	</div>
{/if}
