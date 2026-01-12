<script lang="ts">
	import { onMount } from 'svelte';
	import CreateHostForm from '$lib/components/home/CreateHostForm.svelte';
	import HostCard from '$lib/components/home/HostCard.svelte';
	import HostCardSkeleton from '$lib/components/home/HostCardSkeleton.svelte';
	import { authStore } from '$lib/stores/auth';
	import { hostsStore } from '$lib/stores/hosts';
	import { t } from '$lib/stores/locale';
	import { createDebouncedLoadingStore, isLoadingAddForm, showAddForm } from '$lib/stores/ui';
	import { goto } from '$lib/utils/navigation';

	let autoPingInterval: ReturnType<typeof setInterval> | undefined;
	let lastPingResults = $state(new Map<string, any>());
	let hasInitialized = $state(false);
	let hasFetchedInitially = $state(false);
	let isMounted = $state(false);
	let createFormRef: CreateHostForm | null = $state(null);

	const debouncedLoading = createDebouncedLoadingStore(350);
	let hasHostsError = $state(false);
	let isLoadingCreateForm = $state(false);

	$effect(() => {
		const unsubscribe = hostsStore.isLoading.subscribe((value) => {
			debouncedLoading.setLoading(value);
		});
		return unsubscribe;
	});

	$effect(() => {
		const unsubscribe = hostsStore.hasError.subscribe((value) => {
			hasHostsError = value;
		});
		return unsubscribe;
	});

	$effect(() => {
		const unsubscribe = isLoadingAddForm.subscribe((value) => {
			isLoadingCreateForm = value;
		});
		return unsubscribe;
	});

	onMount(async () => {
		isMounted = true;
	});

	$effect(() => {
		// Wait for component to mount and auth state to load
		if (!isMounted || $authStore.isLoading) {
			return;
		}

		// If server is unreachable, don't try to fetch hosts or show cached data
		if ($authStore.serverUnreachable) {
			if (autoPingInterval) {
				clearInterval(autoPingInterval);
				autoPingInterval = undefined;
				hasInitialized = false;
			}
			// Clear hosts to prevent showing stale cached data
			hostsStore.fetchHosts();
			return;
		}

		// Stop auto-ping if user becomes unauthenticated
		if ($authStore.useAuth && !$authStore.isAuthenticated) {
			if (autoPingInterval) {
				clearInterval(autoPingInterval);
				autoPingInterval = undefined;
				hasInitialized = false;
			}
		}

		// Only fetch once on initial load
		if (!hasFetchedInitially) {
			hasFetchedInitially = true;
			checkAndFetchHosts();
		}
	});

	async function checkAndFetchHosts() {
		// Don't redirect if still loading - wait for auth state
		if ($authStore.isLoading) {
			return;
		}

		// If server is unreachable, don't fetch
		if ($authStore.serverUnreachable) {
			return;
		}

		// If auth is required and user is not authenticated, redirect to auth
		if ($authStore.useAuth && !$authStore.isAuthenticated) {
			// Stop auto-ping immediately before redirecting
			if (autoPingInterval) {
				clearInterval(autoPingInterval);
				autoPingInterval = undefined;
				hasInitialized = false;
			}
			goto('/auth');
			return;
		}

		// Fetch hosts if authenticated or if no auth is required
		if ($authStore.isAuthenticated || !$authStore.useAuth) {
			hostsStore.fetchHosts();

			// Start automatic bulk ping every 15 seconds (only once)
			if (!hasInitialized) {
				startAutoBulkPing();
				hasInitialized = true;
			}
		}
	}

	async function performBackgroundBulkPing() {
		// Don't ping if server is unreachable
		if ($authStore.serverUnreachable) {
			console.log('[AutoPing] Skipping ping - server unreachable');
			return;
		}

		// Don't ping if not authenticated
		if ($authStore.useAuth && !$authStore.isAuthenticated) {
			console.log('[AutoPing] Skipping ping - not authenticated');
			return;
		}

		try {
			await hostsStore.bulkPing((result) => {
				// Update UI immediately for each host as response arrives
				lastPingResults = new Map(lastPingResults).set(result.host_id, result);
			});
		} catch (error) {
			console.error('Background bulk ping failed:', error);
		}
	}

	function startAutoBulkPing() {
		// Stop existing interval if any
		if (autoPingInterval) {
			clearInterval(autoPingInterval);
		}

		// Perform initial bulk ping
		performBackgroundBulkPing();

		// Start auto bulk ping every 15 seconds
		autoPingInterval = setInterval(() => {
			performBackgroundBulkPing();
		}, 15000);
	}

	// Cleanup on unmount
	$effect(() => {
		return () => {
			if (autoPingInterval) {
				clearInterval(autoPingInterval);
			}
		};
	});

	// Refresh network interfaces when form is shown
	$effect(() => {
		if ($showAddForm && createFormRef) {
			createFormRef.refreshInterfaces();
		}
	});
</script>

<main class="pb-20 pt-24">
	<div class="space-y-4 px-4">
		{#if $showAddForm && !$authStore.isReadOnly}
			<CreateHostForm
				bind:this={createFormRef}
				class="mx-auto max-w-[40em]"
				isLoading={isLoadingCreateForm}
				onSuccess={() => {
					showAddForm.set(false);
					isLoadingAddForm.set(false);
				}}
			/>
		{/if}

		<ul class="space-y-2">
			{#if hasHostsError && !$debouncedLoading}
				<div class="mx-auto max-w-[40em] rounded-lg border p-8 text-center">
					<div class="mb-4 rounded-full bg-destructive/10 p-3">
						<svg
							class="mx-auto h-8 w-8 text-destructive"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							xmlns="http://www.w3.org/2000/svg"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							></path>
						</svg>
					</div>
					<h3 class="mb-2 text-lg font-semibold">{$t.messages.error.networkError}</h3>
					<p class="mb-4 text-sm text-muted-foreground">{$t.messages.error.serverError}</p>
					<button
						onclick={() => hostsStore.fetchHosts()}
						class="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
					>
						{$t.ui.common.retry}
					</button>
				</div>
			{:else if $debouncedLoading}
				{#each Array(3) as _, i (i)}
					<HostCardSkeleton class="mx-auto max-w-[40em]" />
				{/each}
			{:else if !$hostsStore || $hostsStore.length === 0}
				<div class="mx-auto max-w-[40em] rounded-lg border border-dashed p-12 text-center">
					<p class="text-lg text-muted-foreground">{$t.ui.host.empty.title}</p>
					<p class="mt-2 text-sm text-muted-foreground">
						{$t.ui.host.empty.description}
					</p>
				</div>
			{:else}
				{#each $hostsStore as host (host.id)}
					<HostCard
						{host}
						class="mx-auto max-w-[40em]"
						bulkPingResult={lastPingResults.get(host.id)}
					/>
				{/each}
			{/if}
		</ul>
	</div>
</main>
