<script>
	import Highlight, { LineNumbers } from 'svelte-highlight';
	import typescript from 'svelte-highlight/languages/typescript';
	import atomOneDark from 'svelte-highlight/styles/atom-one-dark';

	const code = 'const add = (a: number';
</script>

<svelte:head>
	{@html atomOneDark}
</svelte:head>

<Highlight language={typescript} {code} let:highlighted>
	<LineNumbers {highlighted} wrapLines hideBorder />
</Highlight>

<style>
	.reveal pre code,
	.hljs {
		background-color: #ffffff !important;
		color: #dcdcdc;
		padding: 5px;
		overflow: auto;
		max-height: 400px;
		word-wrap: normal;
	}
	.reveal pre,
	.hljs {
		box-shadow: none !important;
	}
</style>
