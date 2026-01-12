/**
 * Custom error class to signal that an error has already been handled
 * (e.g., toast notifications have been shown to the user).
 *
 * Supports both single and multiple errors from API responses.
 *
 * Usage:
 *   // Single error already shown to user
 *   throw new HandledError();
 *
 *   // Multiple errors already shown to user
 *   throw new HandledError('Multiple validation errors', { errors: [...] });
 */
export class HandledError extends Error {
	public readonly errors?: Array<{ code?: string; message: string; field?: string }>;

	constructor(
		message?: string,
		details?: { errors?: Array<{ code?: string; message: string; field?: string }> }
	) {
		super(message || 'Error has already been handled');
		this.name = 'HandledError';
		this.errors = details?.errors;

		// Maintains proper stack trace for where our error was thrown (only available on V8)
		if (Error.captureStackTrace) {
			Error.captureStackTrace(this, HandledError);
		}
	}
}

/**
 * Type guard to check if an error is a HandledError
 */
export function isHandledError(error: unknown): error is HandledError {
	return error instanceof HandledError;
}
