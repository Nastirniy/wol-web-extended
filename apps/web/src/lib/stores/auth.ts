import { derived, writable } from 'svelte/store';
import { browser } from '$app/environment';
import type { AppConfig, AuthStatus } from '$lib/types/api';
import { buildApiUrl } from '$lib/utils/api';

// Default config to avoid null checks
const defaultConfig: AppConfig = {
	os: 'unknown',
	url_prefix: '',
	use_auth: true,
	readonly_mode: false
};

export function createAuthStore() {
	const config = writable<AppConfig>(defaultConfig);
	const isLoading = writable(true);
	const serverUnreachable = writable(false); // Track if server is unreachable
	let configLoadPromise: Promise<void> | null = null; // Track loading promise to prevent duplicates
	let isLoadingConfig = false; // Mutex to prevent race conditions

	// Derived stores for easy access - no null checks needed
	const useAuth = derived(config, ($config) => $config.use_auth);

	const isAuthenticated = derived(config, ($config) => {
		if (!browser) return false; // On server, assume not authenticated
		if (!$config.use_auth) return true; // No auth required
		return $config.user != null; // Check if user info is present
	});
	const currentUser = derived(config, ($config) => $config.user);

	// Detect readonly mode by checking global readonly mode or user's readonly field
	const isReadOnly = derived(config, ($config) => {
		// Global readonly mode affects everyone
		if ($config.readonly_mode) return true;

		if (!$config.use_auth) return false; // No auth = not readonly (unless global readonly)
		if (!$config.user) return false; // Not authenticated = not readonly (should redirect to login instead)
		// Use the actual readonly field from the user object
		return $config.user?.readonly === true;
	});

	// Show sensitive data only when NOT in readonly mode
	const showSensitiveData = derived([config, isReadOnly], ([$config, $isReadOnly]) => {
		// If readonly mode is enabled (globally or per-user), hide sensitive data
		if ($isReadOnly) return false;

		if (!$config.use_auth) return true; // No auth and not readonly = show all
		if (!$config.user) return false; // Not authenticated = hide
		return true; // Authenticated and not readonly = show
	});

	async function loadConfig(force = false) {
		// If already loading and not forcing, wait for existing load to complete
		if (isLoadingConfig && !force) {
			while (isLoadingConfig) {
				await new Promise((resolve) => setTimeout(resolve, 50));
			}
			return;
		}

		// If we have a pending promise and not forcing, return it
		if (configLoadPromise && !force) {
			return configLoadPromise;
		}

		// If forcing, wait for any existing load to complete first
		if (force && isLoadingConfig) {
			while (isLoadingConfig) {
				await new Promise((resolve) => setTimeout(resolve, 50));
			}
		}

		if (!browser) {
			// On server, use default config
			config.set(defaultConfig);
			isLoading.set(false);
			return;
		}

		// Set loading flag
		isLoadingConfig = true;

		// Create and store the loading promise
		configLoadPromise = (async () => {
			try {
				isLoading.set(true);
				serverUnreachable.set(false); // Reset server unreachable state

				// First, check authentication status
				// Don't use authFetch for initial load to avoid redirect during startup
				const authResponse = await fetch(buildApiUrl('/api/auth/me'), {
					method: 'GET',
					headers: {
						Accept: 'application/json'
					},
					credentials: 'include'
				});

				let authData: AuthStatus | null = null;
				if (authResponse.ok) {
					try {
						authData = await authResponse.json();
					} catch (e) {
						console.error('Failed to parse auth response as JSON:', e);
					}
				}

				// Then, get configuration
				// Don't use authFetch for initial load to avoid redirect during startup
				const configResponse = await fetch(buildApiUrl('/api/config'), {
					method: 'GET',
					headers: {
						Accept: 'application/json'
					},
					credentials: 'include'
				});

				let configData: AppConfig;

				if (configResponse.ok) {
					try {
						configData = await configResponse.json();
					} catch (e) {
						console.error('Failed to parse config response as JSON:', e);
						// If parsing fails but we have auth data, use minimal config
						configData = {
							os: 'unknown',
							url_prefix: '',
							use_auth: authData?.auth_enabled ?? true
						};
					}
				} else {
					// If config fails but we have auth data, use minimal config
					configData = {
						os: 'unknown',
						url_prefix: '',
						use_auth: authData?.auth_enabled ?? true
					};
				}

				// Merge auth data into config
				if (authData && authData.authenticated) {
					configData.user = authData.user;
				}

				config.set(configData);
				serverUnreachable.set(false); // Server is reachable
			} catch (error) {
				console.error('Failed to load config:', error);
				// Network error - server is unreachable
				serverUnreachable.set(true);
				// Set default config if loading fails
				config.set(defaultConfig);
			} finally {
				isLoading.set(false);
				isLoadingConfig = false; // Clear loading flag
				configLoadPromise = null; // Clear promise after completion
			}
		})();

		return configLoadPromise;
	}

	async function logout() {
		if (!browser) return;

		try {
			// Don't use authFetch for logout to avoid redirect loop
			await fetch(buildApiUrl('/api/auth/logout'), {
				method: 'POST',
				headers: {
					Accept: 'application/json'
				},
				credentials: 'include'
			});
		} catch (error) {
			console.error('Logout request failed:', error);
		}

		// Clear config - reset to default
		config.set(defaultConfig);
		configLoadPromise = null;
		isLoadingConfig = false;
	}

	async function login(
		username: string,
		password: string
	): Promise<{ success: boolean; error?: string }> {
		if (!browser) return { success: false, error: 'Not in browser context' };

		try {
			// Don't use authFetch for login to avoid redirect loop
			const response = await fetch(buildApiUrl('/api/auth/login'), {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Accept: 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify({ username, password })
			});

			if (response.ok) {
				await loadConfig(true); // Force reload config to get user info
				return { success: true };
			} else {
				// Log server error for debugging but don't expose to user
				const errorData = await response.text();
				console.error('Login failed:', errorData);

				// Return generic user-friendly message
				return { success: false, error: 'Invalid username or password' };
			}
		} catch (error) {
			console.error('Login request failed:', error);
			return { success: false, error: 'Unable to connect to server. Please try again.' };
		}
	}

	// Create a combined store that exposes all properties
	const combined = derived(
		[
			config,
			isLoading,
			useAuth,
			isAuthenticated,
			currentUser,
			isReadOnly,
			showSensitiveData,
			serverUnreachable
		],
		([
			$config,
			$isLoading,
			$useAuth,
			$isAuthenticated,
			$currentUser,
			$isReadOnly,
			$showSensitiveData,
			$serverUnreachable
		]) => ({
			config: $config,
			isLoading: $isLoading,
			useAuth: $useAuth,
			isAuthenticated: $isAuthenticated,
			currentUser: $currentUser,
			isReadOnly: $isReadOnly,
			showSensitiveData: $showSensitiveData,
			serverUnreachable: $serverUnreachable
		})
	);

	return {
		subscribe: combined.subscribe,
		loadConfig,
		logout,
		login
	};
}

export const authStore = createAuthStore();
