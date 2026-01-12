import { writable } from 'svelte/store';

export const showAddForm = writable(false);
export const isLoadingAddForm = writable(false);

/**
 * Creates a debounced loading store that ensures skeleton loaders
 * display for a minimum duration to prevent blinking on fast loads.
 *
 * @param minDisplayMs - Minimum time in milliseconds to show loading state (default: 350ms)
 * @returns Store with setLoading method to control loading state
 */
export function createDebouncedLoadingStore(minDisplayMs = 350) {
	let minDisplayStartTime: number | null = null;
	const { subscribe, set } = writable(true);

	return {
		subscribe,
		setLoading: async (isLoading: boolean) => {
			if (isLoading) {
				// Start loading immediately
				minDisplayStartTime = Date.now();
				set(true);
			} else if (minDisplayStartTime) {
				// Ensure minimum display time before hiding
				const elapsed = Date.now() - minDisplayStartTime;
				const remaining = Math.max(0, minDisplayMs - elapsed);

				if (remaining > 0) {
					await new Promise((resolve) => setTimeout(resolve, remaining));
				}
				minDisplayStartTime = null;
				set(false);
			} else {
				// No start time tracked, just update
				set(false);
			}
		}
	};
}
