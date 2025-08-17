<script>
	import Highlight from 'svelte-highlight';
	import markdown from 'svelte-highlight/languages/markdown';

	export let code = 'code';
</script>

<div class="relative font-mono text-4xl leading-snug">
	<!-- Highlighted code (not clipped) -->
	<div class="code-layer pointer-events-none">
		<Highlight language={markdown} {code} />
	</div>
</div>
