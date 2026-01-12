<script lang="ts">
	import { Skeleton } from '$lib/components/ui/skeleton';
</script>

<div class="mx-auto mb-6 w-full max-w-4xl space-y-4">
	<div>
		<Skeleton class="mb-2 h-7 w-40" />
		<Skeleton class="h-5 w-64" />
	</div>

	<div class="grid gap-4">
		<!-- Username field skeleton -->
		<div class="grid gap-2">
			<Skeleton class="h-5 w-20" />
			<Skeleton class="h-10 w-full" />
		</div>

		<!-- Password field skeleton -->
		<div class="grid gap-2">
			<Skeleton class="h-5 w-20" />
			<Skeleton class="h-10 w-full" />
		</div>

		<!-- Confirm Password field skeleton -->
		<div class="grid gap-2">
			<Skeleton class="h-5 w-32" />
			<Skeleton class="h-10 w-full" />
		</div>

		<!-- Checkbox skeleton -->
		<div class="flex items-center space-x-2">
			<Skeleton class="h-4 w-4 rounded" />
			<Skeleton class="h-5 w-80" />
		</div>
	</div>

	<!-- Submit button skeleton -->
	<Skeleton class="mt-4 h-10 w-full" />
</div>
