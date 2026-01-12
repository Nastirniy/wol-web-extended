/**
 * Normalizes MAC address to lowercase with colon separators
 * @param mac MAC address in any format (with colons, hyphens, or no separators)
 * @returns Normalized MAC address in lowercase with colons (e.g., "aa:bb:cc:dd:ee:ff")
 */
export function normalizeMACAddress(mac: string): string {
	// Remove all separators (colons, hyphens, spaces)
	const cleanMac = mac.replace(/[:\-\s]/g, '').toLowerCase();

	// Validate length
	if (cleanMac.length !== 12) {
		return mac; // Return as-is if invalid length
	}

	// Validate hex characters
	if (!/^[0-9a-f]{12}$/i.test(cleanMac)) {
		return mac; // Return as-is if invalid characters
	}

	// Add colons every 2 characters
	const result = [];
	for (let i = 0; i < 6; i++) {
		result.push(cleanMac.substring(i * 2, i * 2 + 2));
	}

	return result.join(':');
}

/**
 * Validates if a string is a valid MAC address
 * @param mac MAC address to validate
 * @returns true if valid, false otherwise
 */
export function isValidMACAddress(mac: string): boolean {
	const normalized = normalizeMACAddress(mac);
	return /^[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}$/i.test(
		normalized
	);
}
