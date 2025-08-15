<script lang="ts">
	import Navbar from '../components/nav/Navbar.svelte';
	import Code from '../components/typing/Code.svelte';
	import Terminal from '../components/typing/Terminal.svelte';
	import Timer from '../components/typing/Timer.svelte';

	//Homepage is the solo typing page
</script>

<Navbar />
<div class="flex h-screen w-full flex-col items-center justify-center bg-[#11111b] p-4 pr-10 pl-10">
	<Terminal />
</div>
