<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import { Button } from '$lib/components/ui/button';
	import { authStore } from '$lib/stores/auth';
	import { t } from '$lib/stores/locale';
	import { goto } from '$lib/utils/navigation';

	let isNavigating = $state(false);
	let navigationError = $state<string | null>(null);
	let configLoadFailed = $state(false);

	onMount(async () => {
		try {
			await authStore.loadConfig();
		} catch (error) {
			console.error('Failed to load config in error page:', error);
			configLoadFailed = true;
		}
	});

	async function handleNavigation() {
		if (isNavigating) return;

		isNavigating = true;
		navigationError = null;

		try {
			// Wait for auth state to load to avoid showing intermediate pages
			if (!configLoadFailed && page.status < 500 && $authStore.isLoading) {
				// Wait a bit for auth to load
				await new Promise((resolve) => setTimeout(resolve, 100));
			}

			// If server is unreachable, stay on error page or reload
			if ($authStore.serverUnreachable && !configLoadFailed) {
				// Just reload the page to retry connection
				if (browser) {
					window.location.reload();
				}
				return;
			}

			// If config failed to load or we're in a server error state, default to /home
			// The destination route will handle auth redirect if needed
			if (configLoadFailed || page.status >= 500) {
				await goto('/home');
			} else if ($authStore.useAuth && !$authStore.isAuthenticated) {
				await goto('/auth');
			} else {
				await goto('/home');
			}
		} catch (error) {
			isNavigating = false;
			navigationError = error instanceof Error ? error.message : 'Navigation failed';
		}
	}

	function handleReload() {
		if (browser) {
			window.location.reload();
		}
	}

	let pageTitle = $derived.by(() => {
		if (page.status === 404) {
			return $t.ui.error.pageNotFound;
		} else if (page.status >= 500) {
			return $t.ui.error.serverError;
		}
		return $t.ui.error.error;
	});

	let errorMessage = $derived.by(() => {
		if (navigationError) {
			return navigationError;
		}
		if (page.error?.message) {
			return page.error.message;
		}
		if (page.status === 404) {
			return $t.ui.error.pageNotFoundMessage;
		} else if (page.status >= 500) {
			return $t.ui.error.serverErrorMessage;
		}
		return $t.ui.error.pageNotFoundMessage;
	});

	let buttonText = $derived.by(() => {
		if (isNavigating) return $t.ui.error.navigating;
		if ($authStore.isLoading && !configLoadFailed) return $t.ui.common.loading;
		// If server is unreachable, show reload option
		if ($authStore.serverUnreachable && !configLoadFailed) return $t.ui.error.reload;
		// If config failed or server error, always show "Go to Home"
		if (configLoadFailed || page.status >= 500) return $t.ui.error.goToHome;
		if ($authStore.useAuth && !$authStore.isAuthenticated) return $t.ui.error.goToLogin;
		return $t.ui.error.goToHome;
	});

	let showReloadButton = $derived(page.status >= 500 && !$authStore.serverUnreachable);
	let isButtonDisabled = $derived(
		isNavigating || ($authStore.isLoading && !configLoadFailed && page.status < 500)
	);
</script>

<div class="flex min-h-screen flex-col items-center justify-center px-4">
	<div class="text-center">
		<h1 class="text-6xl font-bold text-foreground/80">{page.status}</h1>
		<p class="mt-4 text-2xl font-semibold text-foreground">
			{pageTitle}
		</p>
		<p class="mt-2 text-foreground/60">
			{errorMessage}
		</p>
		<div class="mt-8 flex flex-col gap-3 sm:flex-row sm:justify-center">
			<Button onclick={handleNavigation} variant="default" disabled={isButtonDisabled}>
				{buttonText}
			</Button>
			{#if showReloadButton}
				<Button onclick={handleReload} variant="outline">
					{$t.ui.error.reload}
				</Button>
			{/if}
		</div>
	</div>
</div>
