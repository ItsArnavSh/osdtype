import { CGrammar } from '$lib/templates/c';
import { generate } from '../../rust-core/pkg/rust_core.js';

export enum Languages {
	TYPESCRIPT,
	CPP,
	GOLANG
}
export function GenerateCode(lang: Languages, seed: number, tokens: number): string[] {
	let result: string[] = [];
	try {
		switch (lang) {
			case Languages.CPP:
				result = generate(CGrammar, seed, tokens);
				break;
		}
	} catch (err) {
		console.log('Wasm Failed: ', err);
	}
	return result;
}
