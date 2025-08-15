<script>
	import Highlight, { LineNumbers } from 'svelte-highlight';
	import atomOneDark from 'svelte-highlight/styles/atom-one-dark';
	import { langHighlighter } from '../../core/lsp';
	export let code = '';
	export let language;
</script>

<svelte:head>
	{@html atomOneDark}
</svelte:head>

<Highlight language={langHighlighter(language)} {code} let:highlighted>
	<LineNumbers {highlighted} wrapLines hideBorder />
</Highlight>
