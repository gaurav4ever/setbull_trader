<script>
	import { formatCurrency, formatNumber } from '../utils/formatting';

	// Props
	export let levels = []; // Array of execution levels
	export let totalQuantity = 0; // Total quantity
	export let tradeSide = 'BUY'; // Trade side (BUY or SELL)
	export let compact = false; // Whether to show in compact mode

	// Determine the CSS classes for the trade side
	$: tradeSideClasses =
		tradeSide === 'BUY'
			? 'text-green-700 bg-green-50 border-green-200'
			: 'text-red-700 bg-red-50 border-red-200';
</script>

<div class="execution-levels {compact ? 'text-sm' : ''}">
	<!-- Total Quantity Summary -->
	<div class="mb-2 flex justify-between items-center">
		<span class="text-gray-700 font-medium">Total Quantity:</span>
		<span class="text-gray-900">{totalQuantity}</span>
	</div>

	<!-- Trade Side Indicator -->
	<div class="mb-3">
		<span class="px-2 py-1 rounded-full text-xs font-medium {tradeSideClasses}">
			{tradeSide}
		</span>
	</div>

	<!-- Execution Levels Table -->
	<table class="min-w-full divide-y divide-gray-200">
		<thead class="bg-gray-50">
			<tr>
				<th
					scope="col"
					class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
				>
					Level
				</th>
				<th
					scope="col"
					class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
				>
					Price
				</th>
				<th
					scope="col"
					class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
				>
					Qty
				</th>
			</tr>
		</thead>
		<tbody class="bg-white divide-y divide-gray-200">
			{#each levels as level, i}
				<tr class={i === 0 ? 'bg-red-50' : ''}>
					<td class="px-3 py-2 whitespace-nowrap text-sm text-gray-900">
						{level.description}
					</td>
					<td class="px-3 py-2 whitespace-nowrap text-sm text-gray-900">
						{formatNumber(level.price, 2)}
					</td>
					<td class="px-3 py-2 whitespace-nowrap text-sm text-gray-900">
						{level.quantity}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>

	<!-- Total Investment Required (if not in compact mode) -->
	{#if !compact && totalQuantity > 0}
		<div class="mt-3 pt-3 border-t border-gray-200">
			<div class="flex justify-between items-center">
				<span class="text-gray-700 font-medium">Total Investment:</span>
				<span class="text-gray-900">
					{formatCurrency(levels.reduce((sum, level) => sum + level.price * level.quantity, 0))}
				</span>
			</div>
		</div>
	{/if}
</div>

<style>
	.execution-levels table {
		border-collapse: collapse;
		width: 100%;
	}
</style>
