<script lang="ts">
	import SettingsModal from '$lib/components/SettingsModal.svelte';
	import { CGrammar } from '$lib/templates/c';
	import { onMount } from 'svelte';
	import { generate } from '../../../rust-core/pkg/rust_core';
	import { GetGrammar, type Language } from '$lib/core/entity/languages';

	let lang: Language = $state('Go');
	let timer = $state(30);
	let modalOpen = $state(false);

	let seed = $derived(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

	let code = $derived(generate(GetGrammar(lang), seed, 1000));

	let display_code = $derived(code.join(''));

	let display_chars = $derived(display_code.split('')); // --- Typing state ---
	let cursor = $state(0);
	let typed: Record<number, string> = $state({});

	// --- Game state ---
	type Phase = 'idle' | 'typing' | 'done';
	let phase = $state<Phase>('idle');
	let timeLeft = $state(timer);
	let wpm = $state(0);
	let intervalId: ReturnType<typeof setInterval> | null = null;

	// --- Final scores ---
	let finalWpm = $state(0);
	let finalRawWpm = $state(0);
	let finalAccuracy = $state(0);
	let finalCorrect = $state(0);
	let finalWrong = $state(0);

	function calcStats() {
		const elapsed = timer - timeLeft;
		const entries = Object.entries(typed);
		const correct = entries.filter(([i, ch]) => ch === display_chars[Number(i)]).length;
		const wrong = entries.filter(([i, ch]) => ch !== display_chars[Number(i)]).length;
		const raw = elapsed > 0 ? Math.round(entries.length / 5 / (elapsed / 60)) : 0;
		const net = elapsed > 0 ? Math.round(correct / 5 / (elapsed / 60)) : 0;
		const acc = entries.length > 0 ? Math.round((correct / entries.length) * 100) : 100;
		return { correct, wrong, raw, net, acc };
	}

	function startGame() {
		if (phase !== 'idle') return;
		phase = 'typing';
		timeLeft = timer;
		intervalId = setInterval(() => {
			timeLeft--;
			const { net } = calcStats();
			wpm = net;
			if (timeLeft <= 0) {
				clearInterval(intervalId!);
				intervalId = null;
				endGame();
			}
		}, 1000);
	}

	function endGame() {
		const { correct, wrong, raw, net, acc } = calcStats();
		finalWpm = net;
		finalRawWpm = raw;
		finalAccuracy = acc;
		finalCorrect = correct;
		finalWrong = wrong;
		phase = 'done';
	}

	function resetGame() {
		if (intervalId) {
			clearInterval(intervalId);
			intervalId = null;
		}
		phase = 'idle';
		timeLeft = timer;
		wpm = 0;
		typed = {};
		cursor = nextReal(0);

		seed = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
		code = generate(CGrammar, seed, 1000);

		display_code = code.join('');
		display_chars = display_code.split('');
	}

	function isSpecial(char: string) {
		return char === ' ' || char === '\n' || char === '\t';
	}
	function nextReal(pos: number) {
		while (pos < display_chars.length && isSpecial(display_chars[pos])) pos++;
		return pos;
	}
	function prevReal(pos: number) {
		pos--;
		while (pos > 0 && isSpecial(display_chars[pos])) pos--;
		return Math.max(0, pos);
	}

	let char_states = $derived(
		display_chars.map((char, i) => {
			if (isSpecial(char)) return 'special';
			if (!(i in typed)) return 'pending';
			return typed[i] === char ? 'correct' : 'wrong';
		})
	);
	// Dot
	let dotTop = $state(0);

	$effect(() => {
		const _ = cursor;
		const span = spanRefs[cursor];
		if (!span || !codeContainer) return;

		const spanOffsetTop = span.offsetTop;
		const targetScrollTop = spanOffsetTop - codeContainer.clientHeight / 2 + span.offsetHeight / 2;
		codeContainer.scrollTop = Math.max(0, targetScrollTop);

		// Dot: position relative to the border container, accounting for scroll
		dotTop = spanOffsetTop - codeContainer.scrollTop + span.offsetHeight / 2;
	});
	// --- Autoscroll ---
	let codeContainer: HTMLDivElement;
	let spanRefs: HTMLSpanElement[] = [];

	$effect(() => {
		const _ = cursor;
		const span = spanRefs[cursor];
		if (!span || !codeContainer) return;
		const spanOffsetTop = span.offsetTop;
		const targetScrollTop = spanOffsetTop - codeContainer.clientHeight / 2 + span.offsetHeight / 2;
		codeContainer.scrollTop = Math.max(0, targetScrollTop);
	});

	// --- Input handling ---
	let inputRef: HTMLTextAreaElement;
	function focusInput() {
		inputRef?.focus();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Tab') {
			e.preventDefault();
			resetGame();
			return;
		}
		if (phase === 'done') return;
		if (e.ctrlKey || e.metaKey || e.altKey) return;
		if (e.key === ' ' || e.key === 'Enter') {
			e.preventDefault();
			return;
		}
		if (e.key === 'Backspace') {
			e.preventDefault();
			if (phase === 'idle') return;
			const prev = prevReal(cursor);
			delete typed[prev];
			cursor = prev;
			return;
		}
		if (e.key.length === 1) {
			e.preventDefault();
			if (phase === 'idle') startGame();
			const pos = nextReal(cursor);
			typed[pos] = e.key;
			cursor = nextReal(pos + 1);
		}
	}

	onMount(() => {
		cursor = nextReal(0);
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});
</script>

