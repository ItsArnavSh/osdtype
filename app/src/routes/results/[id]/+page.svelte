<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import ExportModal from '$lib/components/ExportModal.svelte';
	import ReplayModal from '$lib/components/ReplayModal.svelte';
	import SpeedGraph from '$lib/components/SpeedGraph.svelte';
	import { acknowledgeResults } from '$lib/core/api/state';
	import { getGame, getGameReplays } from '$lib/core/api/game';
	import { errorMessage } from '$lib/core/api/client';
	import { LANGUAGE_LABELS } from '$lib/core/api/types';
	import type { GameView, ReplayBundle, ReplayRun, ResultView } from '$lib/core/api/types';
	import { deriveSamples, syntheticSamples } from '$lib/core/replay';
	import type { ExportStat, SpeedSeries } from '$lib/core/stats';
	import { appState, applyState } from '$lib/core/state/state.svelte';

	const gameID = $derived(Number(page.params.id ?? 0));
	let game = $state<GameView | null>(null);
	let replays = $state<ReplayBundle | null>(null);
	let selectedRun = $state<ReplayRun | null>(null);
	let exportOpen = $state(false);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');

	let results = $derived<ResultView[]>(
		(game?.leaderboard?.length ? game.leaderboard : game?.players ?? [])
			.map((entry, index) => ({
				user_id: entry.user_id,
				username: entry.username,
				avatar_url: entry.avatar_url,
				rank: 'rank' in entry ? entry.rank : index + 1,
				wpm: entry.wpm,
				raw: 'raw' in entry ? entry.raw : entry.wpm,
				accuracy: entry.accuracy,
				correct: 'correct' in entry ? entry.correct : 0,
				wrong: 'wrong' in entry ? entry.wrong : 0,
				rating_before: 'rating_before' in entry ? entry.rating_before : 0,
				rating_after: 'rating_after' in entry ? entry.rating_after : 0,
				rating_delta: 'rating_delta' in entry ? entry.rating_delta : 0,
				flagged: entry.flagged,
				left: 'left' in entry ? entry.left : false,
				is_bot: entry.is_bot
			}))
			.sort((a, b) => a.rank - b.rank)
	);
	let myID = $derived(appState.user?.id ?? 0);
	let mine = $derived(results.find((entry) => entry.user_id === myID) ?? null);

	let graphSeries = $derived.by<SpeedSeries[]>(() => {
		const currentGame = game;
		if (!currentGame) return [];
		return results.map((result) => {
			const run = runFor(result.user_id);
			const samples =
				run && run.events.length > 0
					? deriveSamples(
							run.events,
							currentGame.snippet,
							currentGame.start_offset,
							currentGame.duration
						)
					: syntheticSamples(result.wpm, result.accuracy, currentGame.duration);
			return {
				label: result.username,
				samples,
				isBot: result.is_bot
			};
		});
	});

	let exportStats = $derived.by<ExportStat[]>(() => {
		const top = results[0];
		return [
			{ label: 'wpm', value: Math.round(top?.wpm ?? 0), accent: true },
			{ label: 'acc', value: `${Math.round((top?.accuracy ?? 0) * 100)}%` },
			{ label: 'players', value: results.length },
			{ label: 'duration', value: `${game?.duration ?? 0}s` }
		];
	});

	onMount(async () => {
		const replayPromise = getGameReplays(gameID).catch(() => null);
		if (appState.state?.results?.id === gameID) {
			game = appState.state.results;
			loading = false;
		} else {
			try {
				game = await getGame(gameID);
			} catch (err) {
				error = errorMessage(err);
			} finally {
				loading = false;
			}
		}
		replays = await replayPromise;
	});

	function runFor(userID: number): ReplayRun | undefined {
		return replays?.runs.find((run) => run.user_id === userID);
	}

	function canReplay(result: ResultView): boolean {
		const run = runFor(result.user_id);
		return Boolean(run && run.events.length > 0);
	}

	function openReplay(result: ResultView) {
		const run = runFor(result.user_id);
		if (run && run.events.length > 0) selectedRun = run;
	}

	function shareURLFor(run: ReplayRun): string {
		if (typeof location === 'undefined' || !game) return '';
		return `${location.origin}/replay/${game.id}/${run.user_id}`;
	}

	async function acknowledge() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			const next = await acknowledgeResults();
			applyState(next);
			if (game?.contest_code) await goto(`/contests/${game.contest_code}`);
			else if (next.state === 'room' && next.room) await goto(`/room/${next.room.code}`);
			else if (next.state === 'game' && next.game) await goto(`/game/${next.game.id}`);
			else if (next.state === 'results' && next.results) await goto(`/results/${next.results.id}`);
			else await goto(resolve('/play'));
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	function delta(value: number): string {
		return value > 0 ? `+${value}` : `${value}`;
	}
</script>

<svelte:head><title>results · OSDTYP</title></svelte:head>

