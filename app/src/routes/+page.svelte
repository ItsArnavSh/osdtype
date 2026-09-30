<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { ping, redirectToGithubLogin, whoAmI } from '$lib/core/api';
	import type { User } from '$lib/core/api';
	import { onMount } from 'svelte';

	// Connection state, probed once on mount. The landing page is the first
	// thing a developer sees, so a silent failure here reads as "the app is
	// broken" rather than "the backend is not running".
	type Status = 'checking' | 'online' | 'offline';
	let status = $state<Status>('checking');
	let user = $state<User | null>(null);

	const MODES = [
		{
			name: 'Ranked',
			tagline: 'May the fastest fingers win',
			desc: 'Matchmaking pairs you with players near your rank. Every result moves your rating.',
			lang: 'C++',
			icon: '/cpp.png',
			href: '/play',
			accent: 'var(--silver)'
		},
		{
			name: 'Contests',
			tagline: 'Everyone starts on the same keystroke',
			desc: 'Scheduled rooms with a shared lobby, a shared snippet, and a leaderboard everyone can read.',
			lang: 'Rust',
			icon: '/python.png',
			href: '/hub',
			accent: 'var(--mer)'
		},
		{
			name: 'Practice',
			tagline: 'Nobody is watching',
			desc: 'A quiet room with your own timer. WPM and accuracy update keystroke by keystroke.',
			lang: 'TypeScript',
			icon: '/html.png',
			href: '/play',
			accent: 'var(--red)'
		}
	] as const;

	const LANGS = [
		{ name: 'C', icon: '/cpp.png' },
		{ name: 'Go', icon: '/cpp.png' },
		{ name: 'C++', icon: '/cpp.png' },
		{ name: 'Java', icon: '/python.png' },
		{ name: 'Rust', icon: '/cpp.png' },
		{ name: 'TypeScript', icon: '/html.png' }
	] as const;

	onMount(async () => {
		try {
			status = (await ping()) === 'pong' ? 'online' : 'offline';
		} catch {
			status = 'offline';
		}

		// The session cookie is optional, so a 401 here is the normal case for
		// a signed-out visitor rather than an error worth surfacing.
		try {
			user = await whoAmI();
		} catch {
			user = null;
		}
	});
</script>

<div class="flex min-h-screen flex-col bg-(--bg) p-8 pb-20">
	<!-- Header -->
	<header class="flex flex-row items-start justify-between">
		<div class="title-text text-9xl text-(--silver)">OSDTYP</div>
		<div class="flex flex-col items-end gap-2">
			<div
				class="flex items-center gap-2 rounded px-2 py-1 text-xs tracking-widest"
				style="border: 1px solid rgba(192,192,192,0.15); color: var(--silver);"
				role="status"
			>
				<span
					class="inline-block h-2 w-2 rounded-full"
					style="background: {status === 'online'
						? 'var(--mer)'
						: status === 'offline'
							? 'var(--red)'
							: 'var(--silver)'};"
				></span>
				{#if status === 'checking'}
					CHECKING
				{:else if status === 'online'}
					SERVER ONLINE
				{:else}
					SERVER OFFLINE
				{/if}
			</div>

			{#if user}
				<button
					onclick={() => goto(resolve('/play'))}
					class="title-text cursor-pointer text-7xl text-(--silver) transition-opacity hover:opacity-70"
				>
					{user.username}
				</button>
			{:else}
				<button
					onclick={() => redirectToGithubLogin()}
					class="title-text cursor-pointer text-7xl text-(--silver) transition-opacity hover:opacity-70"
				>
					LOGIN
				</button>
			{/if}
		</div>
	</header>

	<!-- Hero -->
	<section class="mt-12 flex flex-col gap-6">
		<div class="flex flex-col gap-4">
			<div class="title-text text-8xl text-(--silver)">TYPE AGAINST OTHER HUMANS</div>
			<div class="max-w-3xl text-lg leading-relaxed text-(--silver) opacity-70">
				OSDType is a multiplayer typing arena. Real code as the snippet, live WPM and accuracy for
				every keystroke, and an Elo rating that keeps finding you opponents worth beating.
			</div>
		</div>

		<div class="flex flex-wrap items-center gap-3">
			<button
				onclick={() => goto(resolve('/play'))}
				class="cursor-pointer rounded px-8 py-4 text-lg transition-opacity hover:opacity-80"
				style="background: var(--red); color: var(--silver);"
			>
				Start Playing
			</button>
			<button
				onclick={() => goto(resolve('/hub'))}
				class="cursor-pointer rounded px-8 py-4 text-lg transition-opacity hover:opacity-80"
				style="background: var(--mer); color: var(--bg);"
			>
				Browse Contests
			</button>
		</div>
	</section>

	<!-- Modes -->
	<section
		class="mt-12 grid flex-1 gap-3"
		style="grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));"
	>
		{#each MODES as mode (mode.name)}
			<button
				onclick={() => goto(resolve(mode.href as '/play'))}
				class="flex cursor-pointer flex-col justify-between gap-6 rounded bg-(--fbg) p-6 text-left transition-transform hover:-translate-y-0.5"
				style="border: 1px solid rgba(192,192,192,0.1); border-top: 2px solid {mode.accent};"
			>
				<div class="flex items-center justify-between">
					<span class="text-xs tracking-widest" style="color: {mode.accent}">{mode.lang}</span>
					<img src={mode.icon} alt={mode.lang} class="h-8 w-8 object-contain" />
				</div>

				<div class="flex flex-col gap-2">
					<div class="title-text text-4xl text-(--silver)">{mode.name}</div>
					<div class="text-sm text-(--silver) opacity-60">{mode.tagline}</div>
				</div>

				<div class="text-sm leading-relaxed text-(--silver) opacity-50">{mode.desc}</div>
			</button>
		{/each}
	</section>

	<!-- Languages + footer -->
	<footer class="mt-12 flex flex-col gap-6">
		<div
			class="flex flex-col gap-3"
			style="border-top: 1px solid rgba(192,192,192,0.1); padding-top: 2rem;"
		>
			<div class="text-xs tracking-widest text-(--silver) opacity-40">
				SIX LANGUAGES, RANDOM SNIPPETS
			</div>
			<div class="flex flex-wrap items-center gap-6">
				{#each LANGS as lang (lang.name)}
					<div class="flex items-center gap-2">
						<img src={lang.icon} alt={lang.name} class="h-6 w-6 object-contain" />
						<span class="text-sm text-(--silver) opacity-60">{lang.name}</span>
					</div>
				{/each}
			</div>
		</div>

		{#if status === 'offline'}
			<div
				class="rounded px-4 py-3 text-sm"
				style="border: 1px solid var(--red); color: var(--red);"
			>
				The backend is not responding. Start it with <code>just dev</code> (or
				<code>docker compose up</code>) and reload.
			</div>
		{/if}
	</footer>
</div>
