<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onDestroy, onMount } from 'svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { getGame, leaveGame } from '$lib/core/api/game';
	import { onSocketEvent, sendSocket, socketConnected } from '$lib/core/api/socket';
	import { KeypressAction, LANGUAGE_LABELS } from '$lib/core/api/types';
	import Avatar from '$lib/components/Avatar.svelte';
	import type { GameView, PlayerProgress, ResultView } from '$lib/core/api/types';
	import { appState, applyState, refreshState } from '$lib/core/state/state.svelte';

	const gameID = $derived(Number(page.params.id ?? 0));

	let game = $state<GameView | null>(null);
	let typed = $state<string[]>([]);
	let pos = $state(0);
	let initialized = $state(false);
	let localFinished = $state(false);
	let loading = $state(true);
	let error = $state('');
	let busy = $state(false);
	let now = $state(Date.now());
	let serverOffset = $state(0);
	let clockTimer: ReturnType<typeof setInterval> | null = null;
	let unsubscribe: (() => void) | null = null;
	let results: ResultView[] = $state([]);

	let chars = $derived(game ? Array.from(game.snippet) : []);
	let startChar = $derived(game ? byteToChar(game.snippet, game.start_offset) : 0);
	let displayStart = $derived(lineStart(startChar));
	let visibleChars = $derived(chars.slice(displayStart));
	let codeScroll = $state<HTMLDivElement>();
	let startsAt = $derived(game?.starts_at ? Date.parse(game.starts_at) : 0);
	let endsAt = $derived(game?.ends_at ? Date.parse(game.ends_at) : 0);
	let serverNow = $derived(now + serverOffset);
	let countdown = $derived(startsAt ? Math.max(0, Math.ceil((startsAt - serverNow) / 1000)) : 0);
	let remaining = $derived(endsAt ? Math.max(0, Math.ceil((endsAt - serverNow) / 1000)) : 0);
	let hasStarted = $derived(startsAt > 0 && serverNow >= startsAt);
	let hasEnded = $derived(endsAt > 0 && serverNow >= endsAt);
	let canType = $derived(Boolean(game) && hasStarted && !hasEnded && !localFinished && !busy);
	let phase = $derived(
		!game ? 'loading' : localFinished || game.status === 'finished' ? 'waiting' : countdown > 0 ? 'countdown' : hasEnded ? 'waiting' : 'typing'
	);
	let myID = $derived(appState.user?.id ?? 0);
	let me = $derived(game?.players.find((p) => p.user_id === myID) ?? null);
	let progressPct = $derived(totalProgress());

	function byteToChar(value: string, bytes: number): number {
		const encoder = new TextEncoder();
		let consumed = 0;
		let index = 0;
		for (const ch of value) {
			if (consumed >= bytes) break;
			consumed += encoder.encode(ch).length;
			index++;
		}
		return index;
	}

	function lineStart(index: number): number {
		let i = Math.min(Math.max(0, index), chars.length);
		while (i > 0 && chars[i - 1] !== '\n') i--;
		return i;
	}

	function nextReal(index: number): number {
		let i = Math.max(0, index);
		while (i < chars.length && /\s/.test(chars[i])) i++;
		return i;
	}

	function prevReal(index: number): number {
		let i = Math.min(chars.length - 1, index - 1);
		while (i > 0 && /\s/.test(chars[i])) i--;
		return i;
	}

	function initialize(view: GameView) {
		game = view;
		const start = byteToChar(view.snippet, view.start_offset);
		const mine = view.players.find((p) => p.user_id === (appState.user?.id ?? 0));
		let current = start;
		if (mine && mine.current_points > view.start_offset) {
			current = byteToChar(view.snippet, mine.current_points);
		}
		const nextTyped = new Array<string>(chars.length).fill('');
		const resolvedChars = Array.from(view.snippet);
		for (let i = nextReal(start); i < current; i = nextReal(i + 1)) {
			if (i < resolvedChars.length) nextTyped[i] = resolvedChars[i];
		}
		typed = nextTyped;
		pos = nextReal(current);
		initialized = true;
		localFinished = pos >= chars.length;
		// Cross-check the server clock captured with the authoritative state
		// snapshot. Without it we still use local time, but the layout always
		// provides a snapshot before a game page is rendered.
		serverOffset = 0;
		if (appState.state?.server_time) {
			serverOffset = Date.parse(appState.state.server_time) - Date.now();
		}
	}

	async function load() {
		try {
			const view = await getGame(gameID);
			initialize(view);
			results = view.leaderboard ?? [];
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	}

	function applyGame(view: GameView) {
		if (view.id !== gameID) return;
		game = view;
		if (!initialized) initialize(view);
		loading = false;
		if (view.leaderboard?.length) results = view.leaderboard;
	}

	async function resync() {
		try {
			const view = await getGame(gameID);
			initialized = false;
			initialize(view);
			await refreshState();
		} catch {
			// The layout will retry state on the next socket reconnect.
		}
	}

	function totalProgress(): number {
		if (!game) return 0;
		const start = startChar;
		const total = Math.max(1, chars.length - start);
		if (me && pos > start) {
			return Math.min(1, (pos - start) / total);
		}
		if (me) {
			const totalBytes = Math.max(1, new TextEncoder().encode(game.snippet).length - game.start_offset);
			return Math.min(1, (me.current_points - game.start_offset) / totalBytes);
		}
		return 0;
	}

	function playerProgress(player: PlayerProgress): number {
		if (!game) return 0;
		const total = Math.max(1, new TextEncoder().encode(game.snippet).length - game.start_offset);
		if (player.user_id === myID) return progressPct;
		return Math.min(1, Math.max(0, (player.current_points - game.start_offset) / total));
	}

	function onKeydown(event: KeyboardEvent) {
		if (!canType) return;
		if (event.ctrlKey || event.metaKey || event.altKey) return;
		const target = event.target as HTMLElement | null;
		if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')) return;

		if (event.key === 'Backspace') {
			event.preventDefault();
			if (pos <= nextReal(startChar)) return;
			const previous = prevReal(pos);
			if (previous < startChar) return;
			if (typed[previous]) {
				const copy = typed.slice();
				copy[previous] = '';
				typed = copy;
			}
			pos = previous;
			sendSocket('keypress', { value: '', action: KeypressAction.BACKSPACE, time_ms: 0 });
			return;
		}

		if (event.key.length !== 1) return;
		if (event.key === ' ' || event.key === '\n' || event.key === '\t') {
			event.preventDefault();
			return;
		}
		event.preventDefault();
		if (pos >= chars.length) return;
		const expected = pos;
		const copy = typed.slice();
		copy[expected] = event.key;
		typed = copy;
		const next = nextReal(expected + 1);
		pos = next;
		sendSocket('keypress', { value: event.key, action: KeypressAction.KEYPRESS, time_ms: 0 });
		if (next >= chars.length) {
			localFinished = true;
		}
	}

	async function openResults() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			for (let attempt = 0; attempt < 4; attempt++) {
				const next = await refreshState();
				if (next?.state === 'results' && next.results?.id === gameID) {
					await goto(`/results/${gameID}`, { replaceState: true });
					return;
				}
				await new Promise((resolve) => setTimeout(resolve, 350));
			}
			error = 'Results are still being written. Try again in a moment.';
		} finally {
			busy = false;
		}
	}

	async function leave() {
		if (busy) return;
		if (!confirm('Leave this race? Your run will be forfeited.')) return;
		busy = true;
		try {
			const result = await leaveGame(gameID);
			if (result.state) applyState(result.state);
			// Keeping the race in the rear-view matters more than showing a
			// forfeited result. Return to the room if one is still active,
			// otherwise to the hub.
			const next = result.state ?? appState.state;
			if (next?.state === 'room' && next.room) await goto(resolve(`/room/${next.room.code}`));
			else await goto(resolve('/play'));
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	function fmt(seconds: number): string {
		const safe = Math.max(0, seconds);
		return `${Math.floor(safe / 60).toString().padStart(2, '0')}:${(safe % 60).toString().padStart(2, '0')}`;
	}

	function charClass(index: number): string {
		if (index < startChar) return 'text-(--silver)/15';
		if (index === pos && canType) return 'current-char text-(--silver)';
		if (typed[index]) {
			return typed[index] === chars[index] ? 'text-(--mer)' : 'text-(--red) underline decoration-(--red)/60';
		}
		return 'text-(--silver)/45';
	}

	// The server picks a start offset into the generated snippet. Hide whole
	// lines before that offset so the player never has to scroll past hundreds
	// of dimmed characters, and keep the active character in view as they type.
	$effect(() => {
		void pos;
		void displayStart;
		void canType;
		const active = codeScroll?.querySelector<HTMLElement>('.current-char');
		if (active) active.scrollIntoView({ block: 'nearest' });
	});

	onMount(() => {
		clockTimer = setInterval(() => {
			now = Date.now();
			if (localFinished && phase === 'waiting' && results.length > 0) {
				// The state event normally navigates first; this is the fallback.
			}
		}, 100);
		unsubscribe = onSocketEvent((event) => {
			if (event.type === 'game_start' && event.data) applyGame(event.data as GameView);
			if (event.type === 'progress' && event.data) {
				const players = event.data as PlayerProgress[];
				if (game) game = { ...game, players };
			}
			if (event.type === 'leaderboard' && event.data) {
				results = event.data as ResultView[];
				if (game) game = { ...game, leaderboard: results, status: 'finished' };
			}
			if (event.type === 'socket_open') void resync();
			if (event.type === 'state' && event.data) {
				const state = event.data as { state: string; game?: GameView; results?: GameView };
				if (state.state === 'game' && state.game) applyGame(state.game);
				if (state.state === 'results' && state.results) {
					void goto(`/results/${state.results.id}`, { replaceState: true });
				}
			}
		});
		if (appState.state?.game?.id === gameID) {
			initialize(appState.state.game);
			loading = false;
		} else void load();
		window.addEventListener('keydown', onKeydown);
		return () => {
			window.removeEventListener('keydown', onKeydown);
			unsubscribe?.();
			if (clockTimer) clearInterval(clockTimer);
		};
	});

	onDestroy(() => {
		window.removeEventListener('keydown', onKeydown);
		unsubscribe?.();
		if (clockTimer) clearInterval(clockTimer);
	});
</script>

<svelte:head><title>{game ? `race ${game.id}` : 'race'} · OSDTYP</title></svelte:head>

<div class="flex h-screen flex-col overflow-hidden bg-(--bg) p-4 md:p-6">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<div class="flex items-center gap-4">
			<div>
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
					{game?.mode ?? 'race'} · {game?.room_code || `game ${gameID}`}
				</div>
				<div class="mt-1 flex items-baseline gap-3">
					<span class="font-mono text-3xl text-(--silver)">{fmt(remaining)}</span>
					<span class="font-mono text-[10px] text-(--silver)/30">
						{game?.duration ?? 0}s · {LANGUAGE_LABELS[game?.language ?? 'go'] ?? game?.language}
					</span>
				</div>
			</div>
			{#if phase === 'countdown'}
				<div class="flex items-baseline gap-2 border-l border-(--border) pl-4">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
						>starts in</span
					>
					<span class="font-mono text-2xl text-(--red)">{countdown}</span>
				</div>
			{/if}
		</div>
		<div class="flex items-center gap-2">
			<span
				class={`h-1.5 w-1.5 ${socketConnected() ? 'bg-(--mer)' : 'bg-(--red)'}`}
				title={socketConnected() ? 'socket connected' : 'socket reconnecting'}
			></span>
			<button
				class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] text-(--silver)/45 hover:border-(--red) hover:text-(--red) disabled:opacity-40"
				onclick={leave}
				disabled={busy}>leave race</button
			>
		</div>
	</header>

	{#if loading}
		<div class="flex flex-1 items-center justify-center">
			<span class="font-mono text-xs text-(--silver)/35">loading race…</span>
		</div>
	{:else if !game}
		<div class="flex flex-1 flex-col items-center justify-center gap-3">
			<div class="font-mono text-sm text-(--red)">{error || 'game not found'}</div>
			<a href={resolve('/play')} class="font-mono text-xs text-(--mer)">return to hub</a>
		</div>
	{:else}
		{#if error}
			<div class="mt-2 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)">{error}</div>
		{/if}
		<main class="grid min-h-0 flex-1 gap-3 pt-3 lg:grid-cols-[1fr_260px]">
			<section class="relative flex min-h-0 flex-col border border-(--border) bg-(--fbg)">
				<div class="flex items-center justify-between border-b border-(--border) px-4 py-2 font-mono text-[10px] text-(--silver)/35">
					<span>seed {game.seed} · offset {game.start_offset}</span>
					<span>{phase === 'typing' ? 'typing live' : phase}</span>
				</div>
				<div
					bind:this={codeScroll}
					class="code-scroll min-h-0 flex-1 overflow-auto p-5 font-mono text-[15px] leading-7 whitespace-pre-wrap md:p-7 md:text-base"
				>
					{#each visibleChars as char, i (displayStart + i)}
						{@const index = displayStart + i}
						<span class={charClass(index)}>{char}</span>
					{/each}
				</div>
				{#if phase === 'countdown'}
					<div class="absolute inset-0 flex flex-col items-center justify-center bg-(--bg)/80">
						<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
							server starts the clock
						</div>
						<div class="countdown-number mt-2 font-mono text-8xl text-(--silver)">{countdown}</div>
						<div class="mt-3 font-mono text-xs text-(--silver)/35">
							wait for zero, then type. spaces are skipped.
						</div>
					</div>
				{:else if phase === 'waiting'}
					<div class="absolute inset-x-0 bottom-0 border-t border-(--border) bg-(--bg)/95 px-5 py-3">
						<div class="flex flex-wrap items-center justify-between gap-3 font-mono text-xs">
							<span class="text-(--silver)/50">
								{localFinished ? 'run complete · waiting for the server' : 'clock over · waiting for results'}
							</span>
							<div class="flex items-center gap-3">
								<span class="text-(--mer)">{fmt(remaining)}</span>
								<button
									class="cursor-pointer border border-(--mer) px-3 py-1 text-[10px] text-(--mer) hover:bg-(--mer)/10 disabled:opacity-40"
									onclick={openResults}
									disabled={busy}>{busy ? 'checking…' : 'view results =>'}</button
								>
							</div>
						</div>
					</div>
				{/if}
			</section>

			<aside class="flex min-h-0 flex-col gap-2 overflow-auto">
				<div class="border border-(--border) bg-(--fbg)">
					<div class="border-b border-(--border) px-4 py-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						race · {game.players.length} players
					</div>
					<div class="divide-y divide-(--border)">
						{#each [...game.players].sort((a, b) => playerProgress(b) - playerProgress(a)) as player (player.user_id)}
							<div class="px-4 py-3">
								<div class="flex items-center justify-between gap-2">
									<div class="flex min-w-0 items-center gap-2">
										<Avatar src={player.avatar_url} name={player.username} class="h-6 w-6" text="text-[9px]" />
										<span class="truncate font-mono text-[11px] text-(--silver)/75"
											>{player.username}{player.user_id === myID ? ' (you)' : ''}</span
										>
										{#if player.is_bot}<span class="font-mono text-[9px] text-(--silver)/25">bot</span>{/if}
									</div>
									<span class="shrink-0 font-mono text-[10px] text-(--silver)/35">
										{Math.round(player.wpm)} wpm
									</span>
								</div>
								<div class="mt-2 h-1 w-full bg-(--border)">
									<div
										class={`h-1 ${player.finished ? 'bg-(--mer)' : 'bg-(--silver)/35'}`}
										style={`width:${Math.round(playerProgress(player) * 100)}%`}
									></div>
								</div>
								<div class="mt-1 flex justify-between font-mono text-[9px] text-(--silver)/25">
									<span>{player.finished ? 'finished' : player.flagged ? 'flagged' : 'typing'}</span>
									<span>{Math.round(playerProgress(player) * 100)}%</span>
								</div>
							</div>
						{/each}
					</div>
				</div>

				<div class="border border-(--border) p-4">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/30 uppercase">your run</div>
					<div class="mt-2 grid grid-cols-2 gap-2">
						<div>
							<div class="font-mono text-[9px] text-(--silver)/30">progress</div>
							<div class="font-mono text-lg text-(--mer)">{Math.round(progressPct * 100)}%</div>
						</div>
						<div>
							<div class="font-mono text-[9px] text-(--silver)/30">wpm</div>
							<div class="font-mono text-lg text-(--silver)">{Math.round(me?.wpm ?? 0)}</div>
						</div>
					</div>
					<div class="mt-3 h-px w-full bg-(--border)">
						<div class="h-px bg-(--mer)" style={`width:${progressPct * 100}%`}></div>
					</div>
					<p class="mt-3 font-mono text-[9px] leading-relaxed text-(--silver)/25">
						Backspace works. Whitespace is skipped automatically. Refreshing resumes from the server's
						authoritative progress.
					</p>
				</div>
			</aside>
		</main>
	{/if}
</div>

<style>
	.current-char {
		background: color-mix(in srgb, var(--mer) 24%, transparent);
		box-shadow: inset 0 -2px 0 var(--mer);
	}
	.countdown-number {
		animation: tick 1s steps(2, end) infinite;
	}
	@keyframes tick {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.45;
		}
	}
</style>
