<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svoast';
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { t } from '$lib/stores/locale';
	import { createDebouncedLoadingStore } from '$lib/stores/ui';
	import { buildApiUrl } from '$lib/utils/api';
	import { handleGenericError } from '$lib/utils/errors';
	import { goto } from '$lib/utils/navigation';

	let username = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let isRegistering = $state(false);
	let hasSuperuser = $state<boolean | null>(null);
	const debouncedLoading = createDebouncedLoadingStore(350);
	let hasChecked = $state(false);

	// Check if superuser exists - only once on mount
	onMount(() => {
		if (!hasChecked) {
			hasChecked = true;
			checkSuperuser();
		}
	});

	async function checkSuperuser() {
		debouncedLoading.setLoading(true);
		try {
			const response = await fetch(buildApiUrl('/api/auth/has-superuser'));
			const data = await response.json();
			hasSuperuser = data.has_superuser;

			// If superuser already exists, redirect to auth page
			if (data.has_superuser) {
				goto('/auth');
			}
		} catch (error) {
			console.error('Failed to check superuser:', error);
			hasSuperuser = false;
		} finally {
			await debouncedLoading.setLoading(false);
		}
	}

	async function setupSuperuser(e: Event) {
		e.preventDefault();

		// Trim whitespace from inputs
		username = username.trim();
		password = password.trim();
		confirmPassword = confirmPassword.trim();

		if (password !== confirmPassword) {
			toast.error($t.messages.setup.passwordMismatch, { closable: true });
			return;
		}

		if (!username || !password) {
			toast.error($t.messages.setup.requiredFields, { closable: true });
			return;
		}

		isRegistering = true;

		try {
			const response = await fetch(buildApiUrl('/api/auth/setup'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});

			if (response.ok) {
				const data = await response.json();
				if (data.success) {
					toast.success($t.messages.setup.success, { closable: true });
					setTimeout(() => {
						goto('/auth');
					}, 1500);
				} else {
					toast.error($t.messages.setup.failed, { closable: true });
				}
			} else {
				const errorText = await response.text();
				handleGenericError('create superuser', errorText);
			}
		} catch (error) {
			console.error('Setup error:', error);
			toast.error($t.messages.setup.failed, { closable: true });

			t;
		}
		isRegistering = false;
	}
</script>

{#if $debouncedLoading}
	<div class="flex h-screen items-center justify-center px-4">
		<Card.Root class="w-96 max-w-full">
			<Card.Header>
				<Skeleton class="mx-auto h-7 w-32" />
				<Skeleton class="mx-auto h-5 w-48" />
			</Card.Header>
			<Card.Content>
				<div class="grid w-full items-center gap-4">
					<Skeleton class="h-10 w-full" />
					<Skeleton class="h-10 w-full" />
					<Skeleton class="h-10 w-full" />
				</div>
			</Card.Content>
			<Card.Footer>
				<Skeleton class="h-10 w-full" />
			</Card.Footer>
		</Card.Root>
	</div>
{:else if !hasSuperuser}
	<div class="flex h-screen items-center justify-center px-4">
		<Card.Root class="w-96 max-w-full">
			<form onsubmit={setupSuperuser}>
				<Card.Header>
					<Card.Title class="text-center">{$t.ui.setup.title}</Card.Title>
					<Card.Description class="text-center">{$t.ui.setup.description}</Card.Description>
				</Card.Header>
				<Card.Content>
					<div class="grid w-full items-center gap-4">
						<div class="flex flex-col space-y-1.5">
							<Input
								id="setup-username"
								placeholder={$t.ui.setup.usernamePlaceholder}
								bind:value={username}
								autofocus
							/>
						</div>
						<div class="flex flex-col space-y-1.5">
							<Input
								id="setup-password"
								type="password"
								placeholder={$t.ui.setup.passwordPlaceholder}
								bind:value={password}
							/>
						</div>
						<div class="flex flex-col space-y-1.5">
							<Input
								id="setup-confirm-password"
								type="password"
								placeholder={$t.ui.setup.confirmPasswordPlaceholder}
								bind:value={confirmPassword}
							/>
						</div>
					</div>
				</Card.Content>
				<Card.Footer>
					<Button type="submit" class="w-full" disabled={isRegistering}>
						{isRegistering ? $t.ui.setup.creatingButton : $t.ui.setup.createButton}
					</Button>
				</Card.Footer>
			</form>
		</Card.Root>
	</div>
{/if}
