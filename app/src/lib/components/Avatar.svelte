<script lang="ts">
	let {
		src = '',
		name = '',
		class: className = '',
		text = 'text-[10px]'
	}: { src?: string; name?: string; class?: string; text?: string } = $props();

	let failed = $state(false);
	let initial = $derived((name.trim().charAt(0) || '?').toUpperCase());
</script>

{#if src && !failed}
	<img
		{src}
		alt=""
		class={`${className} border border-(--border) object-cover`}
		onerror={() => (failed = true)}
	/>
{:else}
	<div
		class={`${className} flex items-center justify-center border border-(--border) bg-(--fbg) font-mono ${text} text-(--silver)/45`}
		aria-label={name}
	>
		{initial}
	</div>
{/if}
