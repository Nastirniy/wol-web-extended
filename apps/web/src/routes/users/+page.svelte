<script lang="ts">
	import { onMount } from 'svelte';
	import { Home, UserPlus, X } from 'lucide-svelte';
	import { toast } from 'svoast';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import CreateUserForm from '$lib/components/users/CreateUserForm.svelte';
	import DeleteUserModal from '$lib/components/users/DeleteUserModal.svelte';
	import UserCard from '$lib/components/users/UserCard.svelte';
	import UserCardSkeleton from '$lib/components/users/UserCardSkeleton.svelte';
	import { authStore } from '$lib/stores/auth';
	import { t } from '$lib/stores/locale';
	import { createDebouncedLoadingStore, showAddForm } from '$lib/stores/ui';
	import { usersStore } from '$lib/stores/users';
	import type { User } from '$lib/types/api';
	import { goto } from '$lib/utils/navigation';

	let isCheckingAuth = $state(true);
	let showCreateForm = $state(false);
	let isLoadingCreateForm = $state(false);
	let showDeleteModal = $state(false);
	let userToDelete = $state<{ id: string; name: string } | null>(null);

	let hasCheckedAuth = false;
	const debouncedLoading = createDebouncedLoadingStore(350);
	let hasUsersError = $state(false);

	$effect(() => {
		const unsubscribe = usersStore.isLoading.subscribe((value) => {
			debouncedLoading.setLoading(value);
		});
		return unsubscribe;
	});

	$effect(() => {
		const unsubscribe = usersStore.hasError.subscribe((value) => {
			hasUsersError = value;
		});
		return unsubscribe;
	});

	onMount(async () => {
		// Reset the home page create form when navigating to users page
		showAddForm.set(false);

		if (!hasCheckedAuth) {
			hasCheckedAuth = true;
			checkAuthAndFetch();
		}
	});

	async function checkAuthAndFetch() {
		try {
			// Wait for auth store to finish loading
			while ($authStore.isLoading) {
				await new Promise((resolve) => setTimeout(resolve, 100));
			}

			// Auth check complete
			isCheckingAuth = false;

			// Check if auth is enabled
			if (!$authStore.useAuth) {
				goto('/home');
				return;
			}

			// Check if user is authenticated
			if (!$authStore.isAuthenticated) {
				goto('/auth');
				return;
			}

			// Check if user is superuser
			if (!$authStore.currentUser?.is_superuser) {
				toast.error($t.messages.auth.accessDenied, { closable: true });
				goto('/home');
				return;
			}

			usersStore.fetchUsers();
		} catch (error) {
			console.error('Error checking auth:', error);
			isCheckingAuth = false;
			// Don't redirect on fetch errors - just show error and try to load users anyway
			usersStore.fetchUsers();
		}
	}

	function showDeleteConfirmation(userId: string, username: string) {
		userToDelete = { id: userId, name: username };
		showDeleteModal = true;
	}

	async function toggleCreateForm() {
		if (!showCreateForm) {
			// Opening the form - show skeleton briefly
			showCreateForm = true;
			isLoadingCreateForm = true;

			// Minimum display time for skeleton (300ms to prevent blink)
			await new Promise((resolve) => setTimeout(resolve, 300));
			isLoadingCreateForm = false;
		} else {
			// Closing the form
			showCreateForm = false;
			isLoadingCreateForm = false;
		}
	}
</script>

<div
	class="fixed top-4 flex h-12 w-full items-center justify-between bg-background px-4 font-mono text-lg font-bold"
>
	<div class="flex items-center gap-2">
		{#if isCheckingAuth}
			<Skeleton class="h-8 w-28" />
		{:else}
			<Button onclick={() => goto('/home')} variant="outline" class="gap-2" size="sm">
				<Home class="h-4 w-4" />
				{$t.ui.nav.users.homepage}
			</Button>
		{/if}
	</div>

	{#if isCheckingAuth}
		<Skeleton class="h-8 w-24" />
	{:else}
		<Button onclick={toggleCreateForm} variant="outline" class="gap-2" size="sm">
			{#if showCreateForm}
				<X class="h-4 w-4" />
				{$t.ui.common.cancel}
			{:else}
				<UserPlus class="h-4 w-4" />
				{$t.ui.nav.users.newUser}
			{/if}
		</Button>
	{/if}
</div>

<div class="container mx-auto mb-20 mt-20 flex flex-col items-center p-4">
	{#if showCreateForm}
		<CreateUserForm
			isLoading={isLoadingCreateForm}
			onSuccess={() => {
				showCreateForm = false;
				isLoadingCreateForm = false;
				usersStore.fetchUsers();
			}}
		/>
	{/if}

	{#if hasUsersError && !$debouncedLoading}
		<div class="mx-auto w-full max-w-4xl rounded-lg border p-8 text-center">
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
				onclick={() => usersStore.fetchUsers()}
				class="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
			>
				{$t.ui.common.retry}
			</button>
		</div>
	{:else if $debouncedLoading}
		<div class="mx-auto w-full max-w-4xl space-y-2">
			{#each Array(3) as _, i (i)}
				<UserCardSkeleton />
			{/each}
		</div>
	{:else if $usersStore.length === 0}
		<p class="py-8 text-center text-muted-foreground">{$t.ui.user.noUsers}</p>
	{:else}
		<div class="mx-auto w-full max-w-4xl space-y-2">
			{#each $usersStore as user}
				<UserCard {user} onDelete={showDeleteConfirmation} />
			{/each}
		</div>
	{/if}
</div>

<DeleteUserModal
	bind:open={showDeleteModal}
	userId={userToDelete?.id ?? null}
	username={userToDelete?.name ?? null}
	onSuccess={() => usersStore.fetchUsers()}
/>
