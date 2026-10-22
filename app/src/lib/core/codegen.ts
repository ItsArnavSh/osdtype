import init, { generate as wasmGenerate, languages as wasmLanguages } from '../../rust-core/pkg/rust_core.js';

/**
 * Client-side code generation backed by the Rust/WASM core.
 *
 * The WASM module must be instantiated once before `generate` can be called.
 * `initCodegen()` is idempotent — calling it repeatedly returns the same
 * promise, so any page can safely `await` it during `onMount`.
 */
let initPromise: Promise<unknown> | null = null;

export function initCodegen(): Promise<unknown> {
	if (!initPromise) {
		initPromise = init();
	}
	return initPromise;
}

/**
 * Generates a snippet for a bundled language id (e.g. `"cpp"`, `"python"`)
 * from a seed. Returns an empty string if the language is unknown.
 */
export function generate(lang: string, seed: number, tokens: number): string {
	return wasmGenerate(lang, seed, tokens);
}

/** Canonical ids of every language bundled with the generator. */
export function bundledLanguages(): string[] {
	return wasmLanguages();
}
