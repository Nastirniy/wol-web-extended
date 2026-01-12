<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { cn } from '$lib/utils';

	let { class: className = '' }: { class?: string } = $props();
</script>

<Card.Root class={cn('', className)}>
	<Card.Content>
		<div class="grid grid-cols-2 gap-2">
			<!-- Name skeleton -->
			<div class="flex min-w-0 items-baseline gap-1">
				<Skeleton class="h-6 w-12" />
				<Skeleton class="h-6 flex-1" />
			</div>
			<!-- Status indicator skeleton -->
			<div class="flex items-center justify-end">
				<Skeleton class="h-3 w-3 rounded-full" />
			</div>
			<!-- Broadcast skeleton -->
			<Skeleton class="col-span-2 h-6 w-full" />
			<!-- MAC Address skeleton -->
			<Skeleton class="col-span-2 h-6 w-full" />
			<!-- Interface skeleton -->
			<Skeleton class="col-span-2 h-6 w-3/4" />
		</div>
	</Card.Content>
	<Card.Footer class="flex justify-between gap-2">
		<!-- Wake button skeleton -->
		<Skeleton class="h-9 flex-1" />
		<!-- Ping button skeleton -->
		<Skeleton class="h-9 flex-1" />
		<!-- Menu button skeleton -->
		<Skeleton class="h-9 w-9" />
	</Card.Footer>
</Card.Root>
