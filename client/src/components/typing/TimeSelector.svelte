<script lang="ts">
	export let time: number;
	let is_hovering = false;

	function setTime(newTime: number) {
		time = newTime;
		is_hovering = false;
	}
</script>

<div class="mt-10 ml-10 flex flex-col items-start justify-end space-y-1 font-mono text-white">
	<!-- Big number with options -->
	<div
		class="flex cursor-pointer flex-row items-center gap-4 text-5xl leading-none font-bold opacity-40 hover:opacity-60"
		style="min-width: 10ch;"
		on:mouseenter={() => (is_hovering = true)}
		on:mouseleave={() => (is_hovering = false)}
	>
		{time}
		{#if is_hovering}
			<div class="slide-in flex gap-2 text-lg">
				<button
					class="fade-up transition-all duration-200 hover:scale-110 hover:opacity-60"
					style="animation-delay: 0ms;"
					on:click={() => setTime(15)}>15</button
				>
				<button
					class="fade-up transition-all duration-200 hover:scale-110 hover:opacity-60"
					style="animation-delay: 100ms;"
					on:click={() => setTime(30)}>30</button
				>
				<button
					class="fade-up transition-all duration-200 hover:scale-110 hover:opacity-60"
					style="animation-delay: 200ms;"
					on:click={() => setTime(45)}>45</button
				>
				<button
					class="fade-up transition-all duration-200 hover:scale-110 hover:opacity-60"
					style="animation-delay: 300ms;"
					on:click={() => setTime(90)}>60</button
				>
			</div>
		{/if}
	</div>
	<!-- Label -->
	<div class="text-sm tracking-widest uppercase">Timer</div>
</div>

<style>
	@keyframes slideIn {
		from {
			opacity: 0;
			transform: translateX(-20px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}

	@keyframes fadeUp {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.slide-in {
		animation: slideIn 0.3s ease-out forwards;
	}

	.fade-up {
		opacity: 0;
		animation: fadeUp 0.4s ease-out forwards;
	}
</style>
