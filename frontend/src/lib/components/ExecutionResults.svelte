<script>
	import { onMount } from 'svelte';
	import { formatCurrency } from '../utils/formatting';

	// Props
	export let results = []; // Array of execution results
	export let visible = true; // Whether the results panel is visible

	// Helper function to get status color classes
	function getStatusClasses(status) {
		switch (status) {
			case 'COMPLETED':
				return 'bg-green-100 text-green-800';
			case 'FAILED':
				return 'bg-red-100 text-red-800';
			case 'EXECUTING':
				return 'bg-blue-100 text-blue-800';
			case 'PENDING':
				return 'bg-yellow-100 text-yellow-800';
			case 'CANCELLED':
				return 'bg-gray-100 text-gray-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}
</script>

{#if visible && results && results.length > 0}
	<div class="execution-results p-4 bg-white rounded-lg shadow-sm border border-gray-200">
		<div class="mb-4">
			<h2 class="text-lg font-medium text-gray-900">Execution Results</h2>
			<p class="text-sm text-gray-500">Results of your trade executions</p>
		</div>

		<div class="space-y-4">
			{#each results as result}
				<div class="border border-gray-200 rounded-md overflow-hidden">
					<!-- Result Header -->
					<div
						class="bg-gray-50 px-4 py-2 border-b border-gray-200 flex justify-between items-center"
					>
						<div>
							<span class="font-medium text-gray-900"
								>{result.stock?.symbol || 'Unknown Stock'}</span
							>
							{#if result.executionPlanId}
								<span class="ml-2 text-xs text-gray-500"
									>Plan: {result.executionPlanId.slice(0, 8)}...</span
								>
							{/if}
						</div>

						<!-- Status Badge -->
						<span
							class="px-2 py-1 text-xs font-medium rounded-full {getStatusClasses(result.status)}"
						>
							{result.status}
						</span>
					</div>

					<!-- Result Details -->
					<div class="p-3">
						{#if result.status === 'FAILED' && result.errorMessage}
							<div class="text-sm text-red-600 mb-2">
								<p>Error: {result.errorMessage}</p>
							</div>
						{/if}

						{#if result.executedAt}
							<div class="text-sm text-gray-500 mb-2">
								<span>Executed at: {new Date(result.executedAt).toLocaleString()}</span>
							</div>
						{/if}

						<!-- Order Details (if available) -->
						{#if result.orders && result.orders.length > 0}
							<div class="mt-2">
								<h4 class="text-sm font-medium text-gray-700 mb-1">Orders Placed:</h4>
								<div class="overflow-x-auto">
									<table class="min-w-full divide-y divide-gray-200 text-sm">
										<thead class="bg-gray-50">
											<tr>
												<th
													scope="col"
													class="px-2 py-1 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
												>
													Level
												</th>
												<th
													scope="col"
													class="px-2 py-1 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
												>
													Price
												</th>
												<th
													scope="col"
													class="px-2 py-1 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
												>
													Qty
												</th>
												<th
													scope="col"
													class="px-2 py-1 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
												>
													Status
												</th>
												<th
													scope="col"
													class="px-2 py-1 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
												>
													Error
												</th>
											</tr>
										</thead>
										<tbody class="bg-white divide-y divide-gray-200">
											{#each result.orders as order}
												<tr>
													<td class="px-2 py-1 whitespace-nowrap text-xs">
														{order.description || 'Order'}
													</td>
													<td class="px-2 py-1 whitespace-nowrap text-xs">
														{formatCurrency(order.price)}
													</td>
													<td class="px-2 py-1 whitespace-nowrap text-xs">
														{order.quantity}
													</td>
													<td class="px-2 py-1 whitespace-nowrap text-xs">
														<span
															class="px-1.5 py-0.5 text-xs rounded-full {getStatusClasses(
																order.status
															)}"
														>
															{order.status}
														</span>
													</td>
													<td class="px-2 py-1 text-xs text-red-600">
														{#if order.error}
															{order.error.includes('Market is Closed')
																? 'Market is Closed! Want to place an offline order?'
																: order.error}
														{/if}
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							</div>
						{:else if result.status === 'COMPLETED'}
							<p class="text-sm text-gray-700">All orders executed successfully.</p>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	</div>
{:else if visible}
	<div class="execution-results p-4 bg-white rounded-lg shadow-sm border border-gray-200">
		<div class="text-center py-6">
			<svg
				class="mx-auto h-12 w-12 text-gray-400"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
				/>
			</svg>
			<h3 class="mt-2 text-sm font-medium text-gray-900">No execution results</h3>
			<p class="mt-1 text-sm text-gray-500">Execute trades to see results here.</p>
		</div>
	</div>
{/if}
