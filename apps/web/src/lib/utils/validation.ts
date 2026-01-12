/**
 * Shared validation utilities for the application
 */

export const IP_REGEX =
	/^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;

export interface ValidationResult {
	valid: boolean;
	error?: string;
}

/**
 * Validates a broadcast address in IP:PORT format
 * @param broadcast - The broadcast address to validate (e.g., "255.255.255.255:9")
 * @returns Validation result with error message if invalid
 */
export function validateBroadcastAddress(broadcast: string): ValidationResult {
	const parts = broadcast.split(':');
	if (parts.length !== 2) {
		return {
			valid: false,
			error: 'Invalid format. Use IP:PORT format like 255.255.255.255:9'
		};
	}

	const [ip, portStr] = parts;
	const port = parseInt(portStr);

	if (!IP_REGEX.test(ip)) {
		return { valid: false, error: 'Invalid IP address format' };
	}

	if (isNaN(port) || port < 1 || port > 65535) {
		return { valid: false, error: 'Port must be between 1 and 65535' };
	}

	return { valid: true };
}

/**
 * Validates an IP address format
 * @param ip - The IP address to validate
 * @returns True if valid IP address format
 */
export function validateIP(ip: string): boolean {
	return IP_REGEX.test(ip);
}

/**
 * Validates an IPv4 address with strict octet checking (0-255)
 * @param ip - The IP address to validate
 * @param allowEmpty - Whether to allow empty strings (default: true)
 * @returns True if valid IPv4 address or empty (if allowed)
 */
export function validateIPv4(ip: string, allowEmpty: boolean = true): boolean {
	if (!ip) return allowEmpty;

	const ipv4Regex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
	const match = ip.match(ipv4Regex);
	if (!match) return false;

	// Check each octet is 0-255
	for (let i = 1; i <= 4; i++) {
		const num = parseInt(match[i], 10);
		if (num < 0 || num > 255) return false;
	}

	return true;
}
