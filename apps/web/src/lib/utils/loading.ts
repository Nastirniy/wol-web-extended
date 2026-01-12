/**
 * Delays showing a loading skeleton to prevent flicker on fast operations
 *
 * @param delayMs Time to wait before showing skeleton (default: 300ms)
 * @returns Object with timeout ID and cancel function
 */
export function delayedSkeleton(delayMs: number = 300): { timeout: number; cancel: () => void } {
	const timeout = window.setTimeout(() => {}, delayMs);

	return {
		timeout,
		cancel: () => clearTimeout(timeout)
	};
}
