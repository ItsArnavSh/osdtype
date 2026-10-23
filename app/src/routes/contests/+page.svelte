<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import ContestCard from '$lib/components/ContestCard.svelte';
	import { appState } from '$lib/core/state/state.svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { listContests } from '$lib/core/api/contests';
	import { onSocketEvent } from '$lib/core/api/socket';
	import type { ContestListView, ContestView } from '$lib/core/api/types';

	let board = $state<ContestListView | null>(null);
	let loading = $state(true);
	let error = $state('');
	let now = $state(Date.now());
	let offset = $state(0);
	let code = $state('');
	let refreshTimer: ReturnType<typeof setInterval> | null = null;
	let clockTimer: ReturnType<typeof setInterval> | null = null;
	let unsubscribe: (() => void) | null = null;
	let refreshing = false;
	let query = $state('');

	function matches(contest: ContestView): boolean {
		const needle = query.trim().toLowerCase();
		if (!needle) return true;
		return [
			contest.title,
			contest.description,
			contest.code,
			contest.language,
			contest.status,
			contest.host?.username
		].some((value) => (value ?? '').toLowerCase().includes(needle));
	}

	let filtered = $derived((board?.contests ?? []).filter(matches));
	let live = $derived(filtered.filter((contest) => contest.status === 'started'));
	let upcoming = $derived(filtered.filter((contest) => contest.status !== 'started'));
	let mine = $derived((board?.mine ?? []).filter(matches));
	let trending = $derived((board?.trending ?? []).filter(matches));

	async function load(silent = false) {
		if (refreshing) return;
		refreshing = true;
		if (!silent) loading = true;
		error = '';
		try {
			const next = await listContests();
			board = next;
			offset = Date.parse(next.server_time) - Date.now();
			now = Date.now() + offset;
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
			refreshing = false;
		}
	}

	function joinByCode() {
		const value = code.trim().toUpperCase();
		if (value) void goto(resolve(`/contests/${value}`));
	}

	onMount(() => {
		void load();
		clockTimer = setInterval(() => (now = Date.now() + offset), 1000);
		refreshTimer = setInterval(() => void load(true), 30_000);
		unsubscribe = onSocketEvent((event) => {
			if (
				event.type === 'contest_started' ||
				event.type === 'contest_enrollment' ||
				event.type === 'contest_cancelled'
			) {
				void load(true);
			}
		});
		return () => {
			if (clockTimer) clearInterval(clockTimer);
			if (refreshTimer) clearInterval(refreshTimer);
			unsubscribe?.();
		};
	});
</script>

<svelte:head><title>contests · OSDTYP</title></svelte:head>

