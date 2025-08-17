<script>
	import LanguageSel from '../typing/LanguageSel.svelte';
	import Timer from '../typing/Timer.svelte';
	import TimeSelector from '../typing/TimeSelector.svelte';
	let start_timer = false;
</script>

<div class="flex flex-1 flex-row justify-between">
	{#if start_timer}
		<Timer time={45} />
	{:else}
		<TimeSelector time={45} />
	{/if}

	<LanguageSel />
</div>
