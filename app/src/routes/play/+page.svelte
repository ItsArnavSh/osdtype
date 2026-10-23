<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import ModeIcon from '$lib/components/ModeIcon.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import { appState, resetSession } from '$lib/core/state/state.svelte';
	import { logout } from '$lib/core/api/auth';
	import { friends } from '$lib/core/api/users';
	import type { FriendView } from '$lib/core/api/types';
	import { onMount } from 'svelte';

	let user = $derived(appState.user);
	let guest = $derived(!user);
	let inviteCount = $derived(appState.invites.length);
	let friendList = $state<FriendView[]>([]);
	let friendError = $state('');
	let onlineFriends = $derived(friendList.filter((friend) => friend.online).length);
	let visibleFriends = $derived(
		[...friendList]
			.sort(
				(a, b) =>
					Number(b.online) - Number(a.online) ||
					Number(b.is_friend) - Number(a.is_friend) ||
					a.user.username.localeCompare(b.user.username)
			)
			.slice(0, 5)
	);

	onMount(async () => {
		if (!appState.user) return;
		try {
			friendList = await friends();
		} catch (err) {
			friendError = err instanceof Error ? err.message : 'could not load friends';
		}
	});

	function loginHref(next: string): string {
		return `/login?next=${encodeURIComponent(next)}`;
	}

	async function signOut() {
		await logout().catch(() => {});
		resetSession();
		void goto('/login');
	}

	const modes = [
		{
			name: 'Ranked',
			icon: 'ranked',
			desc: 'Timed head-to-head races with Elo on the line.',
			href: '/play/ranked',
			cta: 'find match',
			meta: '30 / 90 / 300s',
			locked: true
		},
		{
			name: 'Rooms',
			icon: 'rooms',
			desc: 'Private or public lobbies. Invite the group.',
			href: '/play/rooms',
			cta: 'open rooms',
			meta: '2–16 players',
			locked: true
		},
		{
			name: 'Practice',
			icon: 'practice',
			desc: 'Solo runs with adaptive rivals chasing your pace.',
			href: '/play/me',
			cta: 'start',
			meta: 'solo · adaptive bots',
			locked: false
		},
		{
			name: 'Zen',
			icon: 'zen',
			desc: 'No timer, no score, no pressure. Just the keys.',
			href: '/play/zen',
			cta: 'drift',
			meta: 'no timer · no stats',
			locked: false
		}
	] as const;
</script>

