export const THEMES = ['carbon', 'mono', 'gruvbox', 'meridian', 'amber', 'paper', 'crt'] as const;
export type Theme = (typeof THEMES)[number];

export interface ThemePalette {
	bg: string;
	fbg: string;
	mer: string;
	silver: string;
	red: string;
	border: string;
	pending: string;
	space: string;
	wrongBg: string;
	syntaxKeyword: string;
	syntaxType: string;
	syntaxLiteral: string;
	syntaxComment: string;
	syntaxMeta: string;
	syntaxPunct: string;
}

/** Mirror of the CSS theme variables for export previews/captures. */
export const THEME_PALETTES: Record<Theme, ThemePalette> = {
	carbon: {
		bg: '#040806',
		fbg: '#0b0f11',
		mer: '#00a19b',
		silver: '#e8eceb',
		red: '#ff4d4d',
		border: 'rgba(232, 236, 235, 0.15)',
		pending: 'rgba(232, 236, 235, 0.56)',
		space: 'rgba(232, 236, 235, 0.18)',
		wrongBg: 'rgba(255, 77, 77, 0.15)',
		syntaxKeyword: '#00bfb3',
		syntaxType: '#e8eceb',
		syntaxLiteral: '#ff8a8a',
		syntaxComment: 'rgba(160, 174, 172, 0.55)',
		syntaxMeta: '#7ad7cf',
		syntaxPunct: 'rgba(232, 236, 235, 0.66)'
	},
	mono: {
		bg: '#070707',
		fbg: '#101010',
		mer: '#f2f2f2',
		silver: '#ededed',
		red: '#ff4d4d',
		border: 'rgba(255, 255, 255, 0.15)',
		pending: 'rgba(255, 255, 255, 0.55)',
		space: 'rgba(255, 255, 255, 0.17)',
		wrongBg: 'rgba(255, 77, 77, 0.14)',
		syntaxKeyword: '#ffffff',
		syntaxType: '#d6d6d6',
		syntaxLiteral: '#a8a8a8',
		syntaxComment: 'rgba(255, 255, 255, 0.36)',
		syntaxMeta: '#f0f0f0',
		syntaxPunct: 'rgba(255, 255, 255, 0.62)'
	},
	meridian: {
		bg: '#0b0c0e',
		fbg: '#16181c',
		mer: '#00a19c',
		silver: '#c0c0c0',
		red: '#d40000',
		border: 'rgba(192, 192, 192, 0.2)',
		pending: 'rgba(192, 192, 192, 0.64)',
		space: 'rgba(192, 192, 192, 0.22)',
		wrongBg: 'rgba(212, 0, 0, 0.16)',
		syntaxKeyword: '#57b0c9',
		syntaxType: '#7ec8a8',
		syntaxLiteral: '#d8a657',
		syntaxComment: 'rgba(192, 192, 192, 0.4)',
		syntaxMeta: '#b08fc7',
		syntaxPunct: 'rgba(192, 192, 192, 0.7)'
	},
	gruvbox: {
		bg: '#1d2021',
		fbg: '#282828',
		mer: '#b8bb26',
		silver: '#ebdbb2',
		red: '#fb4934',
		border: 'rgba(235, 219, 178, 0.2)',
		pending: 'rgba(235, 219, 178, 0.66)',
		space: 'rgba(235, 219, 178, 0.24)',
		wrongBg: 'rgba(251, 73, 52, 0.16)',
		syntaxKeyword: '#83a598',
		syntaxType: '#b8bb26',
		syntaxLiteral: '#d3869b',
		syntaxComment: 'rgba(146, 131, 116, 0.9)',
		syntaxMeta: '#fabd2f',
		syntaxPunct: 'rgba(235, 219, 178, 0.68)'
	},
	amber: {
		bg: '#120d06',
		fbg: '#1d1509',
		mer: '#ffb000',
		silver: '#e8d5a3',
		red: '#d1603d',
		border: 'rgba(232, 213, 163, 0.2)',
		pending: 'rgba(232, 213, 163, 0.68)',
		space: 'rgba(232, 213, 163, 0.26)',
		wrongBg: 'rgba(209, 96, 61, 0.2)',
		syntaxKeyword: '#d9a441',
		syntaxType: '#a8bf6f',
		syntaxLiteral: '#e08b5a',
		syntaxComment: 'rgba(232, 213, 163, 0.38)',
		syntaxMeta: '#c8a2d8',
		syntaxPunct: 'rgba(232, 213, 163, 0.68)'
	},
	paper: {
		bg: '#f3f0e8',
		fbg: '#fffdf7',
		mer: '#0f766e',
		silver: '#2b2924',
		red: '#b3261e',
		border: 'rgba(43, 41, 36, 0.22)',
		pending: 'rgba(43, 41, 36, 0.72)',
		space: 'rgba(43, 41, 36, 0.28)',
		wrongBg: 'rgba(179, 38, 30, 0.12)',
		syntaxKeyword: '#0b6b8f',
		syntaxType: '#0f766e',
		syntaxLiteral: '#b3261e',
		syntaxComment: 'rgba(43, 41, 36, 0.45)',
		syntaxMeta: '#7c3aed',
		syntaxPunct: 'rgba(43, 41, 36, 0.68)'
	},
	crt: {
		bg: '#040806',
		fbg: '#071009',
		mer: '#6bff9e',
		silver: '#b9ffcf',
		red: '#ff6b5e',
		border: 'rgba(107, 255, 158, 0.22)',
		pending: 'rgba(185, 255, 207, 0.56)',
		space: 'rgba(185, 255, 207, 0.2)',
		wrongBg: 'rgba(255, 107, 94, 0.18)',
		syntaxKeyword: '#62ffd6',
		syntaxType: '#b7ff6b',
		syntaxLiteral: '#ffe66b',
		syntaxComment: 'rgba(150, 255, 190, 0.42)',
		syntaxMeta: '#ff9be0',
		syntaxPunct: 'rgba(185, 255, 207, 0.68)'
	}
};

export function paletteStyle(palette: ThemePalette): string {
	return [
		`--bg:${palette.bg}`,
		`--fbg:${palette.fbg}`,
		`--mer:${palette.mer}`,
		`--silver:${palette.silver}`,
		`--red:${palette.red}`,
		`--border:${palette.border}`,
		`--pending:${palette.pending}`,
		`--space:${palette.space}`,
		`--wrong-bg:${palette.wrongBg}`,
		`--syntax-keyword:${palette.syntaxKeyword}`,
		`--syntax-type:${palette.syntaxType}`,
		`--syntax-literal:${palette.syntaxLiteral}`,
		`--syntax-comment:${palette.syntaxComment}`,
		`--syntax-meta:${palette.syntaxMeta}`,
		`--syntax-punct:${palette.syntaxPunct}`,
		`--completed:color-mix(in srgb, ${palette.silver} 27%, transparent)`
	].join(';');
}

const STORAGE_KEY = 'osdtype.theme';

function isTheme(value: string | null): value is Theme {
	return value !== null && (THEMES as readonly string[]).includes(value);
}

export const themeState = $state<{ current: Theme }>({ current: 'carbon' });

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
	applyTheme(isTheme(stored) ? stored : 'carbon');
}

export function cycleTheme(): void {
	const i = THEMES.indexOf(themeState.current);
	applyTheme(THEMES[(i + 1) % THEMES.length]);
}