<div class="min-h-screen bg-(--bg) p-6 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
				{game?.mode ?? 'match'} results {game?.room_code ? `· ${game.room_code}` : ''}
				{game?.contest_stage_name ? `· ${game.contest_stage_name}` : ''}
			</div>
			<div class="title-text text-7xl leading-none text-(--silver) md:text-8xl">RESULTS</div>
			{#if game}
				<div class="mt-2 font-mono text-[10px] text-(--silver)/30">
					game {game.id} · {game.duration}s · {LANGUAGE_LABELS[game.language] ?? game.language} · seed {game.seed}
				</div>
				{#if game.contest_code}
					<a
						href={resolve(`/contests/${game.contest_code}`)}
						class="mt-2 inline-block border border-(--border) px-2.5 py-1 font-mono text-[9px] tracking-[0.1em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer)"
						>← contest funnel</a
					>
				{/if}
			{/if}
		</div>
		{#if mine}
			<div class="text-right">
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">placement</div>
				<div class="font-mono text-4xl text-(--mer)">#{mine.rank}</div>
			</div>
		{/if}
	</header>

	{#if error}
		<div class="mt-4 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)">{error}</div>
	{/if}

	<main class="mx-auto mt-6 max-w-4xl">
		{#if loading}
			<div class="border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35">
				loading results…
			</div>
		{:else if !game}
			<div class="border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35">
				No result for this game.
			</div>
		{:else}
			<div class="border border-(--border) bg-(--fbg)">
				<div class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) px-5 py-3">
					<div>
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/40 uppercase">
							speed graph · every player
						</div>
						<div class="mt-1 font-mono text-[10px] text-(--silver)/30">
							labelled net wpm over {game.duration}s · click replay on a row for its recording
						</div>
					</div>
					<button
						class="cursor-pointer border border-(--mer) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
						onclick={() => (exportOpen = true)}>export image</button
					>
				</div>
				<div class="px-3 py-4 md:px-5">
					<SpeedGraph series={graphSeries} duration={game.duration} />
				</div>
			</div>

			<div class="mt-4 border border-(--border) bg-(--fbg)">
				<div class="grid grid-cols-[40px_1fr_70px_70px_80px] gap-2 border-b border-(--border) px-5 py-2 font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase md:grid-cols-[50px_1fr_90px_90px_110px]">
					<span>#</span>
					<span>player</span>
					<span class="text-right">wpm</span>
					<span class="text-right">acc</span>
					<span class="text-right">rating</span>
				</div>
				<div class="divide-y divide-(--border)">
					{#each results as result (result.user_id)}
						<div
							class={`grid grid-cols-[40px_1fr_70px_70px_80px] items-center gap-2 px-5 py-3 md:grid-cols-[50px_1fr_90px_90px_110px] ${result.user_id === myID ? 'bg-(--mer)/5' : ''}`}
						>
							<span class="font-mono text-sm text-(--silver)/45">{result.rank}</span>
							<div class="flex min-w-0 items-center gap-3">
								<Avatar src={result.avatar_url} name={result.username} class="h-8 w-8" text="text-xs" />
								<div class="min-w-0">
									<a
										href={`/profile/${encodeURIComponent(result.username)}`}
										class="block truncate text-sm text-(--silver) hover:text-(--mer)"
										>{result.username}{result.user_id === myID ? ' (you)' : ''}</a
									>
									<div class="mt-0.5 flex flex-wrap items-center gap-2 font-mono text-[9px] text-(--silver)/25">
										{#if result.is_bot}<span>bot</span>{/if}
										{#if result.left}<span>left early</span>{/if}
										{#if result.flagged}<span class="text-(--red)/70">anti-cheat flag</span>{/if}
										{#if canReplay(result)}
											<button
												class="cursor-pointer border border-(--border) px-1.5 py-0.5 text-[9px] tracking-[0.08em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer)"
												onclick={() => openReplay(result)}>replay</button
											>
										{:else}
											<span class="text-[9px] uppercase">no recording</span>
										{/if}
									</div>
								</div>
							</div>
							<span class="text-right font-mono text-sm text-(--silver)">{Math.round(result.wpm)}</span>
							<span class="text-right font-mono text-xs text-(--silver)/50">{Math.round(result.accuracy * 100)}%</span>
							<div class="text-right">
								{#if game.mode === 'ranked'}
									<div class="font-mono text-sm {result.rating_delta >= 0 ? 'text-(--mer)' : 'text-(--red)'}">
										{delta(result.rating_delta)}
									</div>
									<div class="font-mono text-[9px] text-(--silver)/25">{result.rating_after}</div>
								{:else}
									<div class="font-mono text-xs text-(--silver)/30">—</div>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>

			<div class="mt-4 flex flex-wrap items-center justify-between gap-3">
				<div class="font-mono text-[10px] text-(--silver)/30">
					{#if game.mode === 'ranked'}
						elo updates are final once you continue. flagged players do not change rating.
					{:else if game.mode === 'contest'}
						contest run · no ranked rating change. return to the funnel for the next attempt.
					{:else}
						room match · no rating change
					{/if}
				</div>
				<button
					class="cursor-pointer border border-(--mer) px-6 py-2.5 font-mono text-sm text-(--mer) hover:bg-(--mer)/10 disabled:opacity-40"
					onclick={acknowledge}
					disabled={busy}
					>{busy
						? 'continuing…'
						: game.contest_code
							? 'back to contest =>'
							: appState.state?.state === 'results' && appState.state.room
								? 'back to room =>'
								: 'continue =>'}</button
				>
			</div>
		{/if}
	</main>
</div>

{#if selectedRun && game}
	<ReplayModal
		run={selectedRun}
		snippet={game.snippet}
		startOffset={game.start_offset}
		duration={game.duration}
		language={game.language}
		shareURL={shareURLFor(selectedRun)}
		onclose={() => (selectedRun = null)}
	/>
{/if}

{#if exportOpen && game}
	<ExportModal
		title={`${game.mode} results`}
		subtitle={`game ${game.id}${game.room_code ? ` · ${game.room_code}` : ''} · ${results.length} players`}
		series={graphSeries}
		duration={game.duration}
		stats={exportStats}
		filename={`osdtype-${game.mode}-${game.id}`}
		onclose={() => (exportOpen = false)}
	/>
{/if}
