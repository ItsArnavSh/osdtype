<script lang="ts">
	import { page } from '$app/state';
	import Avatar from '$lib/components/Avatar.svelte';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { follow, profile, unfollow } from '$lib/core/api/users';
	import { LANGUAGE_LABELS } from '$lib/core/api/types';
	import type { ProfileView } from '$lib/core/api/types';
	import { appState } from '$lib/core/state/state.svelte';

	const username = $derived(decodeURIComponent(page.params.username ?? ''));

	let data = $state<ProfileView | null>(null);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');

	async function load() {
		loading = true;
		try {
			data = await profile(username);
		} catch (err) {
			error = errorMessage(err);
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
			data = await profile(username);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
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

	onMount(load);
</script>

<svelte:head><title>{username} · profile · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-6 md:p-8">
	<header class="flex flex-wrap items-end gap-4">
		<a
			href={resolve('/leaderboard')}
			class="mb-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
			>← ladder</a
		>
		<div class="title-text text-7xl leading-none text-(--silver) md:text-8xl">PROFILE</div>
	</header>

	{#if error}
		<div class="mt-4 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)">{error}</div>
	{/if}

	{#if loading}
		<div class="mt-6 border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35">
			loading profile…
		</div>
	{:else if data}
		<main class="mt-5 grid flex-1 gap-3 lg:grid-cols-[1fr_360px]">
			<section class="border border-(--border) bg-(--fbg)">
				<div class="flex flex-wrap items-center gap-5 border-b border-(--border) p-6">
					<Avatar src={data.user.avatar_url} name={data.user.username} class="h-20 w-20" text="text-2xl" />
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-3">
							<h1 class="truncate text-3xl text-(--silver)">{data.user.username}</h1>
							{#if data.user.is_bot}
								<span class="border border-(--border) px-2 py-0.5 font-mono text-[9px] text-(--silver)/30 uppercase">bot</span>
							{/if}
							{#if data.user.id === appState.user?.id}
								<span class="font-mono text-[9px] text-(--mer) uppercase">you</span>
							{/if}
						</div>
						<div class="mt-2 flex flex-wrap items-center gap-4 font-mono text-[11px] text-(--silver)/45">
							<span class="text-(--mer)">{data.user.rank.label}</span>
							<span>#{data.position} global</span>
							<span>{data.user.rating} rating</span>
							<span class="flex items-center gap-1"
								><span class={`h-1.5 w-1.5 ${data.online ? 'bg-(--mer)' : 'bg-(--silver)/25'}`}></span
								>{data.online ? 'online' : 'offline'}</span
							>
						</div>
					</div>
					{#if data.user.id !== appState.user?.id}
						<button
							class="cursor-pointer border border-(--border) px-4 py-2 font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer) disabled:opacity-40"
							onclick={toggleFollow}
							disabled={busy}>{data.is_followed ? 'unfollow' : 'follow'}</button
						>
					{/if}
				</div>

				<div class="grid grid-cols-2 divide-x divide-(--border) border-b border-(--border) md:grid-cols-4">
					<div class="p-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">games</div>
						<div class="mt-1 font-mono text-xl text-(--silver)">{data.user.games_played}</div>
					</div>
					<div class="p-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">wins</div>
						<div class="mt-1 font-mono text-xl text-(--mer)">{data.user.wins}</div>
					</div>
					<div class="p-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">peak</div>
						<div class="mt-1 font-mono text-xl text-(--silver)">{data.user.peak_rating}</div>
					</div>
					<div class="p-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">next tier</div>
						<div class="mt-1 font-mono text-xl text-(--silver)">{data.user.rank.next}</div>
					</div>
				</div>

				<div class="px-6 py-4">
					<div class="flex items-center justify-between font-mono text-[10px] text-(--silver)/35">
						<span>{data.user.rank.label}</span>
						<span>{Math.round(data.user.rank.progress * 100)}%</span>
					</div>
					<div class="mt-2 h-px w-full bg-(--border)">
						<div class="h-px bg-(--mer)" style={`width:${data.user.rank.progress * 100}%`}></div>
					</div>
				</div>
			</section>

			<aside class="flex flex-col gap-3">
				<div class="border border-(--border) bg-(--fbg)">
					<div class="border-b border-(--border) px-5 py-3 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						recent matches
					</div>
					{#if data.history.length === 0}
						<div class="p-6 font-mono text-[11px] text-(--silver)/30">No finished games yet.</div>
					{:else}
						<div class="divide-y divide-(--border)">
							{#each data.history as match (match.game_id)}
								<div class="px-5 py-3">
									<div class="flex items-center justify-between gap-2">
										<span class="font-mono text-[11px] text-(--silver)/70">
											{match.mode}{match.room_code ? ` · ${match.room_code}` : ''}
										</span>
										<span class="font-mono text-[10px] text-(--silver)/25">{fmtDate(match.played_at)}</span>
									</div>
									<div class="mt-1 flex flex-wrap gap-3 font-mono text-[10px] text-(--silver)/35">
										<span>#{match.rank}/{match.players}</span>
										<span>{Math.round(match.wpm)} wpm</span>
										<span>{Math.round(match.accuracy * 100)}%</span>
										<span>{LANGUAGE_LABELS[match.language] ?? match.language}</span>
										{#if match.mode === 'ranked'}
											<span class={match.rating_delta >= 0 ? 'text-(--mer)' : 'text-(--red)'}
												>{delta(match.rating_delta)}</span
											>
										{/if}
										{#if match.flagged}<span class="text-(--red)/70">flagged</span>{/if}
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				{#if data.user.id === appState.user?.id}
					<a
						href={resolve('/play')}
						class="border border-(--mer) px-4 py-2.5 text-center font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
						>back to hub</a
					>
				{/if}
			</aside>
		</main>
	{/if}
</div>
