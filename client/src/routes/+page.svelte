<script lang="ts">
	import { onMount } from 'svelte';
	import { generate } from '../rust-core/pkg/rust_core.js';
	import { CGrammar } from '$lib/core/templates/c.js';

	let result: string[] = [];
	let loading = true;
	onMount(async () => {
		try {
			result = generate(CGrammar, 12312, 1000);
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
