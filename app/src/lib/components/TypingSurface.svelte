<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import SettingsModal from '$lib/components/SettingsModal.svelte';
	import ExportModal from '$lib/components/ExportModal.svelte';
	import ReplayModal from '$lib/components/ReplayModal.svelte';
	import ResultGraph from '$lib/components/ResultGraph.svelte';
	import SpeedGraph from '$lib/components/SpeedGraph.svelte';
	import { initCodegen, generate, randomSeed, deriveSeed } from '$lib/core/codegen';
	import {
		LANGUAGES,
		LANGUAGE_LABELS,
		TIMES,
		type Language,
		type Time
	} from '$lib/core/entity/languages';
	import type { ExportStat, SpeedSeries, WpmSample } from '$lib/core/stats';
	import { highlightCode } from '$lib/core/syntax';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import { appState } from '$lib/core/state/state.svelte';
	import { createPracticeRun, savePracticeRun } from '$lib/core/api/game';
	import { errorMessage } from '$lib/core/api/client';
	import { KeypressAction } from '$lib/core/api/types';
	import type { ClientKeypress, PracticeRunView, ReplayRun } from '$lib/core/api/types';
	import { profile } from '$lib/core/api/users';
	import {
		DEFAULT_PACE_WPM,
		PRACTICE_BOTS,
		estimatePace,
		targetForBot,
		type PracticeBotID,
		type PracticeBotProfile
	} from '$lib/core/practice-bots';
	import { mapTypedToDisplay, nonWhitespace, syntheticSamples } from '$lib/core/replay';
	import { onDestroy, onMount } from 'svelte';

	/** Zen: no timer, no results — just keep typing. */
	let { zen = false }: { zen?: boolean } = $props();

	let guest = $derived(!appState.user);
	let loginHref = $derived(`/login?next=${encodeURIComponent(page.url.pathname)}`);

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
	/** Non-whitespace characters from the start point, shared by the bots. */
	let targetCharacters = $derived(nonWhitespace(display_chars.slice(start_cursor)));

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

	// ─── Practice bots (logged-in singleplayer) ──────────────────────────────
	let selectedBots = $state<PracticeBotID[]>([]);
	let effectiveBots = $derived(guest ? [] : selectedBots);
	let paceBase = $state(DEFAULT_PACE_WPM);
	let paceRuns = $state(0);
	let racePaceBase = $state(DEFAULT_PACE_WPM);
	let racePaceRuns = $state(0);
	let paceLoading = $state(false);
	let paceRequest = 0;

	interface LiveBot {
		id: PracticeBotID;
		label: string;
		color: string;
		targetWpm: number;
		accuracy: number;
		typed: string[];
		correct: number;
		wrong: number;
		totalTyped: number;
		finished: boolean;
		backspacePending: boolean;
		accum: number;
		rngState: number;
		samples: WpmSample[];
		lastSampleSec: number;
	}

	let liveBots = $state<LiveBot[]>([]);
	let botElapsedMs = $state(0);
	let botIntervalId: ReturnType<typeof setInterval> | null = null;

	// ─── Recorded practice run / replay / export ─────────────────────────────
	let runEvents = $state<ClientKeypress[]>([]);
	let runStartedAt = 0;
	let runGeneration = 0;
	let practiceRun = $state<PracticeRunView | null>(null);
	let practiceSaving = $state(false);
	let practiceSaveBusy = $state(false);
	let practiceError = $state('');
	let practiceSaved = $derived(practiceRun !== null && !practiceRun.temporary);
	let replayOpen = $state(false);
	let exportOpen = $state(false);

	function botProfile(id: PracticeBotID): PracticeBotProfile {
		return PRACTICE_BOTS.find((bot) => bot.id === id) ?? PRACTICE_BOTS[0];
	}

	function botTarget(id: PracticeBotID): number {
		return targetForBot(botProfile(id), paceBase);
	}

	function toggleBot(id: PracticeBotID) {
		if (guest) return;
		selectedBots = selectedBots.includes(id)
			? selectedBots.filter((bot) => bot !== id)
			: [...selectedBots, id];
	}

	$effect(() => {
		const uid = appState.user?.id ?? 0;
		if (!uid || zen) return;
		void loadPace();
	});

	async function loadPace() {
		if (guest || zen || !appState.user) return;
		const request = ++paceRequest;
		paceLoading = true;
		try {
			const data = await profile(appState.user.username);
			const pace = estimatePace(data.history);
			if (request === paceRequest) {
				paceBase = pace.wpm;
				paceRuns = pace.runs;
			}
		} catch {
			// The default pace remains a playable fallback if history is unavailable.
		} finally {
			if (request === paceRequest) paceLoading = false;
		}
	}

	function nextBotRandom(bot: LiveBot): number {
		bot.rngState = (Math.imul(bot.rngState, 1664525) + 1013904223) >>> 0;
		return bot.rngState / 4294967296;
	}

	function stepBot(bot: LiveBot, deltaMs: number, elapsedMs: number) {
		if (bot.finished) return;
		bot.accum += ((bot.targetWpm * 5) / 60) * (deltaMs / 1000);
		let guard = 0;
		while (bot.accum >= 1 && guard < 40 && !bot.finished) {
			bot.accum -= 1;
			guard++;

			if (bot.backspacePending) {
				bot.typed.pop();
				bot.backspacePending = false;
				continue;
			}

			const expected = targetCharacters[bot.typed.length];
			if (expected === undefined) {
				bot.finished = true;
				break;
			}

			bot.totalTyped++;
			if (nextBotRandom(bot) * 100 >= bot.accuracy) {
				let wrong = 'x';
				if (expected === wrong) wrong = 'z';
				if (expected === wrong) wrong = 'q';
				bot.typed.push(wrong);
				bot.wrong++;
				bot.backspacePending = true;
			} else {
				bot.typed.push(expected);
				bot.correct++;
			}
		}

		const second = Math.min(timer, Math.floor(elapsedMs / 1000));
		while (bot.lastSampleSec < second) {
			bot.lastSampleSec++;
			const t = bot.lastSampleSec;
			bot.samples.push({
				t,
				raw: (bot.totalTyped / 5) / (t / 60),
				net: (bot.correct / 5) / (t / 60),
				errors: bot.wrong
			});
		}
	}

	function stepBots() {
		if (phase !== 'typing' || liveBots.length === 0) return;
		const elapsed = Math.min(timer * 1000, Date.now() - runStartedAt);
		if (elapsed <= botElapsedMs) return;
		const delta = Math.min(1500, elapsed - botElapsedMs);
		botElapsedMs = elapsed;
		for (const bot of liveBots) stepBot(bot, delta, elapsed);
	}

	function stopBots() {
		if (botIntervalId) {
			clearInterval(botIntervalId);
			botIntervalId = null;
		}
	}

	function beginBotRace() {
		stopBots();
		botElapsedMs = 0;
		racePaceBase = paceBase;
		racePaceRuns = paceRuns;
		liveBots = effectiveBots.map((id, index) => {
			const profile = botProfile(id);
			const targetWpm = targetForBot(profile, paceBase);
			return {
				id,
				label: profile.label,
				color: profile.color,
				targetWpm,
				accuracy: profile.accuracy,
				typed: [],
				correct: 0,
				wrong: 0,
				totalTyped: 0,
				finished: false,
				backspacePending: false,
				accum: 0,
				rngState: ((runGeneration + 1) * 2654435761 + index * 7919 + targetWpm * 97) >>> 0,
				samples: [{ t: 0, raw: 0, net: 0, errors: 0 }],
				lastSampleSec: 0
			};
		});
		if (liveBots.length === 0) return;
		botIntervalId = setInterval(stepBots, 100);
	}

	function finalizeBotRace() {
		if (liveBots.length === 0) return;
		const elapsed = Math.max(1000, (finalTime || timer) * 1000);
		const remaining = Math.max(0, elapsed - botElapsedMs);
		for (const bot of liveBots) {
			if (remaining > 0) stepBot(bot, remaining, elapsed);
			if (bot.backspacePending) {
				bot.typed.pop();
				bot.backspacePending = false;
			}
			while (bot.lastSampleSec < timer) {
				bot.lastSampleSec++;
				const t = bot.lastSampleSec;
				bot.samples.push({
					t,
					raw: (bot.totalTyped / 5) / (t / 60),
					net: (bot.correct / 5) / (t / 60),
					errors: bot.wrong
				});
			}
		}
		botElapsedMs = elapsed;
		stopBots();
	}

	function botLiveWpm(bot: LiveBot): number {
		const seconds = Math.max(1, botElapsedMs / 1000);
		return Math.round((bot.correct / 5) / (seconds / 60));
	}

	function botFinalWpm(bot: LiveBot): number {
		const last = bot.samples[bot.samples.length - 1];
		return last ? Math.round(last.net) : 0;
	}

	function botProgress(bot: LiveBot): number {
		return Math.min(100, Math.round((bot.correct / Math.max(1, targetCharacters.length)) * 100));
	}

	function recordEvent(action: KeypressAction, value: string) {
		if (zen) return;
		runEvents.push({
			value,
			action,
			time_ms: Math.max(0, Date.now() - runStartedAt)
		});
	}

	function typedText(): string {
		return Object.entries(typed)
			.sort((a, b) => Number(a[0]) - Number(b[0]))
			.map(([, char]) => char)
			.join('');
	}

	function startOffsetBytes(): number {
		return new TextEncoder().encode(code.slice(0, start_cursor)).length;
	}

	async function storePracticeRun(generation: number) {
		if (guest || runEvents.length === 0) return;
		practiceSaving = true;
		practiceError = '';
		try {
			const run = await createPracticeRun({
				language: lang,
				duration: timer,
				duration_ms: timer * 1000,
				seed: seed.toString(),
				snippet: code,
				start_offset: startOffsetBytes(),
				final: typedText(),
				events: [...runEvents]
			});
			if (generation === runGeneration) practiceRun = run;
		} catch (err) {
			if (generation === runGeneration) practiceError = errorMessage(err);
		} finally {
			if (generation === runGeneration) practiceSaving = false;
		}
	}

	async function saveRun() {
		if (!practiceRun || practiceSaved || practiceSaveBusy) return;
		practiceSaveBusy = true;
		practiceError = '';
		try {
			practiceRun = await savePracticeRun(practiceRun.id);
			void loadPace();
		} catch (err) {
			practiceError = errorMessage(err);
		} finally {
			practiceSaveBusy = false;
		}
	}

	function practiceShareURL(): string {
		if (!practiceRun || typeof location === 'undefined' || !appState.user) return '';
		return `${location.origin}/replay/${practiceRun.id}/${appState.user.id}`;
	}

	let localRun = $derived.by<ReplayRun | null>(() => {
		if (finalTime <= 0 || runEvents.length === 0) return null;
		return {
			user_id: appState.user?.id ?? 0,
			username: appState.user?.username ?? 'guest',
			avatar_url: appState.user?.avatar_url ?? '',
			wpm: finalWpm,
			raw: finalRawWpm,
			accuracy: finalAccuracy / 100,
			correct: finalCorrect,
			wrong: finalWrong,
			is_bot: false,
			final: typedText(),
			events: [...runEvents]
		};
	});

	let resultSeries = $derived.by<SpeedSeries[]>(() => {
		const duration = finalTime || timer;
		const userSamples =
			samples.length > 1
				? samples
				: syntheticSamples(finalWpm, finalAccuracy / 100, duration);
		return [
			{
				label: appState.user?.username ?? 'you',
				samples: userSamples,
				color: 'var(--mer)'
			},
			...liveBots.map((bot) => ({
				label: bot.label,
				samples:
					bot.samples.length > 1
						? bot.samples
						: syntheticSamples(botFinalWpm(bot), bot.accuracy / 100, duration),
				color: bot.color,
				isBot: true
			}))
		];
	});

	let resultStats = $derived<ExportStat[]>([
		{ label: 'wpm', value: finalWpm, accent: true },
		{ label: 'acc', value: `${finalAccuracy}%` },
		{ label: 'raw', value: finalRawWpm },
		{ label: 'time', value: `${finalTime}s` }
	]);

	interface RaceResult {
		id: string;
		label: string;
		color: string;
		wpm: number;
		accuracy: number;
		progress: number;
		isYou: boolean;
		place: number;
	}

	let raceResults = $derived.by<RaceResult[]>(() => {
		if (liveBots.length === 0) return [];
		const total = Math.max(1, targetCharacters.length);
		const entries = [
			{
				id: 'you',
				label: appState.user?.username ?? 'you',
				color: 'var(--mer)',
				wpm: finalWpm,
				accuracy: finalAccuracy,
				progress: Math.min(100, Math.round((finalCorrect / total) * 100)),
				isYou: true
			},
			...liveBots.map((bot) => ({
				id: bot.id,
				label: bot.label,
				color: bot.color,
				wpm: botFinalWpm(bot),
				accuracy: bot.accuracy,
				progress: botProgress(bot),
				isYou: false
			}))
		];
		return entries
			.sort((a, b) => b.wpm - a.wpm)
			.map((entry, index) => ({ ...entry, place: index + 1 }));
	});

	/** Each bot's current display cursor, used to paint its coloured caret. */
	let botDisplays = $derived.by(() => {
		const offset = startOffsetBytes();
		return liveBots.map((bot) => ({
			bot,
			display: mapTypedToDisplay(code, offset, bot.typed)
		}));
	});

	let botCarets = $derived.by(() => {
		const byIndex = new Map<number, LiveBot[]>();
		for (const entry of botDisplays) {
			const index = entry.display.cursor;
			if (index >= display_chars.length) continue;
			const list = byIndex.get(index) ?? [];
			list.push(entry.bot);
			byIndex.set(index, list);
		}
		return byIndex;
	});

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
		runGeneration++;
		runEvents = [];
		runStartedAt = Date.now();
		practiceRun = null;
		practiceSaving = false;
		practiceError = '';
		practiceSaveBusy = false;
		replayOpen = false;
		exportOpen = false;
		phase = 'typing';
		if (zen) {
			liveBots = [];
			return;
		}

		beginBotRace();
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
		finalizeBotRace();
		phase = 'done';
		void storePracticeRun(runGeneration);
	}

	function resetGame() {
		if (intervalId) {
			clearInterval(intervalId);
			intervalId = null;
		}
		stopBots();
		liveBots = [];
		botElapsedMs = 0;
		runGeneration++;
		phase = 'idle';
		timeLeft = timer;
		wpm = 0;
		typed = {};
		samples = [];
		runEvents = [];
		practiceRun = null;
		practiceSaving = false;
		practiceError = '';
		practiceSaveBusy = false;
		replayOpen = false;
		exportOpen = false;
		newSnippet();
		cursor = start_cursor;
	}

	/** Replays the current snippet without generating a new one. */
	function repeatTest() {
		if (intervalId) {
			clearInterval(intervalId);
			intervalId = null;
		}
		stopBots();
		liveBots = [];
		botElapsedMs = 0;
		runGeneration++;
		phase = 'idle';
		timeLeft = timer;
		wpm = 0;
		typed = {};
		samples = [];
		runEvents = [];
		practiceRun = null;
		practiceSaving = false;
		practiceError = '';
		practiceSaveBusy = false;
		replayOpen = false;
		exportOpen = false;
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
			recordEvent(KeypressAction.BACKSPACE, '');
			return;
		}
		if (e.key.length === 1) {
			e.preventDefault();
			if (phase === 'idle') startGame();
			recordEvent(KeypressAction.KEYPRESS, e.key);
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
		stopBots();
	});
