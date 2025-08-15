<script>
	import Highlight, { LineNumbers } from 'svelte-highlight';
	import markdown from 'svelte-highlight/languages/markdown';

	export let code = '';
</script>

<div class="">
	{#if code}
		<Highlight language={markdown} {code} let:highlighted>
			<LineNumbers {highlighted} wrapLines hideBorder />
		</Highlight>
	{/if}
</div>
