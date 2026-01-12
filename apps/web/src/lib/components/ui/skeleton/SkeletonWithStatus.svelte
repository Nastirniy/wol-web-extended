<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { t } from '$lib/stores/locale';
	import { cn } from '$lib/utils';
	import { Skeleton } from './index';

	type Props = {
		isLoading?: boolean;
		hasError?: boolean;
		errorTitle?: string;
		errorDescription?: string;
		onRetry?: () => void;
		skeletonCount?: number;
		skeletonComponent?: any;
		className?: string;
		skeletonProps?: Record<string, any>;
	};

	let {
		isLoading: isLoadingProp,
		hasError: hasErrorProp,
		errorTitle: errorTitleProp,
		errorDescription: errorDescriptionProp,
		onRetry,
		skeletonCount: skeletonCountProp,
		skeletonComponent,
		className = '',
		skeletonProps = {}
	}: Props = $props();

	let isLoading = $state(isLoadingProp ?? true);
	let hasError = $state(hasErrorProp ?? false);
	let errorTitle = $state(errorTitleProp ?? '');
	let errorDescription = $state(errorDescriptionProp ?? '');
	let skeletonCount = $state(skeletonCountProp ?? 1);

	let showSkeleton = $derived(isLoading && !hasError);
	let showError = $derived(hasError && !isLoading);

	// Set default error messages if not provided
	$effect(() => {
		if (hasError && !errorTitle) {
			errorTitle = $t.messages.error.networkError;
		}
		if (hasError && !errorDescription) {
			errorDescription = $t.messages.error.serverError;
		}
	});
</script>

{#if showSkeleton}
	{#if skeletonComponent}
		{#each Array.from({ length: skeletonCount }) as _, i}
			<skeletonComponent {...skeletonProps} class={cn(className, i > 0 ? 'mt-4' : '')}
			></skeletonComponent>
		{/each}
	{:else}
		<div class={cn('space-y-4', className)}>
			{#each Array.from({ length: skeletonCount }) as _, i}
				<Skeleton class="h-16 w-full" />
			{/each}
		</div>
	{/if}
{:else if showError}
	<div
		class={cn(
			'flex flex-col items-center justify-center rounded-lg border p-8 text-center',
			className
		)}
	>
		<div class="mb-4 rounded-full bg-destructive/10 p-3">
			<svg
				class="h-8 w-8 text-destructive"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				></path>
			</svg>
		</div>
		<h3 class="mb-2 text-lg font-semibold">{errorTitle}</h3>
		<p class="mb-4 text-sm text-muted-foreground">{errorDescription}</p>
		{#if onRetry}
			<Button onclick={onRetry} variant="default">
				{$t.ui.common.retry || 'Retry'}
			</Button>
		{/if}
	</div>
{/if}
