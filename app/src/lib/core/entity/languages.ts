import { CGrammar } from '$lib/templates/c';
import { TSGrammar } from '$lib/templates/cpp';
import { GoGrammar } from '$lib/templates/go';
import { JavaGrammar } from '$lib/templates/java';
import { RustGrammar } from '$lib/templates/rs';
import { CppGrammar } from '$lib/templates/ts';

export const LANGUAGES = ['C', 'Go', 'CPP', 'Java', 'Rust', 'TypeScript'] as const;
export type Language = (typeof LANGUAGES)[number];

export const TIMES = [30, 90, 300] as const;
export type Time = (typeof TIMES)[number];

export function GetGrammar(lang: Language): string {
	switch (lang) {
		case 'C':
			return CGrammar;
		case 'Go':
			return GoGrammar;
		case 'CPP':
			return CppGrammar;
		case 'Java':
			return JavaGrammar;
		case 'Rust':
			return RustGrammar;
		case 'TypeScript':
			return TSGrammar;

		default:
			return CGrammar;
	}
}