<svelte:head><title>play · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex items-center gap-3">
			<a href={resolve('/play')} class="title-text text-5xl leading-none text-(--silver) md:text-7xl">
				OSDTYP
			</a>
			<button
				class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[10px] text-(--silver)/55 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
				onclick={cycleTheme}
				title="cycle theme"
			>{themeState.current}</button
			>
		</div>

		<nav class="flex flex-wrap items-center gap-2">
			<a
				href={resolve('/contests')}
				class="border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer)"
				>contests</a
			>
			<a
				href={resolve('/docs')}
				class="border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer)"
				>docs</a
			>
			<a
				href={resolve('/about')}
				class="border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer)"
				>about</a
			>
			{#if user}
				<a
					href={resolve('/leaderboard')}
					class="border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:border-(--silver)/40 hover:text-(--silver)"
					>leaderboard</a
				>
				<a
					href={resolve('/friends')}
					class="border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:border-(--silver)/40 hover:text-(--silver)"
					>friends</a
				>
				<a
					href={resolve(`/profile/${user.username}`)}
					class="flex min-w-0 max-w-[12rem] items-center gap-2 border border-(--border) px-2 py-1 hover:border-(--silver)/40 sm:max-w-none"
				>
					<Avatar
						src={user.avatar_url}
						name={user.username}
						class="h-6 w-6 shrink-0"
						text="text-[9px]"
					/>
					<span class="truncate font-mono text-[11px] text-(--silver)">{user.username}</span>
				</a>
				<button
					class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/35 uppercase hover:border-(--silver)/40 hover:text-(--silver)"
					onclick={signOut}>log out</button
				>
			{:else}
				<a
					href={loginHref('/play')}
					class="border border-(--mer) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase transition-colors hover:bg-(--mer)/10"
					>sign up / log in =></a
				>
			{/if}
		</nav>
	</header>

	<div class="mt-8 flex flex-wrap items-end justify-between gap-4 border-b border-(--border) pb-4">
		<div>
			<div class="font-mono text-[10px] tracking-[0.18em] text-(--red) uppercase">hub</div>
			<div class="title-text mt-1 text-5xl leading-none text-(--silver) md:text-6xl">
				PICK A MODE
			</div>
		</div>
		<div class="flex items-center gap-2 font-mono text-[10px] text-(--silver)/35">
			<span class={`h-1.5 w-1.5 ${appState.state ? 'bg-(--mer)' : 'bg-(--silver)/25'}`}></span>
			{#if guest}
				guest session
			{:else if inviteCount > 0}
				{inviteCount} pending invite{inviteCount === 1 ? '' : 's'}
			{:else}
				session live
			{/if}
		</div>
	</div>

	<section class="mt-4">
		{#if user}
			<div class="grid grid-cols-2 divide-(--border) border border-(--border) bg-(--fbg) sm:grid-cols-4 sm:divide-x">
				<div class="p-4">
					<div class="font-mono text-[9px] tracking-[0.16em] text-(--silver)/35 uppercase">
						best elo
					</div>
					<div class="mt-1 flex items-baseline gap-2">
						<span class="font-mono text-2xl text-(--mer)">{user.rating}</span>
						<span class="font-mono text-[9px] text-(--silver)/25">of 3 clocks</span>
					</div>
				</div>
				<div class="border-t border-(--border) p-4 sm:border-t-0">
					<div class="font-mono text-[9px] tracking-[0.16em] text-(--silver)/35 uppercase">
						peak elo
					</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">{user.peak_rating}</div>
				</div>
				<div class="border-t border-(--border) p-4 sm:border-t-0">
					<div class="font-mono text-[9px] tracking-[0.16em] text-(--silver)/35 uppercase">
						record
					</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">
						{user.wins}<span class="text-sm text-(--silver)/30">W</span>
						{Math.max(0, user.games_played - user.wins)}<span class="text-sm text-(--silver)/30">L</span>
					</div>
				</div>
				<div class="border-t border-(--border) p-4 sm:border-t-0">
					<div class="font-mono text-[9px] tracking-[0.16em] text-(--silver)/35 uppercase">
						games
					</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">{user.games_played}</div>
				</div>
			</div>
		{:else}
			<div class="flex flex-wrap items-center justify-between gap-4 border border-(--border) bg-(--fbg) px-5 py-4">
				<div class="flex items-start gap-3">
					<span class="mt-1 h-2 w-2 shrink-0 bg-(--mer)"></span>
					<div>
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
							guest mode
						</div>
						<p class="mt-1 font-mono text-xs leading-relaxed text-(--silver)/50">
							Practice and Zen are open. Sign in for Elo ranked races, rooms, friends and saved runs.
						</p>
					</div>
				</div>
				<a
					href={loginHref('/play')}
					class="border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
					>create account =></a
				>
			</div>
		{/if}
	</section>

	<main class="mt-4 grid flex-1 content-start gap-3 lg:grid-cols-[minmax(0,1fr)_340px]">
		<section class="min-w-0">
			<div class="mb-2 font-mono text-[10px] tracking-[0.16em] text-(--silver)/30 uppercase">
				play
			</div>
			<div class="grid gap-3 sm:grid-cols-2">
				{#each modes as mode (mode.name)}
					<div class="relative">
						<button
							class={`group flex h-full min-h-52 w-full cursor-pointer flex-col justify-between border border-(--border) bg-(--fbg) p-5 text-left transition-colors hover:border-(--mer)/60 ${
								mode.locked && guest ? 'pointer-events-none opacity-30 blur-[1.5px]' : ''
							}`}
							onclick={() => goto(resolve(mode.href))}
							disabled={mode.locked && guest}
							aria-hidden={mode.locked && guest}
							tabindex={mode.locked && guest ? -1 : 0}
						>
							<div class="flex w-full items-start justify-between gap-3">
								<div class="text-3xl text-(--silver)">{mode.name}</div>
								<ModeIcon name={mode.icon} class="h-11 w-11 shrink-0 text-(--mer)/75" />
							</div>
							<p class="mt-3 max-w-[28ch] font-mono text-[11px] leading-relaxed text-(--silver)/50">
								{mode.desc}
							</p>
							<div class="mt-6 flex items-end justify-between gap-3">
								<span class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
									{#if mode.name === 'Ranked' && user}
										best {user.rating} elo
									{:else}
										{mode.meta}
									{/if}
								</span>
								<span
									class="border border-(--mer) px-3 py-1.5 font-mono text-[11px] text-(--mer) transition-colors group-hover:bg-(--mer)/10"
									>{mode.cta} =></span
								>
							</div>
						</button>
						{#if mode.locked && guest}
							<a
								href={loginHref(mode.href)}
								class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-(--bg)/30 p-4 text-center transition-colors hover:bg-(--bg)/10"
							>
								<span class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/45 uppercase">
									login to continue
								</span>
								<span class="border border-(--mer) bg-(--bg)/80 px-4 py-1.5 font-mono text-xs text-(--mer)">
									sign in for {mode.name.toLowerCase()} =>
								</span>
							</a>
						{/if}
					</div>
				{/each}
			</div>

			<a
				href={resolve('/contests')}
				class="group mt-3 flex flex-wrap items-center justify-between gap-3 border border-(--border) bg-(--fbg) px-5 py-4 transition-colors hover:border-(--mer)"
			>
				<div class="flex items-center gap-3">
					<span class="font-mono text-[9px] tracking-[0.16em] text-(--red) uppercase">events</span>
					<span class="text-lg text-(--silver)">Contests</span>
					<span class="hidden font-mono text-[10px] text-(--silver)/35 sm:inline"
						>standalone scheduled races · public or invite-only</span
					>
				</div>
				<span
					class="border border-(--mer) px-3 py-1.5 font-mono text-[10px] text-(--mer) transition-colors group-hover:bg-(--mer)/10"
					>contest board =></span
				>
			</a>
		</section>

		<aside class="flex flex-col gap-3 lg:mt-[23px]">
			{#if user}
				<div class="border border-(--border) bg-(--fbg)">
					<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							friends
						</div>
						<a href={resolve('/friends')} class="font-mono text-[10px] text-(--silver)/40 hover:text-(--silver)">
							manage
						</a>
					</div>
					{#if friendError}
						<p class="p-5 font-mono text-[11px] text-(--red)/70">{friendError}</p>
					{:else if visibleFriends.length === 0}
						<p class="p-5 font-mono text-[11px] leading-relaxed text-(--silver)/30">
							No friends yet. Follow players from a profile or the leaderboard.
						</p>
					{:else}
						<div class="divide-y divide-(--border)">
							{#each visibleFriends as friend (friend.user.id)}
								<a
									href={resolve(`/profile/${friend.user.username}`)}
									class="flex items-center gap-2.5 px-5 py-2.5 font-mono text-xs transition-colors hover:bg-(--mer)/5"
								>
									<span
										class={`h-1.5 w-1.5 shrink-0 ${friend.online ? 'bg-(--mer)' : 'bg-(--silver)/20'}`}
									></span>
									<span class="truncate text-(--silver)/70">{friend.user.username}</span>
									{#if friend.is_friend}
										<span class="text-[9px] tracking-[0.1em] text-(--mer)/50 uppercase">friend</span>
									{:else}
										<span class="text-[9px] tracking-[0.1em] text-(--silver)/25 uppercase">following</span>
									{/if}
									<span class="ml-auto text-[10px] text-(--silver)/30">{friend.user.rating}</span>
								</a>
							{/each}
						</div>
						<div class="border-t border-(--border) px-5 py-2 font-mono text-[10px] text-(--silver)/25">
							{onlineFriends} online now
						</div>
					{/if}
				</div>

				<div class="border border-(--border) bg-(--fbg)">
					<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							invites
						</div>
						<a href={resolve('/invites')} class="font-mono text-[10px] text-(--silver)/40 hover:text-(--silver)">
							view all
						</a>
					</div>
					{#if appState.invites.length === 0}
						<p class="p-5 font-mono text-[11px] text-(--silver)/30">No pending room invites.</p>
					{:else}
						<div class="divide-y divide-(--border)">
							{#each appState.invites.slice(0, 3) as invite (invite.id)}
								<div class="flex items-center justify-between gap-3 px-5 py-2.5 font-mono text-xs">
									<span class="truncate text-(--silver)/70">{invite.room_name}</span>
									<span class="shrink-0 text-(--silver)/30">{invite.from}</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<div class="flex flex-1 flex-col border border-(--border) bg-(--fbg) p-5">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						competitive
					</div>
					<p class="mt-2 font-mono text-[11px] leading-relaxed text-(--silver)/40">
						Three clocks, three independent Elos. Everyone starts at 400 on each ladder, and
						the number above is your highest.
					</p>
					<a
						href={resolve('/leaderboard')}
						class="mt-auto flex justify-center border border-(--border) px-3 py-2 pt-2 font-mono text-[10px] tracking-[0.12em] text-(--silver)/50 uppercase hover:border-(--mer) hover:text-(--mer)"
						>view leaderboard =></a
					>
				</div>
			{:else}
				<div
					class="border border-(--border) bg-(--fbg) p-5"
					style="border-top-color: var(--mer);"
				>
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
						with an account
					</div>
					<div class="mt-1 text-2xl text-(--silver)">Unlock the rest.</div>
					<ul class="mt-3 flex flex-col gap-2 font-mono text-[11px] leading-relaxed text-(--silver)/45">
						<li>· Elo ranked and the leaderboard</li>
						<li>· rooms, invites and contests</li>
						<li>· saved runs, pins and replay links</li>
					</ul>
					<a
						href={loginHref('/play')}
						class="mt-4 flex justify-center border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
						>sign up free =></a
					>
				</div>
			{/if}
		</aside>
	</main>

	<footer class="mt-5 flex flex-wrap items-center justify-between gap-2 border-t border-(--border) pt-3 font-mono text-[10px] text-(--silver)/25">
		<span>server-authoritative · explicit leave only</span>
		<span>resrap-rs 1.0.1 · 10 languages</span>
	</footer>
</div>
