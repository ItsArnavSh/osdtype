<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { joinRankedLobby } from '$lib/core/api';
	import { OSDTypeSession } from '$lib/core/api/session';
	import { gameState } from '$lib/core/state/state.svelte';
	import { generate } from '../../../rust-core/pkg/rust_core';
	import { GetGrammar, LANGUAGES } from '$lib/core/entity/languages';
	import type { Language } from '$lib/core/entity/languages';
	import type { Leaderboard, GameplayBroadcastFrame } from '$lib/core/api/types';

	// ─── Phase ───────────────────────────────────────────────────────────────
	type Phase = 'searching' | 'countdown' | 'idle' | 'typing' | 'done';
	let phase = $state<Phase>('searching');
	let searchElapsed = $state(0);
	let countdown = $state(3);

	// ─── Game config ─────────────────────────────────────────────────────────
	let timer = gameState.mode as number; // 30 | 90 | 300
	let lang = $state<Language>('Go');
	let seed = $state(0);

	// ─── Generated code ──────────────────────────────────────────────────────
	let code = $state<string[]>([]);
	let display_code = $derived(code.join(''));
	let display_chars = $derived(display_code.split(''));

	// ─── Typing state ────────────────────────────────────────────────────────
	let cursor = $state(0);
	let typed: Record<number, string> = $state({});
	let timeLeft = $state(timer);
	let wpm = $state(0);
	let intervalId: ReturnType<typeof setInterval> | null = null;

	// ─── Final scores ────────────────────────────────────────────────────────
	let finalWpm = $state(0);
	let finalRawWpm = $state(0);
	let finalAccuracy = $state(0);
	let finalCorrect = $state(0);
	let finalWrong = $state(0);
	let serverLeaderboard = $state<Leaderboard>([]);

	// ─── Opponents ───────────────────────────────────────────────────────────
	// Map of player_id → { cursor position on the left-border (px), username }
	interface Opponent {
		playerId: number;
		username: string;
		dotTop: number;
		currentPoints: number;
	}
	let opponents = $state<Record<number, Opponent>>({});

	// ─── DOM refs ────────────────────────────────────────────────────────────
	let codeContainer: HTMLDivElement;
	let spanRefs: HTMLSpanElement[] = [];
	let inputRef: HTMLTextAreaElement;
	let dotTop = $state(0);

	// ─── Session ─────────────────────────────────────────────────────────────
	const session = new OSDTypeSession({
		onOpen: () => {
			searchTimer = setInterval(() => searchElapsed++, 1000);
		},

		onSeed: (raw) => {
			clearInterval(searchTimer);
			const numeric = parseInt(raw, 10);
			seed = numeric;
			lang = LANGUAGES[numeric % 6];
			code = generate(GetGrammar(lang), numeric, 1000);
			cursor = nextReal(0);
			phase = 'countdown';
			startCountdown();
		},

		onBroadcast: (frame: GameplayBroadcastFrame) => {
			// Full state snapshot — rebuild opponents map from the whole array
			const updated: typeof opponents = {};
			for (const msg of frame) {
				const span = spanRefs[msg.current_points];
				const top =
					span && codeContainer
						? span.offsetTop - codeContainer.scrollTop + span.offsetHeight / 2
						: (opponents[msg.player_id]?.dotTop ?? 0);
				updated[msg.player_id] = {
					playerId: msg.player_id,
					username: String(msg.player_id), // swap for real GitHub username when available
					dotTop: top,
					currentPoints: msg.current_points
				};
			}
			opponents = updated;
		},

		onLeaderboard: (board) => {
			serverLeaderboard = board;
			endGame();
		},

		onControl: () => {
			session.disconnect();
		},

		onClose: () => {
			clearAllIntervals();
		}
	});

	let searchTimer: ReturnType<typeof setInterval>;

	function clearAllIntervals() {
		clearInterval(searchTimer);
		if (intervalId) {
			clearInterval(intervalId);
			intervalId = null;
		}
	}

	// ─── Countdown ───────────────────────────────────────────────────────────
	function startCountdown() {
		countdown = 3;
		const cd = setInterval(() => {
			countdown--;
			if (countdown <= 0) {
				clearInterval(cd);
				phase = 'idle';
			}
		}, 1000);
	}

	// ─── Game logic (mirrors practice page) ──────────────────────────────────
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

	// ─── Dot + autoscroll ────────────────────────────────────────────────────
	$effect(() => {
		const _ = cursor;
		const span = spanRefs[cursor];
		if (!span || !codeContainer) return;
		const spanOffsetTop = span.offsetTop;
		const targetScrollTop = spanOffsetTop - codeContainer.clientHeight / 2 + span.offsetHeight / 2;
		codeContainer.scrollTop = Math.max(0, targetScrollTop);
		dotTop = spanOffsetTop - codeContainer.scrollTop + span.offsetHeight / 2;
	});

	// ─── Keyboard ────────────────────────────────────────────────────────────
	function focusInput() {
		inputRef?.focus();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (phase === 'done' || phase === 'searching' || phase === 'countdown') return;
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

	// ─── Lifecycle ───────────────────────────────────────────────────────────
	onMount(async () => {
		await joinRankedLobby(gameState.mode);
		await session.connect();
		window.addEventListener('keydown', handleKeydown);
	});

	onDestroy(() => {
		session.disconnect();
		clearAllIntervals();
		window.removeEventListener('keydown', handleKeydown);
	});

	function formatTime(s: number) {
		const m = Math.floor(s / 60)
			.toString()
			.padStart(2, '0');
		const sec = (s % 60).toString().padStart(2, '0');
		return `${m}:${sec}`;
	}

	function ghAvatar(username: string) {
		return `https://github.com/${username}.png?size=32`;
	}
</script>

<div class="m-0 min-h-screen overflow-hidden bg-(--bg)">
	<!-- ── Header ── -->
	<div class="mt-10 flex flex-col p-2 text-white">
		<div class="flex flex-row items-end">
			<div class="title-text flex-1 text-9xl">OSDTYP</div>
			<div class="my-8 flex flex-col items-end gap-1 px-6">
				<div class="font-mono text-(--red)">→Ranked</div>
				<div class="flex flex-row items-end gap-6">
					<div class="flex flex-col items-center">
						<div class="font-mono text-3xl text-(--mer)">
							{phase === 'searching' ? formatTime(searchElapsed) : timeLeft}
						</div>
						<div class="text-xs text-(--silver)/50">
							{phase === 'searching' ? 'Searching' : 'Timer'}
						</div>
					</div>
					{#if phase !== 'searching'}
						<div class="flex flex-col items-end">
							<div class="font-mono text-3xl text-(--mer)">{lang}</div>
							<div class="text-xs text-(--silver)/50">Language</div>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<textarea
		bind:this={inputRef}
		class="pointer-events-none absolute resize-none opacity-0"
		autocomplete="off"
		autocorrect="off"
		spellcheck="false"
		rows="1"
	/>

	<!-- ── Main panel ── -->
	<div
		class="relative border-t border-b border-(--silver) bg-(--fbg)"
		onclick={focusInput}
		role="none"
		style="height: 75vh;"
	>
		<!-- SEARCHING -->
		{#if phase === 'searching'}
			<div class="flex h-full flex-col items-center justify-center gap-6">
				<div class="searching-rings">
					<div class="ring"></div>
					<div class="delay ring"></div>
					<span class="ring-icon">⌨</span>
				</div>
				<div class="font-mono text-xl tracking-widest text-(--silver)/60">finding a match…</div>
				<div class="font-mono text-sm text-(--silver)/30">{formatTime(searchElapsed)}</div>
			</div>

			<!-- COUNTDOWN -->
		{:else if phase === 'countdown'}
			<div class="flex h-full flex-col items-center justify-center gap-4">
				<div class="font-mono text-sm tracking-widest text-(--silver)/40 uppercase">
					match found — starting in
				</div>
				<div
					class="countdown-number font-mono text-(--mer)"
					style="font-size: 8rem; line-height:1;"
				>
					{countdown}
				</div>
				<div class="font-mono text-sm text-(--silver)/30">{lang} · {timer}s</div>
			</div>

			<!-- DONE -->
		{:else if phase === 'done'}
			<div class="flex h-full flex-col items-center justify-center gap-10 overflow-y-auto px-16">
				<!-- Your stats -->
				<div class="flex flex-col items-center">
					<div class="font-mono text-8xl text-(--mer)">{finalWpm}</div>
					<div class="mt-1 font-mono text-sm tracking-widest text-(--silver)/40 uppercase">wpm</div>
				</div>
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

				<!-- Server leaderboard -->
				{#if serverLeaderboard.length > 0}
					<div class="w-full max-w-lg">
						<div class="mb-3 font-mono text-xs tracking-widest text-(--silver)/30 uppercase">
							leaderboard
						</div>
						<div class="flex flex-col gap-2">
							{#each serverLeaderboard as entry, i}
								<div
									class="flex flex-row items-center gap-4 border border-(--silver)/10 px-4 py-2 font-mono text-sm"
									class:border-mer={i === 0}
								>
									<span class="w-6 text-(--silver)/30">#{i + 1}</span>
									<span class="flex-1 text-(--silver)">{entry.id}</span>
									<span class="text-(--mer)">{entry.wpm.toFixed(1)} wpm</span>
									<span class="text-(--silver)/50">{(entry.accuracy * 100).toFixed(0)}%</span>
									<span class="text-(--red)">{entry.wrong} err</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<!-- TYPING / IDLE -->
		{:else}
			<div class="relative ml-10 h-full border-l border-(--silver) bg-(--fbg)">
				<!-- Your dot -->
				<div
					class="dot your-dot absolute rounded-full transition-all duration-100"
					style="left: -6px; top: {dotTop}px; transform: translateY(-50%); background-color: var(--mer);"
				></div>

				<!-- Opponent dots (GitHub avatars) -->
				{#each Object.values(opponents) as opp (opp.playerId)}
					<div
						class="opponent-dot absolute transition-all duration-200"
						style="left: -16px; top: {opp.dotTop}px; transform: translateY(-50%);"
					>
						<img
							src={ghAvatar(opp.username)}
							alt={opp.username}
							class="rounded-full border border-(--silver)/30"
							style="width:24px; height:24px; object-fit:cover;"
							title={opp.username}
						/>
					</div>
				{/each}

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

	<!-- ── Bottom bar ── -->
	<div class="flex flex-row items-center px-4 py-3">
		<div class="font-mono text-sm text-(--silver)/30">ranked match · {timer}s · {lang}</div>
		<div class="flex-1"></div>
		{#if phase === 'typing'}
			<div class="text-right font-mono">
				<span class="text-3xl text-(--mer)">{wpm}</span>
				<span class="ml-1 text-sm text-(--silver)/40">wpm</span>
			</div>
		{/if}
	</div>
</div>

<style>
	.searching-rings {
		position: relative;
		width: 120px;
		height: 120px;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.ring {
		position: absolute;
		width: 100%;
		height: 100%;
		border-radius: 50%;
		border: 1px solid color-mix(in srgb, var(--mer) 25%, transparent);
		animation: pulse 2.4s ease-out infinite;
	}
	.ring.delay {
		animation-delay: 1.2s;
	}
	.ring-icon {
		position: relative;
		font-size: 2.5rem;
		z-index: 1;
	}
	@keyframes pulse {
		0% {
			transform: scale(0.5);
			opacity: 0.8;
		}
		100% {
			transform: scale(2.2);
			opacity: 0;
		}
	}

	.countdown-number {
		animation: pop 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
	}
	@keyframes pop {
		from {
			transform: scale(0.4);
			opacity: 0;
		}
		to {
			transform: scale(1);
			opacity: 1;
		}
	}

	.your-dot {
		width: 12px;
		height: 12px;
	}

	.opponent-dot img {
		display: block;
	}

	.border-mer {
		border-color: var(--mer);
	}
</style>
