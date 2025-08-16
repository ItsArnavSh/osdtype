<script>
	import Highlight, { LineNumbers } from 'svelte-highlight';
	import { langHighlighter } from '../../core/lsp';
	export let expected = '';
	export let typed = '';
	import materialPalenight from 'svelte-highlight/styles/material-palenight';
	export let language = 'typescript';

	function getDifferingLines(str1, str2) {
		const lines1 = str1.split('\n');
		const lines2 = str2.split('\n');
		const maxLen = Math.max(lines1.length, lines2.length);

		const differingLines = [];

		for (let i = 0; i < maxLen; i++) {
			const line1 = lines1[i] ?? ''; // default empty if missing
			const line2 = lines2[i] ?? '';
			if (line1 !== line2) {
				differingLines.push(i); // index starts at 0
			}
		}

		return differingLines;
	}
	let lines = getDifferingLines(expected, typed);
</script>

<svelte:head>
	{@html materialPalenight}
</svelte:head>

<Highlight language={langHighlighter(language)} code={typed} let:highlighted>
	<LineNumbers
		{highlighted}
		highlightedLines={[0, 2]}
		startingLineNumber={100}
		--highlighted-background="rgba(239, 42, 42, 0.2)"
	/>
</Highlight>
