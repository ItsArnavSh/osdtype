<script lang="ts">
	import { resolve } from '$app/paths';
	import { onDestroy, onMount } from 'svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { friends, searchUsers } from '$lib/core/api/users';
	import type { FriendView, UserView } from '$lib/core/api/types';

	let list = $state<FriendView[]>([]);
	let loading = $state(true);
	let error = $state('');

	let query = $state('');
	let results = $state<UserView[]>([]);
	let searching = $state(false);
	let searched = $state(false);
	let searchError = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | null = null;
	let searchSeq = 0;

	let mutual = $derived(list.filter((friend) => friend.is_friend));
	let following = $derived(list.filter((friend) => !friend.is_friend));
	let onlineCount = $derived(list.filter((friend) => friend.online).length);

	onMount(async () => {
		try {
			list = await friends();
			list = [...list].sort(
				(a, b) =>
					Number(b.online) - Number(a.online) ||
					Number(b.is_friend) - Number(a.is_friend) ||
					a.user.username.localeCompare(b.user.username)
			);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	});

	onDestroy(() => {
		if (searchTimer) clearTimeout(searchTimer);
	});

	function scheduleSearch() {
		if (searchTimer) clearTimeout(searchTimer);
		const q = query.trim();
		searched = false;
		searchError = '';
		if (!q) {
			results = [];
			searching = false;
			return;
		}
		searching = true;
		const seq = ++searchSeq;
		searchTimer = setTimeout(() => runSearch(q, seq), 220);
	}

	async function runSearch(q: string, seq: number) {
		try {
			const found = await searchUsers(q);
			if (seq !== searchSeq) return;
			results = found;
			searched = true;
		} catch (err) {
			if (seq !== searchSeq) return;
			results = [];
			searchError = errorMessage(err);
			searched = true;
		} finally {
			if (seq === searchSeq) searching = false;
		}
	}

	function submitSearch(event: SubmitEvent) {
		event.preventDefault();
		if (searchTimer) clearTimeout(searchTimer);
		const q = query.trim();
		if (!q) return;
		searching = true;
		searched = false;
		searchError = '';
		void runSearch(q, ++searchSeq);
	}

	function clearSearch() {
		if (searchTimer) clearTimeout(searchTimer);
		searchSeq++;
		query = '';
		results = [];
		searching = false;
		searched = false;
		searchError = '';
	}
</script>

<svelte:head><title>friends · OSDTYP</title></svelte:head>

<div class="min-h-screen bg-(--bg) p-6 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<a
				href={resolve('/play')}
				class="border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
				>← hub</a
			>
			<div class="title-text mt-2 text-7xl leading-none text-(--silver) md:text-8xl">FRIENDS</div>
		</div>
		<div class="text-right">
			<div class="font-mono text-3xl text-(--mer)">{onlineCount}</div>
			<div class="mt-1 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
				online now
			</div>
		</div>
	</header>

	{#if error}
		<div class="mt-4 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)">{error}</div>
	{/if}

	<section class="mx-auto mt-5 max-w-5xl border border-(--border) bg-(--fbg)">
		<div class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-5 py-3">
			<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
				find players
			</div>
			<span class="font-mono text-[10px] text-(--silver)/25">search by username</span>
		</div>
		<form class="flex flex-wrap gap-2 p-3" onsubmit={submitSearch}>
			<div class="relative min-w-0 flex-1">
				<input
					class="w-full border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none placeholder:text-(--silver)/25 focus:border-(--mer)"
					placeholder="username…"
					autocomplete="off"
					spellcheck="false"
					bind:value={query}
					oninput={scheduleSearch}
				/>
				{#if query}
					<button
						type="button"
						class="absolute top-1/2 right-2 -translate-y-1/2 cursor-pointer px-1 font-mono text-[10px] text-(--silver)/30 hover:text-(--silver)"
						onclick={clearSearch}
						aria-label="clear search">×</button
					>
				{/if}
			</div>
			<button
				type="submit"
				class="cursor-pointer border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
				>search =></button
			>
		</form>
		{#if searching}
			<div class="border-t border-(--border) px-5 py-4 font-mono text-[11px] text-(--silver)/30">
				searching…
			</div>
		{:else if searchError}
			<div class="border-t border-(--border) px-5 py-4 font-mono text-[11px] text-(--red)/70">
				{searchError}
			</div>
		{:else if searched && results.length === 0}
			<div class="border-t border-(--border) px-5 py-4 font-mono text-[11px] text-(--silver)/30">
				No players found for “{query.trim()}”.
			</div>
		{:else if results.length > 0}
			<div class="grid border-t border-(--border) sm:grid-cols-2">
				{#each results as user (user.id)}
					<a
						href={resolve(`/profile/${user.username}`)}
						class="group flex items-center gap-3 border-b border-(--border) px-5 py-3 transition-colors hover:bg-(--mer)/5 sm:odd:border-r"
					>
						<Avatar
							src={user.avatar_url}
							name={user.username}
							class="h-9 w-9"
							text="text-[10px]"
						/>
						<div class="min-w-0 flex-1">
							<div class="truncate text-sm text-(--silver)">{user.username}</div>
							<div class="mt-0.5 font-mono text-[10px] text-(--silver)/35">
								{user.rating} elo · {user.games_played} games
							</div>
						</div>
						<span class="font-mono text-[10px] text-(--mer)/40 group-hover:text-(--mer)"
							>profile =></span
						>
					</a>
				{/each}
			</div>
		{/if}
	</section>

	<main class="mx-auto mt-3 grid max-w-5xl gap-3 lg:grid-cols-2">
		{#if loading}
			<div class="border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35 lg:col-span-2">
				loading friends…
			</div>
		{:else}
			<section class="border border-(--border) bg-(--fbg)">
				<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						mutual friends
					</div>
					<span class="font-mono text-[10px] text-(--mer)/50">{mutual.length}</span>
				</div>
				{#if mutual.length === 0}
					<div class="p-6 font-mono text-[11px] leading-relaxed text-(--silver)/30">
						Friends are mutual follows. Follow a player from their profile; when they follow you
						back, they appear here and become available as room invite suggestions.
					</div>
				{:else}
					<div class="divide-y divide-(--border)">
						{#each mutual as friend (friend.user.id)}
							<a
								href={resolve(`/profile/${friend.user.username}`)}
								class="flex items-center gap-3 px-5 py-3 transition-colors hover:bg-(--mer)/5"
							>
								<Avatar
									src={friend.user.avatar_url}
									name={friend.user.username}
									class="h-9 w-9"
									text="text-[10px]"
								/>
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2">
										<span class="truncate text-sm text-(--silver)">{friend.user.username}</span>
										<span
											class={`h-1.5 w-1.5 shrink-0 ${friend.online ? 'bg-(--mer)' : 'bg-(--silver)/20'}`}
										></span>
										<span class="font-mono text-[9px] text-(--silver)/30 uppercase">
											{friend.online ? 'online' : 'offline'}
										</span>
									</div>
									<div class="mt-0.5 font-mono text-[10px] text-(--silver)/35">
										{friend.user.rating} elo
									</div>
								</div>
								{#if friend.follows_you}
									<span class="font-mono text-[9px] text-(--silver)/25 lowercase">follows you</span>
								{/if}
								<span class="font-mono text-[10px] text-(--mer)/40">profile =></span>
							</a>
						{/each}
					</div>
				{/if}
			</section>

			<section class="border border-(--border) bg-(--fbg)">
				<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						following
					</div>
					<span class="font-mono text-[10px] text-(--silver)/35">{following.length}</span>
				</div>
				{#if following.length === 0}
					<div class="p-6 font-mono text-[11px] text-(--silver)/30">
						You are not following anyone yet.
					</div>
				{:else}
					<div class="divide-y divide-(--border)">
						{#each following as friend (friend.user.id)}
							<a
								href={resolve(`/profile/${friend.user.username}`)}
								class="flex items-center gap-3 px-5 py-3 transition-colors hover:bg-(--mer)/5"
							>
								<Avatar
									src={friend.user.avatar_url}
									name={friend.user.username}
									class="h-9 w-9"
									text="text-[10px]"
								/>
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2">
										<span class="truncate text-sm text-(--silver)">{friend.user.username}</span>
										<span
											class={`h-1.5 w-1.5 shrink-0 ${friend.online ? 'bg-(--mer)' : 'bg-(--silver)/20'}`}
										></span>
									</div>
									<div class="mt-0.5 font-mono text-[10px] text-(--silver)/35">
										{friend.user.rating} elo
									</div>
								</div>
								<span class="font-mono text-[9px] text-(--silver)/25 uppercase">followed</span>
							</a>
						{/each}
					</div>
				{/if}
			</section>
		{/if}
	</main>

	<p class="mx-auto mt-4 max-w-5xl font-mono text-[10px] leading-relaxed text-(--silver)/25">
		Saved practice runs and finished matches are visible on every profile. Open any run to watch its
		simulated replay or copy its public share link.
	</p>
</div>
