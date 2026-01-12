import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import type { User } from '$lib/types/api';
import { HandledError } from '$lib/utils/HandledError';
import { buildApiUrl } from '$lib/utils/api';
import { handleGenericError, handleMutationError } from '$lib/utils/errors';
import { authFetch } from '$lib/utils/fetch';

export function createUsersStore() {
	const users = writable<User[]>([]);
	const isLoading = writable<boolean>(true); // Start with true so skeletons show immediately
	const hasError = writable<boolean>(false);

	async function fetchUsers() {
		if (!browser) return;

		isLoading.set(true);
		hasError.set(false); // Reset error state on new fetch

		try {
			const response = await authFetch(buildApiUrl('/api/users'), {
				method: 'GET',
				headers: {
					Accept: 'application/json'
				},
				credentials: 'include'
			});

			if (response.ok) {
				const data: User[] = await response.json();
				users.set(data);
				hasError.set(false);
			} else {
				console.error('Failed to fetch users:', response.status);
				users.set([]);
				hasError.set(true);
			}
		} catch (error) {
			console.error('Error fetching users:', error);
			users.set([]);
			hasError.set(true);
		} finally {
			isLoading.set(false);
		}
	}

	async function createUser(userData: {
		username: string;
		password?: string;
		readonly?: boolean;
		is_superuser?: boolean;
	}) {
		if (!browser) return;

		try {
			const response = await authFetch(buildApiUrl('/api/users'), {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Accept: 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify(userData)
			});

			if (response.ok) {
				const result = await response.json();
				await fetchUsers();
				return result;
			} else {
				const errorText = await response.text();
				handleMutationError(response, errorText, 'create', 'user');
				throw new HandledError();
			}
		} catch (error: unknown) {
			if (error instanceof HandledError) {
				throw error;
			}
			console.error('Error creating user:', error);
			throw error;
		}
	}

	async function updateUser(
		userId: string,
		userData: { name?: string; password?: string; readonly?: boolean; is_superuser?: boolean }
	) {
		if (!browser) return;

		try {
			const response = await authFetch(buildApiUrl(`/api/users/${userId}`), {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Accept: 'application/json'
				},
				credentials: 'include',
				body: JSON.stringify(userData)
			});

			if (response.ok) {
				await fetchUsers();
			} else {
				const errorText = await response.text();
				handleMutationError(response, errorText, 'update', 'user');
				throw new HandledError();
			}
		} catch (error: unknown) {
			if (error instanceof HandledError) {
				throw error;
			}
			console.error('Error updating user:', error);
			const errorText = error instanceof Error ? error.message : 'Unknown error';
			handleMutationError({ status: 500 } as Response, errorText, 'update', 'user');
			throw error;
		}
	}

	async function deleteUser(userId: string) {
		if (!browser) return;

		try {
			const response = await authFetch(buildApiUrl(`/api/users/${userId}`), {
				method: 'DELETE',
				headers: {
					Accept: 'application/json'
				},
				credentials: 'include'
			});

			if (response.ok) {
				await fetchUsers();
			} else {
				const errorText = await response.text();
				handleMutationError(response, errorText, 'delete', 'user');
				throw new HandledError();
			}
		} catch (error: unknown) {
			if (error instanceof HandledError) {
				throw error;
			}
			console.error('Error deleting user:', error);
			const errorText = error instanceof Error ? error.message : 'Network error';
			handleGenericError('delete user', errorText);
			throw error;
		}
	}

	return {
		subscribe: users.subscribe,
		isLoading: { subscribe: isLoading.subscribe },
		hasError: { subscribe: hasError.subscribe },
		fetchUsers,
		createUser,
		updateUser,
		deleteUser
	};
}

export const usersStore = createUsersStore();
