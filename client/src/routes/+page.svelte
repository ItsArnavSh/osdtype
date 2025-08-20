<script lang="ts">
	import { onMount } from 'svelte';
	import Config from '../components/config/Config.svelte';
	import Menu from '../components/nav/Menu.svelte';
	import Terminal from '../components/typing/Terminal.svelte';
	let started = false;
</script>

<div class="relative bg-[#202432]">
	<div
		class="bebas fixed top-[-10%] left-[10%] z-0 text-[430px] font-bold tracking-[.02em] text-[#292d3e] opacity-[60%] drop-shadow-[0_4px_6px_rgba(0,0,0,0.05)]"
	>
		OSDTYPE
	</div>

	<div class="group fixed top-[1%] right-[2%] z-50">
		<div
			class="relative flex items-center justify-center text-xl tracking-widest text-gray-100 uppercase"
			aria-label="menu"
		>
			<Menu />
		</div>
	</div>

	<div class="relative z-10 flex h-lvh flex-col">
		<div class="flex w-full flex-1 flex-col p-4 pr-10 pl-10">
			<Config />
			<div class="mt-auto">
				{#if started}
					<div>
						<div
							class=" 700 relative m-3 h-[66.66vh] w-full overflow-hidden rounded-2xl bg-[#292d3e] p-4 font-mono text-xl text-[#CDD6F4] shadow-2xl"
						></div>
					</div>
				{/if}
				<Terminal />
			</div>
			<p class="mb-2 text-center font-mono text-white">Made with ☕️ by Arnav</p>
		</div>
	</div>
</div>
