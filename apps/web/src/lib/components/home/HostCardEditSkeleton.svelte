<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
</script>

<Card.Content>
	<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
		<!-- Name field -->
		<div class="flex flex-col gap-2">
			<Skeleton class="h-5 w-20" />
			<Skeleton class="h-10 w-full" />
		</div>

		<!-- MAC Address field -->
		<div class="flex flex-col gap-2">
			<Skeleton class="h-5 w-28" />
			<div class="flex items-center gap-1">
				<Skeleton class="h-10 flex-1" />
				<Skeleton class="h-10 w-10 shrink-0" />
			</div>
		</div>

		<!-- Broadcast Address field -->
		<div class="flex flex-col gap-2 md:col-span-2">
			<Skeleton class="h-5 w-36" />
			<div class="flex items-center gap-1">
				<Skeleton class="h-10 flex-1" />
				<Skeleton class="h-10 w-10 shrink-0" />
			</div>
		</div>
	</div>
</Card.Content>

<!-- Footer buttons -->
<Card.Footer class="flex justify-end gap-2">
	<Skeleton class="h-10 w-20" />
	<Skeleton class="h-10 w-16" />
</Card.Footer>
