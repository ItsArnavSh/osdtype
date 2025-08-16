<script lang="ts">
	import Page from '../../routes/+page.svelte';
	import BlandCode from './BlandCode.svelte';
	import Code from './Code.svelte';
	let scrollContainer: HTMLDivElement;
	let expected_code, typed_code: string;
	let myTextarea;

	function focusTextarea() {
		myTextarea.focus();
	}
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
	function block_event(event) {
		event.preventDefault();
	}
</script>

<div
	class="700 relative m-4 h-[66.66vh] w-full overflow-hidden rounded-2xl bg-[#292d3e] p-4 font-mono text-2xl text-[#CDD6F4] shadow-2xl"
	onclick={focusTextarea}
>
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

<textarea
	class="flxed z-10 opacity-0"
	bind:this={myTextarea}
	bind:value={typed_code}
	onpaste={block_event}
	onselect={block_event}
	oncontextmenu={block_event}
	ondrop={block_event}
	onkeypress={(event) => {
		if (event.ctrlKey) {
			block_event(event);
		}
	}}
></textarea>
