<script lang="ts">
	import { onMount } from 'svelte';
	import { stockGroupsStore } from '../stores/stockGroupsStore.js';
	import StockGroupForm from './StockGroupForm.svelte';
	import { formatStockForStockGroupDisplay } from '../utils/stockFormatting.js';

	let showCreateModal = false;
	let groups: any[] = [];
	let loading = true;
	let error = '';

	const unsubscribe = stockGroupsStore.subscribe((state) => {
		groups = state.groups;
		loading = state.loading;
		error = state.error;
	});

	onMount(() => {
		stockGroupsStore.loadGroups();
		return () => unsubscribe();
	});

	function openCreateModal() {
		showCreateModal = true;
	}
	function closeCreateModal() {
		showCreateModal = false;
	}
	function handleGroupCreated() {
		showCreateModal = false;
		stockGroupsStore.loadGroups();
	}
</script>

<div class="bg-white rounded-lg shadow p-4 mb-6">
	<div class="flex justify-between items-center mb-2">
		<h2 class="text-lg font-semibold">Stock Groups</h2>
		<div class="flex gap-2">
			<button
				class="bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700"
				on:click={openCreateModal}>Create Group</button
			>
			<a href="/groups" class="text-blue-600 hover:underline px-2 py-1">View All</a>
		</div>
	</div>
	{#if loading}
		<div class="text-gray-500 py-4">Loading groups...</div>
	{:else if error}
		<div class="text-red-600 py-4">{error}</div>
	{:else if groups && groups.length > 0}
		<table class="min-w-full text-sm">
			<thead>
				<tr class="text-left text-gray-600 border-b">
					<th class="py-2 pr-4">Entry Type</th>
					<th class="py-2 pr-4">Stocks</th>
					<th class="py-2 pr-4">Status</th>
				</tr>
			</thead>
			<tbody>
				{#each groups.slice(0, 3) as group}
					<tr class="border-b last:border-0 hover:bg-gray-50 transition">
						<td class="py-2 pr-4 font-medium">{group.entryType}</td>
						<td class="py-2 pr-4">
							{#if group.stocks && group.stocks.length > 0}
								<ul class="flex flex-wrap gap-1">
									{#each group.stocks as stock}
										<li class="bg-blue-100 text-blue-800 rounded-full px-2 py-0.5 text-xs">
											{typeof stock === 'string' ? stock : formatStockForStockGroupDisplay(stock)}
										</li>
									{/each}
								</ul>
							{:else}
								<span class="text-gray-400">-</span>
							{/if}
						</td>
						<td class="py-2 pr-4">
							<span
								class="inline-block px-2 py-0.5 rounded text-xs font-semibold
									{group.status === 'executing'
									? 'bg-yellow-100 text-yellow-800'
									: group.status === 'completed'
										? 'bg-green-100 text-green-800'
										: group.status === 'failed'
											? 'bg-red-100 text-red-800'
											: 'bg-gray-100 text-gray-800'}"
							>
								{group.status}
							</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else}
		<div class="text-gray-500 py-4">No stock groups found.</div>
	{/if}

	<StockGroupForm
		show={showCreateModal}
		on:close={closeCreateModal}
		on:created={handleGroupCreated}
	/>
</div>
