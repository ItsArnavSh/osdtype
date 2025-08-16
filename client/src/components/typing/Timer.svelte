<script lang="ts">
	import { onMount } from 'svelte';
	export let time: number;
	let last_secs: number = time / 4;
	let remaining_time = time;
	onMount(() => {
		let interval_id = setInterval(() => {
			remaining_time--;
			if (remaining_time <= last_secs) {
				clearInterval(interval_id);
				final_countdown();
			}
			if (remaining_time <= 0) {
				clearInterval(interval_id);
			}
		}, 1000);
	});
	function final_countdown() {
		remaining_time *= 1000;
		last_secs *= 1000; //Now they are last milliseconds
		let interval_id = setInterval(() => {
			remaining_time -= 50;
			if (remaining_time <= 0) {
				remaining_time = 0;
				clearInterval(interval_id);
			}
		}, 50);
	}
</script>

<div class="mt-10 ml-10 flex flex-col items-start justify-end space-y-2 font-mono text-white">
	<!-- Big number -->
	<div
		class="text-6xl leading-none font-bold"
		class:opacity-40={remaining_time >= last_secs}
		class:text-red-300={remaining_time < last_secs}
		style="min-width: 10ch;"
	>
		{#if remaining_time >= last_secs}
			{remaining_time}
		{:else}
			{Math.floor(remaining_time / 1000)}<span class="text-3xl"
				>.{Math.floor(remaining_time % 1000)}
			</span>
		{/if}
	</div>
	<!-- Label -->
	<div class=" text-sm tracking-widest uppercase">Timer</div>
</div>
