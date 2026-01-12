<script lang="ts">
	import { ChevronDown, X } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Popover from '$lib/components/ui/popover';
	import { t } from '$lib/stores/locale';
	import { cn } from '$lib/utils.js';

	interface Option {
		value: string;
		label: string;
		description?: string;
	}

	let {
		options = [],
		selectedValues = $bindable([]),
		placeholder = $bindable($t.ui.common.selectOptionsPlaceholder),
		disabled = false,
		class: className = '',
		name = ''
	}: {
		options: Option[];
		selectedValues?: string[];
		placeholder?: string;
		disabled?: boolean;
		class?: string;
		name?: string;
	} = $props();

	let open = $state(false);

	function toggleOption(value: string) {
		if (selectedValues.includes(value)) {
			selectedValues = selectedValues.filter((v) => v !== value);
		} else {
			selectedValues = [...selectedValues, value];
		}
	}

	function removeOption(value: string, event: MouseEvent) {
		event.stopPropagation();
		selectedValues = selectedValues.filter((v) => v !== value);
	}

	function clearAll(event?: MouseEvent | KeyboardEvent) {
		event?.stopPropagation();
		selectedValues = [];
	}

	const selectedOptions = $derived(options.filter((opt) => selectedValues.includes(opt.value)));

	const displayText = $derived(
		selectedOptions.length === 0
			? placeholder
			: selectedOptions.length === 1
				? selectedOptions[0].label
				: `${selectedOptions.length} {$t.ui.common.selectedCount}`
	);
</script>

<Popover.Root bind:open>
	<Popover.Trigger
		class={cn(
			'flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-base ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
			className
		)}
		type="button"
		role="combobox"
		aria-expanded={open}
		aria-controls="multi-select-content"
		{disabled}
	>
		<div class="flex flex-1 items-center justify-between gap-2 overflow-hidden">
			{#if selectedOptions.length === 0}
				<span class="text-muted-foreground">{placeholder}</span>
			{:else}
				<span class="text-sm">
					{selectedOptions.length}
					{$t.ui.common.selectedCount}
				</span>
				{#if selectedOptions.length > 0}
					<span
						role="button"
						tabindex="-1"
						onclick={clearAll}
						onkeydown={(e) => e.key === 'Enter' && clearAll(e)}
						class="cursor-pointer text-xs text-muted-foreground hover:text-destructive"
					>
						{$t.ui.common.clearButton}
					</span>
				{/if}
			{/if}
		</div>
		<ChevronDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
	</Popover.Trigger>
	<Popover.Content
		id="multi-select-content"
		class="w-[--bits-popover-trigger-width] p-0"
		align="start"
	>
		<div class="max-h-64 overflow-y-auto p-1">
			{#if options.length === 0}
				<div class="px-2 py-6 text-center text-sm text-muted-foreground">
					{$t.ui.common.noOptionsAvailable}
				</div>
			{:else}
				{#each options as option}
					<button
						type="button"
						class="flex w-full items-start gap-2 rounded px-2 py-2 text-left text-sm hover:bg-accent hover:text-accent-foreground"
						onclick={() => toggleOption(option.value)}
						title={option.label}
					>
						<div class="flex h-4 w-4 shrink-0 items-center justify-center rounded border">
							{#if selectedValues.includes(option.value)}
								<div class="h-2 w-2 rounded-sm bg-primary"></div>
							{/if}
						</div>
						<div class="min-w-0 flex-1">
							<div class="truncate font-medium">{option.label}</div>
							{#if option.description}
								<div class="truncate text-xs text-muted-foreground">{option.description}</div>
							{/if}
						</div>
					</button>
				{/each}
			{/if}
		</div>
		{#if selectedOptions.length > 0}
			<div class="border-t p-2">
				<button
					type="button"
					class="w-full text-center text-xs text-muted-foreground hover:text-destructive"
					onclick={clearAll}
				>
					{$t.ui.common.clearAllButton} ({selectedOptions.length})
				</button>
			</div>
		{/if}
	</Popover.Content>
</Popover.Root>

<!-- Hidden input for form submission -->
{#if name}
	<input type="hidden" {name} value={selectedValues.join(',')} />
{/if}
