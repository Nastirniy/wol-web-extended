<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
</script>

<Card.Root class="max-w-4xl">
	<Card.Content class="p-4">
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<!-- Name skeleton -->
			<Skeleton class="h-7 w-48 flex-shrink-0" />

			<div class="flex flex-col gap-2 sm:flex-1 sm:flex-row sm:items-center sm:justify-center">
				<div class="flex flex-wrap items-center gap-2">
					<!-- User type badge skeleton -->
					<Skeleton class="h-6 w-16" />
					<!-- Access type badge skeleton -->
					<Skeleton class="h-6 w-20" />
					<!-- Created date skeleton -->
					<Skeleton class="h-4 w-40" />
				</div>
			</div>

			<!-- Buttons skeleton -->
			<div class="flex flex-wrap gap-2 sm:flex-nowrap">
				<Skeleton class="h-9 w-20" />
				<Skeleton class="h-9 w-32" />
				<Skeleton class="h-9 w-20" />
			</div>
		</div>
	</Card.Content>
</Card.Root>
