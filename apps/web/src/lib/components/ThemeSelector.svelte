<script lang="ts">
	import { Monitor, Moon, Smartphone, Sun } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Popover from '$lib/components/ui/popover';
	import { t } from '$lib/stores/locale';
	import { type Theme, themeStore } from '$lib/stores/theme';

	const themes: { value: Theme; labelKey: string; icon: any }[] = [
		{ value: 'light', labelKey: 'light', icon: Sun },
		{ value: 'dark', labelKey: 'dark', icon: Moon },
		{ value: 'amoled', labelKey: 'amoled', icon: Moon }
	];

	function selectTheme(theme: Theme) {
		themeStore.set(theme);
	}

	function getThemeLabel(key: string) {
		// @ts-ignore
		return $t.ui.theme[key] || key;
	}
</script>

<Popover.Root>
	<Popover.Trigger>
		<Button size="icon" variant="outline" title={$t.ui.footer.theme} class="h-8 w-8">
			{#if $themeStore === 'light'}
				<Sun class="h-4 w-4" />
			{:else}
				<Moon class="h-4 w-4" />
			{/if}
		</Button>
	</Popover.Trigger>
	<Popover.Content class="w-48 p-2">
		<div class="mb-2 text-sm font-semibold">{$t.ui.footer.theme}</div>
		<div class="space-y-1">
			{#each themes as theme}
				<button
					onclick={() => selectTheme(theme.value)}
					class="flex w-full items-center justify-between rounded px-3 py-2 text-sm transition-colors hover:bg-accent {$themeStore ===
					theme.value
						? 'bg-accent font-medium'
						: ''}"
				>
					<span class="flex items-center gap-2">
						{#if theme.value === 'light'}
							<Sun class="h-4 w-4" />
						{/if}
						{#if theme.value === 'dark'}
							<Moon class="h-4 w-4" />
						{/if}
						{#if theme.value === 'amoled'}
							<Moon class="h-4 w-4 fill-current" />
						{/if}
						{getThemeLabel(theme.labelKey)}
					</span>
					{#if $themeStore === theme.value}
						<svg
							class="h-4 w-4"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							xmlns="http://www.w3.org/2000/svg"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M5 13l4 4L19 7"
							></path>
						</svg>
					{/if}
				</button>
			{/each}
		</div>
	</Popover.Content>
</Popover.Root>
