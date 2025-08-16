<script>
	import Highlight, { LineNumbers } from 'svelte-highlight';
	import markdown from 'svelte-highlight/languages/markdown';

	export let code = '';
</script>

<div class="">
	{#if code}
		<Highlight language={markdown} let:highlighted {code}>
			<LineNumbers {highlighted} startingLineNumber={100} />
		</Highlight>
	{/if}
</div>
