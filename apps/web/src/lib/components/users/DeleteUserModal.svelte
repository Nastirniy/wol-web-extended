<script lang="ts">
	import { fade } from 'svelte/transition';
	import { toast } from 'svoast';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { t } from '$lib/stores/locale';
	import { usersStore } from '$lib/stores/users';

	let {
		open = $bindable(false),
		userId,
		username,
		onSuccess
	}: {
		open?: boolean;
		userId: string | null;
		username: string | null;
		onSuccess?: () => void;
	} = $props();

	async function confirmDelete() {
		if (!userId) return;

		try {
			await usersStore.deleteUser(userId);
			toast.success($t.messages.user.deleteSuccess, { closable: true });
			close();
			onSuccess?.();
		} catch (error) {
			// Error already handled by store with specific error code message
			console.error('Error deleting user:', error);
		}
	}

	function close() {
		open = false;
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		transition:fade={{ duration: 150 }}
		class="fixed bottom-0 left-0 right-0 top-0 z-[9999] flex items-center justify-center"
		style="background: rgba(0, 0, 0, 0.2); backdrop-filter: blur(4px);"
		onclick={(e) => e.target === e.currentTarget && close()}
		onkeydown={(e) => e.key === 'Escape' && close()}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<Card.Root class="mx-4 w-full max-w-md border-2 shadow-2xl">
			<Card.Header class="pt-6">
				<Card.Title>{$t.ui.user.deleteTitle}</Card.Title>
				<Card.Description>
					{#if username}
						{$t.ui.user.deleteDescription.replace('{name}', username)}
					{/if}
				</Card.Description>
			</Card.Header>
			<Card.Footer class="flex justify-end gap-2">
				<Button variant="outline" onclick={close}>{$t.ui.common.cancel}</Button>
				<Button variant="destructive" onclick={confirmDelete}>{$t.ui.user.deleteMenuButton}</Button>
			</Card.Footer>
		</Card.Root>
	</div>
{/if}
