<script lang="ts">
	import { Edit, EllipsisVertical, InfoIcon, Trash2 } from 'lucide-svelte';
	import { toast } from 'svoast';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Popover from '$lib/components/ui/popover/index';
	import { t } from '$lib/stores/locale';
	import { usersStore } from '$lib/stores/users';
	import type { User } from '$lib/types/api';

	let {
		user,
		onDelete
	}: {
		user: User;
		onDelete: (userId: string, username: string) => void;
	} = $props();

	let isEditing = $state(false);
	let showOptionsMenu = $state(false);
	let isSubmitting = $state(false);
	let menuButtonRef: HTMLButtonElement | null = $state(null);

	let formData = $state({
		username: '',
		password: '',
		confirmPassword: '',
		readonly: false,
		is_superuser: false
	});

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleString();
	}

	function startEdit() {
		isEditing = true;
		formData = {
			username: user.name,
			password: '',
			confirmPassword: '',
			readonly: user.readonly,
			is_superuser: user.is_superuser
		};
	}

	function cancelEdit() {
		isEditing = false;
		formData = {
			username: '',
			password: '',
			confirmPassword: '',
			readonly: false,
			is_superuser: false
		};
	}

	async function saveEdit() {
		if (isSubmitting) return;

		formData.username = formData.username.trim();
		formData.password = formData.password.trim();
		formData.confirmPassword = formData.confirmPassword.trim();

		if (!formData.username) {
			toast.error($t.messages.user.usernameRequired, { closable: true });
			return;
		}

		if (formData.password && formData.password !== formData.confirmPassword) {
			toast.error($t.messages.user.passwordMismatch, { closable: true });
			return;
		}

		isSubmitting = true;
		try {
			await usersStore.updateUser(user.id, {
				name: formData.username,
				password: formData.password,
				readonly: formData.readonly,
				is_superuser: formData.is_superuser
			});

			toast.success($t.messages.user.updateSuccess, { closable: true });
			// Update local user object
			user.name = formData.username;
			user.readonly = formData.readonly;
			user.is_superuser = formData.is_superuser;
			cancelEdit();
		} catch (error: any) {
			if (error.message !== 'HANDLED') {
				console.error('Error updating user:', error);
				toast.error($t.messages.user.updateError, { closable: true });
			}
		} finally {
			isSubmitting = false;
		}
	}

	async function toggleReadonly() {
		try {
			// Store the new readonly value before update
			const newReadonlyValue = !user.readonly;

			// Backend requires ALL fields for update (name, readonly, is_superuser)
			await usersStore.updateUser(user.id, {
				name: user.name,
				readonly: newReadonlyValue,
				is_superuser: user.is_superuser
			});

			toast.success(user.readonly ? $t.ui.user.grantFullAccess : $t.ui.user.makeReadonlyTitle, {
				closable: true
			});
			user.readonly = newReadonlyValue;
		} catch (error) {
			// Error already handled by store with specific error code message
			console.error('Error toggling readonly:', error);
		}
	}

	// Close menu when clicking outside
	$effect(() => {
		if (!showOptionsMenu) return;

		function handleClickOutside(event: MouseEvent) {
			const target = event.target as Node;
			if (menuButtonRef?.contains(target)) return;
			showOptionsMenu = false;
		}

		document.addEventListener('click', handleClickOutside);
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});
</script>

