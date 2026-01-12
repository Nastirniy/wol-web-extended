<script lang="ts">
	import { AlertCircle, RefreshCw } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import { t } from '$lib/stores/locale';
	import { cn } from '$lib/utils';

	let {
		onRetry,
		message,
		class: className
	}: {
		onRetry?: () => void;
		message?: string;
		class?: string;
	} = $props();

	let isRetrying = $state(false);

	async function handleRetry() {
		if (!onRetry || isRetrying) return;
		isRetrying = true;
		try {
			await onRetry();
		} finally {
			// Keep retrying state for a bit to prevent double-clicks
			setTimeout(() => {
				isRetrying = false;
			}, 500);
		}
	}
</script>

<div
	class={cn(
		'flex items-center gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-3',
		className
	)}
>
	<AlertCircle class="h-8 w-8 flex-shrink-0 text-destructive" />
	<div class="flex-1 space-y-0.5 text-left">
		<h3 class="text-sm font-semibold">{$t.ui.error.serverUnreachable}</h3>
		<p class="text-xs text-muted-foreground">
			{message || $t.ui.error.serverUnreachableDescription}
		</p>
	</div>
	{#if onRetry}
		<Button
			onclick={handleRetry}
			disabled={isRetrying}
			variant="outline"
			size="sm"
			class="flex-shrink-0 gap-1.5"
		>
			<RefreshCw class={cn('h-3.5 w-3.5', isRetrying && 'animate-spin')} />
			{isRetrying ? $t.ui.error.retrying : $t.ui.common.retry}
		</Button>
	{/if}
</div>