<div class="m-0 min-h-screen overflow-hidden bg-(--bg)">
	<div class=" mt-10 flex flex-col p-2 text-white">
		<div class="flex flex-row items-end">
			<div class="title-text flex-1 text-9xl">OSDTYP</div>
			<div class="my-8 flex flex-col items-end gap-1 px-6">
				<div class="font-mono text-(--red)">→Hub</div>
				<button
					onclick={() => (modalOpen = true)}
					class="flex cursor-pointer flex-row items-end gap-6 opacity-100 transition-opacity hover:opacity-70"
				>
					<div class="flex flex-col items-center">
						<div class="font-mono text-3xl text-(--mer)">{timeLeft}</div>
						<div class="text-xs text-(--silver)/50">Timer</div>
					</div>
					<div class="flex flex-col items-end">
						<div class="font-mono text-3xl text-(--mer)">{lang}</div>
						<div class="text-xs text-(--silver)/50">Language</div>
					</div>
				</button>
			</div>
		</div>
	</div>

	{#if modalOpen}
		<SettingsModal bind:lang bind:timer bind:open={modalOpen} />
	{/if}

	<textarea
		bind:this={inputRef}
		class="pointer-events-none absolute resize-none opacity-0"
		autocomplete="off"
		autocorrect="off"
		spellcheck="false"
		rows="1"
	/>

	<!-- Code display / results -->
	<div
		class="relative border-t border-b border-(--silver) bg-(--fbg)"
		onclick={focusInput}
		role="none"
		style="height: 75vh;"
	>
		{#if phase === 'done'}
			<!-- Results screen -->
			<div class="flex h-full flex-col items-center justify-center gap-10 px-16">
				<!-- Primary stat -->
				<div class="flex flex-col items-center">
					<div class="font-mono text-8xl text-(--mer)">{finalWpm}</div>
					<div class="mt-1 font-mono text-sm tracking-widest text-(--silver)/40 uppercase">wpm</div>
				</div>

				<!-- Secondary stats row -->
				<div class="flex flex-row gap-16">
					<div class="flex flex-col items-center">
						<div class="font-mono text-4xl text-(--silver)">{finalRawWpm}</div>
						<div class="mt-1 font-mono text-xs tracking-widest text-(--silver)/40 uppercase">
							raw
						</div>
					</div>
					<div class="flex flex-col items-center">
						<div class="font-mono text-4xl text-(--silver)">{finalAccuracy}%</div>
						<div class="mt-1 font-mono text-xs tracking-widest text-(--silver)/40 uppercase">
							acc
						</div>
					</div>
					<div class="flex flex-col items-center">
						<div class="font-mono text-4xl text-green-400">{finalCorrect}</div>
						<div class="mt-1 font-mono text-xs tracking-widest text-(--silver)/40 uppercase">
							correct
						</div>
					</div>
					<div class="flex flex-col items-center">
						<div class="font-mono text-4xl text-(--red)">{finalWrong}</div>
						<div class="mt-1 font-mono text-xs tracking-widest text-(--silver)/40 uppercase">
							wrong
						</div>
					</div>
					<div class="flex flex-col items-center">
						<div class="font-mono text-4xl text-(--silver)">{timer}s</div>
						<div class="mt-1 font-mono text-xs tracking-widest text-(--silver)/40 uppercase">
							time
						</div>
					</div>
				</div>

				<div class="font-mono text-sm tracking-widest text-(--silver)/30">tab to restart</div>
			</div>
		{:else}
			<!-- Code display -->
			<!-- Code display -->
			<div class="relative ml-10 h-full border-l border-(--silver) bg-(--fbg)">
				<!-- Dot on the border -->
				<div
					class="absolute rounded-full transition-all duration-100"
					style="width: 12px; height: 12px; left: -6px; top: {dotTop}px; transform: translateY(-50%); background-color: var(--mer);"
				></div>

				<div
					bind:this={codeContainer}
					class="h-full overflow-hidden p-5 font-mono text-2xl leading-relaxed"
				>
					{#each display_chars as char, i (i)}
						<span
							bind:this={spanRefs[i]}
							class="whitespace-pre-wrap {char_states[i] === 'special'
								? 'text-(--silver)/20'
								: char_states[i] === 'correct'
									? 'text-(--silver)'
									: char_states[i] === 'wrong'
										? 'bg-red-950/40 text-(--red)'
										: 'text-(--silver)/40'}">{char}</span
						>
					{/each}
				</div>
			</div>
			<!-- Idle overlay -->
			{#if phase === 'idle'}
				<div
					class="absolute inset-0 flex items-center justify-center"
					style="background: rgba(0,0,0,0.5);"
				>
					<span class="font-mono text-xl tracking-widest text-(--silver)/70"
						>start typing to begin</span
					>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Bottom bar -->
	<div class="flex flex-row items-center px-4 py-3">
		<button
			onclick={resetGame}
			class="flex cursor-pointer flex-row items-center gap-2 font-mono text-sm text-(--silver)/50 opacity-100 transition-opacity hover:opacity-70"
		>
			<span class="rounded border border-(--silver)/20 px-1.5 py-0.5 text-xs text-(--silver)/30"
				>tab</span
			>
			<span>reset</span>
		</button>

		<div class="flex-1"></div>

		{#if phase === 'typing'}
			<div class="text-right font-mono">
				<span class="text-3xl text-(--mer)">{wpm}</span>
				<span class="ml-1 text-sm text-(--silver)/40">wpm</span>
			</div>
		{/if}
	</div>
</div>
