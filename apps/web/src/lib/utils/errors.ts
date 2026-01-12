import { get } from 'svelte/store';
import { toast } from 'svoast';
import { t } from '$lib/stores/locale';

export interface ParsedError {
	code?: string;
	message: string;
	field?: string;
}

export interface ParsedErrorResponse {
	errors: ParsedError[];
	isBulk: boolean; // true if multiple errors, false if single error
}

/**
 * Parse error response to extract error code(s) and message(s)
 * Server can send single error or array of errors for bulk validation
 *
 * Single error format:
 * { "code": "ERR_001", "error": "message" }
 * OR
 * { "error": "message" } // without code
 *
 * Multiple errors format:
 * { "errors": [{ "code": "ERR_001", "message": "...", "field": "username" }, ...] }
 */
function parseErrorResponse(errorText: string): ParsedErrorResponse {
	try {
		const errorData = JSON.parse(errorText);

		// Check for multiple errors (bulk validation)
		if (errorData.errors && Array.isArray(errorData.errors)) {
			return {
				errors: errorData.errors.map((err: any) => ({
					code: err.code, // Optional - may be undefined
					message: err.message || err.error || 'Unknown error',
					field: err.field
				})),
				isBulk: true
			};
		}

		// Single error
		return {
			errors: [
				{
					code: errorData.code, // Optional - may be undefined
					message: errorData.error || errorData.message || 'Unknown error'
				}
			],
			isBulk: false
		};
	} catch {
		// Fallback for non-JSON responses (raw text errors)
		return {
			errors: [{ message: errorText || 'Unknown error' }],
			isBulk: false
		};
	}
}

/**
 * Get localized error message from error code
 * Returns the localized message if code exists and translation available, otherwise returns fallback
 */
function getLocalizedErrorMessage(code: string | undefined, fallback: string): string {
	if (!code) return fallback;

	const locale = get(t);
	const localizedMessage =
		locale.messages.error.codes[code as keyof typeof locale.messages.error.codes];

	return localizedMessage || fallback;
}

/**
 * Show an error toast with optional description
 */
function showError(title: string, description?: string) {
	if (description) {
		toast.error(`${title}\n${description}`, { closable: true });
	} else {
		toast.error(title, { closable: true });
	}
}

/**
 * Show multiple error toasts (for bulk validation)
 */
function showErrors(errors: ParsedError[], operation: string) {
	const locale = get(t);

	if (errors.length === 1) {
		// Single error - show as normal
		const error = errors[0];
		const message = getLocalizedErrorMessage(error.code, error.message);
		const title = error.field
			? `${locale.messages.error.failedToOperation.replace('{operation}', operation)} (${error.field})`
			: locale.messages.error.failedToOperation.replace('{operation}', operation);

		showError(title, message);
	} else {
		// Multiple errors - show first error with count
		const firstError = errors[0];
		const message = getLocalizedErrorMessage(firstError.code, firstError.message);
		const title = `${locale.messages.error.failedToOperation.replace('{operation}', operation)} (${errors.length} errors)`;

		showError(title, message);

		// Log remaining errors to console for debugging
		console.error(`Additional errors (${errors.length - 1}):`, errors.slice(1));
	}
}

/**
 * Handle 403 Forbidden errors (readonly mode)
 */
export function handleForbiddenError(operation: string) {
	const message = get(t).messages.error.readonlyMode.replace('{operation}', operation);
	showError(message);
}

/**
 * Handle 429 Rate Limit errors
 */
export function handleRateLimitError(errorText?: string) {
	if (errorText) {
		const errorData = parseErrorResponse(errorText);
		const firstError = errorData.errors[0];
		console.error(
			'Rate limit error:',
			firstError.message,
			firstError.code ? `[${firstError.code}]` : ''
		);
	}
	const locale = get(t);
	showError(locale.messages.error.rateLimitTitle, locale.messages.error.rateLimitMessage);
}

/**
 * Handle 404 Not Found errors
 */
export function handleNotFoundError(resource: string) {
	const message = get(t).messages.error.notFoundResource.replace('{resource}', resource);
	showError(message);
}

/**
 * Handle 401 Unauthorized errors
 */
export function handleUnauthorizedError() {
	const locale = get(t);
	showError(locale.messages.error.unauthorizedTitle, locale.messages.error.unauthorizedMessage);
}

