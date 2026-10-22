/**
 * Languages bundled with the resrap-rs code generator.
 *
 * These ids are the canonical template names exposed by the Rust crate
 * (`resrap_rs::templates`), so they can be handed straight to the WASM
 * `generate` call. Keep this list in sync with `templates::names()`.
 */
export const LANGUAGES = [
	'c',
	'cpp',
	'go',
	'haskell',
	'html',
	'java',
	'js',
	'python',
	'rust',
	'sql'
] as const;

export type Language = (typeof LANGUAGES)[number];

/** Presentable names for the language ids. */
export const LANGUAGE_LABELS: Record<Language, string> = {
	c: 'C',
	cpp: 'C++',
	go: 'Go',
	haskell: 'Haskell',
	html: 'HTML',
	java: 'Java',
	js: 'JavaScript',
	python: 'Python',
	rust: 'Rust',
	sql: 'SQL'
};

export function languageLabel(lang: Language): string {
	return LANGUAGE_LABELS[lang] ?? lang;
}

export const TIMES = [30, 90, 300] as const;
export type Time = (typeof TIMES)[number];
