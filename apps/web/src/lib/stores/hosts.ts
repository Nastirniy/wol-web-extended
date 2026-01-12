import { get, writable } from 'svelte/store';
import { toast } from 'svoast';
import { browser } from '$app/environment';
import { t } from '$lib/stores/locale';
import type { AppConfig, BulkPingResult, Host, NetworkInterface, PingResult } from '$lib/types/api';
import { HandledError } from '$lib/utils/HandledError';
import { buildApiUrl } from '$lib/utils/api';
import {
	handleGenericError,
	handleMutationError,
	handleRateLimitError,
	handleWakeError
} from '$lib/utils/errors';
import { authFetch } from '$lib/utils/fetch';

// Re-export types for components that import from this module
export type { Host, NetworkInterface, AppConfig, PingResult, BulkPingResult };

export function createHostsStore() {
	const hosts = writable<Host[]>([]);
	const isLoading = writable<boolean>(true); // Start with true so skeletons show immediately
	const hasError = writable<boolean>(false);
	let configCache: AppConfig | null = null;
	let configPromise: Promise<AppConfig> | null = null;
	let loadingStartTime: number | null = null;
	let hasShownPingRateLimitToast = false; // Track if we've shown the ping rate limit toast

	async function fetchHosts() {
		if (!browser) return;

		isLoading.set(true);
		hasError.set(false); // Reset error state on new fetch
		loadingStartTime = performance.now();
		const minDisplayTime = 150; // Minimum 150ms to prevent blink

		try {
			const response = await authFetch(buildApiUrl('/api/hosts'), {
				method: 'GET',
				headers: {
					Accept: 'application/json'
				},
				credentials: 'include'
			});

			if (response.ok) {
				const data: Host[] = await response.json();
				hosts.set(data);
				hasError.set(false);
			} else if (response.status === 401) {
				// Authentication error - don't show network error, just clear hosts
				// The authFetch already handles redirect to login
				console.log('Authentication required, redirecting to login');
				hosts.set([]);
				hasError.set(false); // Don't show error UI for auth issues
			} else {
				// Real network/server error
				console.error('Failed to fetch hosts:', response.status);
				hosts.set([]);
				hasError.set(true);
			}
		} catch (error) {
			console.error('Error fetching hosts:', error);
			hosts.set([]);
			hasError.set(true);
		} finally {
			// Ensure skeleton displays for at least 150ms
			const elapsedTime = performance.now() - (loadingStartTime || 0);
			const remainingTime = Math.max(0, minDisplayTime - elapsedTime);

			setTimeout(() => {
				isLoading.set(false);
			}, remainingTime);
		}
	}

	async function createHost(host: {
		broadcast: string;
		mac: string;
		name: string;
		interface?: string;
	}) {
		if (!browser) return;

		try {
			const response = await authFetch(buildApiUrl('/api/hosts'), {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Accept: 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify(host)
			});

			if (response.ok) {
				const result = await response.json();
				await fetchHosts();
				return result;
			} else {
				const errorText = await response.text();
				handleMutationError(response, errorText, 'create', 'host');
				throw new HandledError();
			}
		} catch (error: unknown) {
			if (error instanceof HandledError) {
				throw error;
			}
			console.error('Error creating host:', error);
			throw error;
		}
	}

	async function updateHost(host: Host) {
		if (!browser) return;

		try {
			const response = await authFetch(buildApiUrl(`/api/hosts/${host.id}`), {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Accept: 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify(host)
			});

			if (response.ok) {
				await fetchHosts();
			} else {
				const errorText = await response.text();
				handleMutationError(response, errorText, 'update', 'host');
				throw new HandledError();
			}
		} catch (error: unknown) {
			if (error instanceof HandledError) {
				throw error;
			}
			console.error('Error updating host:', error);
			const errorText = error instanceof Error ? error.message : 'Unknown error';
			handleMutationError({ status: 500 } as Response, errorText, 'update', 'host');
			throw error;
		}
	}

	async function deleteHost(host: Host) {
		if (!browser) return;

		try {
			const response = await authFetch(buildApiUrl(`/api/hosts/${host.id}`), {
				method: 'DELETE',
				headers: {
					Accept: 'application/json'
				},
				credentials: 'include'
			});

			if (response.ok) {
				await fetchHosts();
			} else {
				const errorText = await response.text();
				handleMutationError(response, errorText, 'delete', 'host');
				throw new HandledError();
			}
		} catch (error: unknown) {
			if (error instanceof HandledError) {
				throw error;
			}
			console.error('Error deleting host:', error);
			const errorText = error instanceof Error ? error.message : 'Network error';
			handleGenericError('delete host', errorText);
			throw error;
		}
	}

	async function wakeHost(host: Host) {
		if (!browser) return;

		try {
			const response = await authFetch(buildApiUrl('/api/wake'), {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Accept: 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify({ id: host.id })
			});

			if (response.ok) {
				toast.success(get(t).messages.host.wakeSuccessShort, { closable: true });
			} else {
				const errorText = await response.text();
				handleWakeError(response, errorText);
			}
		} catch (error: any) {
			console.error('Error waking host:', error);
			toast.error(get(t).messages.host.wakeError, { closable: true });
		}
	}

	async function pingHost(host: Host): Promise<PingResult> {
		if (!browser) {
			return {
				ping_success: false,
				arp_success: false
			};
		}

		try {
			const response = await authFetch(buildApiUrl('/api/ping'), {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Accept: 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify({ id: host.id })
			});

			if (response.status === 429) {
				// Rate limit exceeded
				const errorText = await response.text();
				if (!hasShownPingRateLimitToast) {
					handleRateLimitError(errorText);
					hasShownPingRateLimitToast = true;
				}
				return {
					ping_success: false,
					arp_success: false,
					rate_limited: true
				};
			}

			if (!response.ok) {
				throw new Error(`HTTP ${response.status}`);
			}

			// Reset the rate limit flag only on successful response (200)
			hasShownPingRateLimitToast = false;
			const result = await response.json();
			return result;
		} catch (error) {
			console.error('Ping failed for host:', host.name, error);
			// Return server unreachable status if API fails (network error, server down, etc.)
			return {
				ping_success: false,
				arp_success: false,
				server_unreachable: true
			};
		}
	}

	async function getNetworkInterfaces(): Promise<NetworkInterface[]> {
		if (!browser) {
			throw new Error('Network interfaces not available on server side');
		}

		try {
			const response = await authFetch(buildApiUrl('/api/network-interfaces'), {
				method: 'GET',
				headers: {
					Accept: 'application/json'
				},
				credentials: 'include'
			});

			if (!response.ok) {
				if (response.status === 403) {
					// Readonly mode or no auth - return empty array
					return [];
				}
				throw new Error(`Network interfaces request failed: ${response.status}`);
			}

			const interfaces = await response.json();
			return interfaces;
		} catch (error) {
			console.error('Error fetching network interfaces:', error);
			throw error; // Re-throw error instead of swallowing it
		}
	}

	async function getConfig(forceRefresh: boolean = false): Promise<AppConfig> {
		if (!browser) {
			throw new Error('Config not available on server side');
		}

		// Clear cache if force refresh is requested
		if (forceRefresh) {
			configCache = null;
		}

		// Return cached config if available
		if (configCache) {
			return configCache;
		}

		// If already fetching, wait for existing promise
		if (configPromise) {
			return configPromise;
		}

		// Fetch config
		configPromise = (async () => {
			try {
				const response = await authFetch(buildApiUrl('/api/config'), {
					method: 'GET',
					headers: {
						Accept: 'application/json'
					},
					credentials: 'include'
				});

				if (!response.ok) {
					throw new Error(`Config request failed: ${response.status}`);
				}

				const config = await response.json();
				configCache = config; // Cache the result
				return config;
			} catch (error) {
				console.error('Error fetching config:', error);
				throw error;
			} finally {
				configPromise = null; // Clear promise after completion
			}
		})();

		return configPromise;
	}

	async function bulkPing(
		onProgress?: (result: BulkPingResult) => void
	): Promise<BulkPingResult[]> {
		if (!browser) {
			return [];
		}

		try {
			const response = await authFetch(buildApiUrl('/api/ping/bulk'), {
				method: 'POST',
				headers: {
					Accept: 'application/json'
				},
				credentials: 'include'
			});

			if (response.status === 429) {
				// Rate limit exceeded for bulk ping
				const errorText = await response.text();
				if (!hasShownPingRateLimitToast) {
					handleRateLimitError(errorText);
					hasShownPingRateLimitToast = true;
				}
				throw new Error(`Bulk ping rate limited: ${response.status}`);
			}

			if (!response.ok) {
				// For 401, authFetch wrapper already handled redirect - silently return empty
				if (response.status === 401) {
					console.log('Bulk ping: Authentication error handled by authFetch wrapper');
					return [];
				}
				// For other errors, throw
				throw new Error(`Bulk ping request failed: ${response.status}`);
			}

			// Reset the rate limit flag only on successful response (200)
			hasShownPingRateLimitToast = false;

			const results: any[] = [];
			const reader = response.body?.getReader();
			if (!reader) {
				throw new Error('Response body not available');
			}

			const decoder = new TextDecoder();
			let buffer = '';

			try {
				while (true) {
					const { done, value } = await reader.read();
					if (done) break;

					buffer += decoder.decode(value, { stream: true });

					// Parse complete JSON objects using a simpler approach
					// Match objects like {"host_id":"xxx","host_name":"xxx",...}
					const objectRegex = /\{[^{}]+\}/g;
					let match;
					let lastIndex = 0;

					while ((match = objectRegex.exec(buffer)) !== null) {
						try {
							const result = JSON.parse(match[0]);
							results.push(result);
							if (onProgress) {
								onProgress(result);
							}
							lastIndex = match.index + match[0].length;
						} catch (e) {
							console.warn('Failed to parse ping result:', match[0], e);
							// Continue processing other results
						}
					}

					// Keep only unprocessed part of buffer
					if (lastIndex > 0) {
						buffer = buffer.slice(lastIndex);
					}
				}
			} finally {
				reader.releaseLock();
			}

			return results;
		} catch (error) {
			console.error('Error during bulk ping:', error);
			// Return error status for all hosts when server is unreachable
			let currentHosts: Host[] = [];
			hosts.subscribe((value) => {
				currentHosts = value;
			})();

			const errorResults: BulkPingResult[] = currentHosts.map((host) => ({
				host_id: host.id,
				host_name: host.name,
				ping_success: false,
				arp_success: false,
				server_unreachable: true
			}));

			// Notify progress callback for each host
			if (onProgress) {
				errorResults.forEach((result) => onProgress(result));
			}

			return errorResults;
		}
	}

	return {
		subscribe: hosts.subscribe,
		isLoading: { subscribe: isLoading.subscribe },
		hasError: { subscribe: hasError.subscribe },
		fetchHosts,
		createHost,
		updateHost,
		deleteHost,
		wakeHost,
		pingHost,
		getConfig,
		getNetworkInterfaces,
		bulkPing
	};
}

export const hostsStore = createHostsStore();
