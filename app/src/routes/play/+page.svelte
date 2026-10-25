<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import ModeIcon from '$lib/components/ModeIcon.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import InstallApp from '$lib/components/InstallApp.svelte';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import { appState, resetSession } from '$lib/core/state/state.svelte';
	import { logout } from '$lib/core/api/auth';
	import { friends } from '$lib/core/api/users';
	import type { FriendView } from '$lib/core/api/types';
	import { onMount } from 'svelte';

	let user = $derived(appState.user);
	let guest = $derived(!user);
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
			.slice(0, 4)
	);

	onMount(async () => {
		if (!appState.user) return;
		try {
			friendList = await friends();
		} catch (err) {
			friendError = err instanceof Error ? err.message : 'could not load friends';
		}
	});

	async function signOut() {
		await logout().catch(() => {});
		resetSession();
		void goto(resolve('/login'));
	}

	// Practice is the primary action, so it gets its own hero. The rest of
	// the modes share one consistent card treatment.
	const modes = [
		{
			name: 'Ranked',
			icon: 'ranked',
			desc: 'Head-to-head code races with Elo on the line.',
			href: '/play/ranked',
			cta: 'find a match',
			meta: '30 / 90 / 300s',
			account: true
		},
		{
			name: 'Rooms',
			icon: 'rooms',
			desc: 'Private or public lobbies. Invite the group.',
			href: '/play/rooms',
			cta: 'open rooms',
			meta: '2–16 players',
			account: true
		},
		{
			name: 'Zen',
			icon: 'zen',
			desc: 'No timer, no score, no pressure. Just the keys.',
			href: '/play/zen',
			cta: 'drift',
			meta: 'no timer · no stats',
			account: false
		}
	] as const;

	function modeCta(mode: (typeof modes)[number]): string {
		return guest && mode.account ? 'sign in' : mode.cta;
	}

	function modeMeta(mode: (typeof modes)[number]): string {
		if (mode.name === 'Ranked' && user) return `best ${user.rating} elo`;
		return mode.meta;
	}
</script>

