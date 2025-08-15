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

{#if remaining_time >= last_secs}
	{remaining_time}
{:else}
	{Math.round(remaining_time / 1000)}.{Math.round(remaining_time % 1000)}
{/if}
