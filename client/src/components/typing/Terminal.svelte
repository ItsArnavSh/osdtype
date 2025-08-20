<script lang="ts">
	import BlandCode from './BlandCode.svelte';
	import { diffChars, createPatch } from 'diff';
	import Code from './Code.svelte';
	import { onMount } from 'svelte';

	let scrollContainer: HTMLDivElement;
	let expected_code: string = '',
		typed_code: string = '';
	let myTextarea;
	let game_started = false;
	// Add these variables for diff tracking
	let previous_typed_code = '';
	let changes_log = [];

	function end_game() {
		game_started = false;
	}

	function start_game() {
		game_started = true;
		// Reset tracking when game starts
		previous_typed_code = '';
		changes_log = [];
	}

	interface ChangeEvent {
		type: 'added' | 'removed';
		changedWords: string;
		timestamp: number;
	}

	function onKeystroke() {
		const timestamp = Date.now();
		const changes = diffChars(previous_typed_code, typed_code);

		changes
			.filter((c) => c.added || c.removed)
			.forEach((change) => {
				const event: ChangeEvent = {
					type: change.added ? 'added' : 'removed',
					changedWords: change.value,
					timestamp: timestamp
				};
				console.log(event);
			});

		previous_typed_code = typed_code;

		// Auto-add spaces if next expected character is a space
		if (typed_code.length < expected_code.length && expected_code[typed_code.length] === ' ') {
			typed_code += ' ';
		}
	}

	function focusTextarea() {
		myTextarea.focus();
	}

	let tokens = [
		'let',
		' ',
		'count',
		':',
		' ',
		'number',
		' ',
		'=',
		' ',
		'0',
		';',
		'\n',
		'function',
		' ',
		'add',
		'(',
		'a',
		':',
		' ',
		'number',
		',',
		' ',
		'b',
		':',
		' ',
		'number',
		')',
		':',
		' ',
		'number',
		' ',
		'{',
		'\n',
		'return',
		' ',
		'a',
		' ',
		'+',
		' ',
		'b',
		';',
		'\n',
		'}',
		'\n',
		'class',
		' ',
		'Person',
		' ',
		'{',
		'\n',
		'name',
		':',
		' ',
		'string',
		';',
		'\n',
		'constructor',
		'(',
		'name',
		':',
		' ',
		'string',
		')',
		' ',
		'{',
		'\n',
		'this',
		'.',
		'name',
		' ',
		'=',
		' ',
		'name',
		';',
		'\n',
		'}',
		'\n',
		'greet',
		'(',
		')',
		':',
		' ',
		'void',
		' ',
		'{',
		'\n',
		'console',
		'.',
		'log',
		'(',
		'"',
		'Hello,',
		' ',
		'"',
		'+',
		' ',
		'this',
		'.',
		'name',
		')',
		';',
		'\n',
		'}',
		'\n',
		'}',
		'\n',
		'const',
		' ',
		'p',
		' ',
		'=',
		' ',
		'new',
		' ',
		'Person',
		'(',
		'"',
		'Arnav',
		'"',
		')',
		';',
		'\n',
		'console',
		'.',
		'log',
		'(',
		'add',
		'(',
		'5',
		',',
		' ',
		'7',
		')',
		')',
		';',
		'\n',
		'p',
		'.',
		'greet',
		'(',
		')',
		';'
	];

	onMount(() => {
		typed_code = '';
		previous_typed_code = '';
	});

	const lineHeight = 32;
	const preScrollZone = 4;
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
	class=" 700 relative m-3 h-[66.66vh] w-full overflow-hidden rounded-2xl bg-[#292d3e] p-4 font-mono text-xl text-[#CDD6F4] shadow-2xl"
	onclick={focusTextarea}
>
	<div class="absolute inset-0 top-[2.5rem] ml-5 overflow-auto p-4" bind:this={scrollContainer}>
		<div class="absolute z-1 opacity-30">
			<BlandCode code={expected_code} />
		</div>
		<div class="relative z-10">
			<Code language="typescript" typed={typed_code} expected={expected_code} />
		</div>
	</div>
</div>

<textarea
	class="fixed z-0 opacity-0"
	bind:this={myTextarea}
	bind:value={typed_code}
	onpaste={block_event}
	onselect={block_event}
	oncontextmenu={block_event}
	ondrop={block_event}
	onkeyup={onKeystroke}
	onkeypress={(event) => {
		if (event.key === ' ') {
			event.preventDefault();
		}
	}}
></textarea>
