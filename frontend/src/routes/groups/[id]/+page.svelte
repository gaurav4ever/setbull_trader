<script lang="ts">
	import { onMount } from 'svelte';
	import { getGroup, executeGroup } from '$lib/services/stockGroupService.js';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	let group: any = null;
	let loading: boolean = true;
	let error: string = '';
	let executing: boolean = false;
	let executionError: string = '';

	// Get group ID from route params
	const groupId: string = $page.params.id;

	async function fetchGroup() {
		loading = true;
		error = '';
		try {
			group = await getGroup(groupId);
		} catch (e: any) {
			error = e.message || 'Failed to load group';
		} finally {
			loading = false;
		}
	}

	onMount(fetchGroup);

	async function handleExecute() {
		executing = true;
		executionError = '';
		try {
			await executeGroup(groupId);
			await fetchGroup();
		} catch (e: any) {
			executionError = e.message || 'Failed to execute group';
		} finally {
			executing = false;
		}
	}

	function goBack() {
		goto('/groups');
	}
</script>

<div class="max-w-2xl mx-auto mt-8">
	<div class="bg-white rounded-lg shadow p-6">
		{#if loading}
			<p class="text-gray-500 py-4">Loading group details...</p>
		{:else if error}
			<p class="text-red-600 py-4">{error}</p>
		{:else if group}
			<button
				on:click={goBack}
				class="bg-gray-200 text-gray-700 px-3 py-1 rounded hover:bg-gray-300 mb-4"
				>&larr; Back to Groups</button
			>
			<h2 class="text-xl font-bold text-gray-900 mb-2">Group: {group.entryType}</h2>
			<p class="mb-2">
				<b>Status:</b>
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
			</p>
			<p class="mb-4">
				<b>Stocks:</b>
				{#if group.stocks && group.stocks.length > 0}
					<ul class="flex flex-wrap gap-1 mt-1">
						{#each group.stocks as s}
							<li class="bg-blue-100 text-blue-800 rounded-full px-2 py-0.5 text-xs">{s.symbol}</li>
						{/each}
					</ul>
				{:else}
					<span class="text-gray-400">-</span>
				{/if}
			</p>
			<button
				on:click={handleExecute}
				class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 mb-4"
				disabled={executing || group.status === 'EXECUTING'}
			>
				{executing ? 'Executing...' : 'Execute Group'}
			</button>
			{#if executionError}
				<p class="text-red-600 text-sm mt-2">{executionError}</p>
			{/if}
			{#if group.status === 'COMPLETED' || group.status === 'FAILED'}
				<h3 class="text-lg font-semibold mt-6 mb-2">Execution Results</h3>
				<!-- Placeholder: Extend/replace with ExecutionResults.svelte if available -->
				<ul class="list-disc list-inside text-sm text-gray-700">
					{#each group.stocks as s}
						<li>{s.symbol} - <i>Result: (see backend for per-stock execution details)</i></li>
					{/each}
				</ul>
			{/if}
		{/if}
	</div>
</div>

<style>
	body {
		background: #f7fafc;
	}
</style>
