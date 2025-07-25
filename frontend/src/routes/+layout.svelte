<svelte:head>
	<script src="https://cdn.tailwindcss.com"></script>
</svelte:head>

<div class="min-h-screen bg-gray-50 flex flex-col">
	<header class="bg-white shadow">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex justify-between h-16">
				<div class="flex items-center">
					<a href="/" class="flex-shrink-0 flex items-center">
						<span class="text-2xl font-bold text-blue-600">SetBull Trader</span>
					</a>

					<nav class="ml-10 hidden md:flex space-x-8">
						<a
							href="/"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Dashboard</a
						>
						<a
							href="/groups"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Stock Groups</a
						>
						<a
							href="/order"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Place Order</a
						>
						<a
							href="/order/modify"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Modify Order</a
						>
						<a
							href="/order/cancel"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Cancel Order</a
						>
						<a
							href="/trades"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Trades</a
						>
						<a
							href="/trades/history"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Trade History</a
						>
						<a
							href="/bbw-dashboard"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>BBW Dashboard</a
						>
						<a
							href="/stocks"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Manage Stocks</a
						>
						<a
							href="/history"
							class="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
							>Execution History</a
						>
					</nav>
				</div>

				<!-- Mobile menu button -->
				<div class="flex items-center md:hidden">
					<button
						type="button"
						class="inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-gray-500 hover:bg-gray-100"
					>
						<span class="sr-only">Open main menu</span>
						<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M4 6h16M4 12h16M4 18h16"
							/>
						</svg>
					</button>
				</div>
			</div>
		</div>
	</header>

	<main class="flex-grow py-8">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
			<slot />
		</div>
	</main>

	<footer class="bg-white border-t border-gray-200 py-6">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="text-center md:text-left text-gray-500 text-sm">
				&copy; {new Date().getFullYear()} SetBull Trader. All rights reserved.
			</div>
		</div>
	</footer>
</div>
