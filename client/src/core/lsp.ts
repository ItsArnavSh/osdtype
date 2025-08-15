import { markdown, type LanguageType } from 'svelte-highlight/languages';
import typescript from 'svelte-highlight/languages/typescript';
export function langHighlighter(language: string): LanguageType<string> {
	switch (language) {
		case 'typescript':
			return typescript;
		default:
			return markdown;
	}
}
