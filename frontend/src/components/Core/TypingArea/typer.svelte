<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	let competitionTime: number = 10;
	let targetText: string = 'Hello';
	targetText = targetText.repeat(5);
	let allowSpace = true;
	const dispatch = createEventDispatcher();
	const newSpace = '\u0131'; // dotless i
	const newEnter = '\u02BC'; // modifier apostrophe
	const newTab = '\u03BC'; // Greek mu
	let yetToStart = true;
	let displayText = targetText
		.replace(/ /g, newSpace)
		.replace(/\n/g, newEnter) // Keep newlines as newEnter for display
		.replace(/\t/g, newTab)
		.split('');
	function runAfter(callback: () => void) {
		setTimeout(callback, competitionTime * 1000);
	}
	let wpm: number;

	let userInput = '';
	let userInpArray: string[] = [];
	let forbiddenArr = allowSpace ? [newEnter, newTab] : [newSpace, newEnter, newTab]; // Keep newEnter in forbidden since it's auto-added
	function updateData() {
		if (yetToStart) {
			dispatch('activateTimer', null);
			yetToStart = false;
		}
		// Convert spaces and user-entered newlines to the space character
		userInput = userInput.replaceAll(' ', allowSpace ? newSpace : '');
		userInput = userInput.replaceAll('\n', allowSpace ? newSpace : ''); // Convert user enters to spaces

		if (
			displayText[userInput.length - 1] == newSpace &&
			userInput[userInput.length - 1] != newSpace &&
			userInput[userInput.length - 1] != displayText[userInput.length - 1]
		) {
			// Remove the last character (which is misspelled)
			userInput = userInput.slice(0, -1);
			// Add the correct characters (space + next character)
			userInput += displayText[userInput.length] + displayText[userInput.length + 1];
		}
		while (forbiddenArr.includes(displayText[userInput.length])) {
			userInput += displayText[userInput.length];
		}
		userInpArray = userInput.split('');
	}
	let blank = ' ';
	let inputRef: HTMLInputElement;
	function focusInput() {
		inputRef?.focus();
	}
	function findAllOccurrences(str: string, char: string) {
		return [...str].map((c, i) => (c === char ? i : -1)).filter((i) => i !== -1);
	}
	// Keep original enter tracking since we still have newlines in target text
	let enters = findAllOccurrences(targetText, '\n');
	enters = [0, ...enters];
	// Get the index in `enters` of the next \n after userInput.length
	function nextEnterIndex(pos: number): number {
		for (let i = 0; i < enters.length; i++) {
			if (enters[i] > pos) return i;
		}
		return enters.length - 1; // fallback to last \n if beyond
	}
	let letLower: number, letHigher: number;
	$: {
		if (enters.length < 9) {
			letLower = 0;
			letHigher = displayText.length;
		} else {
			const currentEnterIndex = nextEnterIndex(userInput.length);
			// Clamp so we don't go negative
			const lowerEnterIndex = Math.max(currentEnterIndex - 3, 0);
			const higherEnterIndex = Math.min(currentEnterIndex + 10, enters.length - 1);
			// Now these are positions (in the actual string):
			letLower = enters[lowerEnterIndex];
			letHigher = enters[higherEnterIndex];
		}
	}
</script>

<div class="terminal hide-scrollbar h-[50%] scroll-m-0 overflow-auto font-mono">
	<button
		class="fixed z-10 h-[40%] w-[60%] opacity-0"
		type="button"
		onclick={() => {
			focusInput();
		}}>s</button
	>
	<div class="screen"></div>
	<div class="flicker"></div>
	<!-- Target text in light opacity -->
	<p class="hide-scrollbar absolute top-20 left-20 z-1 overflow-auto text-gray-500">
		{#each displayText as char, i (i)}
			{#if i >= letLower && i <= letHigher}
				{#if (i < userInput.length ? userInput[i] : char) == newEnter}
					<br />
				{:else if (i < userInput.length ? userInput[i] : char) == newSpace}
					{blank}
				{:else if (i < userInput.length ? userInput[i] : char) == newTab}
					<span>&nbsp;&nbsp</span>
				{:else}
					<span class="terminal-text">{i < userInput.length ? userInput[i] : char}</span>
				{/if}
			{/if}
		{/each}
	</p>

	<!-- User input layered on top -->
	<p class="absolute top-20 left-20 z-2 overflow-auto">
		{#each userInpArray as char, i (i)}
			{#if i >= letLower && i <= letHigher}
				{#if (i < userInput.length ? userInput[i] : char) == newEnter}
					{#if userInpArray.length - 1 === i}<span class="cursor"></span>{/if}<br />
					<span id="_{i}" class=""></span>
				{:else if (i < userInput.length ? userInput[i] : char) == newSpace}
					{blank}{#if userInpArray.length - 1 === i}<span class="cursor"></span>{/if}
				{:else if (i < userInput.length ? userInput[i] : char) == newTab}
					<span>&nbsp;&nbsp;</span>{#if userInpArray.length - 1 === i}<span class="cursor"
						></span>{/if}
				{:else}
					<span
						class={char == displayText[i]
							? 'terminal-text response correct'
							: 'terminal-text error wrong'}
					>
						{char}{#if userInpArray.length - 1 === i}<span class="cursor"></span>{/if}
					</span>
				{/if}
			{/if}
		{/each}
	</p>
	<input
		class="pointer-events-none absolute z-5 w-full bg-none opacity-0"
		type="text"
		oninput={() => {
			updateData();
		}}
		onkeydown={(e) => {
			if (e.key == 'Backspace') {
				while ([newTab, newEnter, newSpace].includes(userInput[userInput.length - 1]))
					userInput = userInput.slice(0, -1);
			}
		}}
		autofocus
		bind:value={userInput}
		bind:this={inputRef}
		autocomplete="off"
		onpaste={(e) => {
			e.preventDefault();
			return false;
		}}
	/>
</div>
