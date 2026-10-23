<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { resolve } from '$app/paths';
	import { errorMessage } from '$lib/core/api/client';
	import { friends, leaderboard, searchLeaderboard } from '$lib/core/api/users';
	import type { FriendView, LeaderboardRow, LobbyDuration } from '$lib/core/api/types';
	import { LOBBY_DURATIONS, userElo } from '$lib/core/api/types';
	import { appState } from '$lib/core/state/state.svelte';

	let duration = $state<LobbyDuration>(90);
	let rows = $state<LeaderboardRow[]>([]);
	let nearby = $state<LeaderboardRow[]>([]);
	let myPosition = $state(0);
	let loading = $state(true);
	let error = $state('');

	let friendList = $state<FriendView[]>([]);
	let friendsLoading = $state(true);
	let friendsError = $state('');

	let query = $state('');
	let searchResults = $state<LeaderboardRow[]>([]);
	let searching = $state(false);
	let searched = $state(false);
	let searchError = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | null = null;
	let searchSeq = 0;
	let loadSeq = 0;

	let topIDs = $derived(new Set(rows.map((row) => row.user.id)));
	let nearbyExtra = $derived(nearby.filter((row) => !topIDs.has(row.user.id)));
	let me = $derived(
		rows.find((row) => row.user.id === appState.user?.id) ??
			nearby.find((row) => row.user.id === appState.user?.id) ??
			null
	);
	let rankLabel = $derived(myPosition > 0 ? `#${myPosition}` : '—');
	let myElo = $derived(appState.user ? userElo(appState.user, duration).rating : null);
	let friendRows = $derived(
		friendList
			.filter((friend) => friend.is_friend)
			.map((friend) => {
				const elo = userElo(friend.user, duration);
				return { friend, elo };
			})
			.sort(
				(a, b) =>
					b.elo.rating - a.elo.rating ||
					b.elo.games_played - a.elo.games_played ||
					a.friend.user.username.localeCompare(b.friend.user.username)
			)
	);
	let onlineFriends = $derived(friendRows.filter(({ friend }) => friend.online).length);

	async function load() {
		const seq = ++loadSeq;
		loading = true;
		error = '';
		try {
			const data = await leaderboard(duration, 0, 5);
			if (seq !== loadSeq) return;
			rows = data.leaderboard ?? [];
			nearby = data.nearby ?? [];
			myPosition = data.my_position ?? 0;
		} catch (err) {
			if (seq !== loadSeq) return;
			error = errorMessage(err);
		} finally {
			if (seq === loadSeq) loading = false;
		}
	}

	async function loadFriends() {
		friendsLoading = true;
		friendsError = '';
		try {
			friendList = await friends();
		} catch (err) {
			friendsError = errorMessage(err);
		} finally {
			friendsLoading = false;
		}
	}

	function selectDuration(next: LobbyDuration) {
		if (next === duration) return;
		duration = next;
		rows = [];
		nearby = [];
		myPosition = 0;
		clearSearch();
		void load();
	}

	function scheduleSearch() {
		if (searchTimer) clearTimeout(searchTimer);
		const q = query.trim();
		searchError = '';
		if (!q) {
			searchSeq++;
			searchResults = [];
			searching = false;
			searched = false;
			return;
		}
		searching = true;
		searched = false;
		const seq = ++searchSeq;
		searchTimer = setTimeout(() => void runSearch(q, seq), 200);
	}

	async function runSearch(q: string, seq: number) {
		try {
			const data = await searchLeaderboard(duration, q);
			if (seq !== searchSeq) return;
			searchResults = data.results ?? [];
			searched = true;
		} catch (err) {
			if (seq !== searchSeq) return;
			searchResults = [];
			searchError = errorMessage(err);
			searched = true;
		} finally {
			if (seq === searchSeq) searching = false;
		}
	}

	function clearSearch() {
		if (searchTimer) clearTimeout(searchTimer);
		searchSeq++;
		query = '';
		searchResults = [];
		searching = false;
		searched = false;
		searchError = '';
	}

	onMount(() => {
		void load();
		void loadFriends();
	});

	onDestroy(() => {
		if (searchTimer) clearTimeout(searchTimer);
	});
