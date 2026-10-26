<script lang="ts">
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
	import { promptState } from '$lib/core/prompts.svelte';
	import {
		DEFAULT_WHITESPACE_MODE,
		WHITESPACE_MODES,
		WHITESPACE_MODE_HINTS,
		WHITESPACE_MODE_LABELS,
		type WhitespaceMode
	} from '$lib/core/whitespace';
	import { onDestroy, onMount } from 'svelte';

	/** Zen: no timer, no results — just keep typing. */
	let { zen = false }: { zen?: boolean } = $props();

	let guest = $derived(!appState.user);
	let saveRunData = $derived(appState.user?.save_run_data ?? true);

	// ─── Config ──────────────────────────────────────────────────────────────
	let lang = $state<Language>('go');
	let timer = $state<Time>(30);
	let whitespaceMode = $state<WhitespaceMode>(DEFAULT_WHITESPACE_MODE);
	let modalOpen = $state(false);
	let botsOpen = $state(false);

	const PREFS_KEY = 'osdtype.test.v1';

	function loadPrefs() {
		try {
			const raw = localStorage.getItem(PREFS_KEY);
			if (!raw) return;
			const saved = JSON.parse(raw) as {
				lang?: string;
				timer?: number;
				whitespaceMode?: string;
			};
			if (LANGUAGES.includes(saved.lang as Language)) lang = saved.lang as Language;
			if (TIMES.includes(saved.timer as Time)) timer = saved.timer as Time;
			if (WHITESPACE_MODES.includes(saved.whitespaceMode as WhitespaceMode)) {
				whitespaceMode = saved.whitespaceMode as WhitespaceMode;
			}
		} catch {
			// Preferences are helpful, not required. Keep the defaults.
		}
	}

	function savePrefs() {
		try {
			localStorage.setItem(PREFS_KEY, JSON.stringify({ lang, timer, whitespaceMode }));
		} catch {
			// Storage can be unavailable in private contexts.
		}
	}

	function setLanguage(next: Language) {
		if (next === lang) return;
		lang = next;
		savePrefs();
		resetGame();
		focusInput();
	}

	function setTimer(next: Time) {
		if (next === timer) return;
		timer = next;
		savePrefs();
		resetGame();
		focusInput();
	}

	function setWhitespaceMode(next: WhitespaceMode) {
		if (next === whitespaceMode || phase === 'typing') return;
		whitespaceMode = next;
		savePrefs();
		if (phase === 'done') resetGame();
		else focusInput();
	}

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

	/** The first character the run actually starts on. The snippet is sliced
	 *  at the chosen line, so this is only the indentation before the cursor. */
	let start_cursor = $state(0);
	/** Non-whitespace characters from the start point, shared by the bots. */
	let targetCharacters = $derived(nonWhitespace(display_chars.slice(start_cursor)));

	// ─── Typing state ────────────────────────────────────────────────────────
	let cursor = $state(0);
	let typed: Record<number, string> = $state({});
	/** Whitespace the player skipped when it was required by the active mode. */
	let missed: Record<number, true> = $state({});
	/** Wrong whitespace keystrokes that did not consume a target character. */
	let extraWrong = $state(0);

	interface UndoStep {
		/** Display index this keystroke wrote, or null for an extra keystroke. */
		typedIndex: number | null;
		value: string;
		/** Whitespace positions skipped by this keystroke that were counted as errors. */
		missed: number[];
		/** Whitespace positions skipped before the typed character, error or not. */
		skipped: number[];
		extra: boolean;
		/** Cursor position restored when this keystroke is undone. */
		restoreTo: number;
	}

	let undoStack = $state<UndoStep[]>([]);
	let typedCount = $derived(
		Object.keys(typed).filter((index) => !isSpecial(display_chars[Number(index)])).length
	);
	let typingProgress = $derived(
		Math.min(100, Math.round((typedCount / Math.max(1, targetCharacters.length)) * 100))
	);

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
				raw: bot.totalTyped / 5 / (t / 60),
				net: bot.correct / 5 / (t / 60),
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
					raw: bot.totalTyped / 5 / (t / 60),
					net: bot.correct / 5 / (t / 60),
					errors: bot.wrong
				});
			}
		}
		botElapsedMs = elapsed;
		stopBots();
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
		if (guest || !saveRunData || runEvents.length === 0) return;
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
			samples.length > 1 ? samples : syntheticSamples(finalWpm, finalAccuracy / 100, duration);
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
		// This map is a local lookup rebuilt by the derivation, not reactive state.
		// eslint-disable-next-line svelte/prefer-svelte-reactivity
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

	function fmtClock(seconds: number): string {
		const safe = Math.max(0, Math.round(seconds));
		return `${Math.floor(safe / 60)
			.toString()
			.padStart(2, '0')}:${(safe % 60).toString().padStart(2, '0')}`;
	}

	function calcStats() {
		const elapsed = finalTime - timeLeft;
		// Correct whitespace is shown in the surface but, like the server's
		// scoring, it does not count toward the printable-character totals.
		// Wrong whitespace still counts because manual mode treats spaces and
		// returns as ordinary characters.
		const entries = Object.entries(typed).filter(([i, ch]) => {
			const target = display_chars[Number(i)];
			return !(isSpecial(ch) && ch === target);
		});
		const correct = entries.filter(([i, ch]) => ch === display_chars[Number(i)]).length;
		const wrong =
			entries.filter(([i, ch]) => ch !== display_chars[Number(i)]).length +
			Object.keys(missed).length +
			extraWrong;
		const raw = elapsed > 0 ? Math.round(entries.length / 5 / (elapsed / 60)) : 0;
		const net = elapsed > 0 ? Math.round(correct / 5 / (elapsed / 60)) : 0;
		const total = correct + wrong;
		const acc = total > 0 ? Math.round((correct / total) * 100) : 100;
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

		// Short fallback snippets have no candidate with a long enough tail.
		// Starting at their last line can point at the final newline, which
		// makes the target empty. Begin at the top instead.
		const startLine =
			candidates.length > 0 ? candidates[Number(pick % BigInt(candidates.length))] : 0;

		let cursorAt = offsets[startLine];
		while (cursorAt < text.length && ' \n\t'.includes(text[cursorAt])) cursorAt++;

		// Keep the middle-of-snippet start, but make the chosen line the first
		// visible line. Only the indentation before the cursor remains as
		// context; no completed code sits above the run.
		const lineStart = offsets[startLine];
		start_cursor = cursorAt - lineStart;
		code = text.slice(lineStart);
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
		typed = {};
		missed = {};
		extraWrong = 0;
		undoStack = [];
		cursor = start_cursor;
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
		missed = {};
		extraWrong = 0;
		undoStack = [];
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
		missed = {};
		extraWrong = 0;
		undoStack = [];
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
		setLanguage(LANGUAGES[(i + 1) % LANGUAGES.length]);
	}

	function nextTimer() {
		const i = TIMES.indexOf(timer);
		setTimer(TIMES[(i + 1) % TIMES.length]);
	}

	// ─── Cursor helpers ──────────────────────────────────────────────────────
	function isSpecial(char: string) {
		return char === ' ' || char === '\n' || char === '\t' || char === '\r';
	}

	function isNewline(char: string | undefined) {
		return char === '\n' || char === '\r';
	}

	function skipAllWhitespace(pos: number) {
		while (pos < display_chars.length && isSpecial(display_chars[pos])) pos++;
		return pos;
	}

	/**
	 * Tabs and the indentation at the start of a line always advance without
	 * input. Everything after the indentation is typed by the player, including
	 * spaces and returns (except in auto mode).
	 */
	function autoWhitespace(from: number) {
		const skipped: number[] = [];
		let pos = from;

		while (pos < display_chars.length && display_chars[pos] === '\t') {
			skipped.push(pos);
			pos++;
		}

		let lineStart = pos;
		while (lineStart > 0 && !isNewline(display_chars[lineStart - 1])) lineStart--;
		let onlyIndent = true;
		for (let i = lineStart; i < pos; i++) {
			if (display_chars[i] !== ' ' && display_chars[i] !== '\t') {
				onlyIndent = false;
				break;
			}
		}
		if (onlyIndent) {
			while (
				pos < display_chars.length &&
				(display_chars[pos] === ' ' || display_chars[pos] === '\t')
			) {
				skipped.push(pos);
				pos++;
			}
		}

		return { pos, skipped };
	}

	/** Place the caret on the next position the active mode expects. */
	function nextCaretAfter(pos: number) {
		const auto = autoWhitespace(pos);
		if (whitespaceMode === 'auto') return skipAllWhitespace(auto.pos);
		return auto.pos;
	}

	/**
	 * Walks whitespace before the next printable character. Tabs and line
	 * indentation always pass. Adaptive also lets spaces pass and counts skipped
	 * returns as mistakes; manual stops at the first non-indentation character.
	 */
	function collectSkippedWhitespace(from: number) {
		const auto = autoWhitespace(from);
		const skipped = [...auto.skipped];
		const missedPositions: number[] = [];
		let pos = auto.pos;

		if (whitespaceMode === 'auto') {
			while (pos < display_chars.length && isSpecial(display_chars[pos])) {
				skipped.push(pos);
				pos++;
			}
		} else if (whitespaceMode === 'adaptive') {
			while (pos < display_chars.length) {
				const char = display_chars[pos];
				if (char === '\t' || char === ' ') {
					skipped.push(pos);
					pos++;
					continue;
				}
				if (isNewline(char)) {
					skipped.push(pos);
					missedPositions.push(pos);
					pos++;
					continue;
				}
				break;
			}
		}

		return { pos, skipped, missedPositions };
	}

	let char_states = $derived(
		display_chars.map((char, i) => {
			if (i in typed) return typed[i] === char ? 'correct' : 'wrong';
			if (i in missed) return 'wrong';
			if (isSpecial(char)) return 'special';
			return 'pending';
		})
	);

	/** Global character indices grouped into visual lines. Newline markers are
	 *  only rendered when returns are part of the run. */
	let lines = $derived.by(() => {
		const out: number[][] = [];
		let current: number[] = [];
		const showNewlines = whitespaceMode !== 'auto';
		for (let i = 0; i < display_chars.length; i++) {
			if (display_chars[i] === '\n') {
				if (showNewlines) current.push(i);
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
		if (state === 'wrong') return 'bg-(--wrong-bg) text-(--red)';
		if (state === 'correct') {
			const char = display_chars[i];
			if (char === ' ') return 'bg-(--mer)/25 text-(--silver)';
			if (char === '\n' || char === '\r') return 'bg-(--mer)/10 text-(--mer)';
			if (char === '\t') return 'bg-(--mer)/15 text-(--silver)';
			const kind = syntax_kinds[i] ?? 'plain';
			// Punctuation is a typed character like any other; keep it at the
			// full foreground contrast instead of the dimmer syntax tone.
			if (kind === 'punct') return 'text-(--silver)';
			return kind === 'plain' || kind === 'ws' ? 'text-(--silver)' : `syn-${kind}`;
		}
		if (state === 'special') return 'text-(--space)';
		// Untyped text stays readable and neutral until it is written.
		return 'text-(--pending)';
	}

	function charGlyph(i: number): string {
		const char = display_chars[i];
		if (char === '\n') return '↵';
		if (char === '\r') return '⏎';
		return char;
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
		const targetScrollTop =
			spanOffsetTop - codeContainer.clientHeight * 0.5 + span.offsetHeight / 2;
		codeContainer.scrollTop = Math.max(0, targetScrollTop);
	});

	// ─── Input handling ──────────────────────────────────────────────────────
	let inputRef: HTMLTextAreaElement;
	function focusInput() {
		if (promptState.blocking) return;
		inputRef?.focus({ preventScroll: true });
	}

	function undoLastAction(): string | null {
		const step = undoStack.pop();
		if (!step) return null;
		if (step.typedIndex !== null) delete typed[step.typedIndex];
		for (const index of step.missed) delete missed[index];
		if (step.extra) extraWrong = Math.max(0, extraWrong - 1);
		cursor = step.restoreTo;
		return step.value;
	}

	function isWhitespaceStep(step: UndoStep): boolean {
		if (step.extra || step.typedIndex === null) return true;
		return isSpecial(step.value);
	}

	function typeCharacter(char: string) {
		const { pos, skipped, missedPositions } = collectSkippedWhitespace(cursor);
		if (pos >= display_chars.length) return;

		for (const index of missedPositions) missed[index] = true;
		typed[pos] = char;
		const restoreTo =
			whitespaceMode === 'auto' || missedPositions.length === 0
				? pos
				: Math.min(pos, ...missedPositions);
		undoStack.push({
			typedIndex: pos,
			value: char,
			missed: missedPositions,
			skipped,
			extra: false,
			restoreTo
		});
		cursor = nextCaretAfter(pos + 1);

		// Top up the text before the end is reached, in every mode, so a long
		// timer never runs out of something to type.
		if (cursor > display_chars.length - 400) {
			const extraSeed = deriveSeed(seed, 1_000_000 + extensions++);
			const extra = generate(lang, extraSeed, SNIPPET_TOKENS);
			if (extra) code = code + '\n' + extra;
		}
	}

	function typeSpace() {
		const cursorBefore = cursor;
		const auto = autoWhitespace(cursor);
		const pos = auto.pos;

		// Spaces are characters in every manual mode. A space written over a
		// non-space target is a visible error and advances the caret.
		if (pos < display_chars.length) {
			typed[pos] = ' ';
			undoStack.push({
				typedIndex: pos,
				value: ' ',
				missed: [],
				skipped: auto.skipped,
				extra: false,
				restoreTo: pos
			});
			cursor = nextCaretAfter(pos + 1);
			return;
		}

		extraWrong++;
		undoStack.push({
			typedIndex: null,
			value: ' ',
			missed: [],
			skipped: [],
			extra: true,
			restoreTo: cursorBefore
		});
	}

	function typeEnter() {
		const cursorBefore = cursor;
		const auto = autoWhitespace(cursor);
		const skipped = [...auto.skipped];
		let pos = auto.pos;

		// Adaptive lets optional spaces pass before a return.
		if (whitespaceMode === 'adaptive') {
			while (
				pos < display_chars.length &&
				(display_chars[pos] === ' ' || display_chars[pos] === '\t')
			) {
				skipped.push(pos);
				pos++;
			}
		}

		if (pos >= display_chars.length) {
			extraWrong++;
			undoStack.push({
				typedIndex: null,
				value: '\n',
				missed: [],
				skipped: [],
				extra: true,
				restoreTo: cursorBefore
			});
			return;
		}

		// Enter is also a character at the current target, but it always
		// advances to the next line. If the target is not a return, that
		// target is marked wrong and the rest of the line is skipped.
		typed[pos] = '\n';
		undoStack.push({
			typedIndex: pos,
			value: '\n',
			missed: [],
			skipped,
			extra: false,
			restoreTo: pos
		});

		let lineEnd = pos;
		while (lineEnd < display_chars.length && !isNewline(display_chars[lineEnd])) lineEnd++;
		cursor = nextCaretAfter(lineEnd < display_chars.length ? lineEnd + 1 : lineEnd);

		if (cursor > display_chars.length - 400) {
			const extraSeed = deriveSeed(seed, 1_000_000 + extensions++);
			const extra = generate(lang, extraSeed, SNIPPET_TOKENS);
			if (extra) code = code + '\n' + extra;
		}
	}

	function deleteCharacter() {
		const deleted = undoLastAction();
		if (deleted === null) return;
		recordEvent(KeypressAction.BACKSPACE, '');
	}

	function deleteWord() {
		const deleted: string[] = [];

		// Drop whitespace immediately before the caret, then the word itself.
		// A character that jumped over whitespace starts a new word, so stop
		// after consuming it even when the whitespace was auto-skipped.
		while (undoStack.length > 0 && isWhitespaceStep(undoStack[undoStack.length - 1])) {
			const value = undoLastAction();
			if (value !== null) deleted.push(value);
		}
		while (undoStack.length > 0) {
			const step = undoStack[undoStack.length - 1];
			if (isWhitespaceStep(step)) break;
			const value = undoLastAction();
			if (value !== null) deleted.push(value);
			if (step.skipped.length > 0) break;
		}

		if (deleted.length === 0) return;
		recordEvent(KeypressAction.BACKSPACE, [...deleted].reverse().join(''));
	}

	function handleKeydown(e: KeyboardEvent) {
		if (promptState.blocking) return;
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

		const target = e.target instanceof HTMLElement ? e.target : null;
		const control = target?.closest(
			'button, a, select, input, textarea, summary, [contenteditable="true"]'
		);
		const onTypingInput = target === inputRef;

		// If a real form control has focus, native keyboard behaviour wins.
		// Letter keys on buttons/links still fall through so "type anywhere"
		// keeps working, and we move focus back to the typing surface below.
		if (control && !onTypingInput) {
			if (
				control instanceof HTMLSelectElement ||
				control instanceof HTMLInputElement ||
				(control instanceof HTMLTextAreaElement && control !== inputRef)
			) {
				return;
			}
			if (e.key === ' ' || e.key === 'Enter' || e.key === 'Tab') return;
		}

		if (e.key === 'Tab') {
			e.preventDefault();
			resetGame();
			return;
		}
		if (phase === 'done') return;

		if (e.key === 'Backspace') {
			if (phase === 'idle') return;
			e.preventDefault();
			if (e.altKey) return;
			focusInput();
			if (mod) deleteWord();
			else deleteCharacter();
			return;
		}

		if (mod || e.altKey) return;

		if (e.key === ' ' || e.key === 'Enter') {
			if (whitespaceMode === 'auto') {
				e.preventDefault();
				return;
			}
			e.preventDefault();
			focusInput();
			if (phase === 'idle') startGame();
			const value = e.key === ' ' ? ' ' : '\n';
			recordEvent(KeypressAction.KEYPRESS, value);
			if (e.key === ' ') typeSpace();
			else typeEnter();
			return;
		}

		if (e.key.length === 1) {
			e.preventDefault();
			focusInput();
			if (phase === 'idle') startGame();
			recordEvent(KeypressAction.KEYPRESS, e.key);
			typeCharacter(e.key);
		}
	}

	onMount(() => {
		loadPrefs();
		window.addEventListener('keydown', handleKeydown);
		timeLeft = timer;
		initCodegen().then(() => {
			wasmReady = true;
			newSnippet();
			cursor = start_cursor;
			focusInput();
		});
		return () => {
			window.removeEventListener('keydown', handleKeydown);
		};
	});

	onDestroy(() => {
		if (intervalId) clearInterval(intervalId);
		stopBots();
	});
</script>

<div class="flex h-screen flex-col overflow-hidden bg-(--bg)">
	<!-- Top bar -->
	<header class="border-b border-(--border)">
		<div class="mx-auto flex h-16 w-full max-w-[80rem] items-center gap-3 px-4 md:h-20 md:px-6">
			<a href={resolve('/play')} class="ui-btn ui-btn-sm" aria-label="Open the mode hub"> ← hub </a>
			<a href={resolve('/play')} class="title-text text-3xl leading-none text-(--silver)">
				OSDTYP
			</a>
			<span
				class="hidden border border-(--border) px-2 py-0.5 font-mono text-[9px] tracking-[0.14em] text-(--silver)/45 uppercase sm:inline-block"
			>
				{zen ? 'zen' : 'practice'}
			</span>

			<div class="flex-1"></div>

			<!-- Live stats. The clock is deliberately the largest thing in the
			     top bar so it stays readable in peripheral vision while typing. -->
			<div class="flex items-center gap-3 md:gap-5">
				<div class="flex flex-col items-end">
					<span class="font-mono text-[9px] tracking-[0.16em] text-(--silver)/35 uppercase">
						{zen ? 'zen mode' : phase === 'idle' ? 'timer' : 'time left'}
					</span>
					<span
						class={`min-w-[5ch] text-right font-mono text-4xl leading-none font-semibold tabular-nums md:text-5xl ${
							phase === 'typing' && !zen && timeLeft <= 5 ? 'text-(--red)' : 'text-(--mer)'
						}`}
						aria-label={zen
							? 'Zen mode, no timer'
							: `${phase === 'idle' ? timer : timeLeft} seconds left`}
					>
						{zen ? '∞' : fmtClock(phase === 'idle' ? timer : timeLeft)}
					</span>
				</div>
				{#if !zen}
					<div class="hidden h-10 w-px bg-(--border) sm:block"></div>
					<div class="hidden flex-col items-end sm:flex">
						<span class="font-mono text-[9px] tracking-[0.16em] text-(--silver)/35 uppercase">
							wpm
						</span>
						<div class="flex h-9 items-end md:h-12">
							<span class="font-mono text-2xl leading-none text-(--mer)">
								{phase === 'idle' ? '—' : phase === 'done' ? finalWpm : wpm}
							</span>
						</div>
					</div>
				{/if}
			</div>

			<div class="ml-1 flex items-center gap-2">
				{#if appState.user}
					<a
						href={resolve(`/profile/${appState.user.username}`)}
						class="ui-btn ui-btn-sm hidden max-w-36 truncate sm:inline-flex"
						title="Your profile">{appState.user.username}</a
					>
				{:else}
					<a
						href={resolve(`/login?next=${encodeURIComponent(page.url.pathname)}` as '/login')}
						class="ui-btn ui-btn-sm ui-btn-primary">sign in</a
					>
				{/if}
				<button class="ui-btn ui-btn-sm" onclick={cycleTheme} title="Cycle theme (ctrl+shift+T)"
					>{themeState.current}</button
				>
				<button
					class="ui-btn ui-btn-sm"
					onclick={() => {
						modalOpen = true;
					}}
					title="Settings (esc)">settings</button
				>
			</div>
		</div>
	</header>

	{#if modalOpen}
		<SettingsModal bind:open={modalOpen} />
	{/if}

	<textarea
		bind:this={inputRef}
		class="pointer-events-none fixed top-0 left-0 h-px w-px resize-none opacity-0"
		autocomplete="off"
		spellcheck="false"
		aria-label="Typing test input"
		tabindex="-1"
	></textarea>

	<main class="mx-auto flex min-h-0 w-full max-w-[80rem] flex-1 flex-col px-4 py-4 md:px-6">
		<!-- Test toolbar: every setting that changes the test is here, in one row. -->
		<div class="flex flex-wrap items-center gap-x-6 gap-y-3 pb-3">
			<div class="flex items-center gap-2">
				<label class="ui-label" for="test-language">language</label>
				<select
					id="test-language"
					class="ui-select"
					value={lang}
					onchange={(e) => setLanguage(e.currentTarget.value as Language)}
					disabled={phase === 'typing'}
				>
					{#each LANGUAGES as l (l)}
						<option value={l}>{LANGUAGE_LABELS[l]}</option>
					{/each}
				</select>
			</div>

			{#if !zen}
				<div class="flex items-center gap-2">
					<span class="ui-label">timer</span>
					<div class="flex items-center gap-1">
						{#each TIMES as t (t)}
							<button
								class="ui-chip"
								aria-pressed={timer === t}
								onclick={() => setTimer(t)}
								disabled={phase === 'typing'}
							>
								{t}s
							</button>
						{/each}
					</div>
				</div>
			{/if}

			<div class="flex items-center gap-2">
				<span class="ui-label">space + enter</span>
				<div class="flex items-center gap-1">
					{#each WHITESPACE_MODES as mode (mode)}
						<button
							class="ui-chip"
							aria-pressed={whitespaceMode === mode}
							onclick={() => setWhitespaceMode(mode)}
							disabled={phase === 'typing'}
							title={WHITESPACE_MODE_HINTS[mode]}
						>
							{WHITESPACE_MODE_LABELS[mode]}
						</button>
					{/each}
				</div>
			</div>

			{#if !zen && !guest}
				<div class="flex items-center gap-2">
					<span class="ui-label">race</span>
					<button class="ui-chip" aria-expanded={botsOpen} onclick={() => (botsOpen = !botsOpen)}>
						{#if effectiveBots.length === 0}
							solo
						{:else}
							{effectiveBots.length} bot{effectiveBots.length === 1 ? '' : 's'}
						{/if}
						<span aria-hidden="true">▾</span>
					</button>
				</div>
			{/if}

			<div class="ml-auto flex items-center gap-2">
				<button class="ui-btn ui-btn-sm" onclick={resetGame} title="Restart (tab)">
					restart <span class="kbd">tab</span>
				</button>
			</div>
		</div>

		{#if botsOpen && !zen}
			<div class="mb-3 border border-(--border) bg-(--bg) p-3">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<div class="flex items-center gap-2">
						<span class="ui-label">practice rivals</span>
						<span class="font-mono text-[10px] text-(--silver)/40">
							{paceLoading
								? 'calibrating pace…'
								: `pace ${paceBase} wpm${paceRuns > 0 ? ` · last ${paceRuns} runs` : ' · default'}`}
						</span>
					</div>
					<span class="font-mono text-[9px] text-(--silver)/30"> applies to the next run </span>
				</div>
				<div class="mt-2 flex flex-wrap gap-1.5">
					{#each PRACTICE_BOTS as bot (bot.id)}
						<button
							class="ui-chip"
							aria-pressed={selectedBots.includes(bot.id)}
							onclick={() => toggleBot(bot.id)}
							title={`${botTarget(bot.id)} wpm · ${bot.accuracy}% accuracy`}
						>
							<span class="h-1.5 w-1.5 shrink-0" style={`background:${bot.color}`}></span>
							{bot.label}
							<span class="opacity-45">{botTarget(bot.id)}</span>
						</button>
					{/each}
				</div>
				<p class="mt-2 font-mono text-[9px] text-(--silver)/30">
					Bot speeds follow an exponential moving average of your latest runs.
				</p>
			</div>
		{/if}

		<!-- The surface itself is a click target that returns focus to the
		     hidden textarea. It is not a control, so the keyboard path is the
		     global keydown handler above. -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			class="relative flex min-h-0 flex-1 flex-col border border-(--border) bg-(--fbg)"
			role="group"
			aria-label="Typing test"
			onclick={(e) => {
				const el = e.target as HTMLElement;
				if (el.closest('button, a, select, input, summary')) return;
				focusInput();
			}}
		>
			<!-- Status strip -->
			<div
				class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-4 py-2 font-mono text-[10px] md:px-5"
			>
				<div class="flex min-w-0 items-center gap-2">
					{#if !wasmReady}
						<span class="h-1.5 w-1.5 bg-(--silver)/35"></span>
						<span class="text-(--silver)/45">loading snippet…</span>
					{:else if phase === 'idle'}
						<span class="h-1.5 w-1.5 bg-(--mer)"></span>
						<span class="font-bold tracking-[0.08em] text-(--mer) uppercase">start typing</span>
						<span class="truncate text-(--silver)/40">
							— the timer starts with your first keystroke
						</span>
					{:else if phase === 'typing'}
						<span class="h-1.5 w-1.5 bg-(--mer)"></span>
						<span class="text-(--silver)/55">{zen ? 'zen mode' : 'running'}</span>
						<span class="truncate text-(--silver)/35">
							· line {currentLine + 1} · {typingProgress}%
						</span>
					{:else}
						<span class="h-1.5 w-1.5 bg-(--mer)"></span>
						<span class="text-(--silver)/55">result</span>
						<span class="text-(--silver)/35">· {LANGUAGE_LABELS[lang]} · {finalTime}s</span>
					{/if}
				</div>
			</div>

			{#if phase === 'done'}
				<div class="min-h-0 flex-1 overflow-y-auto">
					<div class="flex min-h-full items-center justify-center px-5 py-5 md:px-8">
						<div class="flex w-full max-w-5xl flex-col gap-5">
							<!-- Headline -->
							<div class="flex flex-wrap items-end gap-x-10 gap-y-4">
								<div>
									<div class="font-mono text-7xl leading-none text-(--mer)">{finalWpm}</div>
									<div
										class="mt-2 font-mono text-[10px] tracking-widest text-(--silver)/45 uppercase"
									>
										wpm
									</div>
								</div>
								<div>
									<div class="font-mono text-5xl leading-none text-(--silver)">
										{finalAccuracy}%
									</div>
									<div
										class="mt-2 font-mono text-[10px] tracking-widest text-(--silver)/45 uppercase"
									>
										acc
									</div>
								</div>
								<div class="ml-auto text-right font-mono text-xs text-(--silver)/45">
									{LANGUAGE_LABELS[lang]} · {finalTime}s
								</div>
							</div>

							<ResultGraph {samples} duration={finalTime} />

							{#if raceResults.length > 0}
								<div class="border border-(--border) bg-(--fbg)">
									<div
										class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-4 py-2.5"
									>
										<div>
											<div
												class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/45 uppercase"
											>
												race result
											</div>
											<div class="mt-1 font-mono text-[9px] text-(--silver)/30">
												bots calibrated from your last {racePaceRuns || 0} run{racePaceRuns === 1
													? ''
													: 's'} · base {racePaceBase} wpm
											</div>
										</div>
										<div class="font-mono text-[9px] text-(--mer)/70 uppercase">
											{raceResults.find((entry) => entry.isYou)?.place === 1
												? 'you took it'
												: 'chase them down'}
										</div>
									</div>
									<div class="divide-y divide-(--border)">
										{#each raceResults as entry (entry.id)}
											<div
												class={`flex flex-wrap items-center gap-3 px-4 py-2.5 ${entry.isYou ? 'bg-(--mer)/5' : ''}`}
											>
												<span class="w-5 font-mono text-[11px] text-(--silver)/35">
													{entry.place}
												</span>
												<span class="h-2 w-2 shrink-0" style={`background:${entry.color}`}></span>
												<span
													class="min-w-0 flex-1 truncate font-mono text-xs"
													style={`color:${entry.isYou ? 'var(--mer)' : 'var(--silver)'}`}
												>
													{entry.label}
												</span>
												{#if entry.isYou}
													<span
														class="font-mono text-[8px] tracking-[0.1em] text-(--mer)/55 uppercase"
														>you</span
													>
												{:else}
													<span
														class="font-mono text-[8px] tracking-[0.1em] text-(--silver)/30 uppercase"
														>bot</span
													>
												{/if}
												<span class="font-mono text-xs text-(--mer)"
													>{entry.wpm}
													<span class="text-[8px] text-(--silver)/30">wpm</span></span
												>
												<span class="font-mono text-[10px] text-(--silver)/40"
													>{entry.accuracy}% acc</span
												>
												<span class="font-mono text-[10px] text-(--silver)/30"
													>{entry.progress}%</span
												>
											</div>
										{/each}
									</div>
								</div>
							{/if}

							{#if effectiveBots.length > 0}
								<div class="border border-(--border) bg-(--fbg) px-3 py-2">
									<div class="flex flex-wrap items-center justify-between gap-2 px-2 pt-1">
										<div>
											<div
												class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/45 uppercase"
											>
												combined speed · you + bots
											</div>
											<div class="mt-1 font-mono text-[9px] text-(--silver)/30">
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
										class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/45 uppercase"
									>
										raw
									</div>
								</div>
								<div class="border border-(--border) px-3 py-2">
									<div class="font-mono text-xl text-(--mer)">{finalCorrect}</div>
									<div
										class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/45 uppercase"
									>
										correct
									</div>
								</div>
								<div class="border border-(--border) px-3 py-2">
									<div class="font-mono text-xl text-(--silver)">{finalWrong}</div>
									<div
										class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/45 uppercase"
									>
										mistakes
									</div>
								</div>
								<div class="border border-(--border) px-3 py-2">
									<div class="font-mono text-xl text-(--silver)">{finalTime}s</div>
									<div
										class="mt-1 font-mono text-[10px] tracking-widest text-(--silver)/45 uppercase"
									>
										time
									</div>
								</div>
							</div>

							<!-- Practice storage -->
							{#if !guest}
								<div
									class="flex flex-wrap items-center gap-3 border border-(--border) bg-(--fbg) px-3 py-2.5"
								>
									<span
										class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
									>
										latest practice
									</span>
									{#if !saveRunData}
										<span class="font-mono text-[11px] text-(--silver)/45">
											save no data is on · this run stays local
										</span>
										<a
											href={resolve('/settings')}
											class="font-mono text-[10px] text-(--silver)/45 underline-offset-2 hover:text-(--mer) hover:underline"
											>change in settings →</a
										>
									{:else if practiceSaving}
										<span class="font-mono text-[11px] text-(--silver)/50"
											>storing temporary run…</span
										>
									{:else if practiceError}
										<span class="font-mono text-[11px] text-(--red)">{practiceError}</span>
									{:else if practiceSaved}
										<span class="font-mono text-[11px] text-(--mer)">saved to profile</span>
										<a
											href={resolve(
												`/profile/${encodeURIComponent(appState.user?.username ?? '')}`
											)}
											class="font-mono text-[10px] text-(--silver)/45 underline-offset-2 hover:text-(--mer) hover:underline"
											>view run →</a
										>
									{:else if practiceRun}
										<span class="font-mono text-[11px] text-(--silver)/45"
											>temporary · replaced next run</span
										>
										<button
											class="ui-btn ui-btn-sm ui-btn-primary"
											onclick={saveRun}
											disabled={practiceSaveBusy}
											>{practiceSaveBusy ? 'saving…' : 'save run →'}</button
										>
									{:else}
										<span class="font-mono text-[11px] text-(--silver)/35">
											no keystrokes stored
										</span>
									{/if}
									<span class="ml-auto font-mono text-[9px] text-(--silver)/30">
										{#if saveRunData}
											temporary runs stay out of history until saved
										{:else}
											no history entry or replay keystrokes were sent
										{/if}
									</span>
								</div>
							{/if}

							<!-- What next -->
							<div class="flex flex-wrap items-center gap-2 border-t border-(--border) pt-5">
								<button class="ui-btn ui-btn-sm ui-btn-primary" onclick={resetGame}>
									next test <span class="kbd">tab</span>
								</button>
								<button class="ui-btn ui-btn-sm" onclick={repeatTest}>repeat</button>
								<button
									class="ui-btn ui-btn-sm"
									onclick={() => (replayOpen = true)}
									disabled={!localRun}
								>
									replay
								</button>
								<button class="ui-btn ui-btn-sm" onclick={() => (modalOpen = true)}>
									settings <span class="kbd">esc</span>
								</button>
								<button class="ui-btn ui-btn-sm" onclick={() => (exportOpen = true)}>
									export png
								</button>
								<a class="ui-btn ui-btn-sm" href={resolve('/play')}>play hub</a>
								<span class="ml-auto font-mono text-[10px] text-(--silver)/35">
									ctrl+shift+t theme · ctrl+shift+l language
								</span>
							</div>
						</div>
					</div>
				</div>
			{:else}
				<!-- Code display -->
				<div
					bind:this={codeContainer}
					class="typing-scroll min-h-0 flex-1 overflow-hidden px-5 py-6 font-mono text-xl leading-relaxed md:px-8 md:text-2xl"
				>
					{#each lines as line, li (li)}
						<div class="code-line" class:code-line-active={li === currentLine}>
							<span class="line-no"><span>{li + 1}</span></span>
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
													></span>{/each}</span
											>{/if}{#if display_chars[i] === '\n' || display_chars[i] === '\r'}<span
												class="newline-glyph"
												aria-hidden="true">{charGlyph(i)}</span
											>{:else}{display_chars[i]}{/if}</span
									>
								{/each}</span
							>
						</div>
					{/each}
				</div>

				{#if !wasmReady}
					<div class="absolute inset-0 flex items-center justify-center bg-(--bg)/50">
						<div class="ui-card px-6 py-4 font-mono text-xs text-(--silver)/55">
							loading the code generator…
						</div>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Footer hints -->
		<div
			class="mt-3 flex flex-wrap items-center gap-x-5 gap-y-2 font-mono text-[10px] text-(--silver)/40"
		>
			<span><span class="kbd">tab</span> restart</span>
			<span><span class="kbd">esc</span> settings</span>
			<span class="hidden sm:inline"><span class="kbd">ctrl+shift+l</span> language</span>
			<span class="hidden sm:inline"><span class="kbd">ctrl+shift+s</span> timer</span>
			{#if effectiveBots.length > 0 && !zen}
				<span class="text-(--mer)/70">
					bots: {effectiveBots.map((id) => botProfile(id).label).join(' · ')}
				</span>
			{/if}
			{#if guest}
				<span class="ml-auto text-(--silver)/35">guest mode · runs are not saved</span>
			{/if}
		</div>
	</main>
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
