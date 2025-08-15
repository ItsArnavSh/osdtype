<script lang="ts">
	import BlandCode from './BlandCode.svelte';
	import Code from './Code.svelte';
	let scrollContainer: HTMLDivElement;
	let expected_code, typed_code: string;
	let tokens = [
		'let',
		' ',
		'x',
		' ',
		'=',
		' ',
		'5',
		';',
		'\n',
		'print',
		'(',
		'"',
		'Hello,',
		' ',
		'World',
		'"',
		')',
		';',
		'\n',
		'print',
		'(',
		'x',
		')',
		';',
		'\n'
	];

	const lineHeight = 32; // px
	const preScrollZone = 4; // start scrolling when caret is within 4 lines from center
	expected_code = tokens.join('');
	$: if (scrollContainer) {
		const linesTyped = typed_code.split('\n').length;
		const caretPos = linesTyped * lineHeight;
		const containerMid = scrollContainer.scrollTop + scrollContainer.clientHeight / 2;
		const triggerZoneStart = containerMid - preScrollZone * lineHeight;
		const triggerZoneEnd = containerMid + preScrollZone * lineHeight;

		if (caretPos > triggerZoneEnd || caretPos < triggerZoneStart) {
			scrollContainer.scrollTo({
				top: caretPos - scrollContainer.clientHeight / 2 + lineHeight,
				behavior: 'smooth'
			});
		}
	}
</script>

<div
	class="rounded-1xl relative m-4 h-[66.66vh] w-full overflow-hidden border border-gray-700 bg-[#1E1E2E] p-4 font-mono text-2xl text-[#CDD6F4] shadow-2xl"
>
	<!-- Terminal Header -->
	<div class="flex items-center gap-3 px-4 py-2">
		<span class="h-4 w-4 rounded-full bg-red-400"></span>
		<span class="h-4 w-4 rounded-full bg-yellow-400"></span>
		<span class="h-4 w-4 rounded-full bg-green-400"></span>
	</div>

	<!-- Code Area -->
	<div class="absolute inset-0 top-[2.5rem] overflow-auto p-4" bind:this={scrollContainer}>
		<div class="absolute z-1 opacity-30">
			<BlandCode code={expected_code} />
		</div>
		<div class="relative z-10">
			<Code language="typescript" code={typed_code} />
		</div>
	</div>
</div>

<textarea class="" bind:value={typed_code}></textarea>
