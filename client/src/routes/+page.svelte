<script lang="ts">
	import Config from '../components/config/Config.svelte';
	import Navbar from '../components/nav/Navbar.svelte';
	import Terminal from '../components/typing/Terminal.svelte';

	//Homepage is the solo typing page
</script>

<div class="flex h-lvh flex-col bg-[#202432]">
	<div class="flex w-full flex-1 flex-col p-4 pr-10 pl-10">
		<Config />

		<!-- pushes terminal to bottom -->
		<div class="mt-auto">
			<Terminal />
		</div>
	</div>
</div>
