<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { devLogin, githubLogin } from '$lib/core/api/auth';
	import { refreshState, appState } from '$lib/core/state/state.svelte';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';

	let username = $state('');
	let busy = $state(false);
	let error = $state('');

	let devBuild = import.meta.env.DEV;

	function nextDestination(): string {
		const next = page.url.searchParams.get('next');
		return next && next.startsWith('/') && !next.startsWith('//') ? next : '/play';
	}

	onMount(() => {
		if (appState.user) {
			void goto(resolve(nextDestination() as '/play'));
			return;
		}
		username = `player-${Math.random().toString(36).slice(2, 6)}`;
		error = page.url.searchParams.get('error') ?? '';
	});

	async function signIn() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			await devLogin(username.trim());
			await refreshState();
			void goto(resolve(nextDestination() as '/play'));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not sign in';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>sign in · OSDTYP</title>
	<meta
		name="description"
		content="Sign in to OSDTYP to save practice runs, play ranked, join rooms, and keep your place in a match."
	/>
</svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg)">
	<header
		class="mx-auto flex w-full max-w-6xl items-center justify-between gap-3 px-5 py-4 md:px-8"
	>
		<a href={resolve('/play')} class="flex items-baseline gap-3">
			<span class="title-text text-3xl leading-none text-(--silver) md:text-4xl">OSDTYP</span>
			<span class="ui-eyebrow hidden sm:inline">code typing arena</span>
		</a>
		<nav class="flex items-center gap-2" aria-label="Account page navigation">
			<button class="ui-btn ui-btn-sm" onclick={cycleTheme} title="Cycle theme">
				theme: {themeState.current}
			</button>
			<a href={resolve('/play')} class="ui-btn ui-btn-sm">← back to play</a>
		</nav>
	</header>

	<main class="mx-auto flex w-full max-w-5xl flex-1 items-center px-5 py-8 md:px-8 md:py-12">
		<div class="grid w-full items-center gap-8 md:grid-cols-[minmax(0,1fr)_24rem] md:gap-14">
			<section class="max-w-xl">
				<div class="ui-eyebrow">your account</div>
				<h1 class="title-text mt-3 text-5xl leading-[0.92] text-(--silver) md:text-7xl">
					SIGN IN.<br />KEEP YOUR RUNS.
				</h1>
				<p class="mt-5 max-w-md text-sm leading-relaxed text-(--silver)/65">
					Save practice history, race ranked, and pick up an unfinished match where you left off.
				</p>

				<ul class="mt-7 grid gap-0 border-t border-(--border)">
					<li class="flex gap-3 border-b border-(--border) py-2.5 font-mono text-xs">
						<span class="text-(--mer)">01</span>
						<span class="text-(--silver)/70">Ranked Elo, rooms and the leaderboard</span>
					</li>
					<li class="flex gap-3 border-b border-(--border) py-2.5 font-mono text-xs">
						<span class="text-(--mer)">02</span>
						<span class="text-(--silver)/70">Saved practice runs and shareable replays</span>
					</li>
					<li class="flex gap-3 border-b border-(--border) py-2.5 font-mono text-xs">
						<span class="text-(--mer)">03</span>
						<span class="text-(--silver)/70">Friends, invites and contest entries</span>
					</li>
				</ul>
			</section>

			<section class="ui-card p-5 md:p-6" style="border-top-color: var(--mer);">
				<div class="ui-eyebrow">welcome</div>
				<h2 class="mt-1 text-2xl leading-tight text-(--silver)">Sign in to OSDTYP.</h2>
				<p class="ui-hint mt-2">
					GitHub is the only sign-in today. It takes a few seconds and creates your account
					automatically.
				</p>

				<button
					class="ui-btn ui-btn-primary mt-5 w-full gap-3 py-3 text-[12px]"
					onclick={githubLogin}
				>
					<svg viewBox="0 0 16 16" class="h-4 w-4" aria-hidden="true" fill="currentColor">
						<path
							d="M8 0C3.58 0 0 3.58 0 8a8 8 0 0 0 5.47 7.59c.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.5-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82a7.4 7.4 0 0 1 4 0c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.28.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8 8 0 0 0 16 8c0-4.42-3.58-8-8-8Z"
						/>
					</svg>
					Continue with GitHub
				</button>

				{#if devBuild}
					<div class="mt-5 border-t border-(--border) pt-4">
						<div class="flex items-center justify-between gap-3">
							<span class="ui-label">developer sign-in</span>
							<span class="font-mono text-[9px] text-(--silver)/30">dev builds only</span>
						</div>
						<form
							class="mt-2 flex flex-col gap-2"
							onsubmit={(e) => {
								e.preventDefault();
								void signIn();
							}}
						>
							<label class="flex flex-col gap-1">
								<span class="sr-only">Username</span>
								<input
									class="ui-input"
									bind:value={username}
									autocomplete="username"
									spellcheck="false"
									maxlength="24"
									placeholder="testuser"
									aria-label="Developer username"
								/>
							</label>
							<button type="submit" disabled={busy || username.trim().length === 0} class="ui-btn">
								{busy ? 'opening session…' : 'sign in with this username →'}
							</button>
						</form>
						<p class="mt-2 font-mono text-[9px] leading-relaxed text-(--silver)/35">
							Only shown in development. GitHub sign-in needs credentials on the API.
						</p>
					</div>
				{/if}

				{#if error}
					<div
						class="mt-4 border border-(--red)/40 bg-(--red)/5 px-3 py-2 font-mono text-xs text-(--red)"
						role="alert"
					>
						{error}
					</div>
				{/if}

				<div class="mt-5 border-t border-(--border) pt-4 text-center">
					<a href={resolve('/play')} class="ui-btn-quiet font-mono text-[11px]">
						or keep practicing as a guest →
					</a>
				</div>
			</section>
		</div>
	</main>

	<footer
		class="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-2 border-t border-(--border) px-5 py-4 font-mono text-[10px] text-(--silver)/35 md:px-8"
	>
		<span>OSDTYP v0.0.1 · experimental</span>
		<a href={resolve('/bugs')} class="transition-colors hover:text-(--mer)">report a bug →</a>
	</footer>
</div>
