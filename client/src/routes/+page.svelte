<script lang="ts">
	import { onMount } from 'svelte';
	import Config from '../components/config/Config.svelte';
	import Menu from '../components/nav/Menu.svelte';
	import Terminal from '../components/typing/Terminal.svelte';
</script>

<div class="relative bg-[#202432]">
	<div
		class="bebas fixed top-[-10%] left-[10%] z-0 text-[430px] font-bold tracking-[.02em] text-[#292d3e] opacity-[60%] drop-shadow-[0_4px_6px_rgba(0,0,0,0.05)]"
	>
		OSDTYPE
	</div>

	<div class="group fixed top-10 right-[2%] z-50">
		<button>
			<div
				class="relative flex items-center justify-center text-xl tracking-widest text-gray-100 uppercase"
				aria-label="menu"
			>
				Menu
				<span
					class="absolute -bottom-1 left-0 h-[2px] w-0 bg-gray-400 transition-all duration-300 group-hover:w-full"
				></span>
			</div>
		</button>
		<div
			class="invisible mt-2 opacity-0 transition-all duration-200 group-hover:visible group-hover:opacity-100"
		>
			<Menu />
		</div>
	</div>

	<div class="relative z-10 flex h-lvh flex-col">
		<div class="flex w-full flex-1 flex-col p-4 pr-10 pl-10">
			<Config />
			<div class="mt-auto">
				<Terminal />
			</div>
		</div>
	</div>
</div>