</script>

{#snippet leaderRow(row: LeaderboardRow)}
	<div
		class={`grid grid-cols-[34px_minmax(0,1fr)_70px] items-center gap-2 px-4 py-2.5 md:grid-cols-[46px_minmax(0,1fr)_110px_64px] md:px-5 ${
			row.user.id === appState.user?.id ? 'bg-(--mer)/5' : ''
		}`}
	>
		<span class={`font-mono text-sm ${row.position <= 3 ? 'text-(--mer)' : 'text-(--silver)/35'}`}>
			{row.position}
		</span>
		<a
			href={resolve(`/profile/${row.user.username}`)}
			class="flex min-w-0 items-center gap-2.5 hover:text-(--mer)"
		>
			<Avatar src={row.user.avatar_url} name={row.user.username} class="h-7 w-7" />
			<span class="truncate text-sm text-(--silver)">{row.user.username}</span>
			{#if row.user.id === appState.user?.id}
				<span class="font-mono text-[9px] tracking-[0.1em] text-(--mer)/70 uppercase">you</span>
			{/if}
			{#if row.user.is_bot}
				<span class="font-mono text-[9px] text-(--silver)/25">bot</span>
			{/if}
		</a>
		<span class="text-right font-mono text-sm text-(--silver)">{row.rating}</span>
		<span class="hidden text-right font-mono text-xs text-(--silver)/35 md:block">
			{row.games_played}g
		</span>
	</div>
{/snippet}

<svelte:head><title>leaderboard · OSDTYP</title></svelte:head>

<div
	class="flex min-h-screen flex-col bg-(--bg) p-5 md:p-8 lg:h-screen lg:min-h-0 lg:overflow-hidden"
>
	<header class="flex shrink-0 flex-wrap items-end justify-between gap-4">
		<div class="flex items-end gap-4">
			<a
				href={resolve('/play')}
				class="mb-1 border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
				>← hub</a
			>
			<div class="title-text text-6xl leading-none text-(--silver) md:text-8xl">LEADERBOARD</div>
		</div>
		<div class="flex items-end gap-6 text-right">
			<div>
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
					your position
				</div>
				<div class="mt-1 font-mono text-2xl text-(--mer)">{rankLabel}</div>
			</div>
			<div>
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
					your {duration}s elo
				</div>
				<div class="mt-1 font-mono text-2xl text-(--silver)">{myElo ?? '—'}</div>
			</div>
		</div>
	</header>

	{#if error}
		<div
			class="mt-4 flex shrink-0 items-center justify-between gap-3 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)"
		>
			<span>{error}</span>
			<button class="cursor-pointer text-(--red)/60 hover:text-(--red)" onclick={load}
				>retry</button
			>
		</div>
	{/if}

	<div class="mt-4 flex shrink-0 flex-wrap items-center justify-between gap-3">
		<div class="flex border border-(--border)">
			{#each LOBBY_DURATIONS as clock (clock)}
				<button
					class={`cursor-pointer px-4 py-2 font-mono text-[11px] tracking-[0.1em] uppercase transition-colors ${
						duration === clock
							? 'bg-(--mer)/10 text-(--mer)'
							: 'text-(--silver)/35 hover:text-(--silver)'
					}`}
					onclick={() => selectDuration(clock)}
				>{clock}s</button>
			{/each}
		</div>
		<span class="font-mono text-[10px] text-(--silver)/25">
			top five · ±five around you · friends on the right
		</span>
	</div>

	<main class="mt-3 grid min-h-0 flex-1 gap-3 lg:grid-cols-[minmax(0,1fr)_340px]">
		<section class="flex min-h-[380px] flex-col border border-(--border) bg-(--fbg) lg:min-h-0">
			<div
				class="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-(--border) px-5 py-3"
			>
				<div>
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						top players
					</div>
					<div class="mt-0.5 font-mono text-[10px] text-(--silver)/25">
						{duration}s ladder · elo
					</div>
				</div>
				<div class="relative w-full sm:w-72">
					<input
						class="w-full border border-(--border) bg-(--bg) px-3 py-2 pr-8 font-mono text-xs text-(--silver) outline-none placeholder:text-(--silver)/25 focus:border-(--mer)"
						placeholder="search players…"
						autocomplete="off"
						spellcheck="false"
						bind:value={query}
						oninput={scheduleSearch}
					/>
					{#if query}
						<button
							type="button"
							class="absolute top-1/2 right-2 -translate-y-1/2 cursor-pointer px-1 font-mono text-[11px] text-(--silver)/30 hover:text-(--silver)"
							onclick={clearSearch}
							aria-label="clear leaderboard search">×</button
						>
					{/if}
					{#if query.trim()}
						<div
							class="absolute top-full right-0 left-0 z-40 mt-1 max-h-72 overflow-y-auto border border-(--border) bg-(--bg) shadow-2xl"
						>
							{#if searching}
								<div class="px-3 py-3 font-mono text-[11px] text-(--silver)/35">
									searching…
								</div>
							{:else if searchError}
								<div class="px-3 py-3 font-mono text-[11px] text-(--red)/70">
									{searchError}
								</div>
							{:else if searched && searchResults.length === 0}
								<div class="px-3 py-3 font-mono text-[11px] text-(--silver)/35">
									No ranked players match “{query.trim()}” on this clock.
								</div>
							{:else if searchResults.length > 0}
								{#each searchResults as row (row.user.id)}
									<a
										href={resolve(`/profile/${row.user.username}`)}
										class="flex items-center gap-2.5 border-b border-(--border)/60 px-3 py-2.5 last:border-b-0 hover:bg-(--mer)/5"
									>
										<span class="w-8 shrink-0 font-mono text-[11px] text-(--silver)/35"
											>#{row.position}</span
										>
										<Avatar
											src={row.user.avatar_url}
											name={row.user.username}
											class="h-7 w-7 shrink-0"
										/>
										<span class="min-w-0 flex-1 truncate text-xs text-(--silver)"
											>{row.user.username}</span
										>
										<span class="font-mono text-xs text-(--mer)">{row.rating}</span>
									</a>
								{/each}
							{/if}
						</div>
					{/if}
				</div>
			</div>

			<div
				class="grid shrink-0 grid-cols-[34px_minmax(0,1fr)_70px] gap-2 border-b border-(--border) px-4 py-2 font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase md:grid-cols-[46px_minmax(0,1fr)_110px_64px] md:px-5"
			>
				<span>#</span>
				<span>player</span>
				<span class="text-right">elo</span>
				<span class="hidden text-right md:block">games</span>
			</div>

			<div class="min-h-0 flex-1 overflow-y-auto">
				{#if loading}
					<div class="p-12 text-center font-mono text-xs text-(--silver)/35">
						loading leaderboard…
					</div>
				{:else if rows.length === 0}
					<div class="flex flex-col items-center gap-3 px-6 py-16 text-center">
						<div class="text-xl text-(--silver)/55">
							No ranked players on the {duration}s clock yet.
						</div>
						<p class="max-w-md font-mono text-[11px] leading-relaxed text-(--silver)/30">
							Everyone begins at 400 Elo on each clock. Finish a {duration}s ranked race to put
							the first number on this ladder.
						</p>
						<a
							href={resolve('/play/ranked')}
							class="mt-2 border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
							>play ranked =></a
						>
					</div>
				{:else}
					<div class="divide-y divide-(--border)">
						{#each rows as row (row.user.id)}
							{@render leaderRow(row)}
						{/each}
					</div>

					{#if myPosition > 5 && nearbyExtra.length > 0}
						<div
							class="flex items-center justify-between border-y border-(--border) bg-(--bg) px-5 py-2"
						>
							<span
								class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/70 uppercase"
								>around your rank</span
							>
							<span class="font-mono text-[10px] text-(--silver)/25">±5 · exact position</span>
						</div>
						<div class="divide-y divide-(--border)">
							{#each nearbyExtra as row (row.user.id)}
								{@render leaderRow(row)}
							{/each}
						</div>
					{:else if myPosition > 0}
						<div
							class="border-t border-(--border) px-5 py-3 font-mono text-[10px] leading-relaxed text-(--silver)/30"
						>
							You are #{myPosition} on the {duration}s ladder and already inside the top five.
						</div>
					{:else}
						<div
							class="border-t border-(--border) px-5 py-3 font-mono text-[10px] leading-relaxed text-(--silver)/30"
						>
							Play a {duration}s ranked race to enter this ladder. Your starting Elo is 400.
						</div>
					{/if}
				{/if}
			</div>

			<div
				class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-t border-(--border) px-5 py-2 font-mono text-[10px] text-(--silver)/25"
			>
				<span>
					{#if me}
						you are #{myPosition} · {me.rating} elo
					{:else}
						unranked on this clock
					{/if}
				</span>
				<span>positions order by elo, then games played</span>
			</div>
		</section>

		<aside class="flex min-h-[260px] flex-col border border-(--border) bg-(--fbg) lg:min-h-0">
			<div
				class="flex shrink-0 items-center justify-between border-b border-(--border) px-5 py-3"
			>
				<div>
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/70 uppercase">
						friends leaderboard
					</div>
					<div class="mt-0.5 font-mono text-[10px] text-(--silver)/25">
						mutual friends · {duration}s
					</div>
				</div>
				<span class="font-mono text-[10px] text-(--silver)/30">
					{onlineFriends}/{friendRows.length} online
				</span>
			</div>

			<div class="min-h-0 flex-1 overflow-y-auto">
				{#if friendsLoading}
					<div class="p-8 text-center font-mono text-[11px] text-(--silver)/35">
						loading friends…
					</div>
				{:else if friendsError}
					<div class="p-5 font-mono text-[11px] leading-relaxed text-(--red)/70">
						{friendsError}
					</div>
				{:else if friendRows.length === 0}
					<div class="flex flex-col gap-3 p-5 font-mono text-[11px] leading-relaxed text-(--silver)/35">
						<span>No mutual friends yet.</span>
						<a href={resolve('/friends')} class="text-(--mer) hover:underline">find friends =></a>
					</div>
				{:else}
					<div class="divide-y divide-(--border)">
						{#each friendRows as { friend, elo }, index (friend.user.id)}
							<a
								href={resolve(`/profile/${friend.user.username}`)}
								class="flex items-center gap-2.5 px-4 py-2.5 transition-colors hover:bg-(--mer)/5"
							>
								<span
									class={`w-5 shrink-0 font-mono text-[11px] ${
										index < 3 ? 'text-(--mer)' : 'text-(--silver)/30'
									}`}
									>{index + 1}</span
								>
								<Avatar
									src={friend.user.avatar_url}
									name={friend.user.username}
									class="h-7 w-7 shrink-0"
								/>
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-1.5">
										<span class="truncate text-xs text-(--silver)">{friend.user.username}</span>
										<span
											class={`h-1.5 w-1.5 shrink-0 ${
												friend.online ? 'bg-(--mer)' : 'bg-(--silver)/20'
											}`}
										></span>
									</div>
									<div class="mt-0.5 font-mono text-[9px] text-(--silver)/25">
										{elo.games_played} ranked games
									</div>
								</div>
								<span class="font-mono text-sm text-(--mer)">{elo.rating}</span>
							</a>
						{/each}
					</div>
				{/if}
			</div>

			<div
				class="shrink-0 border-t border-(--border) px-5 py-2 font-mono text-[10px] leading-relaxed text-(--silver)/25"
			>
				Only this friends list scrolls. The main ladder stays top five plus your window.
			</div>
		</aside>
	</main>
</div>
