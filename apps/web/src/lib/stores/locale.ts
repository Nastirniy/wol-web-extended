import { derived, get, writable } from 'svelte/store';
import { browser } from '$app/environment';
// Import all translation files
import en from '$lib/i18n/en.json';
import uk from '$lib/i18n/uk.json';

// Type for translation structure
export type TranslationKeys = typeof en;

// Available languages with display names
export const AVAILABLE_LANGUAGES = [
	{ code: 'en', name: 'English', nativeName: 'English' },
	{ code: 'uk', name: 'Ukrainian', nativeName: 'Українська' }
] as const;

export type LanguageCode = (typeof AVAILABLE_LANGUAGES)[number]['code'];

// Translation data map
const translations: Record<LanguageCode, TranslationKeys> = {
	en,
	uk
};

// Default language
const DEFAULT_LANGUAGE: LanguageCode = 'en';

// Storage key for localStorage
const STORAGE_KEY = 'wol-locale';

/**
 * Get initial language from localStorage or browser
 */
function getInitialLanguage(): LanguageCode {
	if (!browser) return DEFAULT_LANGUAGE;

	// Try localStorage first
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored && isValidLanguageCode(stored)) {
		return stored as LanguageCode;
	}

	// Try browser language
	const browserLang = navigator.language.toLowerCase();
	const langCode = browserLang.split('-')[0];

	if (isValidLanguageCode(langCode)) {
		return langCode as LanguageCode;
	}

	return DEFAULT_LANGUAGE;
}

/**
 * Check if a string is a valid language code
 */
function isValidLanguageCode(code: string): code is LanguageCode {
	return AVAILABLE_LANGUAGES.some((lang) => lang.code === code);
}

/**
 * Create the locale store
 */
function createLocaleStore() {
	const { subscribe, set, update } = writable<LanguageCode>(getInitialLanguage());

	return {
		subscribe,
		/**
		 * Set the current language
		 */
		setLanguage: (lang: LanguageCode) => {
			if (!isValidLanguageCode(lang)) {
				console.warn(`Invalid language code: ${lang}`);
				return;
			}

			set(lang);

			// Persist to localStorage
			if (browser) {
				localStorage.setItem(STORAGE_KEY, lang);
				// Update HTML lang attribute
				document.documentElement.lang = lang;
			}
		},
		/**
		 * Get the current language
		 */
		getCurrentLanguage: () => {
			return get({ subscribe });
		}
	};
}

// Create the locale store instance
export const locale = createLocaleStore();

/**
 * Derived store with current translation messages
 */
export const t = derived(locale, ($locale) => {
	const messages = translations[$locale] || translations[DEFAULT_LANGUAGE];

	/**
	 * Get a translation by key path (e.g., "ui.common.add")
	 */
	function get(key: string): string {
		const keys = key.split('.');
		let value: any = messages;

		for (const k of keys) {
			if (value && typeof value === 'object' && k in value) {
				value = value[k];
			} else {
				console.warn(`Translation key not found: ${key}`);
				return key;
			}
		}

		return typeof value === 'string' ? value : key;
	}

	/**
	 * Get a translation with variable interpolation
	 * Example: t.interpolate("messages.found", { count: 5 }) => "Found 5 device(s)"
	 */
	function interpolate(key: string, vars: Record<string, string | number>): string {
		let text = get(key);

		Object.entries(vars).forEach(([key, value]) => {
			text = text.replace(new RegExp(`\\{${key}\\}`, 'g'), String(value));
		});

		return text;
	}

	return {
		get,
		interpolate,
		// Direct access to translation structure for better type safety
		ui: (messages as any).ui,
		messages: (messages as any).messages
	};
});

/**
 * Helper to get language display name
 */
export function getLanguageName(code: LanguageCode): string {
	return AVAILABLE_LANGUAGES.find((lang) => lang.code === code)?.name || code;
}

/**
 * Helper to get language native name
 */
export function getLanguageNativeName(code: LanguageCode): string {
	return AVAILABLE_LANGUAGES.find((lang) => lang.code === code)?.nativeName || code;
}
