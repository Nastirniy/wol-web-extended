<script lang="ts">
	import { cn } from '$lib/utils.js';

	let { class: className, ...restProps }: { class?: string; [key: string]: any } = $props();
</script>

<div class={cn('shimmer-skeleton rounded-md', className)} {...restProps}></div>

<style>
	.shimmer-skeleton {
		background: linear-gradient(
			110deg,
			hsl(var(--muted)) 0%,
			hsl(var(--muted-foreground) / 0.12) 25%,
			hsl(var(--muted)) 50%
		);
		background-size: 200% 100%;
		animation: shimmer 2.5s infinite;
	}

	@keyframes shimmer {
		0% {
			background-position: -200% 0;
		}
		100% {
			background-position: 200% 0;
		}
	}
</style>
