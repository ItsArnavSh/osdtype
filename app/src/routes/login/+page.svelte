<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { devLogin, githubLogin } from '$lib/core/api/auth';
	import { refreshState, appState } from '$lib/core/state/state.svelte';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import VersionNotice from '$lib/components/VersionNotice.svelte';

	let username = $state('');
	let busy = $state(false);
	let error = $state('');

	function nextDestination(): string {
		const next = page.url.searchParams.get('next');
		return next && next.startsWith('/') && !next.startsWith('//') ? next : '/play';
	}

	onMount(() => {
		if (appState.user) {
			void goto(nextDestination());
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
			void goto(nextDestination());
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not sign in';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>login · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-5 pb-8 md:p-12">
	<header class="flex items-start justify-between">
		<div class="flex items-end gap-4">
			<div class="title-text text-6xl leading-none text-(--silver) md:text-9xl">OSDTYP</div>
			<button
				class="mb-3 cursor-pointer rounded border border-(--border) px-2.5 py-1 font-mono text-[11px] text-(--silver)/55 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
				onclick={cycleTheme}
				title="cycle theme">{themeState.current}</button
			>
		</div>
		<div class="hidden flex-col items-end gap-1 md:flex">
			<span class="font-mono text-xs tracking-widest text-(--red) uppercase">terminal</span>
			<div class="title-text text-7xl leading-none text-(--silver)">SIGN IN</div>
		</div>
	</header>

	<div class="mt-6">
		<VersionNotice />
	</div>

	<main class="flex flex-1 items-center justify-center py-10">
		<div class="w-full max-w-md border border-(--border) bg-(--fbg)">
			<div class="border-b border-(--border) px-6 py-4">
				<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
					<span class="md:hidden">profile · replays · friends</span>
					<span class="hidden md:inline">ranked · rooms · practice</span>
				</div>
				<h1 class="mt-2 text-2xl text-(--silver)">Punch in.</h1>
				<p class="mt-1 font-mono text-xs leading-relaxed text-(--silver)/45">
					<span class="md:hidden">
						Your profile, shared replays and friends are waiting. Live typing modes stay on the
						desktop build for now.
					</span>
					<span class="hidden md:inline">
						Your rating, rooms and unfinished races are waiting when you return. Jump back in
						wherever you left off.
					</span>
				</p>
			</div>

			<div class="flex flex-col gap-5 p-6">
				<button
					class="w-full cursor-pointer border border-(--silver)/50 px-4 py-3 text-left font-mono text-sm text-(--silver) transition-colors hover:border-(--mer) hover:text-(--mer)"
					onclick={githubLogin}
				>
					Continue with GitHub <span class="float-right text-(--silver)/30">↗</span>
				</button>

				{#if import.meta.env.DEV}
					<div class="flex items-center gap-3">
						<div class="h-px flex-1 bg-(--border)"></div>
						<span class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/30 uppercase"
							>or dev login</span
						>
						<div class="h-px flex-1 bg-(--border)"></div>
					</div>

					<form
						class="flex flex-col gap-3"
						onsubmit={(e) => {
							e.preventDefault();
							void signIn();
						}}
					>
						<label class="flex flex-col gap-1">
							<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
								>username</span
							>
							<input
								class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
								bind:value={username}
								autocomplete="username"
								spellcheck="false"
								maxlength="24"
								placeholder="testuser"
							/>
						</label>
						<button
							type="submit"
							disabled={busy || username.trim().length === 0}
							class="cursor-pointer border border-(--mer) px-4 py-2.5 font-mono text-sm text-(--mer) transition-colors hover:bg-(--mer)/10 disabled:cursor-not-allowed disabled:opacity-40"
						>
							{busy ? 'opening session…' : 'enter terminal =>'}
						</button>
					</form>
				{/if}

				{#if error}
					<div class="border border-(--red)/40 px-3 py-2 font-mono text-xs text-(--red)">
						{error}
					</div>
				{/if}

				{#if import.meta.env.DEV}
					<p class="font-mono text-[10px] leading-relaxed text-(--silver)/25">
						GitHub sign-in is unavailable in this environment. Use the developer login below to test
						the app.
					</p>
				{/if}
			</div>
		</div>
	</main>
</div>