/**
 * Handle generic API errors with custom operation message
 * Supports both single and multiple errors
 * Works with or without error codes
 */
export function handleGenericError(operation: string, errorText: string) {
	const errorData = parseErrorResponse(errorText);

	if (errorData.isBulk) {
		showErrors(errorData.errors, operation);
	} else {
		const firstError = errorData.errors[0];
		console.error(
			`Error during ${operation}:`,
			firstError.message,
			firstError.code ? `[${firstError.code}]` : '(no error code)'
		);
		const title = get(t).messages.error.failedToOperation.replace('{operation}', operation);
		const message = getLocalizedErrorMessage(firstError.code, firstError.message);
		showError(title, message !== firstError.message ? message : undefined);
	}
}

/**
 * Generic API error handler that determines the appropriate error handling based on status code
 */
export function handleAPIError(response: Response, errorText: string, operation: string) {
	switch (response.status) {
		case 401:
			handleUnauthorizedError();
			break;
		case 403:
			handleForbiddenError(operation);
			break;
		case 404:
			handleNotFoundError(operation);
			break;
		case 429:
			handleRateLimitError(errorText);
			break;
		default:
			handleGenericError(operation, errorText);
			break;
	}
}

/**
 * Handle Wake-on-LAN specific errors with detailed messages
 */
export function handleWakeError(response: Response, errorText: string) {
	if (response.status === 429) {
		handleRateLimitError(errorText);
	} else if (response.status === 403) {
		handleForbiddenError('send Wake-on-LAN packets');
	} else {
		const errorData = parseErrorResponse(errorText);
		const firstError = errorData.errors[0];
		console.error(
			'Wake error:',
			firstError.message,
			firstError.code ? `[${firstError.code}]` : '(no error code)'
		);
		showError(get(t).messages.host.wakeError);
	}
}

/**
 * Handle create/update/delete errors with validation message support
 * Supports both single and multiple errors (bulk validation)
 * Works with or without error codes
 */
export function handleMutationError(
	response: Response,
	errorText: string,
	operation: 'create' | 'update' | 'delete',
	resource: string
) {
	const errorData = parseErrorResponse(errorText);
	const firstError = errorData.errors[0];
	const operationName = `${operation} ${resource}`;

	console.error(
		`Mutation error (${operationName}):`,
		firstError.message,
		firstError.code ? `[${firstError.code}]` : '(no error code)'
	);

	// Log additional errors if bulk
	if (errorData.isBulk && errorData.errors.length > 1) {
		console.error(
			`Additional validation errors (${errorData.errors.length - 1}):`,
			errorData.errors.slice(1)
		);
	}

	const locale = get(t);

	if (response.status === 403) {
		// Forbidden - check for specific error code, otherwise assume readonly mode
		if (firstError.code) {
			const localizedMessage = getLocalizedErrorMessage(
				firstError.code,
				locale.messages.error.readonlyMode.replace('{operation}', operationName)
			);
			showError(localizedMessage);
		} else {
			handleForbiddenError(operationName);
		}
	} else if (response.status === 400) {
		// Validation error - show all errors or first error
		if (errorData.isBulk) {
			showErrors(errorData.errors, operationName);
		} else {
			// Try to get localized message, fallback to original message
			const localizedMessage = getLocalizedErrorMessage(firstError.code, firstError.message);
			const genericFallback = locale.messages.error.invalidResourceData.replace(
				'{resource}',
				resource
			);

			// If we have a meaningful message from server, use it; otherwise use generic
			showError(
				localizedMessage !== firstError.message ? localizedMessage : genericFallback,
				localizedMessage === firstError.message ? firstError.message : undefined
			);
		}
	} else if (response.status === 409) {
		// Conflict error (e.g., duplicate username)
		const localizedMessage = getLocalizedErrorMessage(firstError.code, firstError.message);
		const genericFallback = locale.messages.error.resourceAlreadyExists.replace(
			'{resource}',
			resource
		);

		showError(
			localizedMessage !== firstError.message ? localizedMessage : genericFallback,
			localizedMessage === firstError.message ? firstError.message : undefined
		);
	} else {
		handleGenericError(operationName, errorText);
	}
}