<div class="min-h-screen bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4 border-b border-(--border) pb-4">
		<div>
			<a
				href={resolve('/play')}
				class="border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
				>← hub</a
			>
			<div class="mt-1 flex flex-wrap items-end gap-x-4 gap-y-2">
				<h1 class="title-text text-6xl leading-none text-(--silver) md:text-8xl">CONTESTS</h1>
				<span class="mb-1 font-mono text-[9px] tracking-[0.16em] text-(--mer) uppercase"
					>public events / private invitations</span
				>
			</div>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<a
				href="/docs#contests"
				class="border border-(--border) px-3 py-2 font-mono text-[10px] tracking-[0.1em] text-(--silver)/40 uppercase hover:border-(--mer) hover:text-(--mer)"
				>contest docs</a
			>
			<form class="flex items-center" onsubmit={(event) => { event.preventDefault(); joinByCode(); }}>
				<input
					class="w-32 border border-(--border) bg-(--fbg) px-3 py-2 font-mono text-[10px] tracking-[0.12em] text-(--silver) uppercase outline-none placeholder:text-(--silver)/20 focus:border-(--mer)"
					bind:value={code}
					placeholder="event code"
					maxlength="16"
				/>
				<button
					type="submit"
					class="cursor-pointer border border-l-0 border-(--border) px-2.5 py-2 font-mono text-[10px] text-(--silver)/40 hover:text-(--silver)"
					title="open by code">→</button
				>
			</form>
			{#if appState.user}
				<a
					href={resolve('/contests/new')}
					class="border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
					>+ host contest</a
				>
			{:else}
				<a
					href={`/login?next=${encodeURIComponent('/contests/new')}`}
					class="border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
					>sign in to host →</a
				>
			{/if}
		</div>
	</header>

	{#if error}
		<div class="mt-4 border border-(--red)/40 px-4 py-3 font-mono text-xs text-(--red)">{error}</div>
	{/if}

	<main class="mt-5 grid gap-4 xl:grid-cols-[minmax(0,1fr)_330px]">
		<section class="min-w-0">
			<div class="flex items-end justify-between gap-3">
				<div>
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--red) uppercase">
						live board
					</div>
					<h2 class="mt-1 text-3xl text-(--silver)">Public events</h2>
				</div>
				<span class="mb-1 font-mono text-[10px] text-(--silver)/25">
					{filtered.length} {query.trim() ? 'matching' : 'listed'}
				</span>
			</div>

			<div class="mt-3 flex flex-wrap items-center gap-2">
				<input
					class="min-w-0 flex-1 border border-(--border) bg-(--fbg) px-3 py-2 font-mono text-xs text-(--silver) outline-none placeholder:text-(--silver)/20 focus:border-(--mer)"
					bind:value={query}
					placeholder="search events: title, code, host, language or status"
					maxlength="80"
				/>
				{#if query.trim()}
					<button
						type="button"
						class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] tracking-[0.08em] text-(--silver)/40 uppercase hover:border-(--mer)/50 hover:text-(--mer)"
						onclick={() => (query = '')}>clear</button
					>
				{/if}
			</div>

			{#if loading}
				<div class="mt-3 grid gap-3 md:grid-cols-2">
					{#each [0, 1, 2, 3] as i (i)}
						<div class="h-72 animate-pulse border border-(--border) bg-(--fbg)"></div>
					{/each}
				</div>
			{:else if !board || board.contests.length === 0}
				<div class="mt-3 border border-(--border) bg-(--fbg) p-8">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
						quiet board
					</div>
					<p class="mt-2 max-w-lg font-mono text-xs leading-relaxed text-(--silver)/40">
						No public contests are scheduled right now. Host the first one, or open a private
						event with its invite link.
					</p>
					{#if appState.user}
						<a
							href={resolve('/contests/new')}
							class="mt-4 inline-block border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
							>host a contest →</a
						>
					{/if}
				</div>
			{:else}
				{#if filtered.length === 0}
					<div class="mt-3 border border-(--border) bg-(--fbg) p-8">
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
							no matches
						</div>
						<p class="mt-2 max-w-lg font-mono text-xs leading-relaxed text-(--silver)/40">
							No event matches “{query.trim()}”. Try a title, event code, host name or
							language.
						</p>
						<button
							type="button"
							class="mt-4 cursor-pointer border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
							onclick={() => (query = '')}>clear search</button
						>
					</div>
				{:else}
					{#if live.length > 0}
						<div class="mt-3">
							<div
								class="mb-2 flex items-center gap-2 font-mono text-[9px] tracking-[0.14em] text-(--mer) uppercase"
							>
								<span class="h-1.5 w-1.5 animate-pulse bg-(--mer)"></span> racing now
							</div>
							<div class="grid gap-3 md:grid-cols-2">
								{#each live as contest (contest.id)}
									<ContestCard {contest} {now} />
								{/each}
							</div>
						</div>
					{/if}
					<div class="mt-4 grid gap-3 md:grid-cols-2">
						{#each upcoming as contest (contest.id)}
							<ContestCard {contest} {now} />
						{/each}
					</div>
				{/if}
			{/if}
		</section>

		<aside class="flex flex-col gap-3">
			<div class="border border-(--border) bg-(--fbg) p-5">
				<div class="flex items-center justify-between">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
						trending
					</div>
					<span class="font-mono text-[9px] text-(--silver)/20">by enrollment</span>
				</div>
				{#if trending.length === 0}
					<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/25">
						Trending events appear as players enroll.
					</p>
				{:else}
					<div class="mt-2 divide-y divide-(--border)">
						{#each trending as contest, index (contest.id)}
							<a
								href={resolve(`/contests/${contest.code}`)}
								class="flex items-start gap-3 py-3 transition-colors hover:bg-(--mer)/5"
							>
								<span class="font-mono text-lg leading-none text-(--silver)/20">0{index + 1}</span>
								<span class="min-w-0 flex-1">
									<span class="block truncate text-[13px] text-(--silver)/75">{contest.title}</span>
									<span class="mt-1 block font-mono text-[9px] text-(--silver)/30">
										{contest.entry_count}/{contest.max_players} enrolled · {contest.duration}s
									</span>
								</span>
								<span class="mt-0.5 h-2 w-2 shrink-0" style={`background:${contest.accent}`}></span>
							</a>
						{/each}
					</div>
				{/if}
			</div>

			<div class="border border-(--border) bg-(--fbg) p-5">
				<div class="flex items-center justify-between">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
						your events
					</div>
					{#if appState.user}
						<a href={resolve('/contests/new')} class="font-mono text-[10px] text-(--mer)/80 hover:text-(--mer)"
							>new</a
						>
					{/if}
				</div>
				{#if !appState.user}
					<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/30">
						Sign in to host, enroll and keep track of your contests.
					</p>
					<a
						href={`/login?next=${encodeURIComponent('/contests')}`}
						class="mt-3 flex justify-center border border-(--border) px-3 py-2 font-mono text-[10px] tracking-[0.1em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer)"
						>sign in →</a
					>
				{:else if mine.length === 0}
					<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/25">
						Nothing yet. Host an event or enroll from a public contest.
					</p>
				{:else}
					<div class="mt-2 divide-y divide-(--border)">
						{#each mine.slice(0, 7) as contest (contest.id)}
							<a
								href={resolve(`/contests/${contest.code}`)}
								class="flex items-center gap-2 py-2.5 transition-colors hover:bg-(--mer)/5"
							>
								<span class="h-1.5 w-1.5 shrink-0" style={`background:${contest.accent}`}></span>
								<span class="min-w-0 flex-1 truncate font-mono text-[11px] text-(--silver)/60"
									>{contest.title}</span
								>
								{#if contest.is_host}
									<span class="font-mono text-[8px] tracking-[0.1em] text-(--mer)/60 uppercase">host</span>
								{/if}
								<span class="font-mono text-[8px] text-(--silver)/25 uppercase">{contest.status}</span>
							</a>
						{/each}
					</div>
				{/if}
			</div>

			<div class="border border-(--border) bg-(--fbg) p-5">
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
					how events work
				</div>
				<ul class="mt-2 flex flex-col gap-2 font-mono text-[10px] leading-relaxed text-(--silver)/30">
					<li><span class="text-(--mer)">01</span> enroll before the start time</li>
					<li><span class="text-(--mer)">02</span> the clock starts on schedule</li>
					<li><span class="text-(--mer)">03</span> two humans minimum, or cancelled</li>
					<li><span class="text-(--mer)">04</span> results and replay stay on the event</li>
				</ul>
			</div>
		</aside>
	</main>
</div>