<svelte:head><title>play · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg)">
	<header class="border-b border-(--border)">
		<div
			class="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-3 px-5 py-4 md:px-8"
		>
			<div class="flex items-center gap-3">
				<a
					href={resolve('/play')}
					class="title-text text-3xl leading-none text-(--silver) md:text-4xl"
				>
					OSDTYP
				</a>
				<span class="ui-eyebrow hidden sm:inline">code typing arena</span>
			</div>

			<nav class="flex flex-wrap items-center gap-2" aria-label="Main navigation">
				<a href={resolve('/contests')} class="ui-btn ui-btn-sm hidden md:inline-flex">contests</a>
				<a href={resolve('/docs')} class="ui-btn ui-btn-sm">docs</a>
				<a href={resolve('/bugs')} class="ui-btn ui-btn-sm">report bug</a>
				<button class="ui-btn ui-btn-sm" onclick={cycleTheme} title="Cycle theme">
					theme: {themeState.current}
				</button>
				<InstallApp />
				{#if user?.is_admin}
					<a href={resolve('/admin')} class="ui-btn ui-btn-sm border-(--red)/40 text-(--red)"
						>admin</a
					>
				{/if}
				{#if user}
					<a href={resolve('/leaderboard')} class="ui-btn ui-btn-sm">leaderboard</a>
					<a href={resolve('/friends')} class="ui-btn ui-btn-sm">friends</a>
					<a href={resolve('/settings')} class="ui-btn ui-btn-sm">settings</a>
					<a
						href={resolve(`/profile/${user.username}`)}
						class="ui-btn ui-btn-sm flex max-w-48 items-center gap-2 normal-case"
					>
						<Avatar
							src={user.avatar_url}
							name={user.username}
							class="h-5 w-5 shrink-0"
							text="text-[9px]"
						/>
						<span class="truncate text-[11px] tracking-normal">{user.username}</span>
					</a>
					<button class="ui-btn ui-btn-sm" onclick={signOut}>log out</button>
				{:else}
					<a
						href={resolve(`/login?next=${encodeURIComponent('/play')}` as '/login')}
						class="ui-btn ui-btn-sm ui-btn-primary">sign in</a
					>
				{/if}
			</nav>
		</div>
	</header>

	<main class="mx-auto w-full max-w-6xl flex-1 px-5 py-6 md:px-8">
		{#if user}
			<section
				class="mb-4 grid grid-cols-2 divide-(--border) border border-(--border) bg-(--fbg) sm:grid-cols-4 sm:divide-x"
			>
				<div class="p-4">
					<div class="ui-label">best elo</div>
					<div class="mt-1 flex items-baseline gap-2">
						<span class="font-mono text-2xl text-(--mer)">{user.rating}</span>
						<span class="font-mono text-[9px] text-(--silver)/40">of 3 clocks</span>
					</div>
				</div>
				<div class="border-t border-(--border) p-4 sm:border-t-0">
					<div class="ui-label">peak elo</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">{user.peak_rating}</div>
				</div>
				<div class="border-t border-(--border) p-4 sm:border-t-0">
					<div class="ui-label">record</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">
						{user.wins}<span class="text-sm text-(--silver)/40">W</span>
						{Math.max(0, user.games_played - user.wins)}<span class="text-sm text-(--silver)/40"
							>L</span
						>
					</div>
				</div>
				<div class="border-t border-(--border) p-4 sm:border-t-0">
					<div class="ui-label">games</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">{user.games_played}</div>
				</div>
			</section>
		{/if}

		<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
			<div class="flex min-w-0 flex-col gap-4">
				<!-- Phones are a companion surface: no typing actions. -->
				<section
					class="ui-card p-6 md:hidden"
					style="border-top-color: var(--mer);"
					aria-label="Mobile companion"
				>
					<div class="ui-eyebrow">phone companion</div>
					<h1 class="title-text mt-2 text-5xl leading-[0.9] text-(--silver)">
						WATCH.<br />DON'T TYPE.
					</h1>
					<p class="mt-3 font-mono text-[11px] leading-relaxed text-(--silver)/50">
						The typing arena is desktop-only. Your phone keeps the ladder, contests, friends,
						profiles and shared replays within reach.
					</p>
					<div class="mt-5 grid grid-cols-2 gap-2">
						<a href={resolve('/leaderboard')} class="ui-btn ui-btn-sm justify-center">leaderboard</a>
						<a href={resolve('/contests')} class="ui-btn ui-btn-sm justify-center">contests</a>
						{#if user}
							<a href={resolve('/friends')} class="ui-btn ui-btn-sm justify-center">friends</a>
							<a href={resolve('/settings')} class="ui-btn ui-btn-sm justify-center">settings</a>
						{:else}
							<a
								href={resolve(`/login?next=${encodeURIComponent('/play')}` as '/login')}
								class="ui-btn ui-btn-sm justify-center">sign in</a
							>
							<a href={resolve('/docs')} class="ui-btn ui-btn-sm justify-center">docs</a>
						{/if}
					</div>
				</section>

				<!-- Primary action: the one thing a new visitor should do. -->
				<section class="ui-card p-6 md:p-8 max-md:hidden" style="border-top-color: var(--mer);">
					<div class="ui-eyebrow">practice · free</div>
					<div class="mt-2 flex flex-wrap items-start justify-between gap-4">
						<div class="min-w-0">
							<h1 class="title-text text-5xl leading-[0.9] text-(--silver) md:text-7xl">
								START TYPING.
							</h1>
							<p class="mt-3 max-w-xl text-sm leading-relaxed text-(--silver)/65">
								Jump into a real code snippet right away. The timer starts with your first keystroke
								— no account, no setup.
							</p>
						</div>
						<ModeIcon name="practice" class="hidden h-16 w-16 shrink-0 text-(--mer)/55 sm:block" />
					</div>

					<div class="mt-6 flex flex-wrap items-center gap-2">
						<a href={resolve('/play/me')} class="ui-btn ui-btn-primary px-5 py-3 text-xs">
							start practice →
						</a>
						<a href={resolve('/play/zen')} class="ui-btn px-4 py-3 text-xs">zen · no timer</a>
						<span
							class="ml-auto font-mono text-[9px] tracking-[0.1em] text-(--silver)/40 uppercase"
						>
							solo · adaptive bots{#if guest}
								· guest{/if}
						</span>
					</div>
				</section>

				<section aria-label="Other modes" class="grid gap-3 max-md:hidden sm:grid-cols-3">
					{#each modes as mode (mode.name)}
						<a
							href={resolve(
								(guest && mode.account
									? `/login?next=${encodeURIComponent(mode.href)}`
									: mode.href) as '/play'
							)}
							class="ui-card group flex min-h-44 flex-col p-5 transition-colors hover:border-(--mer)/60"
							aria-label={`${mode.name}${guest && mode.account ? ' — account required' : ''}`}
						>
							<div class="flex items-start justify-between gap-3">
								<h2 class="text-2xl leading-none text-(--silver)">{mode.name}</h2>
								<ModeIcon name={mode.icon} class="h-10 w-10 shrink-0 text-(--mer)/70" />
							</div>
							<p class="mt-3 flex-1 text-[11px] leading-relaxed text-(--silver)/55">
								{mode.desc}
							</p>
							<div class="mt-5 flex items-end justify-between gap-2">
								<span class="font-mono text-[9px] tracking-[0.12em] text-(--silver)/40 uppercase">
									{modeMeta(mode)}
								</span>
								<span class="font-mono text-[10px] tracking-[0.08em] text-(--mer) uppercase">
									{modeCta(mode)} →
								</span>
							</div>
						</a>
					{/each}
				</section>

				<a
					href={resolve('/contests')}
					class="ui-card group flex flex-wrap items-center justify-between gap-3 px-5 py-4 transition-colors hover:border-(--mer)/60"
				>
					<div class="flex flex-wrap items-center gap-3">
						<span class="ui-eyebrow">events</span>
						<span class="text-lg text-(--silver)">Contests</span>
						<span class="hidden font-mono text-[10px] text-(--silver)/45 sm:inline">
							scheduled races · public or invite-only
						</span>
					</div>
					<span class="ui-btn ui-btn-sm">contest board →</span>
				</a>
			</div>

			<aside class="flex flex-col gap-3" aria-label="Account and social">
				{#if user}
					<div class="ui-card">
						<div class="flex items-center justify-between border-b border-(--border) px-4 py-2.5">
							<div class="ui-label">friends</div>
							<a href={resolve('/friends')} class="ui-btn-quiet font-mono text-[10px]">manage</a>
						</div>
						{#if friendError}
							<p class="p-4 font-mono text-[11px] text-(--red)/80">{friendError}</p>
						{:else if visibleFriends.length === 0}
							<p class="p-4 font-mono text-[11px] leading-relaxed text-(--silver)/40">
								No friends yet. Follow players from a profile or the leaderboard.
							</p>
						{:else}
							<div class="divide-y divide-(--border)">
								{#each visibleFriends as friend (friend.user.id)}
									<a
										href={resolve(`/profile/${friend.user.username}`)}
										class="flex items-center gap-2.5 px-4 py-2.5 font-mono text-xs transition-colors hover:bg-(--mer)/5"
									>
										<span
											class={`h-1.5 w-1.5 shrink-0 ${friend.online ? 'bg-(--mer)' : 'bg-(--silver)/25'}`}
										></span>
										<span class="truncate text-(--silver)/75">{friend.user.username}</span>
										<span class="ml-auto text-[10px] text-(--silver)/40">{friend.user.rating}</span>
									</a>
								{/each}
							</div>
							<div
								class="border-t border-(--border) px-4 py-2 font-mono text-[10px] text-(--silver)/35"
							>
								{onlineFriends} online now
							</div>
						{/if}
					</div>

					<div class="ui-card">
						<div class="flex items-center justify-between border-b border-(--border) px-4 py-2.5">
							<div class="ui-label">invites</div>
							<a href={resolve('/invites')} class="ui-btn-quiet font-mono text-[10px]">view all</a>
						</div>
						{#if appState.invites.length === 0}
							<p class="p-4 font-mono text-[11px] text-(--silver)/40">No pending room invites.</p>
						{:else}
							<div class="divide-y divide-(--border)">
								{#each appState.invites.slice(0, 3) as invite (invite.id)}
									<div
										class="flex items-center justify-between gap-3 px-4 py-2.5 font-mono text-xs"
									>
										<span class="truncate text-(--silver)/75">{invite.room_name}</span>
										<span class="shrink-0 text-(--silver)/40">{invite.from}</span>
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<div class="ui-card flex flex-1 flex-col p-5">
						<div class="ui-label">competitive</div>
						<p class="mt-2 font-mono text-[11px] leading-relaxed text-(--silver)/50">
							Three clocks, three independent Elos. Everyone starts at 400 on each ladder, and the
							number above is your highest.
						</p>
						<a href={resolve('/leaderboard')} class="ui-btn ui-btn-sm mt-auto pt-2">
							view leaderboard →
						</a>
					</div>
				{:else}
					<section class="ui-card p-5" style="border-top-color: var(--mer);">
						<div class="ui-eyebrow">with an account</div>
						<h2 class="mt-1 text-2xl leading-tight text-(--silver)">Keep your progress.</h2>
						<ul
							class="mt-3 flex flex-col gap-2 font-mono text-[11px] leading-relaxed text-(--silver)/55"
						>
							<li class="flex gap-2">
								<span class="text-(--mer)">·</span>Ranked Elo and the leaderboard
							</li>
							<li class="flex gap-2">
								<span class="text-(--mer)">·</span>Rooms, invites and contests
							</li>
							<li class="flex gap-2">
								<span class="text-(--mer)">·</span>Saved runs and replay links
							</li>
						</ul>
						<a
							href={resolve(`/login?next=${encodeURIComponent('/play')}` as '/login')}
							class="ui-btn ui-btn-primary mt-4 w-full">create account →</a
						>
						<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/40">
							Practice and Zen stay free without one.
						</p>
					</section>
				{/if}
			</aside>
		</div>
	</main>

	<footer class="mx-auto mt-6 w-full max-w-6xl border-t border-(--border) px-5 py-5 md:px-8">
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<div>
				<p class="font-mono text-[10px] tracking-[0.08em] text-(--silver)/40">
					OSDTYP v0.0.1 · experimental · a typing arena for people who write code.
				</p>
				<p class="mt-1 font-mono text-[10px] tracking-[0.08em] text-(--red)/65">
					First public build — bugs expected.
					<a href={resolve('/bugs')} class="underline">Report them.</a>
				</p>
			</div>
			<nav class="flex flex-wrap gap-2 font-mono text-[9px] uppercase">
				<a href={resolve('/welcome')} class="ui-btn ui-btn-sm">features</a>
				<a href="https://osdc.dev/" target="_blank" rel="noreferrer" class="ui-btn ui-btn-sm"
					>osdc.dev</a
				>
				<a
					href="https://itsarnavsh.github.io/"
					target="_blank"
					rel="noreferrer"
					class="ui-btn ui-btn-sm">author site</a
				>
				<a
					href="https://github.com/ItsArnavSh/osdtype"
					target="_blank"
					rel="noreferrer"
					class="ui-btn ui-btn-sm border-(--mer)/40 text-(--mer)">github</a
				>
			</nav>
		</div>
	</footer>
</div>
