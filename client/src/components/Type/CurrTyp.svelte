<script lang="ts">
	let { txt_arr = [] }: { txt_arr: string[] } = $props();
	const code = txt_arr.join('');
	let written = $state('');
	let display_code = $derived(code.substring(0, written.length));
	$effect(() => {
		console.log(display_code);
	});
	let textarea: HTMLTextAreaElement;
</script>

<div class="relative cursor-text text-green-400" onclick={() => textarea?.focus()}>
	{#each display_code as word, i (i)}
		{#if word == '\n'}
			<br />
		{:else if written[i] === code[i]}
			<span class="text-white">{word}</span>
		{:else}
			<span class="text-red-600">{word}</span>
		{/if}
	{/each}

	<textarea
		bind:this={textarea}
		class="absolute inset-0 h-full w-full resize-none opacity-0 outline-none"
		bind:value={written}
		autofocus
		spellcheck="false"
		autocomplete="off"
	></textarea>
</div>

<style>
	/* Optional: make the whole area feel clickable */
	div:hover {
		filter: brightness(1.05);
	}
</style>
