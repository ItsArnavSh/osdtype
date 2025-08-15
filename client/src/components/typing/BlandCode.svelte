<script>
	import Highlight, { LineNumbers } from 'svelte-highlight';
	import markdown from 'svelte-highlight/languages/markdown';

	const code = 'const add = (a: number, b: number) => a + b;';
</script>

<div class="">
	<Highlight language={markdown} {code} let:highlighted>
		<LineNumbers {highlighted} wrapLines hideBorder />
	</Highlight>
</div>
