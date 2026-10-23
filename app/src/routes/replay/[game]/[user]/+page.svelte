<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import ReplayPlayer from '$lib/components/ReplayPlayer.svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { getPublicReplay } from '$lib/core/api/game';
	import { LANGUAGE_LABELS } from '$lib/core/api/types';
	import type { ReplayBundle } from '$lib/core/api/types';

	const gameID = $derived(Number(page.params.game ?? 0));
	const userID = $derived(Number(page.params.user ?? 0));

	let bundle = $state<ReplayBundle | null>(null);
	let loading = $state(true);
	let error = $state('');

	$effect(() => {
		const game = gameID;
		const user = userID;
		if (!game || !user) {
			error = 'This replay link is incomplete.';
			loading = false;
			return;
		}
		loading = true;
		getPublicReplay(game, user)
			.then((data) => {
				bundle = data;
			})
			.catch((err) => {
				error = errorMessage(err);
			})
			.finally(() => {
				loading = false;
			});
	});

	function shareURL(): string {
		if (typeof location === 'undefined') return '';
		return location.href;
	}
</script>

<svelte:head>
	<title>{bundle?.runs[0]?.username ?? 'shared'} replay · OSDTYP</title>
</svelte:head>

<div class="min-h-screen bg-(--bg) p-5 md:p-8">
	<header class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4">
		<a href={resolve('/')} class="flex items-baseline gap-3">
			<span class="title-text text-4xl leading-none text-(--silver)">OSDTYP</span>
			<span class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">shared replay</span>
		</a>
		<a
			href={resolve('/play')}
			class="border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
			>← hub</a
		>
	</header>

	<main class="mx-auto mt-6 max-w-6xl">
		{#if loading}
			<div class="border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35">
				loading recording…
			</div>
		{:else if error || !bundle || bundle.runs.length === 0}
			<div class="border border-(--border) bg-(--fbg) p-12 text-center">
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--red) uppercase">replay unavailable</div>
				<p class="mt-2 font-mono text-xs text-(--silver)/40">{error || 'No recording is available for this run.'}</p>
			</div>
		{:else}
			{@const game = bundle.game}
			{@const run = bundle.runs[0]}
			<div class="border border-(--border) bg-(--fbg) p-4 md:p-6">
				<div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
					<div class="font-mono text-[10px] text-(--silver)/35">
						{game.mode} · {game.duration}s · {LANGUAGE_LABELS[game.language] ?? game.language}
						{#if game.temporary}
							<span class="ml-2 text-(--mer)/70">temporary practice</span>
						{/if}
					</div>
					<div class="font-mono text-[10px] text-(--silver)/25">replay · {game.duration}s</div>
				</div>
				<ReplayPlayer
					{run}
					snippet={game.snippet}
					startOffset={game.start_offset}
					duration={game.duration}
					language={game.language}
					shareURL={shareURL()}
				/>
			</div>
			<p class="mt-3 font-mono text-[10px] text-(--silver)/25">
				A key-by-key replay of the run. Share links open without an account.
			</p>
		{/if}
	</main>
</div>
