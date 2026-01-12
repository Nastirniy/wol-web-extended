<script lang="ts">
	import { InfoIcon } from 'lucide-svelte';
	import { toast } from 'svoast';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Popover from '$lib/components/ui/popover/index';
	import { t } from '$lib/stores/locale';
	import { usersStore } from '$lib/stores/users';
	import CreateUserFormSkeleton from './CreateUserFormSkeleton.svelte';

	let { onSuccess, isLoading = false }: { onSuccess?: () => void; isLoading?: boolean } = $props();

	let formData = $state({
		username: '',
		password: '',
		confirmPassword: '',
		readonly: false,
		is_superuser: false
	});

	let isSubmitting = $state(false);

	async function createUser() {
		if (isSubmitting) return;
		// Trim whitespace from inputs
		formData.username = formData.username.trim();
		formData.password = formData.password.trim();
		formData.confirmPassword = formData.confirmPassword.trim();

		if (!formData.username || !formData.password) {
			toast.error($t.messages.user.createUserRequiredFields, { closable: true });
			return;
		}

		if (formData.password !== formData.confirmPassword) {
			toast.error($t.messages.user.createUserPasswordMismatch, { closable: true });
			return;
		}

		isSubmitting = true;
		try {
			await usersStore.createUser({
				username: formData.username,
				password: formData.password,
				readonly: formData.readonly,
				is_superuser: formData.is_superuser
			});

			toast.success($t.messages.user.createSuccess, { closable: true });
			resetForm();
			onSuccess?.();
		} catch (error: any) {
			if (error.message !== 'HANDLED') {
				console.error('Error creating user:', error);
				toast.error($t.messages.user.createError, { closable: true });
			}
		} finally {
			isSubmitting = false;
		}
	}

	function resetForm() {
		formData = {
			username: '',
			password: '',
			confirmPassword: '',
			readonly: false,
			is_superuser: false
		};
	}
</script>

{#if isLoading}
	<CreateUserFormSkeleton />
{:else}
	<div class="mx-auto mb-6 w-full max-w-4xl space-y-4">
		<div>
			<h2 class="text-lg font-semibold">{$t.ui.user.createNewUserTitle}</h2>
			<p class="text-sm text-muted-foreground">{$t.ui.user.createNewUserDescription}</p>
		</div>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				createUser();
			}}
		>
			<div class="grid gap-4">
				<div class="grid gap-2">
					<Label for="username">{$t.ui.user.usernameLabel}</Label>
					<Input
						id="username"
						bind:value={formData.username}
						placeholder={$t.ui.user.usernamePlaceholder}
						autofocus
					/>
				</div>
				<div class="grid gap-2">
					<Label for="password">{$t.ui.user.passwordLabel}</Label>
					<Input
						id="password"
						type="password"
						bind:value={formData.password}
						placeholder={$t.ui.user.passwordPlaceholder}
					/>
				</div>
				<div class="grid gap-2">
					<Label for="confirmPassword">{$t.ui.user.confirmPasswordLabel}</Label>
					<Input
						id="confirmPassword"
						type="password"
						bind:value={formData.confirmPassword}
						placeholder={$t.ui.user.confirmPasswordPlaceholder}
					/>
				</div>
				<div class="flex items-center space-x-2">
					<input
						type="checkbox"
						id="readonly"
						bind:checked={formData.readonly}
						class="h-4 w-4 rounded border-gray-300"
					/>
					<Label for="readonly" class="cursor-pointer">
						{$t.ui.user.readonlyDescription}
					</Label>
					<Popover.Root>
						<Popover.Trigger>
							<Button variant="secondary" size="icon" type="button" class="shrink-0"
								><InfoIcon class="h-4 w-4" /></Button
							>
						</Popover.Trigger>
						<Popover.Content>
							<p class="mb-2 text-sm font-semibold">{$t.ui.user.info.readonlyTitle}</p>
							<p class="mb-1 text-sm">{$t.ui.user.info.readonlyDescription}</p>
							<ul class="ml-4 list-disc space-y-1 text-sm">
								<li>{$t.ui.user.info.readonlyPermission1}</li>
								<li>{$t.ui.user.info.readonlyPermission2}</li>
								<li>{$t.ui.user.info.readonlyPermission3}</li>
							</ul>
							<p class="mb-1 mt-2 text-sm font-semibold text-orange-600">
								{$t.ui.user.info.readonlyRestriction}
							</p>
							<ul class="ml-4 list-disc space-y-1 text-sm text-muted-foreground">
								<li>{$t.ui.user.info.readonlyRestriction1}</li>
								<li>{$t.ui.user.info.readonlyRestriction2}</li>
							</ul>
						</Popover.Content>
					</Popover.Root>
				</div>
				<div class="flex items-center space-x-2">
					<input
						type="checkbox"
						id="is-superuser"
						bind:checked={formData.is_superuser}
						class="h-4 w-4 rounded border-gray-300"
					/>
					<Label for="is-superuser" class="cursor-pointer">
						{$t.ui.user.card.superuser}
					</Label>
				</div>
			</div>
			<Button type="submit" class="mt-4 w-full" disabled={isSubmitting}>
				{isSubmitting ? $t.ui.user.creatingButton : $t.ui.user.createButton}
			</Button>
		</form>
	</div>
{/if}
