import { goto as svelteGoto } from '$app/navigation';
import { getUrlPrefix } from './api';

/**
 * Navigate to a path with URL prefix support
 * This wraps SvelteKit's goto() to automatically prepend the URL prefix
 *
 * @param path - The path to navigate to (e.g., "/home", "/auth")
 * @param opts - SvelteKit navigation options
 */
export function goto(
	path: string,
	opts?: {
		replaceState?: boolean;
		noScroll?: boolean;
		keepFocus?: boolean;
		invalidateAll?: boolean;
		state?: any;
	}
) {
	const prefix = getUrlPrefix();
	const fullPath = prefix + path;
	return svelteGoto(fullPath, opts);
}
