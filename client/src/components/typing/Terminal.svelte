<script lang="ts">
	import BlandCode from './BlandCode.svelte';
	import Code from './Code.svelte';
</script>

<div
	class="rounded-1xl relative m-4 h-[66.66vh] w-full overflow-hidden border border-gray-700 bg-[#1E1E2E] p-4 font-mono text-2xl text-[#CDD6F4] shadow-2xl"
>
	<!-- Terminal Header -->
	<div class="flex items-center gap-2 border-b border-gray-700 bg-[#181825] px-4 py-2">
		<span class="h-3 w-3 rounded-full bg-red-400"></span>
		<span class="h-3 w-3 rounded-full bg-yellow-400"></span>
		<span class="h-3 w-3 rounded-full bg-green-400"></span>
		<span class="ml-4 text-sm text-gray-400 select-none">Made with ☕ by Arnav</span>
	</div>

	<!-- Code Area -->
	<div class="absolute inset-0 top-[2.5rem] overflow-auto p-4">
		<div class="absolute z-1 opacity-30">
			<BlandCode />
		</div>
		<div class="relative z-10">
			<Code />
		</div>
	</div>
</div>
