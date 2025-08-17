<script lang="ts">
	import { onMount } from 'svelte';
	export let time: number;
	let remaining_time = time;
	onMount(() => {
		let interval_id = setInterval(() => {
			remaining_time--;
			if (remaining_time <= 0) {
				clearInterval(interval_id);
			}
		}, 1000);
	});
</script>

<div class="mt-10 ml-10 flex flex-col items-start justify-end space-y-1 font-mono text-white">
	<!-- Big number -->
	<div class="flex flex-row text-5xl leading-none font-bold opacity-40" style="min-width: 10ch;">
		{remaining_time}
	</div>
	<!-- Label -->
	<div class=" text-sm tracking-widest uppercase">Timer</div>
</div>
