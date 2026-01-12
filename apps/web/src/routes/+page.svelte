<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$lib/utils/navigation';

	onMount(() => {
		goto('/home');
	});
</script>
