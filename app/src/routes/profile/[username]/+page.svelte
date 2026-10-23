<script lang="ts">
	import { page } from '$app/state';
	import Avatar from '$lib/components/Avatar.svelte';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { follow, profile, unfollow, updateRunMeta } from '$lib/core/api/users';
	import { LANGUAGE_LABELS, LOBBY_DURATIONS, userElo } from '$lib/core/api/types';
	import type { LobbyDuration, MatchHistory, ProfileView } from '$lib/core/api/types';
	import { appState } from '$lib/core/state/state.svelte';

	const username = $derived(decodeURIComponent(page.params.username ?? ''));

	let data = $state<ProfileView | null>(null);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');

	let editing = $state<MatchHistory | null>(null);
	let editName = $state('');
	let editDescription = $state('');
	let editError = $state('');
	let saving = $state(false);
	let pinBusy = $state<number | null>(null);

	let isOwn = $derived(data?.user.id === appState.user?.id);
	let pinned = $derived(data?.pinned ?? []);
	let history = $derived(data?.history ?? []);

	async function load(showSpinner = true) {
		if (showSpinner) loading = true;
		error = '';
		try {
			data = await profile(username);
		} catch (err) {
			error = errorMessage(err);
			data = null;
		} finally {
			loading = false;
		}
	}

	async function toggleFollow() {
		if (!data || busy) return;
		busy = true;
		error = '';
		try {
			if (data.is_followed) await unfollow(username);
			else await follow(username);
			await load(false);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function togglePin(match: MatchHistory) {
		if (!isOwn || pinBusy !== null) return;
		pinBusy = match.game_id;
		error = '';
		try {
			await updateRunMeta(match.game_id, { pinned: !match.pinned });
			await load(false);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			pinBusy = null;
		}
	}

	function openEdit(match: MatchHistory) {
		editing = match;
		editName = match.run_name ?? '';
		editDescription = match.run_description ?? '';
		editError = '';
	}

	function closeEdit() {
		editing = null;
		editError = '';
	}

	async function saveEdit() {
		if (!editing || saving) return;
		saving = true;
		editError = '';
		try {
			await updateRunMeta(editing.game_id, {
				name: editName.trim(),
				description: editDescription.trim()
			});
			editing = null;
			await load(false);
		} catch (err) {
			editError = errorMessage(err);
		} finally {
			saving = false;
		}
	}

	function delta(value: number): string {
		return value > 0 ? `+${value}` : `${value}`;
	}

	function fmtDate(value: string): string {
		const date = new Date(value);
		if (Number.isNaN(date.getTime())) return '—';
		return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}

	function fmtWpm(value: number): string {
		return Number.isInteger(value) ? `${value}` : value.toFixed(1);
	}

	function clockPosition(duration: LobbyDuration): number {
		return data?.positions?.[String(duration) as '30' | '90' | '300'] ?? 0;
	}

	function runTitle(match: MatchHistory): string {
		if (match.run_name?.trim()) return match.run_name.trim();
		if (match.mode === 'practice') return 'saved practice';
		if (match.mode === 'ranked') return 'ranked race';
		if (match.mode === 'room') return match.room_code ? `room · ${match.room_code}` : 'room run';
		if (match.mode === 'contest') return 'contest run';
		return `${match.mode} run`;
	}

	function runMode(match: MatchHistory): string {
		if (match.mode === 'practice') return 'practice';
		if (match.mode === 'room') return match.room_code ? `room ${match.room_code}` : 'room';
		return match.mode;
	}

	function replayHref(match: MatchHistory): string {
		return `/replay/${match.game_id}/${match.user_id || data?.user.id}`;
	}

	onMount(() => load());
</script>

<svelte:head><title>{username} · profile · OSDTYP</title></svelte:head>

{#snippet runCard(match: MatchHistory)}
	<div class="group border border-(--border) bg-(--fbg) p-4 transition-colors hover:border-(--mer)/45">
		<div class="flex items-start gap-4">
			<a href={replayHref(match)} class="min-w-0 flex-1" title="open replay">
				<div class="flex flex-wrap items-center gap-2">
					<span class="truncate text-base text-(--silver)">{runTitle(match)}</span>
					{#if match.pinned}
						<span class="border border-(--mer)/35 px-1.5 py-0.5 font-mono text-[8px] tracking-[0.1em] text-(--mer)/70 uppercase"
							>pinned</span
						>
					{/if}
				</div>
				{#if match.run_description}
					<p class="mt-1 line-clamp-2 font-mono text-[11px] leading-relaxed text-(--silver)/40">
						{match.run_description}
					</p>
				{/if}
				<div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[10px] text-(--silver)/35">
					<span class="text-(--silver)/50">{runMode(match)}</span>
					<span>{fmtDate(match.played_at)}</span>
					<span class="text-(--mer)/80">{fmtWpm(match.wpm)} wpm</span>
					<span>{Math.round(match.accuracy * 100)}%</span>
					{#if match.mode !== 'practice'}
						<span>#{match.rank}/{match.players}</span>
					{:else}
						<span class="text-(--mer)/50">solo</span>
					{/if}
					<span>{LANGUAGE_LABELS[match.language] ?? match.language}</span>
					{#if match.mode === 'ranked'}
						<span class={match.rating_delta >= 0 ? 'text-(--mer)' : 'text-(--red)'}>
							{delta(match.rating_delta)} elo
						</span>
					{/if}
					{#if match.flagged}<span class="text-(--red)/70">flagged</span>{/if}
					<span class="ml-auto text-(--mer)/35 group-hover:text-(--mer)/70">replay =></span>
				</div>
			</a>
			{#if isOwn}
				<div class="flex shrink-0 flex-wrap justify-end gap-1">
					<button
						class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[9px] tracking-[0.08em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer) disabled:opacity-40"
						onclick={() => togglePin(match)}
						disabled={pinBusy !== null}
					>
						{pinBusy === match.game_id ? '…' : match.pinned ? 'unpin' : 'pin'}
					</button>
					<button
						class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[9px] tracking-[0.08em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer)"
						onclick={() => openEdit(match)}>edit</button
					>
				</div>
			{/if}
		</div>
	</div>
{/snippet}

<div class="flex min-h-screen flex-col bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex items-end gap-4">
			<a
				href={resolve('/leaderboard')}
				class="mb-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← leaderboard</a
			>
			<div class="title-text text-6xl leading-none text-(--silver) md:text-7xl">PROFILE</div>
		</div>
		<a
			href={resolve('/play')}
			class="mb-1 border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
			>hub →</a
		>
	</header>

	{#if error}
		<div class="mt-4 flex items-center justify-between gap-3 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)">
			<span>{error}</span>
			<button class="cursor-pointer text-(--red)/60 hover:text-(--red)" onclick={() => load()}>retry</button>
		</div>
	{/if}

	{#if loading}
		<div class="mt-6 border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35">
			loading profile…
		</div>
	{:else if data}
		<main class="mt-5 flex flex-1 flex-col gap-3">
			<section class="border border-(--border) bg-(--fbg)">
				<div class="flex flex-wrap items-center gap-5 border-b border-(--border) p-6">
					<Avatar
						src={data.user.avatar_url}
						name={data.user.username}
						class="h-20 w-20"
						text="text-2xl"
					/>
					<div class="min-w-0 flex-1">
						<div class="flex flex-wrap items-center gap-3">
							<h1 class="truncate text-3xl text-(--silver)">{data.user.username}</h1>
							{#if data.user.is_bot}
								<span class="border border-(--border) px-2 py-0.5 font-mono text-[9px] text-(--silver)/30 uppercase"
									>bot</span
								>
							{/if}
							{#if isOwn}
								<span class="font-mono text-[9px] text-(--mer) uppercase">you</span>
							{/if}
							{#if data.is_friend}
								<span class="border border-(--mer)/40 px-2 py-0.5 font-mono text-[9px] text-(--mer) uppercase"
									>friend</span
								>
							{:else if data.is_followed}
								<span class="border border-(--border) px-2 py-0.5 font-mono text-[9px] text-(--silver)/35 uppercase"
									>following</span
								>
							{/if}
							{#if data.follows_you}
								<span class="font-mono text-[9px] text-(--silver)/30 lowercase">follows you</span>
							{/if}
						</div>
						<div class="mt-2 flex flex-wrap items-center gap-4 font-mono text-[11px] text-(--silver)/45">
							<span class="text-(--mer)">{data.user.rating} best elo</span>
							<span>highest of 30 / 90 / 300</span>
							<span class="flex items-center gap-1">
								<span class={`h-1.5 w-1.5 ${data.online ? 'bg-(--mer)' : 'bg-(--silver)/25'}`}></span>
								{data.online ? 'online' : 'offline'}
							</span>
						</div>
					</div>
					{#if !isOwn}
						<button
							class="cursor-pointer border border-(--border) px-4 py-2 font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer) disabled:opacity-40"
							onclick={toggleFollow}
							disabled={busy}>{data.is_followed ? 'unfollow' : 'follow'}</button
						>
					{:else}
						<a
							href={resolve('/play')}
							class="border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
							>play =></a
						>
					{/if}
				</div>

				<div class="grid grid-cols-2 divide-(--border) border-b border-(--border) sm:grid-cols-3 md:grid-cols-5 md:divide-x">
					<div class="p-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
							best elo
						</div>
						<div class="mt-1 font-mono text-2xl text-(--mer)">{data.user.rating}</div>
					</div>
					<div class="border-t border-(--border) p-4 sm:border-t-0">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
							peak elo
						</div>
						<div class="mt-1 font-mono text-2xl text-(--silver)">{data.user.peak_rating}</div>
					</div>
					<div class="border-t border-(--border) p-4 sm:border-t-0">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
							top wpm
						</div>
						<div class="mt-1 font-mono text-2xl text-(--mer)">
							{data.top_wpm > 0 ? fmtWpm(data.top_wpm) : '—'}
						</div>
					</div>
					<div class="border-t border-(--border) p-4 sm:border-t-0">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
							games
						</div>
						<div class="mt-1 font-mono text-2xl text-(--silver)">{data.user.games_played}</div>
					</div>
					<div class="border-t border-(--border) p-4 sm:border-t-0">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
							wins
						</div>
						<div class="mt-1 font-mono text-2xl text-(--silver)">{data.user.wins}</div>
					</div>
				</div>

				<div class="grid grid-cols-1 divide-y divide-(--border) border-b border-(--border) sm:grid-cols-3 sm:divide-x sm:divide-y-0">
					{#each LOBBY_DURATIONS as duration (duration)}
						<div class="p-4">
							<div class="flex items-center justify-between gap-2">
								<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
									{duration}s ladder
								</div>
								<span class="font-mono text-[9px] text-(--silver)/25">
									{clockPosition(duration) > 0 ? `#${clockPosition(duration)}` : 'unranked'}
								</span>
							</div>
							<div class="mt-1 flex items-baseline gap-2">
								<span class="font-mono text-2xl text-(--mer)">
									{userElo(data.user, duration).rating}
								</span>
								<span class="font-mono text-[10px] text-(--silver)/30">
									peak {userElo(data.user, duration).peak_rating}
								</span>
							</div>
							<div class="mt-1 font-mono text-[10px] text-(--silver)/25">
								{userElo(data.user, duration).games_played} games ·
								{userElo(data.user, duration).wins} wins
							</div>
						</div>
					{/each}
				</div>

				{#if isOwn}
					<div class="px-6 py-3 font-mono text-[10px] leading-relaxed text-(--silver)/30">
						pin the runs you want on your profile. edit any run to give it a name and description.
					</div>
				{/if}
			</section>

			<div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_360px]">
				<section class="border border-(--border) bg-(--fbg)">
					<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							pinned runs
						</div>
						<span class="font-mono text-[10px] text-(--silver)/30">{pinned.length}</span>
					</div>
					{#if pinned.length === 0}
						<div class="p-6 font-mono text-[11px] leading-relaxed text-(--silver)/30">
							{#if isOwn}
								No pinned runs yet. Use the pin button on any run below.
							{:else}
								This player has not pinned any runs yet.
							{/if}
						</div>
					{:else}
						<div class="flex flex-col gap-2 p-3">
							{#each pinned as match (match.game_id)}
								{@render runCard(match)}
							{/each}
						</div>
					{/if}
				</section>

				<aside class="flex flex-col gap-3">
					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) px-5 py-3 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							best run
						</div>
						{#if data.best}
							<a href={replayHref(data.best)} class="block p-5 transition-colors hover:bg-(--mer)/5">
								<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
									top wpm
								</div>
								<div class="mt-1 flex items-end gap-2">
									<span class="font-mono text-4xl text-(--mer)">{fmtWpm(data.top_wpm)}</span>
									<span class="mb-1 font-mono text-[10px] text-(--silver)/35">wpm</span>
								</div>
								<div class="mt-4 truncate text-base text-(--silver)">{runTitle(data.best)}</div>
								{#if data.best.run_description}
									<p class="mt-1 line-clamp-3 font-mono text-[11px] leading-relaxed text-(--silver)/40">
										{data.best.run_description}
									</p>
								{/if}
								<div class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[10px] text-(--silver)/35">
									<span>{runMode(data.best)}</span>
									<span>{fmtDate(data.best.played_at)}</span>
									<span>{Math.round(data.best.accuracy * 100)}%</span>
									<span class="ml-auto text-(--mer)/50">replay =></span>
								</div>
							</a>
						{:else}
							<div class="p-5 font-mono text-[11px] text-(--silver)/30">No finished runs yet.</div>
						{/if}
					</section>

					<section class="border border-(--border) bg-(--fbg) p-5">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							record
						</div>
						<div class="mt-3 grid grid-cols-2 gap-3 font-mono text-xs">
							<div class="border border-(--border) p-3">
								<div class="text-[9px] text-(--silver)/30 uppercase">wins</div>
								<div class="mt-1 text-lg text-(--mer)">{data.user.wins}</div>
							</div>
							<div class="border border-(--border) p-3">
								<div class="text-[9px] text-(--silver)/30 uppercase">losses</div>
								<div class="mt-1 text-lg text-(--silver)">
									{Math.max(0, data.user.games_played - data.user.wins)}
								</div>
							</div>
						</div>
						{#if data.position > 0}
							<div class="mt-3 font-mono text-[10px] text-(--silver)/30">
								best ladder rank #{data.position}
							</div>
						{:else if data.user.games_played === 0}
							<div class="mt-3 font-mono text-[10px] text-(--silver)/30">
								Starts at 400 Elo on every clock. Play a ranked race to enter a ladder.
							</div>
						{/if}
					</section>
				</aside>
			</div>

			<section class="border border-(--border) bg-(--fbg)">
				<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						recent runs
					</div>
					<span class="font-mono text-[10px] text-(--silver)/30">{history.length}</span>
				</div>
				{#if history.length === 0}
					<div class="p-8 text-center font-mono text-[11px] text-(--silver)/30">
						No finished runs yet.
					</div>
				{:else}
					<div class="flex flex-col gap-2 p-3">
						{#each history as match (match.game_id)}
							{@render runCard(match)}
						{/each}
					</div>
				{/if}
			</section>
		</main>
	{:else}
		<div class="mt-6 border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35">
			Profile not found.
		</div>
	{/if}
</div>

{#if editing}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
		<form
			class="w-full max-w-lg border border-(--border) bg-(--bg) p-5 shadow-2xl"
			onsubmit={(event) => {
				event.preventDefault();
				saveEdit();
			}}
		>
			<div class="flex items-start justify-between gap-4">
				<div>
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">edit run</div>
					<div class="mt-1 text-xl text-(--silver)">{runTitle(editing)}</div>
				</div>
				<button
					type="button"
					class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[10px] text-(--silver)/40 hover:text-(--silver)"
					onclick={closeEdit}>close</button
				>
			</div>

			{#if editError}
				<div class="mt-4 border border-(--red)/40 px-3 py-2 font-mono text-[11px] text-(--red)">
					{editError}
				</div>
			{/if}

			<label class="mt-5 block">
				<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase">name</span>
				<input
					class="mt-2 w-full border border-(--border) bg-(--fbg) px-3 py-2 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
					maxlength="80"
					bind:value={editName}
					placeholder="give this run a name"
				/>
				<span class="mt-1 block text-right font-mono text-[9px] text-(--silver)/25"
					>{editName.length}/80</span
				>
			</label>

			<label class="mt-2 block">
				<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
					>description</span
				>
				<textarea
					class="mt-2 min-h-28 w-full resize-y border border-(--border) bg-(--fbg) px-3 py-2 font-mono text-sm leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
					maxlength="500"
					bind:value={editDescription}
					placeholder="what made this run interesting?"
				></textarea>
				<span class="mt-1 block text-right font-mono text-[9px] text-(--silver)/25"
					>{editDescription.length}/500</span
				>
			</label>

			<div class="mt-5 flex justify-end gap-2">
				<button
					type="button"
					class="cursor-pointer border border-(--border) px-4 py-2 font-mono text-xs text-(--silver)/50 hover:border-(--silver)/40 hover:text-(--silver)"
					onclick={closeEdit}>cancel</button
				>
				<button
					type="submit"
					class="cursor-pointer border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10 disabled:opacity-40"
					disabled={saving}>{saving ? 'saving…' : 'save run'}</button
				>
			</div>
		</form>
	</div>
{/if}
