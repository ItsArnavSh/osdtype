<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import ModeIcon from '$lib/components/ModeIcon.svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import { appState, resetSession } from '$lib/core/state/state.svelte';
	import { logout } from '$lib/core/api/auth';

	let user = $derived(appState.user);
	let guest = $derived(!user);
	let inviteCount = $derived(appState.invites.length);

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
			desc: 'Timed head-to-head races. Elo on the line.',
			href: '/play/ranked',
			cta: 'find match',
			meta: '30 / 90 / 300s'
		},
		{
			name: 'Rooms',
			icon: 'rooms',
			desc: 'Private or public lobbies. Invite the group.',
			href: '/play/rooms',
			cta: 'open rooms',
			meta: '2–16 players'
		}
	] as const;

	function rankLabel(): string {
		return user?.rank.label ?? 'Unranked';
	}
</script>

<svelte:head><title>play · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-6 md:p-8">
	<header class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between md:gap-6">
		<div class="flex w-full items-end justify-between gap-4 md:w-auto md:justify-start">
			<div class="title-text text-6xl leading-none text-(--silver) sm:text-7xl md:text-9xl">OSDTYP</div>
			<button
				class="mb-2 shrink-0 cursor-pointer rounded border border-(--border) px-2.5 py-1 font-mono text-[11px] text-(--silver)/55 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
				onclick={cycleTheme}
				title="cycle theme"
			>{themeState.current}</button>
		</div>
		<div class="flex w-full flex-col items-start gap-3 md:w-auto md:items-end">
			<div class="flex w-full flex-wrap items-center gap-2 md:w-auto md:gap-3">
				{#if user}
					<a
						href={resolve('/leaderboard')}
						class="border border-(--border) px-3 py-1 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:border-(--silver)/40 hover:text-(--silver)"
						>leaderboard</a
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
						class="cursor-pointer border border-(--border) px-3 py-1 font-mono text-[10px] tracking-[0.12em] text-(--silver)/35 uppercase hover:border-(--silver)/40 hover:text-(--silver)"
						onclick={signOut}>log out</button
					>
				{:else}
					<a
						href={loginHref('/play')}
						class="border border-(--mer) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase transition-colors hover:bg-(--mer)/10"
						>sign up / log in =></a
					>
				{/if}
			</div>
			<div class="text-left md:text-right">
				<div class="font-mono text-[10px] tracking-[0.18em] text-(--red) uppercase">hub</div>
				<div class="title-text text-4xl leading-none text-(--silver) sm:text-5xl md:text-6xl">PICK A MODE</div>
			</div>
		</div>
	</header>

	<section class="mt-6 border-y border-(--border) py-3">
		{#if user}
			<div class="flex flex-wrap items-center justify-between gap-4">
				<div class="flex items-center gap-5">
					<div>
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
							rank
						</div>
						<div class="mt-0.5 text-lg text-(--mer)">{rankLabel()}</div>
					</div>
					<div class="h-8 w-px bg-(--border)"></div>
					<div>
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
							rating
						</div>
						<div class="mt-0.5 font-mono text-lg text-(--silver)">{user.rating}</div>
					</div>
					<div class="h-8 w-px bg-(--border)"></div>
					<div>
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
							record
						</div>
						<div class="mt-0.5 font-mono text-sm text-(--silver)/70">
							{user.wins}W / {Math.max(0, user.games_played - user.wins)}L
						</div>
					</div>
				</div>
				<div class="flex items-center gap-2 font-mono text-[10px] text-(--silver)/35">
					<span class={`h-1.5 w-1.5 ${appState.state ? 'bg-(--mer)' : 'bg-(--silver)/25'}`}></span>
					{inviteCount > 0
						? `${inviteCount} pending invite${inviteCount === 1 ? '' : 's'}`
						: 'session live'}
				</div>
			</div>
		{:else}
			<div class="flex flex-wrap items-center justify-between gap-4">
				<div class="flex items-start gap-3">
					<span class="mt-0.5 h-2 w-2 shrink-0 bg-(--mer)"></span>
					<div>
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
							guest mode
						</div>
						<p class="mt-1 font-mono text-xs leading-relaxed text-(--silver)/50">
							Singleplayer and Zen are open. Sign up to save stats, play ranked, join rooms
							and keep your history.
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

	<main class="mt-4 grid min-h-0 flex-1 gap-3 lg:grid-cols-[1.35fr_1fr]">
		<div class="grid gap-3 md:grid-cols-2">
			{#each modes as mode (mode.name)}
				<div class="relative">
					<button
						class={`group flex w-full cursor-pointer flex-col justify-between border border-(--border) bg-(--fbg) p-6 text-left transition-colors hover:border-(--mer)/60 ${guest ? 'pointer-events-none opacity-30 blur-[1.5px]' : ''}`}
						style="border-top-color: var(--mer);"
						onclick={() => goto(resolve(mode.href))}
						disabled={guest}
						aria-hidden={guest}
						tabindex={guest ? -1 : 0}
					>
						<div class="flex w-full items-start justify-between">
							<div class="text-4xl text-(--silver)">{mode.name}</div>
							<ModeIcon name={mode.icon} class="h-14 w-14 text-(--mer)/75" />
						</div>
						<p class="mt-4 max-w-[24ch] font-mono text-xs leading-relaxed text-(--silver)/50">
							{mode.desc}
						</p>
						<div class="mt-8 flex items-end justify-between">
							<span class="font-mono text-[10px] tracking-[0.15em] text-(--silver)/30 uppercase"
								>{mode.meta}</span
							>
							<span
								class="border border-(--mer) px-4 py-1.5 font-mono text-xs text-(--mer) transition-colors group-hover:bg-(--mer)/10"
								>{mode.cta} =></span
							>
						</div>
					</button>
					{#if guest}
						<a
							href={loginHref(mode.href)}
							class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-(--bg)/30 p-4 text-center transition-colors hover:bg-(--bg)/10"
						>
							<span class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/45 uppercase"
								>login to continue</span
							>
							<span
								class="border border-(--mer) bg-(--bg)/80 px-4 py-1.5 font-mono text-xs text-(--mer)"
								>sign in for {mode.name.toLowerCase()} =></span
							>
						</a>
					{/if}
				</div>
			{/each}
		</div>

		<div class="grid gap-3">
			<button
				class="flex cursor-pointer items-center justify-between border border-(--border) bg-(--fbg) px-6 py-5 text-left transition-colors hover:border-(--mer)/60"
				onclick={() => goto(resolve('/play/me'))}
			>
				<div class="flex items-center gap-4">
					<ModeIcon name="practice" class="h-10 w-10 text-(--mer)" />
					<div>
						<div class="text-2xl text-(--silver)">Practice</div>
						<div class="mt-1 font-mono text-[11px] text-(--silver)/40">
							solo · any language · no rating
						</div>
					</div>
				</div>
				<span class="font-mono text-xs text-(--mer)">start =></span>
			</button>

			<button
				class="flex cursor-pointer items-center justify-between border border-(--border) bg-(--fbg) px-6 py-5 text-left transition-colors hover:border-(--mer)/60"
				onclick={() => goto(resolve('/play/zen'))}
			>
				<div class="flex items-center gap-4">
					<ModeIcon name="zen" class="h-10 w-10 text-(--mer)" />
					<div>
						<div class="text-2xl text-(--silver)">Zen</div>
						<div class="mt-1 font-mono text-[11px] text-(--silver)/40">
							no timer · no stats · just keys
						</div>
					</div>
				</div>
				<span class="font-mono text-xs text-(--mer)">drift =></span>
			</button>

			{#if user}
				<div class="border border-(--border) bg-(--fbg) px-6 py-5">
					<div class="flex items-center justify-between">
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
							invites
						</div>
						<a
							href={resolve('/invites')}
							class="font-mono text-[10px] text-(--silver)/40 hover:text-(--silver)"
							>view all</a
						>
					</div>
					{#if appState.invites.length === 0}
						<p class="mt-3 font-mono text-xs text-(--silver)/30">No pending room invites.</p>
					{:else}
						<div class="mt-3 flex flex-col gap-2">
							{#each appState.invites.slice(0, 3) as invite (invite.id)}
								<div class="flex items-center justify-between gap-3 font-mono text-xs">
									<span class="truncate text-(--silver)/70">{invite.room_name}</span>
									<span class="shrink-0 text-(--silver)/30">{invite.from}</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<div class="border border-(--border) px-6 py-4">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/30 uppercase">
						next rank
					</div>
					<div class="mt-2 flex items-center justify-between font-mono text-[11px] text-(--silver)/45">
						<span>{rankLabel()}</span>
						<span>{user.rank.next ?? '—'}</span>
					</div>
					<div class="mt-2 h-px w-full bg-(--border)">
						<div
							class="h-px bg-(--mer)"
							style={`width:${Math.min(100, Math.max(0, user.rank.progress * 100))}%`}
						></div>
					</div>
				</div>
			{:else}
				<div
					class="border border-(--border) bg-(--fbg) px-6 py-5"
					style="border-top-color: var(--mer);"
				>
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
						with an account
					</div>
					<div class="mt-1 text-xl text-(--silver)">Unlock the rest.</div>
					<ul class="mt-3 flex flex-col gap-2 font-mono text-[11px] leading-relaxed text-(--silver)/45">
						<li>· ranked Elo and leaderboards</li>
						<li>· rooms, invites and contests</li>
						<li>· saved results across sessions</li>
					</ul>
					<a
						href={loginHref('/play')}
						class="mt-4 flex justify-center border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
						>sign up free =></a
					>
				</div>
			{/if}
		</div>
	</main>

	<footer class="mt-5 flex flex-wrap items-center justify-between gap-2 border-t border-(--border) pt-3 font-mono text-[10px] text-(--silver)/25">
		<span>server-authoritative · explicit leave only</span>
		<span>resrap-rs 1.0.0 · 10 languages</span>
	</footer>
</div>