</script>

<div class="flex h-screen flex-col overflow-hidden bg-(--bg)">
	<!-- Top bar -->
	<header class="flex items-center gap-4 border-b border-(--border) px-6 py-4">
		<button
			class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase transition-colors hover:border-(--mer) hover:text-(--mer)"
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

	{#if !zen && phase === 'typing' && liveBots.length > 0}
		<div
			class="flex flex-wrap items-center gap-x-5 gap-y-1 border-b border-(--border) bg-(--fbg) px-6 py-2 font-mono text-[10px]"
		>
			<span class="tracking-[0.14em] text-(--silver)/30 uppercase">race</span>
			{#each liveBots as bot (bot.id)}
				<div class="flex items-center gap-1.5">
					<span class="h-2 w-2" style={`background:${bot.color}`}></span>
					<span class="text-(--silver)/70">{bot.label}</span>
					<span class="text-(--mer)/60">{botLiveWpm(bot)} wpm</span>
					<span class="text-(--silver)/22">{botProgress(bot)}%</span>
				</div>
			{/each}
			<span class="ml-auto text-[9px] text-(--silver)/20">
				bot pace follows your {racePaceBase} wpm form
			</span>
		</div>
	{/if}

	<!-- Code / results -->
	<div class="relative min-h-0 flex-1 border-b border-(--border)" onclick={focusInput} role="none">
		{#if phase === 'done'}
			<div class="h-full overflow-y-auto">
				<div class="flex min-h-full items-center justify-center px-8 py-5">
					<div class="flex w-full max-w-5xl flex-col gap-5">
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

						{#if raceResults.length > 0}
							<div class="border border-(--border) bg-(--fbg)">
								<div class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-4 py-2.5">
									<div>
										<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase">
											race result
										</div>
										<div class="mt-1 font-mono text-[9px] text-(--silver)/25">
											bots calibrated from your last {racePaceRuns || 0} run{racePaceRuns === 1 ? '' : 's'} · base {racePaceBase} wpm
										</div>
									</div>
									<div class="font-mono text-[9px] text-(--mer)/60 uppercase">
										{raceResults.find((entry) => entry.isYou)?.place === 1 ? 'you took it' : 'chase them down'}
									</div>
								</div>
								<div class="divide-y divide-(--border)">
									{#each raceResults as entry (entry.id)}
										<div
											class={`flex flex-wrap items-center gap-3 px-4 py-2.5 ${entry.isYou ? 'bg-(--mer)/5' : ''}`}
										>
											<span class="w-5 font-mono text-[11px] text-(--silver)/30">{entry.place}</span>
											<span class="h-2 w-2 shrink-0" style={`background:${entry.color}`}></span>
											<span class="min-w-0 flex-1 truncate font-mono text-xs" style={`color:${entry.isYou ? 'var(--mer)' : 'var(--silver)'}`}>
												{entry.label}
											</span>
											{#if entry.isYou}
												<span class="font-mono text-[8px] tracking-[0.1em] text-(--mer)/50 uppercase">you</span>
											{:else}
												<span class="font-mono text-[8px] tracking-[0.1em] text-(--silver)/25 uppercase">bot</span>
											{/if}
											<span class="font-mono text-xs text-(--mer)">{entry.wpm} <span class="text-[8px] text-(--silver)/25">wpm</span></span>
											<span class="font-mono text-[10px] text-(--silver)/35">{entry.accuracy}% acc</span>
											<span class="font-mono text-[10px] text-(--silver)/25">{entry.progress}%</span>
										</div>
									{/each}
								</div>
							</div>
						{/if}

						{#if effectiveBots.length > 0}
							<div class="border border-(--border) bg-(--fbg) px-3 py-2">
								<div class="flex flex-wrap items-center justify-between gap-2 px-2 pt-1">
									<div>
										<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase">
											combined speed · you + bots
										</div>
										<div class="mt-1 font-mono text-[9px] text-(--silver)/25">
											coloured lines are the bots' actual simulated race
										</div>
									</div>
									<div class="font-mono text-[9px] text-(--mer)/70 uppercase">
										{effectiveBots.length} bot{effectiveBots.length === 1 ? '' : 's'}
									</div>
								</div>
								<SpeedGraph series={resultSeries} duration={finalTime} />
							</div>
						{/if}

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

						<!-- Practice storage -->
						{#if !guest}
							<div class="flex flex-wrap items-center gap-3 border border-(--border) bg-(--fbg) px-3 py-2.5">
								<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/35 uppercase">
									latest practice
								</span>
								{#if practiceSaving}
									<span class="font-mono text-[11px] text-(--silver)/45">storing temporary run…</span>
								{:else if practiceError}
									<span class="font-mono text-[11px] text-(--red)">{practiceError}</span>
								{:else if practiceSaved}
									<span class="font-mono text-[11px] text-(--mer)">saved to profile</span>
									<a
										href={`/profile/${encodeURIComponent(appState.user?.username ?? '')}`}
										class="font-mono text-[10px] text-(--silver)/40 underline-offset-2 hover:text-(--mer) hover:underline"
										>view run =></a
									>
								{:else if practiceRun}
									<span class="font-mono text-[11px] text-(--silver)/40">temporary · replaced next run</span>
									<button
										class="cursor-pointer border border-(--mer) px-3 py-1 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10 disabled:opacity-40"
										onclick={saveRun}
										disabled={practiceSaveBusy}
										>{practiceSaveBusy ? 'saving…' : 'save run =>'}</button
									>
								{:else}
									<span class="font-mono text-[11px] text-(--silver)/30">
										no keystrokes stored
									</span>
								{/if}
								<span class="ml-auto font-mono text-[9px] text-(--silver)/25">
									temporary runs stay out of history until saved
								</span>
							</div>
						{/if}

						<!-- What next -->
						<div
							class="flex flex-wrap items-center gap-2 border-t border-(--border) pt-5"
						>
							<button class="result-action" onclick={resetGame}>
								next test <span class="kbd">tab</span>
							</button>
							<button class="result-action" onclick={repeatTest}>repeat</button>
							<button
								class="result-action disabled:cursor-default disabled:opacity-40"
								onclick={() => (replayOpen = true)}
								disabled={!localRun}
							>
								replay
							</button>
							<button class="result-action" onclick={() => (modalOpen = true)}>
								settings <span class="kbd">esc</span>
							</button>
							<button
								class="result-action"
								onclick={() => (exportOpen = true)}
							>
								export png
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
								>{#each line as i (i)}
									{@const carets = botCarets.get(i) ?? []}<span
										use:trackSpan={i}
										class={charClass(i)}
										class:char-caret={i === cursor}
										class:bot-caret-host={carets.length > 0}
										>{#if carets.length > 0}<span class="bot-caret-stack" aria-hidden="true"
												>{#each carets as bot, depth (bot.id)}<span
														class="bot-caret-mark"
														style={`left:${depth * 3}px;background:${bot.color}`}
													></span
												>{/each}</span
											>{/if}{display_chars[i]}</span
									>
								{/each}</span
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
						{#if !guest && !zen}
							<div class="mt-2 w-full border-t border-(--border) pt-3">
								<div class="flex items-center justify-between gap-3">
									<span class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
										add bots · singleplayer
									</span>
									<span class="font-mono text-[9px] text-(--silver)/25">
										{paceLoading
											? 'calibrating pace…'
											: `your pace ${paceBase} wpm${paceRuns > 0 ? ` · last ${paceRuns}` : ' · default'}`}
									</span>
								</div>
								<div class="mt-2 flex flex-wrap justify-center gap-1.5">
									{#each PRACTICE_BOTS as bot (bot.id)}
										<button
											class={`flex cursor-pointer items-center gap-1.5 border px-2 py-1 font-mono text-[10px] transition-colors ${
												selectedBots.includes(bot.id)
													? 'border-(--silver)/40 text-(--silver)'
													: 'border-(--border) text-(--silver)/40 hover:border-(--silver)/40 hover:text-(--silver)'
											}`}
											onclick={() => toggleBot(bot.id)}
											title={`${botTarget(bot.id)} wpm · ${bot.accuracy}% accuracy`}
										>
											<span
												class="h-1.5 w-1.5 shrink-0"
												style={`background:${bot.color}`}
											></span>
											{selectedBots.includes(bot.id) ? '−' : '+'} {bot.label}
											<span class="text-[9px] opacity-50">{botTarget(bot.id)}</span>
										</button>
									{/each}
								</div>
								<div class="mt-1.5 text-center font-mono text-[9px] text-(--silver)/20">
									bot speeds follow an exponential moving average of your latest runs
								</div>
							</div>
						{/if}
						{#if guest}
							<a
								href={loginHref}
								class="mt-1 border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase transition-colors hover:border-(--mer) hover:text-(--mer)"
							>
								guest mode · sign up to save stats =>
							</a>
						{/if}
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<!-- Shortcut bar -->
	<div
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
		{#if effectiveBots.length > 0 && !zen}
			<span class="flex items-center gap-2 text-(--mer)/60">
				bots: {effectiveBots.map((id) => botProfile(id).label).join(' · ')}
			</span>
		{/if}
		{#if guest}
			<a
				href={loginHref}
				class="ml-auto text-(--mer)/70 transition-colors hover:text-(--mer)"
			>
				guest mode · sign up for ranked + saved stats =>
			</a>
		{/if}
	</div>
</div>

{#if replayOpen && localRun}
	<ReplayModal
		run={localRun}
		snippet={code}
		startOffset={startOffsetBytes()}
		duration={finalTime}
		language={lang}
		shareURL={practiceShareURL()}
		onclose={() => (replayOpen = false)}
	/>
{/if}

{#if exportOpen}
	<ExportModal
		title={`practice · ${appState.user?.username ?? 'guest'}`}
		subtitle={`${LANGUAGE_LABELS[lang]} · ${finalTime}s · ${finalWpm} wpm`}
		series={resultSeries}
		duration={finalTime}
		stats={resultStats}
		filename={`osdtype-practice-${finalWpm}wpm`}
		onclose={() => (exportOpen = false)}
	/>
{/if}
