/**
 * Centralized fetch wrapper with automatic auth checking and redirection
 */
import { goto } from '$app/navigation';
import { authStore } from '$lib/stores/auth';
import { buildApiUrl, getUrlPrefix } from './api';

// Global flag to prevent multiple simultaneous redirects
let isRedirecting = false;

/**
 * Custom fetch wrapper that automatically handles auth errors
 * and redirects to login page when session expires
 */
export async function authFetch(
	input: string | URL | Request,
	init?: RequestInit
): Promise<Response> {
	// Determine the URL
	let url: string;
	if (input instanceof Request) {
		url = input.url;
	} else if (input instanceof URL) {
		url = input.toString();
	} else {
		url = input;
	}

	// Make the fetch request
	const response = await fetch(url, init);

	// Check for authentication errors (401 Unauthorized)
	if (response.status === 401) {
		// Session expired or not authenticated
		const urlPrefix = getUrlPrefix();

		// Clear the expired session cookie to prevent repeated 401s
		if (typeof document !== 'undefined') {
			// Delete the session_id cookie by setting it to expire in the past
			// Try multiple path combinations to ensure cookie is cleared
			const paths = ['/', urlPrefix || '/'];
			for (const path of paths) {
				document.cookie = `session_id=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=${path}; SameSite=Strict`;
			}
		}

		// Force reload auth state to clear stale authentication data
		authStore.loadConfig(true);

		const authPath = urlPrefix ? `${urlPrefix}/auth` : '/auth';

		// Only redirect if we're not already on the auth or setup page
		// This prevents redirect loops when auth endpoints fail
		if (typeof window !== 'undefined') {
			const currentPath = window.location.pathname;
			const normalizedPath = urlPrefix ? currentPath.replace(urlPrefix, '') : currentPath;

			if (!normalizedPath.startsWith('/auth') && !normalizedPath.startsWith('/setup')) {
				// Redirect to auth page (only once per 401)
				if (!isRedirecting) {
					isRedirecting = true;
					console.log('[AuthFetch] Session expired, redirecting to auth page');
					goto(authPath, { replaceState: true });
					// Reset flag after a short delay to allow the redirect to complete
					setTimeout(() => {
						isRedirecting = false;
					}, 1000);
				} else {
					console.log('[AuthFetch] Redirect already in progress, ignoring');
				}
			} else {
				// Already on auth/setup page, just log
				console.log('[AuthFetch] Got 401 while on auth/setup page, ignoring redirect');
			}
		}

		// Return the response anyway (caller might want to handle it)
		return response;
	}

	return response;
}

/**
 * Helper to build API URL and use authFetch in one call
 */
export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
	const url = buildApiUrl(path);
	return authFetch(url, {
		...init,
		credentials: init?.credentials || 'include',
		headers: {
			Accept: 'application/json',
			...init?.headers
		}
	});
}
