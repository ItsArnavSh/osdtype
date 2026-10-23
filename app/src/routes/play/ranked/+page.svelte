<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onDestroy, onMount } from 'svelte';
	import { appState, refreshState } from '$lib/core/state/state.svelte';
	import { joinQueue, leaveQueue, queueStatus } from '$lib/core/api/queue';
	import { onSocketEvent, sendSocket, socketConnected } from '$lib/core/api/socket';
	import { errorMessage } from '$lib/core/api/client';
	import type { EloView, LobbyDuration, QueueStatus } from '$lib/core/api/types';
	import { LOBBY_DURATIONS, userElo } from '$lib/core/api/types';

	let selected = $state<LobbyDuration>(90);
	let queue = $state<QueueStatus | null>(null);
	let queuedAt = $state<number>(0);
	let now = $state(Date.now());
	let busy = $state(false);
	let error = $state('');
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let clockTimer: ReturnType<typeof setInterval> | null = null;
	let unsubscribe: (() => void) | null = null;

	let stateQueue = $derived(appState.state?.state === 'queue' ? appState.state.queue : null);
	let activeQueue = $derived(queue ?? stateQueue ?? null);
	let activeDuration = $derived((activeQueue?.duration ?? selected) as LobbyDuration);

	function eloFor(duration: LobbyDuration): EloView {
		return userElo(appState.user, duration);
	}

	function startPolling() {
		if (pollTimer) clearInterval(pollTimer);
		pollTimer = setInterval(async () => {
			try {
				const status = await queueStatus(activeDuration);
				if (status) {
					queue = status;
					queuedAt = Date.parse(status.joined_at);
				} else if (appState.state?.state === 'queue') {
					await refreshState();
				}
			} catch {
				// The socket remains the primary transport; polling is a fallback.
			}
		}, 3000);
	}

	function stopPolling() {
		if (pollTimer) clearInterval(pollTimer);
		pollTimer = null;
	}

	async function join(duration: LobbyDuration) {
		if (busy) return;
		busy = true;
		error = '';
		selected = duration;
		try {
			const status = await joinQueue(duration);
			queue = status;
			queuedAt = Date.parse(status.joined_at);
			await refreshState();
			startPolling();
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function cancel() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			await leaveQueue(activeDuration);
			queue = null;
			stopPolling();
			await refreshState();
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	function fmt(seconds: number): string {
		const safe = Math.max(0, seconds);
		const m = Math.floor(safe / 60).toString().padStart(2, '0');
		const s = Math.floor(safe % 60).toString().padStart(2, '0');
		return `${m}:${s}`;
	}

	let waited = $derived(queuedAt > 0 ? Math.max(0, Math.floor((now - queuedAt) / 1000)) : 0);

	onMount(() => {
		clockTimer = setInterval(() => (now = Date.now()), 1000);
		unsubscribe = onSocketEvent((event) => {
			if (event.type === 'queue_status') {
				const data = event.data as {
					duration: number;
					position: number;
					waiting: number;
					waited: number;
				};
				if (data && appState.state?.state === 'queue') {
					queue = {
						duration: data.duration,
						position: data.position,
						waiting: data.waiting,
						joined_at: new Date(Date.now() - data.waited * 1000).toISOString()
					};
					queuedAt = Date.now() - data.waited * 1000;
				}
			}
		});
		if (appState.state?.state === 'queue') {
			const q = appState.state.queue;
			if (q) {
				queue = q;
				queuedAt = Date.parse(q.joined_at);
				selected = q.duration as LobbyDuration;
			}
			startPolling();
		}
		return () => {
			stopPolling();
			if (clockTimer) clearInterval(clockTimer);
			unsubscribe?.();
		};
	});

	onDestroy(() => {
		stopPolling();
		if (clockTimer) clearInterval(clockTimer);
		unsubscribe?.();
	});
</script>

<svelte:head><title>ranked · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-6 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex items-end gap-4">
			<a
				href={resolve('/play')}
				class="mb-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← hub</a
			>
			<div class="title-text text-7xl leading-none text-(--silver) md:text-8xl">RANKED</div>
		</div>
		<div class="flex items-end gap-6 text-right">
			<div>
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
					{activeDuration}s elo
				</div>
				<div class="mt-1 font-mono text-lg text-(--mer)">{eloFor(activeDuration).rating}</div>
			</div>
			<div>
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">record</div>
				<div class="mt-1 font-mono text-lg text-(--silver)">
					{appState.user?.wins ?? 0}W · {Math.max(
						0,
						(appState.user?.games_played ?? 0) - (appState.user?.wins ?? 0)
					)}L
				</div>
			</div>
		</div>
	</header>

	<main class="mt-6 grid flex-1 gap-3 lg:grid-cols-[1fr_340px]">
		<section class="flex min-h-[480px] flex-col border border-(--border) bg-(--fbg)" style="border-top-color: var(--mer);">
			{#if activeQueue}
				<div class="flex flex-1 flex-col items-center justify-center gap-8 px-8 py-12">
					<div class="searching-rings">
						<div class="ring"></div>
						<div class="delay ring"></div>
						<span class="ring-icon">⌨</span>
					</div>
					<div>
						<div class="text-center text-2xl text-(--silver)">Finding a match…</div>
						<div class="mt-2 text-center font-mono text-xs text-(--silver)/40">
							{activeDuration}s · {activeQueue.waiting} in queue
						</div>
					</div>
					<div class="grid w-full max-w-lg grid-cols-3 gap-3">
						<div class="border border-(--border) px-4 py-3 text-center">
							<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/30 uppercase">
								waited
							</div>
							<div class="mt-1 font-mono text-xl text-(--silver)">{fmt(waited)}</div>
						</div>
						<div class="border border-(--border) px-4 py-3 text-center">
							<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/30 uppercase">
								position
							</div>
							<div class="mt-1 font-mono text-xl text-(--mer)">
								#{activeQueue.position + 1}
							</div>
						</div>
						<div class="border border-(--border) px-4 py-3 text-center">
							<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/30 uppercase">
								players
							</div>
							<div class="mt-1 font-mono text-xl text-(--silver)">{activeQueue.waiting}</div>
						</div>
					</div>
					<div class="flex items-center gap-3">
						<button
							class="cursor-pointer border border-(--border) px-5 py-2 font-mono text-xs text-(--silver)/55 uppercase hover:border-(--red) hover:text-(--red) disabled:opacity-40"
							onclick={cancel}
							disabled={busy}
						>cancel queue</button
						>
						<button
							class="cursor-pointer border border-(--border) px-5 py-2 font-mono text-xs text-(--silver)/55 uppercase hover:border-(--silver)/40 hover:text-(--silver)"
							onclick={() => sendSocket('state')}
							disabled={!socketConnected()}
						>resync</button
						>
					</div>
					<div class="font-mono text-[10px] text-(--silver)/25">
						the server will start the match automatically. leaving this page does not leave the
						queue.
					</div>
				</div>
			{:else}
				<div class="flex flex-1 flex-col justify-center px-8 py-10 md:px-14">
					<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
						three clocks · three ladders
					</div>
					<h1 class="mt-3 max-w-xl text-3xl leading-tight text-(--silver) md:text-4xl">
						Pick a clock.<br />Build its Elo.
					</h1>
					<p class="mt-4 max-w-lg font-mono text-xs leading-relaxed text-(--silver)/45">
						Every clock has its own independent Elo. Everyone in a match types the same
						server-generated snippet, ratings update after the race, and flagged runs do not
						move Elo. If nobody is around, a bot can fill the lobby.
					</p>

					<div class="mt-8 grid gap-3 sm:grid-cols-3">
						{#each LOBBY_DURATIONS as duration (duration)}
							<button
								class="cursor-pointer border px-5 py-4 text-left transition-colors"
								class:border-(--mer)={selected === duration}
								class:border-(--border)={selected !== duration}
								onclick={() => (selected = duration)}
							>
								<div class="flex items-baseline justify-between gap-2">
									<span class="font-mono text-2xl text-(--silver)">{duration}s</span>
									<span class="font-mono text-sm text-(--mer)">{eloFor(duration).rating}</span>
								</div>
								<div class="mt-1 font-mono text-[10px] tracking-[0.12em] text-(--silver)/35 uppercase">
									{duration === 30 ? 'sprint' : duration === 90 ? 'standard' : 'marathon'}
								</div>
								<div class="mt-2 font-mono text-[10px] text-(--silver)/25">
									{eloFor(duration).games_played} ranked games
								</div>
							</button>
						{/each}
					</div>

					<div class="mt-6 flex flex-wrap items-center gap-4">
						<button
							class="cursor-pointer border border-(--mer) px-6 py-2.5 font-mono text-sm text-(--mer) hover:bg-(--mer)/10 disabled:cursor-not-allowed disabled:opacity-40"
							onclick={() => join(selected)}
							disabled={busy}
						>{busy ? 'joining…' : 'enter queue =>'}</button
						>
						<span class="font-mono text-[10px] text-(--silver)/30">
							Everyone starts at 400 on each clock · bots after 20s
						</span>
					</div>
				</div>
			{/if}
		</section>

		<aside class="flex flex-col gap-3">
			<div class="border border-(--border) bg-(--fbg) p-5">
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
					match rules
				</div>
				<ul class="mt-3 flex flex-col gap-2 font-mono text-[11px] leading-relaxed text-(--silver)/50">
					<li>· server chooses seed, language, snippet and start offset</li>
					<li>· 3 second countdown after the lobby fills</li>
					<li>· live progress over websocket</li>
					<li>· anti-cheat flags freeze rating changes</li>
					<li>· refresh or reconnect and the server resumes your game</li>
				</ul>
			</div>

			<div class="border border-(--border) bg-(--fbg) p-5">
				<div class="flex items-center justify-between">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
						your ladders
					</div>
					<span class="font-mono text-[10px] text-(--silver)/25">elo</span>
				</div>
				<div class="mt-3 flex flex-col gap-1.5 font-mono text-[11px]">
					{#each LOBBY_DURATIONS as duration (duration)}
						<div class="flex items-center justify-between border-b border-(--border)/40 pb-1.5">
							<span class="text-(--silver)/55">{duration}s</span>
							<span class="text-(--mer)">{eloFor(duration).rating}</span>
						</div>
					{/each}
				</div>
				<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/25">
					Each ladder moves only when you finish a ranked race on that clock. The hub shows
					your highest.
				</p>
			</div>

			{#if error}
				<div class="border border-(--red)/40 px-4 py-3 font-mono text-xs text-(--red)">{error}</div>
			{/if}
		</aside>
	</main>

	<footer class="mt-4 flex items-center justify-between border-t border-(--border) pt-3 font-mono text-[10px] text-(--silver)/25">
		<span>ranked queue · redis-backed</span>
		<a class="hover:text-(--silver)/60" href={resolve('/leaderboard')}>leaderboard →</a>
	</footer>
</div>

<style>
	.searching-rings {
		position: relative;
		width: 110px;
		height: 110px;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.ring {
		position: absolute;
		inset: 0;
		border-radius: 50%;
		border: 1px solid color-mix(in srgb, var(--mer) 25%, transparent);
		animation: pulse 2.4s ease-out infinite;
	}
	.ring.delay {
		animation-delay: 1.2s;
	}
	.ring-icon {
		position: relative;
		font-size: 2rem;
		z-index: 1;
	}
	@keyframes pulse {
		0% {
			transform: scale(0.55);
			opacity: 0.8;
		}
		100% {
			transform: scale(2.1);
			opacity: 0;
		}
	}
</style>
