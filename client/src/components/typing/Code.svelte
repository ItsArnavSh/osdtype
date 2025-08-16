<script>
	import Highlight, { LineNumbers } from 'svelte-highlight';
	import { langHighlighter } from '../../core/lsp';
	export let code = '';
	import materialPalenight from 'svelte-highlight/styles/material-palenight';
	export let language;
</script>

<svelte:head>
	{@html materialPalenight}
</svelte:head>

<Highlight language={langHighlighter(language)} {code} let:highlighted>
	<LineNumbers {highlighted} wrapLines hideBorder />
</Highlight>
