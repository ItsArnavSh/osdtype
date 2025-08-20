<script>
	import Highlight from 'svelte-highlight';
	import { langHighlighter } from '../../core/lsp';
	import materialPalenight from 'svelte-highlight/styles/material-palenight';

	export let expected = '';
	export let typed = '';
	export let language = 'typescript';

	// produce an array of chars with a "diff" flag
	$: diffed = typed.split('').map((ch, i) => ({
		char: ch,
		isDiff: expected[i] !== ch
	}));
</script>

<svelte:head>
	{@html materialPalenight}
</svelte:head>

<div class="relative font-mono text-3xl leading-snug opacity-90">
	<!-- Highlighted code -->
	<div class="code-layer pointer-events-none absolute inset-0 z-0">
		<Highlight language={langHighlighter(language)} code={expected.substring(0, typed.length)} />
	</div>

	<!-- Overlay typed text -->
	<code class="code-layer absolute inset-0 z-10 whitespace-pre caret-white">
		{#each diffed as d, i}
			<span class={d.isDiff ? 'errline text-red-500' : 'text-transparent'}>
				{expected[i]}
			</span>
		{/each}
	</code>
</div>
