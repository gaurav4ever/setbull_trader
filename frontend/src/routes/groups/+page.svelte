<script>
	import { onMount } from 'svelte';
	import { stockGroupsStore } from '$lib/stores/stockGroupsStore.js';
	import StockGroupForm from '$lib/components/StockGroupForm.svelte';
	import { deleteGroup, getGroup, executeGroup } from '$lib/services/stockGroupService.js';
	import { goto } from '$app/navigation';

	let showCreateModal = false;
	let showEditModal = false;
	let editingGroup = null;
	let deletingGroupId = null;
	let error = '';
	let storeState = null;
	const unsubscribe = stockGroupsStore.subscribe((state) => {
		storeState = state;
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
	function handleCreated(newGroup) {
		showCreateModal = false;
		if (newGroup && newGroup.id) {
			// Add new group to local list
			storeState.groups = [newGroup, ...storeState.groups];
		}
	}
	async function editGroup(id) {
		try {
			const group = await getGroup(id);
			editingGroup = group;
			showEditModal = true;
		} catch (e) {
			error = e.message || 'Failed to load group for editing';
		}
	}
	function closeEditModal() {
		showEditModal = false;
		editingGroup = null;
	}
	function handleEdited(evt) {
		const updated = evt.detail;
		showEditModal = false;
		editingGroup = null;
		// Update group in local list
		storeState.groups = storeState.groups.map((g) => (g.id === updated.id ? updated : g));
	}
	async function deleteGroupHandler(id) {
		deletingGroupId = id;
		error = '';
		try {
			const response = await deleteGroup(id);
			if (response && response.success) {
				storeState.groups = storeState.groups.filter((g) => g.id !== id);
				deletingGroupId = null;
			} else {
				error = response ? response.error : 'Failed to delete group';
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete group';
			deletingGroupId = null;
		}
	}
	function goToGroup(id) {
		goto(`/groups/${id}`);
	}

	async function executeGroupHandler(id) {
		try {
			// call the executeGroup api

			const response = await executeGroup(id);
			console.log('executeGroup response', response);
			if (response && response.success) {
				storeState.groups = storeState.groups.map((g) => (g.id === id ? response.data : g));
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to execute group';
		}
	}
</script>

<!-- Group List Card -->
<div class="max-w-4xl mx-auto mt-8">
	<div class="bg-white rounded-lg shadow p-6">
		<div class="flex justify-between items-center mb-4">
			<h1 class="text-2xl font-bold text-gray-900">Stock Groups</h1>
			<button
				on:click={openCreateModal}
				class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition"
				disabled={storeState && storeState.loading}>Create New Group</button
			>
		</div>

		{#if error}
			<p class="text-red-600 py-2">{error}</p>
		{/if}
		{#if storeState && storeState.loading}
			<p class="text-gray-500 py-4">Loading groups...</p>
		{:else if storeState && storeState.error}
			<p class="text-red-600 py-4">{storeState.error}</p>
		{:else if storeState && storeState.groups && storeState.groups.length > 0}
			<table class="min-w-full text-sm">
				<thead>
					<tr class="text-left text-gray-600 border-b">
						<th class="py-2 pr-4">Entry Type</th>
						<th class="py-2 pr-4">Stocks</th>
						<th class="py-2 pr-4">Status</th>
						<th class="py-2 pr-4">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each storeState.groups as group (group.id)}
						<tr class="border-b last:border-0 hover:bg-gray-50 transition">
							<td class="py-2 pr-4 font-medium">{group.entryType}</td>
							<td class="py-2 pr-4">
								{#if group.stocks && group.stocks.length > 0}
									<ul class="flex flex-wrap gap-1">
										{#each group.stocks as s}
											<li class="bg-blue-100 text-blue-800 rounded-full px-2 py-0.5 text-xs">
												{s.symbol}
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
									{group.status === 'EXECUTING'
										? 'bg-yellow-100 text-yellow-800'
										: group.status === 'COMPLETED'
											? 'bg-green-100 text-green-800'
											: group.status === 'FAILED'
												? 'bg-red-100 text-red-800'
												: 'bg-gray-100 text-gray-800'}"
								>
									{group.status}
								</span>
							</td>
							<td class="py-2 pr-4">
								<button
									on:click={() => goToGroup(group.id)}
									class="bg-gray-200 text-gray-700 px-3 py-1 rounded hover:bg-gray-300 mr-2"
									>View</button
								>
								<button
									on:click={() => executeGroupHandler(group.id)}
									class="bg-green-200 text-green-700 px-3 py-1 rounded hover:bg-green-300 mr-2"
									>Execute</button
								>
								<button
									on:click={() => editGroup(group.id)}
									class="bg-blue-200 text-blue-700 px-3 py-1 rounded hover:bg-blue-300 mr-2"
								>
									Edit
								</button>
								<button
									on:click={() => deleteGroupHandler(group.id)}
									class="bg-red-200 text-red-700 px-3 py-1 rounded hover:bg-red-300 mr-2"
									disabled={deletingGroupId === group.id}
								>
									{deletingGroupId === group.id ? 'Deleting...' : 'Delete'}
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="text-gray-500 py-4">No stock groups found.</p>
		{/if}
	</div>
</div>

<StockGroupForm
	show={showCreateModal}
	mode="create"
	on:close={closeCreateModal}
	on:created={handleCreated}
/>
{#if showEditModal}
	<StockGroupForm
		show={showEditModal}
		mode="edit"
		group={editingGroup}
		on:close={closeEditModal}
		on:edited={handleEdited}
	/>
{/if}

<style>
	body {
		background: #f7fafc;
	}
</style>
