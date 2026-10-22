<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import SettingsModal from '$lib/components/SettingsModal.svelte';
	import ResultGraph from '$lib/components/ResultGraph.svelte';
	import { initCodegen, generate, randomSeed, deriveSeed } from '$lib/core/codegen';
	import {
		LANGUAGES,
		LANGUAGE_LABELS,
		TIMES,
		type Language,
		type Time
	} from '$lib/core/entity/languages';
	import type { WpmSample } from '$lib/core/stats';
	import { highlightCode } from '$lib/core/syntax';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import { onDestroy, onMount, tick } from 'svelte';
	import { toPng } from 'html-to-image';

	/** Zen: no timer, no results — just keep typing. */
	let { zen = false }: { zen?: boolean } = $props();

	// ─── Config ──────────────────────────────────────────────────────────────
	let lang = $state<Language>('go');
	let timer = $state<Time>(30);
	let modalOpen = $state(false);

	/** Token budget for a snippet. Small enough to stay readable, large
	 *  enough that a fast test never runs out of text. */
	const SNIPPET_TOKENS = 220;

	/** Number of lines skipped at the top of every generated program. */
	const SKIP_LINES = 15;

	// ─── Generated code ──────────────────────────────────────────────────────
	let code = $state('');
	let seed = $state<bigint>(0n);
	let extensions = 0;
	let wasmReady = $state(false);
	let display_chars = $derived(code.split(''));
	let syntax_kinds = $derived(highlightCode(code, lang));

	/** The first character the run actually starts on; everything before it
	 *  is dimmed context and is not counted. */
	let start_cursor = $state(0);

	// ─── Typing state ────────────────────────────────────────────────────────
	let cursor = $state(0);
	let typed: Record<number, string> = $state({});

	// ─── Game state ──────────────────────────────────────────────────────────
	type Phase = 'idle' | 'typing' | 'done';
	let phase = $state<Phase>('idle');
	let timeLeft = $state(0);
	let wpm = $state(0);
	let intervalId: ReturnType<typeof setInterval> | null = null;
	let samples = $state<WpmSample[]>([]);

	// ─── Final scores ────────────────────────────────────────────────────────
	let finalWpm = $state(0);
	let finalRawWpm = $state(0);
	let finalAccuracy = $state(0);
	let finalCorrect = $state(0);
	let finalWrong = $state(0);
	let finalTime = $state(0);

	// ─── Result export ───────────────────────────────────────────────────────
	let resultRef = $state<HTMLDivElement>();
	let actionsRef = $state<HTMLDivElement>();
	let exporting = $state(false);

	async function exportResult() {
		if (!resultRef || exporting) return;
		exporting = true;
		const actions = actionsRef;
		try {
			// Drop the action bar from the capture so the image ends cleanly
			// after the stats instead of leaving a gap.
			if (actions) actions.style.display = 'none';
			await tick();

			const background =
				getComputedStyle(document.documentElement).getPropertyValue('--bg').trim() || '#000000';
			const width = resultRef.clientWidth + 96;
			const height = resultRef.clientHeight + 96;
			const dataUrl = await toPng(resultRef, {
				backgroundColor: background,
				width,
				height,
				pixelRatio: 2,
				cacheBust: true,
				style: { overflow: 'visible', padding: '48px' }
			});
			const link = document.createElement('a');
			link.download = `osdtype-${finalWpm}wpm-${finalAccuracy}acc-${themeState.current}.png`;
			link.href = dataUrl;
			link.click();
		} catch (err) {
			console.error('Could not export result', err);
		} finally {
			if (actions) actions.style.display = '';
			exporting = false;
		}
	}

	function calcStats() {
		const elapsed = finalTime - timeLeft;
		const entries = Object.entries(typed);
		const correct = entries.filter(([i, ch]) => ch === display_chars[Number(i)]).length;
		const wrong = entries.filter(([i, ch]) => ch !== display_chars[Number(i)]).length;
		const raw = elapsed > 0 ? Math.round(entries.length / 5 / (elapsed / 60)) : 0;
		const net = elapsed > 0 ? Math.round(correct / 5 / (elapsed / 60)) : 0;
		const acc = entries.length > 0 ? Math.round((correct / entries.length) * 100) : 100;
		return { correct, wrong, raw, net, acc };
	}

	function newSnippet() {
		if (!wasmReady) return;
		seed = randomSeed();
		extensions = 0;
		const text = generate(lang, seed, SNIPPET_TOKENS);
		const parts = text.split('\n');

		// Offset of the start of every line.
		const offsets: number[] = [];
		let acc = 0;
		for (const part of parts) {
			offsets.push(acc);
			acc += part.length + 1;
		}

		// Skip the boilerplate, then start on a line that actually has
		// something to type (not an empty brace or a blank line). Any
		// qualifying line past the skip is fair game, which keeps runs varied.
		const candidates: number[] = [];
		for (let i = SKIP_LINES; i < parts.length; i++) {
			if (parts[i].trim().length <= 4) continue;
			if (text.length - offsets[i] < 1400) continue;
			candidates.push(i);
		}

		// Pick the starting line from the seed rather than Math.random, so a
		// seed always produces the same slice and the choice spreads across
		// the whole generated program instead of favouring early lines.
		const pick = deriveSeed(seed, 1);

		const startLine =
			candidates.length > 0
				? candidates[Number(pick % BigInt(candidates.length))]
				: Math.min(SKIP_LINES, Math.max(0, parts.length - 1));

		let cursorAt = offsets[startLine];
		while (cursorAt < text.length && ' \n\t'.includes(text[cursorAt])) cursorAt++;

		start_cursor = cursorAt;
		code = text;
	}

	function startGame() {
		if (phase !== 'idle') return;
		samples = [];
		phase = 'typing';
		if (zen) return;

		timeLeft = timer;
		finalTime = timer;
		intervalId = setInterval(() => {
			timeLeft--;
			const stats = calcStats();
			wpm = stats.net;
			samples = [
				...samples,
				{
					t: finalTime - timeLeft,
					raw: stats.raw,
					net: stats.net,
					errors: stats.wrong
				}
			];
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
		samples = [];
		newSnippet();
		cursor = start_cursor;
	}

	/** Replays the current snippet without generating a new one. */
	function repeatTest() {
		if (intervalId) {
			clearInterval(intervalId);
			intervalId = null;
		}
		phase = 'idle';
		timeLeft = timer;
		wpm = 0;
		typed = {};
		samples = [];
		cursor = start_cursor;
	}

	// ─── Shortcut actions ────────────────────────────────────────────────────
	function nextLanguage() {
		const i = LANGUAGES.indexOf(lang);
		lang = LANGUAGES[(i + 1) % LANGUAGES.length];
		resetGame();
	}

	function nextTimer() {
		const i = TIMES.indexOf(timer);
		timer = TIMES[(i + 1) % TIMES.length];
		resetGame();
	}

	// ─── Cursor helpers ──────────────────────────────────────────────────────
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

	/** Global character indices grouped into visual lines (newlines excluded). */
	let lines = $derived.by(() => {
		const out: number[][] = [];
		let current: number[] = [];
		for (let i = 0; i < display_chars.length; i++) {
			if (display_chars[i] === '\n') {
				out.push(current);
				current = [];
			} else {
				current.push(i);
			}
		}
		out.push(current);
		return out;
	});

	/** Line that currently holds the caret, for the row highlight. */
	let currentLine = $derived.by(() => {
		let line = 0;
		for (let i = 0; i < cursor && i < display_chars.length; i++) {
			if (display_chars[i] === '\n') line++;
		}
		return line;
	});

	/** Colour class for a character: syntax colour only once it is typed. */
	function charClass(i: number): string {
		if (i < start_cursor) return 'text-(--completed)';
		const state = char_states[i];
		if (state === 'special') return 'text-(--space)';
		if (state === 'wrong') return 'bg-(--wrong-bg) text-(--red)';
		if (state === 'correct') {
			const kind = syntax_kinds[i] ?? 'plain';
			return kind === 'plain' || kind === 'ws' ? 'text-(--silver)' : `syn-${kind}`;
		}
		// Untyped text stays readable and neutral until it is written.
		return 'text-(--pending)';
	}

	// ─── Autoscroll ──────────────────────────────────────────────────────────
	let codeContainer = $state<HTMLDivElement | null>(null);
	let spanRefs: (HTMLSpanElement | undefined)[] = [];

	/** Registers each rendered character so the view can follow the caret. */
	function trackSpan(node: HTMLSpanElement, index: number) {
		spanRefs[index] = node;
		return {
			destroy() {
				spanRefs[index] = undefined;
			}
		};
	}

	$effect(() => {
		const pos = cursor;
		const span = spanRefs[pos];
		if (!span || !codeContainer) return;

		const spanOffsetTop = span.offsetTop;
		const targetScrollTop = spanOffsetTop - codeContainer.clientHeight / 2 + span.offsetHeight / 2;
		codeContainer.scrollTop = Math.max(0, targetScrollTop);
	});

	// ─── Input handling ──────────────────────────────────────────────────────
	let inputRef: HTMLTextAreaElement;
	function focusInput() {
		inputRef?.focus();
	}

	function handleKeydown(e: KeyboardEvent) {
		const mod = e.ctrlKey || e.metaKey;

		// App-wide shortcuts (usable mid-run, so they use a modifier).
		if (mod && e.shiftKey) {
			const key = e.key.toLowerCase();
			if (key === 't') {
				e.preventDefault();
				cycleTheme();
				return;
			}
			if (key === 'l') {
				e.preventDefault();
				nextLanguage();
				return;
			}
			if (key === 's') {
				e.preventDefault();
				nextTimer();
				return;
			}
		}

		if (e.key === 'Escape') {
			e.preventDefault();
			modalOpen = !modalOpen;
			return;
		}

		// While settings are open the typing surface is inert.
		if (modalOpen) return;

		if (e.key === 'Tab') {
			e.preventDefault();
			resetGame();
			return;
		}
		if (phase === 'done') return;
		if (mod || e.altKey) return;
		if (e.key === ' ' || e.key === 'Enter') {
			e.preventDefault();
			return;
		}
		if (e.key === 'Backspace') {
			e.preventDefault();
			if (phase === 'idle' || cursor <= start_cursor) return;
			const prev = Math.max(prevReal(cursor), start_cursor);
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

			// Top up the text before the end is reached, in every mode, so a
			// long timer never runs out of something to type.
			if (cursor > display_chars.length - 400) {
				const extraSeed = deriveSeed(seed, 1_000_000 + extensions++);
				const extra = generate(lang, extraSeed, SNIPPET_TOKENS);
				if (extra) code = code + '\n' + extra;
			}
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		timeLeft = timer;
		initCodegen().then(() => {
			wasmReady = true;
			newSnippet();
			cursor = start_cursor;
			focusInput();
		});
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	onDestroy(() => {
		if (intervalId) clearInterval(intervalId);
	});
</script>

<div class="flex h-screen flex-col overflow-hidden bg-(--bg)">
	<!-- Top bar -->
	<header class="flex items-center gap-4 border-b border-(--border) px-6 py-4">
		<button
			class="cursor-pointer font-mono text-xs text-(--silver)/45 transition-colors hover:text-(--silver)"
			onclick={() => goto(resolve('/play'))}
		>
			← hub
		</button>
		<div class="title-text text-4xl leading-none text-(--silver)">OSDTYP</div>
		<span
			class="hidden font-mono text-[10px] tracking-widest text-(--silver)/35 uppercase sm:inline"
		>
			{zen ? 'zen' : 'singleplayer'}
		</span>

		<div class="flex-1"></div>

		<!-- Live stats -->
		<div class="flex items-center gap-5">
			<div class="flex flex-col items-end">
				<span
					class="font-mono text-lg leading-none {phase === 'typing' && !zen && timeLeft <= 5
						? 'text-(--red)'
						: 'text-(--mer)'}">{zen ? '∞' : phase === 'idle' ? timer : timeLeft}</span
				>
				<span class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/35 uppercase"
					>{zen ? 'zen' : 'timer'}</span
				>
			</div>
			{#if !zen}
				<div class="h-7 w-px bg-(--border)"></div>
				<div class="flex flex-col items-end">
					<span class="font-mono text-lg leading-none text-(--mer)"
						>{phase === 'idle' ? '—' : phase === 'done' ? finalWpm : wpm}</span
					>
					<span class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/35 uppercase"
						>wpm</span
					>
				</div>
			{/if}
			<div class="h-7 w-px bg-(--border)"></div>
			<div class="flex flex-col items-end">
				<span class="font-mono text-lg leading-none text-(--mer)">{LANGUAGE_LABELS[lang]}</span>
				<span class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/35 uppercase"
					>language</span
				>
			</div>
		</div>

		<div class="ml-2 flex items-center gap-2">
			<button
				class="cursor-pointer rounded border border-(--border) px-2.5 py-1 font-mono text-[11px] text-(--silver)/55 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
				onclick={cycleTheme}
				title="cycle theme (ctrl+shift+T)"
			>
				{themeState.current}
			</button>
			<button
				class="cursor-pointer rounded border border-(--border) px-2.5 py-1 font-mono text-[11px] text-(--silver)/55 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
				onclick={() => (modalOpen = true)}
				title="settings (esc)"
			>
				settings
			</button>
		</div>
	</header>

	{#if modalOpen}
		<SettingsModal bind:lang bind:timer bind:open={modalOpen} onchange={() => resetGame()} />
	{/if}

	<textarea
		bind:this={inputRef}
		class="pointer-events-none absolute resize-none opacity-0"
		autocomplete="off"
		spellcheck="false"
		rows="1"
	></textarea>

	<!-- Code / results -->
	<div class="relative min-h-0 flex-1 border-b border-(--border)" onclick={focusInput} role="none">
		{#if phase === 'done'}
			<div class="h-full overflow-y-auto">
				<div class="flex min-h-full items-center justify-center px-8 py-5">
					<div bind:this={resultRef} class="flex w-full max-w-5xl flex-col gap-5">
						<!-- Headline -->
						<div class="flex flex-wrap items-end gap-x-10 gap-y-4">
							<div>
								<div class="font-mono text-7xl leading-none text-(--mer)">{finalWpm}</div>
								<div
									class="mt-2 font-mono text-[10px] tracking-widest text-(--silver)/40 uppercase"
								>
									wpm
								</div>
							</div>
							<div>
								<div class="font-mono text-5xl leading-none text-(--silver)">{finalAccuracy}%</div>
								<div
									class="mt-2 font-mono text-[10px] tracking-widest text-(--silver)/40 uppercase"
								>
									acc
								</div>
							</div>
							<div class="ml-auto text-right font-mono text-xs text-(--silver)/40">
								{LANGUAGE_LABELS[lang]} · {finalTime}s
							</div>
						</div>

						<ResultGraph {samples} duration={finalTime} />

						<!-- Secondary stats -->
						<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
							<div class="border border-(--border) px-3 py-2">
								<div class="font-mono text-xl text-(--silver)">{finalRawWpm}</div>
								<div
									class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/40 uppercase"
								>
									raw
								</div>
							</div>
							<div class="border border-(--border) px-3 py-2">
								<div class="font-mono text-xl text-(--mer)">{finalCorrect}</div>
								<div
									class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/40 uppercase"
								>
									correct
								</div>
							</div>
							<div class="border border-(--border) px-3 py-2">
								<div class="font-mono text-xl text-(--silver)">{finalWrong}</div>
								<div
									class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/40 uppercase"
								>
									mistakes
								</div>
							</div>
							<div class="border border-(--border) px-3 py-2">
								<div class="font-mono text-xl text-(--silver)">{finalTime}s</div>
								<div
									class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/40 uppercase"
								>
									time
								</div>
							</div>
						</div>

						<!-- What next -->
						<div
							bind:this={actionsRef}
							class="flex flex-wrap items-center gap-2 border-t border-(--border) pt-5"
						>
							<button class="result-action" onclick={resetGame}>
								next test <span class="kbd">tab</span>
							</button>
							<button class="result-action" onclick={repeatTest}>repeat</button>
							<button class="result-action" onclick={() => (modalOpen = true)}>
								settings <span class="kbd">esc</span>
							</button>
							<button
								class="result-action disabled:cursor-default disabled:opacity-40"
								onclick={exportResult}
								disabled={exporting}
							>
								{exporting ? 'exporting…' : 'export png'}
							</button>
							<button class="result-action" onclick={() => goto(resolve('/play'))}>hub</button>
							<span class="ml-auto font-mono text-[11px] text-(--silver)/30">
								ctrl+shift+t theme · ctrl+shift+l language
							</span>
						</div>
					</div>
				</div>
			</div>
		{:else}
			<!-- Code display -->
			<div class="relative h-full border-l border-(--border)">
				<div
					bind:this={codeContainer}
					class="h-full overflow-hidden px-8 py-6 font-mono text-2xl leading-relaxed"
				>
					{#each lines as line, li (li)}
						<div class="code-line" class:code-line-active={li === currentLine}>
							<span class="line-no">{li + 1}</span>
							<span class="line-code"
								>{#each line as i (i)}<span
										use:trackSpan={i}
										class={charClass(i)}
										class:char-caret={i === cursor}>{display_chars[i]}</span
									>{/each}</span
							>
						</div>
					{/each}
				</div>
			</div>

			<!-- Idle overlay -->
			{#if phase === 'idle'}
				<div
					class="absolute inset-0 flex items-center justify-center"
					style="background: color-mix(in srgb, var(--bg) 52%, transparent);"
				>
					<div
						class="flex flex-col items-center gap-3 rounded border border-(--border) bg-(--fbg) px-8 py-6"
					>
						<span class="font-mono text-sm tracking-widest text-(--silver)/70 uppercase">
							press any key to begin
						</span>
						<span class="font-mono text-[11px] text-(--silver)/35">
							<span class="kbd">tab</span> restart · <span class="kbd">esc</span> settings
						</span>
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Shortcut bar -->
	<footer
		class="flex flex-wrap items-center gap-x-5 gap-y-2 px-6 py-3 font-mono text-[11px] text-(--silver)/45"
	>
		<button
			class="flex cursor-pointer items-center gap-2 transition-colors hover:text-(--silver)"
			onclick={resetGame}
		>
			<span class="kbd">tab</span> restart
		</button>
		<span class="flex items-center gap-2"><span class="kbd">esc</span> settings</span>
		<span class="flex items-center gap-2"><span class="kbd">ctrl+shift+T</span> theme</span>
		<span class="flex items-center gap-2"><span class="kbd">ctrl+shift+L</span> language</span>
		<span class="flex items-center gap-2"><span class="kbd">ctrl+shift+S</span> timer</span>
	</footer>
</div>
