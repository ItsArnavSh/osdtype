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
 * from a 64-bit seed. Returns an empty string if the language is unknown.
 */
export function generate(lang: string, seed: bigint, tokens: number): string {
	return wasmGenerate(lang, seed, tokens);
}

const U64_MASK = (1n << 64n) - 1n;

/** A fresh 64-bit seed from the browser CSPRNG. */
export function randomSeed(): bigint {
	const words = new Uint32Array(2);
	crypto.getRandomValues(words);
	const seed = (BigInt(words[0]) << 32n) | BigInt(words[1]);
	// resrap treats 0 as "seed yourself", so avoid handing it zero.
	return seed === 0n ? 0x9e3779b97f4a7c15n : seed;
}

/**
 * Derives another well-mixed 64-bit seed from `seed` and a salt. Used for
 * every random choice inside a run so the whole snippet is reproducible from
 * its seed alone.
 */
export function deriveSeed(seed: bigint, salt: number | bigint): bigint {
	let z = (seed + BigInt(salt) * 0x9e3779b97f4a7c15n + 0x9e3779b97f4a7c15n) & U64_MASK;
	z = ((z ^ (z >> 30n)) * 0xbf58476d1ce4e5b9n) & U64_MASK;
	z = ((z ^ (z >> 27n)) * 0x94d049bb133111ebn) & U64_MASK;
	return (z ^ (z >> 31n)) & U64_MASK;
}

/** Canonical ids of every language bundled with the generator. */
export function bundledLanguages(): string[] {
	return wasmLanguages();
}
