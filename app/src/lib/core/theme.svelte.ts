export const THEMES = ['gruvbox','meridian', 'amber', 'paper', 'crt'] as const;
export type Theme = (typeof THEMES)[number];

const STORAGE_KEY = 'osdtype.theme';

function isTheme(value: string | null): value is Theme {
	return value !== null && (THEMES as readonly string[]).includes(value);
}

export const themeState = $state<{ current: Theme }>({ current: 'gruvbox' });

export function applyTheme(theme: Theme): void {
	themeState.current = theme;
	if (typeof document !== 'undefined') {
		document.documentElement.dataset.theme = theme;
	}
	try {
		localStorage.setItem(STORAGE_KEY, theme);
	} catch {
		// Storage can be unavailable (private mode, SSR); the theme still applies.
	}
}

export function initTheme(): void {
	let stored: string | null = null;
	try {
		stored = localStorage.getItem(STORAGE_KEY);
	} catch {
		stored = null;
	}
	applyTheme(isTheme(stored) ? stored : 'gruvbox');
}

export function cycleTheme(): void {
	const i = THEMES.indexOf(themeState.current);
	applyTheme(THEMES[(i + 1) % THEMES.length]);
}
