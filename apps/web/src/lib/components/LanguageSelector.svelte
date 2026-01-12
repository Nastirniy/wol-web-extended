<script lang="ts">
	import { Languages } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Popover from '$lib/components/ui/popover';
	import { AVAILABLE_LANGUAGES, type LanguageCode, locale, t } from '$lib/stores/locale';

	function selectLanguage(code: LanguageCode) {
		locale.setLanguage(code);
	}

	let currentLang = $derived($locale);
	let currentLangName = $derived(
		AVAILABLE_LANGUAGES.find((lang) => lang.code === currentLang)?.nativeName || 'English'
	);
</script>

<Popover.Root>
	<Popover.Trigger>
		<Button size="icon" variant="outline" title={$t.ui.footer.language} class="h-8 w-8">
			<Languages class="h-4 w-4" />
		</Button>
	</Popover.Trigger>
	<Popover.Content class="w-48 p-2">
		<div class="mb-2 text-sm font-semibold">{$t.ui.footer.language}</div>
		<div class="space-y-1">
			{#each AVAILABLE_LANGUAGES as lang}
				<button
					onclick={() => selectLanguage(lang.code)}
					class="flex w-full items-center justify-between rounded px-3 py-2 text-sm transition-colors hover:bg-accent {currentLang ===
					lang.code
						? 'bg-accent font-medium'
						: ''}"
				>
					<span>{lang.nativeName}</span>
					{#if currentLang === lang.code}
						<svg
							class="h-4 w-4"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							xmlns="http://www.w3.org/2000/svg"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M5 13l4 4L19 7"
							></path>
						</svg>
					{/if}
				</button>
			{/each}
		</div>
	</Popover.Content>
</Popover.Root>
