<script lang="ts">
	let { txt_arr = [] }: { txt_arr: string[] } = $props();
	const code = txt_arr.join('');
	console.log(code);
</script>

<div class="text-black">
	{#each code as word, i (i)}
		{#if word == '\n'}
			<br />
		{:else}
			{word}
		{/if}
	{/each}
</div>

<style>
	/* Optional: make the whole area feel clickable */
	div:hover {
		filter: brightness(1.05);
	}
</style>
