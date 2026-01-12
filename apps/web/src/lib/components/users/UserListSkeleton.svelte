<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { SkeletonWithStatus } from '$lib/components/ui/skeleton';
	import UserCardSkeleton from '$lib/components/users/UserCardSkeleton.svelte';
	import { t } from '$lib/stores/locale';
	import { cn } from '$lib/utils';

	type Props = {
		isLoading?: boolean;
		hasError?: boolean;
		onRetry?: () => void;
		skeletonCount?: number;
		className?: string;
	};

	let {
		isLoading: isLoadingProp,
		hasError: hasErrorProp,
		onRetry,
		skeletonCount: skeletonCountProp,
		className = ''
	}: Props = $props();

	let isLoading = $state(isLoadingProp ?? true);
	let hasError = $state(hasErrorProp ?? false);
	let skeletonCount = $state(skeletonCountProp ?? 3);
</script>

<SkeletonWithStatus
	{isLoading}
	{hasError}
	{onRetry}
	{skeletonCount}
	skeletonComponent={UserCardSkeleton}
	errorTitle={$t.messages.error.networkError}
	errorDescription={$t.messages.error.serverError}
	className={cn(className)}
/>
