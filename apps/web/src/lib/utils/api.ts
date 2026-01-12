/**
 * Build API URL with proper prefix handling
 * The URL prefix is automatically detected from the HTML <base> tag
 *
 * Examples:
 * - <base href="/"> -> prefix: ""
 * - <base href="/myapp/"> -> prefix: "/myapp"
 */

let detectedUrlPrefix: string | null = null;

/**
 * Detect URL prefix from the HTML base tag
 * Falls back to pathname detection if no base tag exists
 */
function detectUrlPrefix(): string {
	if (typeof window === 'undefined') {
		return '';
	}

	// First, try to get prefix from <base> tag (injected by backend)
	const baseElement = document.querySelector('base');
	if (baseElement) {
		const baseHref = baseElement.getAttribute('href') || '/';
		// Remove trailing slash and return
		let prefix = baseHref.replace(/\/$/, '');
		// If prefix is just empty or "/", return empty string
		return prefix === '' || prefix === '/' ? '' : prefix;
	}

	// Fallback: detect from pathname (for development without backend)
	const pathname = window.location.pathname;

	// If we're at root or just /auth, no prefix
	if (pathname === '/' || pathname === '/auth' || pathname === '/home') {
		return '';
	}

	// Extract prefix by taking the first segment
	const segments = pathname.split('/').filter((s) => s.length > 0);

	if (segments.length === 0) {
		return '';
	}

	// If first segment is a known route (home, auth, users, setup), no prefix
	const knownRoutes = ['home', 'auth', 'users', 'setup', 'api'];
	if (knownRoutes.includes(segments[0])) {
		return '';
	}

	// Otherwise, first segment is the prefix
	return '/' + segments[0];
}

/**
 * Build API URL with automatically detected prefix
 */
export function buildApiUrl(path: string): string {
	// Detect prefix on first call
	if (detectedUrlPrefix === null) {
		detectedUrlPrefix = detectUrlPrefix();
	}

	// Ensure path starts with /
	const normalizedPath = path.startsWith('/') ? path : '/' + path;

	// Return prefixed URL
	return detectedUrlPrefix + normalizedPath;
}

/**
 * Get current URL prefix (for testing/debugging)
 */
export function getUrlPrefix(): string {
	if (detectedUrlPrefix === null) {
		detectedUrlPrefix = detectUrlPrefix();
	}
	return detectedUrlPrefix;
}
