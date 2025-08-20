<script lang="ts">
	import BlandCode from './BlandCode.svelte';
	import { diffChars, createPatch } from 'diff';
	import Code from './Code.svelte';
	import { onMount } from 'svelte';
	import { getsnippet } from '../../api/snippet';

	let scrollContainer: HTMLDivElement;
	let typed_code: string = '';
	let expected_code: string = ''; // Add this declaration
	let myTextarea: HTMLTextAreaElement;
	let previous_typed_code = '';

	// Add type definition if not imported
	interface SnippetResponse {
		Snippet: string[];
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
		myTextarea?.focus();
	}

	let snippet: SnippetResponse | null = null; // Initialize as null
	let tokens: string[] = [];

	onMount(async () => {
		typed_code = '';
		previous_typed_code = '';
		try {
			snippet = await getsnippet('typescript');
			console.log('Raw snippet:', snippet);

			// Parse the JSON string to get the actual array
			tokens = JSON.parse(snippet.Snippet);
			expected_code = tokens.join('');

			console.log('Parsed tokens:', tokens);
			console.log('Expected code:', expected_code);
		} catch (error) {
			console.error('Failed to load snippet:', error);
		}
	});

	function block_event(event: Event) {
		event.preventDefault();
	}
</script>

<div
	class="relative m-3 h-[66.66vh] w-full overflow-hidden rounded-2xl bg-[#292d3e] p-4 font-mono text-xl text-[#CDD6F4] shadow-2xl"
	on:click={focusTextarea}
	role="button"
	tabindex="0"
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
	on:paste={block_event}
	on:select={block_event}
	on:contextmenu={block_event}
	on:drop={block_event}
	on:keyup={onKeystroke}
	on:keypress={(event) => {
		// Only prevent spaces if we're not expecting a space
		if (event.key === ' ' && expected_code[typed_code.length] !== ' ') {
			event.preventDefault();
		}
	}}
></textarea>
