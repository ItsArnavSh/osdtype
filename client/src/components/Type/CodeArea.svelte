<script lang="ts">
	import CurrTyp from './CurrTyp.svelte';
	import TypBottom from './TypBottom.svelte';

	let { txt_arr = [] }: { txt_arr: string[] } = $props();
</script>

<div class="overlay-grid">
	<TypBottom {txt_arr} />
	<CurrTyp {txt_arr} />
</div>

<style>
	.overlay-grid {
		display: grid;
		grid-template-columns: 1fr;
		grid-template-rows: 1fr;
		position: relative; /* Good to have for containment */
	}

	.overlay-grid > :global(*) {
		grid-area: 1 / 1 / 2 / 2; /* Both overlap perfectly */
	}

	/* Bottom layer (visible text) */
	.overlay-grid > :global(:nth-child(1)) {
		z-index: 1;
	}

	/* Top layer (typing input) */
	.overlay-grid > :global(:nth-child(2)) {
		z-index: 2;
	}
</style>
