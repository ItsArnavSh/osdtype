<script>
	import { GenerateCode, Languages } from '$lib/rust_bridge/rs';
	import '../../layout.css';
	import CodeArea from '../../../components/Type/CodeArea.svelte';
	import Selector from '../../../components/Type/Selector.svelte';
	import { onMount } from 'svelte';
	let text_arr = GenerateCode(Languages.CPP, 30, 1000);
	let config = {
		Time: 120,
		Language: 'Rust',
		Editable: true
	};
	let started = false;
	function handleKeydown() {
		if (!started) started = true;
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});
</script>

<div class="h-screen overflow-hidden bg-(--bg)">
	<!-- Header -->
	<div class="flex flex-row">
		<div class="bebas-neue m-10 mb-0 flex-1 text-9xl text-(--silver)">OSDTYPE</div>

		<div class="mt-10 mr-20 flex flex-col text-xl">
			<p class="fira-code text-xl text-(--red)">/Hub</p>
			<Selector {config} />
		</div>
	</div>

	<!-- Editor Wrapper (shorter than screen) -->
	<div
		class="relative h-[calc(100vh-140px)] overflow-y-scroll border-t border-(--silver) bg-(--fbg)"
	>
		<div class="fira-code relative ml-20 h-full overflow-hidden border-l border-(--silver) p-5">
			{#if !started}
				<!-- Overlay -->
				<div class="absolute inset-0 z-10 w-full">
					<!-- Translucent bg -->
					<div class="absolute inset-0 bg-(--fbg) opacity-80"></div>

					<!-- Text -->
					<p
						class="relative flex h-full w-full items-center justify-center
						text-center text-3xl text-(--silver)"
					>
						Start Typing to begin
					</p>
				</div>
			{/if}

			<CodeArea txt_arr={text_arr} />
		</div>
	</div>
	<div>Hello</div>
</div>
