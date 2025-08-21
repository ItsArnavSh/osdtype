<script lang="ts">
	import type { Confi } from '../../api/snippet';
	import LanguageSel from '../typing/LanguageSel.svelte';
	import Timer from '../typing/Timer.svelte';
	import TimeSelector from '../typing/TimeSelector.svelte';
	let start_timer = false;
	export let config: Confi;
</script>

<div class="flex flex-1 flex-row justify-between">
	{#if start_timer}
		<Timer time={45} />
	{:else}
		<TimeSelector bind:config />
	{/if}

	<LanguageSel bind:config />
</div>
