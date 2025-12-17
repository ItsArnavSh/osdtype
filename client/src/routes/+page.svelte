<script lang="ts">
	import { onMount } from 'svelte';
	// Only import your actual Rust functions – NO init!
	import { add } from '../rust-core/pkg/rust_core.js'; // Or absolute '/rust-core/pkg/rust_core.js'

	let result = 0;
	let loading = true;

	onMount(async () => {
		try {
			// No await init() needed – the plugin makes functions usable directly
			result = add(2, 3); // Works immediately (async under the hood, handled by plugin)
		} catch (err) {
			console.error('Wasm failed:', err);
		} finally {
			loading = false;
		}
	});
</script>

{#if loading}
	<h1>Loading Wasm...</h1>
{:else}
	<h1>Result: {result}</h1>
{/if}