<Card.Root class="max-w-4xl">
	{#if isEditing}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				saveEdit();
			}}
		>
			<Card.Content class="p-4">
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
					<div class="flex flex-col gap-2">
						<Label for="edit-username">{$t.ui.user.usernameLabel}</Label>
						<Input
							id="edit-username"
							bind:value={formData.username}
							placeholder={$t.ui.user.usernamePlaceholder}
						/>
					</div>
					{#if !user.is_superuser}
						<div class="flex flex-col gap-2">
							<Label for="edit-password">{$t.ui.user.newPasswordLabel}</Label>
							<Input
								id="edit-password"
								type="password"
								bind:value={formData.password}
								placeholder={$t.ui.user.passwordKeepPlaceholder}
							/>
						</div>
						{#if formData.password}
							<div class="flex flex-col gap-2">
								<Label for="edit-confirm-password">{$t.ui.user.confirmNewPasswordLabel}</Label>
								<Input
									id="edit-confirm-password"
									type="password"
									bind:value={formData.confirmPassword}
									placeholder={$t.ui.user.confirmNewPasswordPlaceholder}
								/>
							</div>
						{/if}
					{:else}
						<div class="flex flex-col gap-2">
							<Label>{$t.ui.user.passwordLabel}</Label>
							<div class="rounded-md bg-muted p-3">
								<p class="text-sm text-muted-foreground">
									{$t.ui.user.superuserPasswordNote}
								</p>
							</div>
						</div>
					{/if}
					<div class="flex items-center gap-2">
						<input
							type="checkbox"
							id="edit-readonly-{user.id}"
							bind:checked={formData.readonly}
							class="h-4 w-4"
						/>
						<Label for="edit-readonly-{user.id}">{$t.ui.user.readonlyLabel}</Label>
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
					<div class="flex items-center gap-2">
						<input
							type="checkbox"
							id="edit-is-superuser-{user.id}"
							bind:checked={formData.is_superuser}
							class="h-4 w-4"
						/>
						<Label for="edit-is-superuser-{user.id}">{$t.ui.user.card.superuser}</Label>
					</div>
				</div>
			</Card.Content>
			<Card.Footer class="flex justify-end gap-2 p-4 pt-0">
				<Button type="button" variant="outline" onclick={cancelEdit} disabled={isSubmitting}>
					{$t.ui.common.cancel}
				</Button>
				<Button type="submit" disabled={isSubmitting}>
					{isSubmitting ? $t.ui.common.saving : $t.ui.common.save}
				</Button>
			</Card.Footer>
		</form>
	{:else}
		<Card.Content class="p-4">
			<div class="flex flex-col gap-3">
				<div class="flex items-start justify-between gap-2">
					<h3 class="min-w-0 flex-1 truncate font-mono text-lg font-bold" title={user.name}>
						{user.name}
					</h3>
					<div class="flex shrink-0 gap-1">
						{#if user.is_superuser}
							<span
								class="rounded bg-purple-100 px-2 py-1 text-xs font-semibold text-purple-800 dark:bg-purple-900 dark:text-purple-200"
							>
								{$t.ui.user.superuserBadge}
							</span>
						{:else}
							<span
								class="rounded bg-gray-100 px-2 py-1 text-xs font-semibold text-gray-800 dark:bg-gray-800 dark:text-gray-200"
							>
								{$t.ui.user.userBadge}
							</span>
						{/if}
					</div>
				</div>
				<div class="flex flex-col gap-2">
					<div class="flex items-center gap-2">
						<span class="text-sm font-medium text-muted-foreground">{$t.ui.user.accessLabel}</span>
						{#if user.readonly}
							<span
								class="rounded bg-orange-100 px-2 py-1 text-xs font-semibold text-orange-800 dark:bg-orange-900 dark:text-orange-200"
							>
								{$t.ui.user.readonlyBadge}
							</span>
						{:else}
							<span
								class="rounded bg-green-100 px-2 py-1 text-xs font-semibold text-green-800 dark:bg-green-900 dark:text-green-200"
							>
								{$t.ui.user.fullAccessBadge}
							</span>
						{/if}
					</div>
					<div class="flex items-center gap-2">
						<span class="text-sm font-medium text-muted-foreground">{$t.ui.user.createdLabel}</span>
						<span class="font-mono text-sm">{formatDate(user.created)}</span>
					</div>
				</div>
			</div>
		</Card.Content>
		<Card.Footer class="relative flex flex-wrap gap-2 p-4 pt-0">
			<Button
				size="sm"
				variant="outline"
				onclick={toggleReadonly}
				title={user.readonly ? $t.ui.user.grantFullAccess : $t.ui.user.makeReadonlyTitle}
				class="min-w-[120px] flex-1"
			>
				{user.readonly ? $t.ui.user.grantAccess : $t.ui.user.makeReadonly}
			</Button>
			<Button
				bind:ref={menuButtonRef}
				size="sm"
				variant="outline"
				onclick={() => (showOptionsMenu = !showOptionsMenu)}
				title={$t.ui.user.moreOptions}
			>
				<EllipsisVertical class="h-4 w-4" />
			</Button>
			{#if showOptionsMenu}
				<div
					class="absolute bottom-full right-0 z-10 mb-1 min-w-[120px] rounded-md border bg-popover shadow-lg"
				>
					<button
						class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground"
						onclick={() => {
							startEdit();
							showOptionsMenu = false;
						}}
					>
						<Edit class="h-4 w-4" />
						{$t.ui.user.editButton}
					</button>
					<button
						class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-red-600 hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50 dark:text-red-500"
						disabled={user.is_superuser}
						title={user.is_superuser
							? $t.ui.user.cannotDeleteSuperuser
							: $t.ui.user.deleteMenuButton}
						onclick={() => {
							if (!user.is_superuser) {
								onDelete(user.id, user.name);
								showOptionsMenu = false;
							}
						}}
					>
						<Trash2 class="h-4 w-4" />
						{$t.ui.user.deleteMenuButton}
					</button>
				</div>
			{/if}
		</Card.Footer>
	{/if}
</Card.Root>
