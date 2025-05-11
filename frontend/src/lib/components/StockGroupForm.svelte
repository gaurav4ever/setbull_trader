<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { createGroup, editGroup } from '$lib/services/stockGroupService.js';
	import { createStock } from '$lib/services/stocksService.js';
	import EnhancedStockSelector from './EnhancedStockSelector.svelte';
	import { formatStockForStockGroupDisplay } from '../utils/stockFormatting.js';

	const ENTRY_TYPES = ['1ST_ENTRY', '2_30_ENTRY'];

	export let show: boolean = false;
	export let mode: 'create' | 'edit' = 'create';
	export let group: any = null; // for edit mode

	// Store selected stocks as objects, not just IDs
	let selectedStocks: any[] = [];
	let entryType: string = '';
	let error: string = '';
	let loading: boolean = false;

	const dispatch = createEventDispatcher();

	onMount(() => {
		if (mode === 'edit' && group) {
			entryType = group.entryType;
			selectedStocks = group.stocks ? [...group.stocks] : [];
		}
	});

	function close() {
		dispatch('close');
	}

	async function handleStockSelected(event: CustomEvent<any>) {
		// add log
		console.log('handleStockSelected', event.detail);
		const stock = event.detail;
		if (!stock) return;
		if (selectedStocks.find((s) => s.symbol === stock.symbol)) return;
		if (selectedStocks.length >= 5) return;

		try {
			const stockData = {
				symbol: stock.symbol,
				name: stock.name,
				securityId: stock.securityId,
				isSelected: true
			};
			console.log('selectedStocks pre', selectedStocks);
			const response = await createStock(stockData);
			if (response.success) {
				stock.id = response.data.id;
				selectedStocks = [...selectedStocks, stock];
			}
			console.log('selectedStocks post', selectedStocks);
		} catch (e: any) {
			console.error('Failed to create stock', e);
			error = e.message || 'Failed to create stock';
		}
	}

	function removeStock(symbol: string) {
		selectedStocks = selectedStocks.filter((s) => s.symbol !== symbol);
	}

	async function handleSubmit() {
		error = '';
		if (!entryType) {
			error = 'Entry type is required.';
			return;
		}
		if (!selectedStocks || selectedStocks.length === 0) {
			error = 'Select at least one stock.';
			return;
		}
		if (selectedStocks.length > 5) {
			error = 'You can select up to 5 stocks only.';
			return;
		}
		loading = true;
		try {
			if (mode === 'edit' && group) {
				await editGroup(
					group.id,
					selectedStocks.map((s) => s.stockId || s.id)
				);
				dispatch('edited', { ...group, entryType, stocks: selectedStocks });
				close();
			} else {
				await createGroup(
					entryType,
					selectedStocks.map((s) => s.id)
				);
				dispatch('created');
				close();
			}
		} catch (e: any) {
			error = e.message || (mode === 'edit' ? 'Failed to edit group' : 'Failed to create group');
		} finally {
			loading = false;
			if (mode !== 'edit') selectedStocks = [];
		}
	}
</script>

{#if show}
	<div class="modal-backdrop" on:click={close}></div>
	<div class="modal" on:click|stopPropagation>
		<h2 class="text-xl font-semibold mb-4">
			{mode === 'edit' ? 'Edit Stock Group' : 'Create Stock Group'}
		</h2>
		<form on:submit|preventDefault={handleSubmit} class="space-y-4">
			<div>
				<label class="block mb-1 font-medium">Entry Type:</label>
				<select bind:value={entryType} required class="w-full border rounded px-2 py-1">
					<option value="" disabled selected>Select entry type</option>
					{#each ENTRY_TYPES as type}
						<option value={type}>{type}</option>
					{/each}
				</select>
			</div>
			<div>
				<label class="block mb-1 font-medium">Stocks:</label>
				<EnhancedStockSelector maxSelectedStocks={5} on:stockSelected={handleStockSelected} />
				{#if selectedStocks.length > 0}
					<ul class="flex flex-wrap gap-2 mt-2">
						{#each selectedStocks as stock}
							<li
								class="flex items-center bg-blue-100 text-blue-800 rounded-full px-3 py-1 text-sm"
							>
								{formatStockForStockGroupDisplay(stock)}
								<button
									type="button"
									class="ml-2 text-blue-600 hover:text-red-600"
									on:click={() => removeStock(stock.symbol)}
									disabled={loading}>&times;</button
								>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
			{#if error}
				<p class="text-red-600 text-sm mt-2">{error}</p>
			{/if}
			<div class="flex gap-2 mt-4 justify-end">
				<button
					type="submit"
					class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
					disabled={loading}
					>{loading
						? mode === 'edit'
							? 'Saving...'
							: 'Creating...'
						: mode === 'edit'
							? 'Save'
							: 'Create'}</button
				>
				<button
					type="button"
					class="bg-gray-200 text-gray-700 px-4 py-2 rounded hover:bg-gray-300"
					on:click={close}
					disabled={loading}>Cancel</button
				>
			</div>
		</form>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.3);
		z-index: 10;
	}
	.modal {
		position: fixed;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		background: #fff;
		padding: 2rem;
		border-radius: 12px;
		z-index: 11;
		min-width: 340px;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.18);
	}
</style>
