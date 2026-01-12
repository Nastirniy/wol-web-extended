import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'amoled';

function createThemeStore() {
	// Get initial theme
	let initialTheme: Theme = 'light';

	if (browser) {
		const stored = localStorage.getItem('theme') as Theme | null;
		if (stored === 'light' || stored === 'dark' || stored === 'amoled') {
			initialTheme = stored;
		} else {
			// Check system preference
			initialTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
		}
	}

	const { subscribe, set, update } = writable<Theme>(initialTheme);

	function applyTheme(theme: Theme) {
		if (browser) {
			const root = document.documentElement;
			root.classList.remove('light', 'dark', 'amoled');

			if (theme === 'light') {
				root.classList.remove('dark');
			} else {
				root.classList.add('dark');
				if (theme === 'amoled') {
					root.classList.add('amoled');
				}
			}
			localStorage.setItem('theme', theme);
			document.cookie = `theme=${theme}; path=/; max-age=31536000; SameSite=Strict`;
		}
	}

	// Apply initial theme
	if (browser) {
		applyTheme(initialTheme);
	}

	return {
		subscribe,
		toggle: () => {
			update((current) => {
				let newTheme: Theme = 'light';
				if (current === 'light') newTheme = 'dark';
				else if (current === 'dark') newTheme = 'amoled';
				else newTheme = 'light';

				applyTheme(newTheme);
				return newTheme;
			});
		},
		set: (theme: Theme) => {
			applyTheme(theme);
			set(theme);
		}
	};
}

export const themeStore = createThemeStore();
