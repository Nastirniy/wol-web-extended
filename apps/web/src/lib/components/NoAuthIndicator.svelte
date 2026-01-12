<script>
	import { Eye, Shield } from 'lucide-svelte';
	import { page } from '$app/state';
	import { authStore } from '$lib/stores/auth';
	import { t } from '$lib/stores/locale';
</script>

{#if !$authStore.useAuth}
	<div
		class="flex items-center gap-2 rounded-md border border-orange-200 bg-orange-50 px-3 py-1.5 text-sm text-orange-700 dark:border-orange-800 dark:bg-orange-950 dark:text-orange-300"
	>
		<Shield class="h-4 w-4" />
		<span class="font-medium">{$t.ui.indicator.noAuth}</span>
	</div>
{:else if $authStore.isReadOnly && page.url.pathname !== '/auth'}
	<div
		class="flex items-center gap-2 rounded-md border border-blue-200 bg-blue-50 px-3 py-1.5 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300"
	>
		<Eye class="h-4 w-4" />
		<span class="font-medium">{$t.ui.indicator.readOnly}</span>
	</div>
{/if}
