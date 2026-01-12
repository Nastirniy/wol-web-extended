<script>
	import { get } from 'svelte/store';
	import { LogOut, Plus, Users, X } from 'lucide-svelte';
	import { page } from '$app/state';
	import { authStore } from '$lib/stores/auth';
	import { t } from '$lib/stores/locale';
	import { isLoadingAddForm, showAddForm } from '$lib/stores/ui';
	import { goto } from '$lib/utils/navigation';
	import NoAuthIndicator from './NoAuthIndicator.svelte';
	import { Button } from './ui/button';
	import { Skeleton } from './ui/skeleton';

	function logout() {
		authStore.logout();
		goto('/auth');
	}

	async function toggleAddForm() {
		const currentState = get(showAddForm);

		if (!currentState) {
			// Opening the form - show skeleton briefly
			showAddForm.set(true);
			isLoadingAddForm.set(true);
			// Minimum display time for skeleton (150ms to prevent blink)
			setTimeout(() => {
				isLoadingAddForm.set(false);
			}, 150);
		} else {
			// Closing the form
			showAddForm.set(false);
			isLoadingAddForm.set(false);
		}
	}

	// Reactive values that update when authStore changes
	let isAuth = $derived($authStore.isAuthenticated);
	let isSuperuser = $derived($authStore.currentUser?.is_superuser ?? false);
	let isReadOnly = $derived($authStore.isReadOnly);
	let useAuth = $derived($authStore.useAuth);
	let isLoading = $derived($authStore.isLoading);
	// Handle URL prefix by checking if pathname ends with /home or /auth
	let isHomePage = $derived(page.url.pathname.endsWith('/home') || page.url.pathname === '/home');
	let isAuthPage = $derived(page.url.pathname.endsWith('/auth') || page.url.pathname === '/auth');
</script>

<div
	class="fixed top-4 flex h-12 w-full items-center justify-between bg-background px-4 font-mono text-lg font-bold"
>
	<div class="flex items-center gap-2">
		{#if isHomePage}
			{#if isLoading}
				<!-- Show skeleton buttons while loading -->
				<Skeleton class="h-8 w-28" />
			{:else}
				{#if !isReadOnly && (isAuth || !useAuth)}
					<Button onclick={toggleAddForm} variant="outline" class="gap-2" size="sm">
						{#if $showAddForm}
							<X class="h-4 w-4" />
							{$t.ui.common.cancel}
						{:else}
							<Plus class="h-4 w-4" />
							{$t.ui.nav.home.addDevice}
						{/if}
					</Button>
				{/if}
				{#if isSuperuser}
					<Button onclick={() => goto('/users')} variant="outline" class="gap-2" size="sm">
						<Users class="h-4 w-4" />
						{$t.ui.nav.home.users}
					</Button>
				{/if}
			{/if}
		{/if}
	</div>

	<!-- Center: Reserved for future use -->
	<div class="flex items-center gap-2"></div>

	{#if isLoading}
		<!-- Show skeleton for logout/indicator while loading -->
		{#if isHomePage}
			<Skeleton class="h-8 w-8" />
		{/if}
	{:else if useAuth && isAuth && !isAuthPage}
		<Button size="icon" variant="outline" class="" onclick={logout} title={$t.ui.nav.home.logout}>
			<LogOut />
		</Button>
	{:else if !useAuth}
		<NoAuthIndicator />
	{:else}
		<span></span>
	{/if}
</div>
