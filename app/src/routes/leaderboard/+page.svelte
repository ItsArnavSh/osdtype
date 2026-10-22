<script lang="ts">
	import { onMount } from 'svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { resolve } from '$app/paths';
	import { errorMessage } from '$lib/core/api/client';
	import { leaderboard } from '$lib/core/api/users';
	import type { LeaderboardRow } from '$lib/core/api/types';
	import { appState } from '$lib/core/state/state.svelte';

	let rows = $state<LeaderboardRow[]>([]);
	let myPosition = $state(0);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const data = await leaderboard(0, 50);
			rows = data.leaderboard;
			myPosition = data.my_position;
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head><title>leaderboard · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-6 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex items-end gap-4">
			<a
				href={resolve('/play')}
				class="mb-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← hub</a
			>
			<div class="title-text text-7xl leading-none text-(--silver) md:text-8xl">LADDER</div>
		</div>
		<div class="text-right">
			<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">your position</div>
			<div class="font-mono text-2xl text-(--mer)">#{myPosition || '—'}</div>
		</div>
	</header>

	{#if error}
		<div class="mt-4 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)">{error}</div>
	{/if}

	<main class="mt-5 max-w-5xl">
		<div class="border border-(--border) bg-(--fbg)">
			<div class="grid grid-cols-[44px_1fr_100px_90px_60px] gap-2 border-b border-(--border) px-5 py-2 font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase md:grid-cols-[56px_1fr_150px_110px_80px]">
				<span>#</span>
				<span>player</span>
				<span>rank</span>
				<span class="text-right">rating</span>
				<span class="text-right">games</span>
			</div>
			{#if loading}
				<div class="p-12 text-center font-mono text-xs text-(--silver)/35">loading ladder…</div>
			{:else if rows.length === 0}
				<div class="p-12 text-center font-mono text-xs text-(--silver)/35">No ranked games yet.</div>
			{:else}
				<div class="divide-y divide-(--border)">
					{#each rows as row (row.user.id)}
						<div
							class={`grid grid-cols-[44px_1fr_100px_90px_60px] items-center gap-2 px-5 py-3 md:grid-cols-[56px_1fr_150px_110px_80px] ${row.user.id === appState.user?.id ? 'bg-(--mer)/5' : ''}`}
						>
							<span class="font-mono text-sm {row.position <= 3 ? 'text-(--mer)' : 'text-(--silver)/35'}"
								>{row.position}</span
							>
							<a
								href={`/profile/${encodeURIComponent(row.user.username)}`}
								class="flex min-w-0 items-center gap-3 hover:text-(--mer)"
							>
								<Avatar src={row.user.avatar_url} name={row.user.username} class="h-7 w-7" />
								<span class="truncate text-sm text-(--silver)">{row.user.username}</span>
								{#if row.user.is_bot}
									<span class="font-mono text-[9px] text-(--silver)/25">bot</span>
								{/if}
							</a>
							<span class="font-mono text-xs text-(--silver)/55">{row.user.rank.label}</span>
							<span class="text-right font-mono text-sm text-(--silver)">{row.user.rating}</span>
							<span class="text-right font-mono text-xs text-(--silver)/30">{row.user.games_played}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</main>
</div>
