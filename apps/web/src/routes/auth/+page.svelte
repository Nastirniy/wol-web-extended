<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svoast';
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { authStore } from '$lib/stores/auth';
	import { t } from '$lib/stores/locale';
	import { createDebouncedLoadingStore } from '$lib/stores/ui';
	import { buildApiUrl } from '$lib/utils/api';
	import { goto } from '$lib/utils/navigation';

	let username = $state('');
	let password = $state('');
	let isLoggingIn = $state(false);
	const debouncedLoading = createDebouncedLoadingStore(350);
	let authEnabled = $state(true);
	let hasChecked = $state(false);

	// Check authentication status when component mounts - only once
	onMount(() => {
		if (!hasChecked) {
			hasChecked = true;
			checkAuthStatus();
		}
	});

	async function checkAuthStatus() {
		debouncedLoading.setLoading(true);
		try {
			const response = await fetch(buildApiUrl('/api/auth/has-superuser'));
			const data = await response.json();
			authEnabled = data.auth_enabled;

			// If no superuser exists, redirect to setup page
			if (!data.has_superuser) {
				goto('/setup');
				return;
			}
		} catch (error) {
			console.error('Failed to check auth status:', error);
		} finally {
			await debouncedLoading.setLoading(false);
		}
	}

	$effect(() => {
		// Wait for both setup check and auth state to load
		if ($debouncedLoading || $authStore.isLoading) {
			return;
		}

		// If no auth is required, redirect to main page
		if (!authEnabled) {
			goto('/home');
			return;
		}

		// If already authenticated, redirect to home
		if ($authStore.isAuthenticated) {
			goto('/home');
			return;
		}
	});

	async function login(e: Event) {
		e.preventDefault();
		isLoggingIn = true;

		// Validate and trim inputs
		const trimmedUsername = username.trim();
		const trimmedPassword = password.trim();

		if (!trimmedUsername || !trimmedPassword) {
			toast.error($t.messages.auth.loginRequiredFields, { closable: true });
			isLoggingIn = false;
			return;
		}

		try {
			const result = await authStore.login(trimmedUsername, trimmedPassword);
			if (result.success) {
				toast.success($t.messages.auth.loginSuccess, { closable: true });
				goto('/home');
			} else {
				toast.error($t.messages.auth.loginError, { closable: true });
				isLoggingIn = false;
			}
		} catch (error) {
			toast.error($t.messages.auth.loginUnexpectedError, { closable: true });
			isLoggingIn = false;
		}
	}
</script>

<div class="flex h-screen items-center justify-center px-4">
	<Card.Root class="w-96 max-w-full">
		{#if $debouncedLoading}
			<Card.Header>
				<Skeleton class="mx-auto h-7 w-24" />
			</Card.Header>
			<Card.Content>
				<div class="grid w-full items-center gap-4">
					<Skeleton class="h-10 w-full" />
					<Skeleton class="h-10 w-full" />
				</div>
			</Card.Content>
			<Card.Footer class="flex flex-col gap-2">
				<Skeleton class="h-10 w-full" />
			</Card.Footer>
		{:else}
			<form onsubmit={login}>
				<Card.Header>
					<Card.Title class="text-center">{$t.ui.auth.login}</Card.Title>
				</Card.Header>
				<Card.Content>
					<div class="grid w-full items-center gap-4">
						<div class="flex flex-col space-y-1.5">
							<Input
								id="username"
								placeholder={$t.ui.auth.usernamePlaceholder}
								bind:value={username}
								autofocus
							/>
						</div>
						<div class="flex flex-col space-y-1.5">
							<Input
								id="password"
								type="password"
								placeholder={$t.ui.auth.passwordPlaceholder}
								bind:value={password}
							/>
						</div>
					</div>
				</Card.Content>
				<Card.Footer class="flex flex-col gap-2">
					<Button type="submit" class="w-full" disabled={isLoggingIn}>
						{isLoggingIn ? $t.ui.auth.loggingIn : $t.ui.auth.continueButton}
					</Button>
				</Card.Footer>
			</form>
		{/if}
	</Card.Root>
</div>
