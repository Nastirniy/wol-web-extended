<script lang="ts">
	import { onMount } from 'svelte';
	import { Toasts } from 'svoast';
	import LanguageSelector from '$lib/components/LanguageSelector.svelte';
	import Navbar from '$lib/components/Navbar.svelte';
	import ThemeSelector from '$lib/components/ThemeSelector.svelte';
	import { authStore } from '$lib/stores/auth';
	import { locale, t } from '$lib/stores/locale';
	import '../app.css';

	let { children } = $props();

	// Initialize locale and set HTML lang attribute
	$effect(() => {
		if (typeof document !== 'undefined') {
			document.documentElement.lang = $locale;
		}
	});

	// Load config on initial page load (public endpoints don't require auth)
	$effect(() => {
		authStore.loadConfig();
	});

	// Hide initial CSS loader once Svelte hydrates (with minimum display time)
	onMount(() => {
		const minDisplayTime = 50; // Minimum 50ms to prevent blink
		const loadStartTime = (window as any).__pageLoadStart || performance.now();

		// Calculate how long to wait
		const elapsedTime = performance.now() - loadStartTime;
		const remainingTime = Math.max(0, minDisplayTime - elapsedTime);

		// Hide loader after minimum display time
		setTimeout(() => {
			document.body.classList.add('app-loaded');
		}, remainingTime);
	});
</script>

<Toasts position="bottom-right" />
<Navbar />
{@render children?.()}
<footer class="fixed bottom-0 flex w-full items-center justify-between bg-background px-3 py-2">
	<span class="text-xs text-foreground/60"
		>{$t.ui.footer.modifiedBy}
		<a target="_blank" href="https://github.com/Nastirniy">Nastirniy</a></span
	>

	<div class="flex items-center gap-2">
		<LanguageSelector />
		<ThemeSelector />
	</div>
</footer>
