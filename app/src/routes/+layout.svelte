<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { initTheme } from '$lib/core/theme.svelte';
	import { page } from '$app/state';
	import { afterNavigate, goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import {
		appState,
		bootSession,
		currentActive,
		refreshState,
		wireGlobalSocket
	} from '$lib/core/state/state.svelte';
	import { connectSocket, disconnectSocket } from '$lib/core/api/socket';
	import { acceptInvite, declineInvite } from '$lib/core/api/rooms';

	let { children } = $props();

	const PUBLIC_PATHS = ['/login'];

	function isPublic(path: string): boolean {
		return path === '/' || PUBLIC_PATHS.includes(path);
	}

	function targetForCurrentState(): string {
		return currentActive();
	}

	function enforce(path: string): void {
		if (!appState.booted) return;
		if (!appState.user) {
			if (!isPublic(path)) {
				void goto(`/login?next=${encodeURIComponent(path + page.url.search)}`, { replaceState: true });
			}
			return;
		}
		if (path === '/' || path === '/login') {
			void goto(targetForCurrentState(), { replaceState: true });
			return;
		}
		const state = appState.state?.state ?? 'idle';
		if (state === 'game' && !path.startsWith('/game/')) {
			void goto(targetForCurrentState(), { replaceState: true });
		} else if (state === 'results' && !path.startsWith('/results/')) {
			void goto(targetForCurrentState(), { replaceState: true });
		} else if (state === 'queue' && path !== '/play/ranked') {
			void goto('/play/ranked', { replaceState: true });
		} else if (state === 'room' && !path.startsWith('/room/')) {
			void goto(targetForCurrentState(), { replaceState: true });
		} else if (state === 'idle' && (path.startsWith('/game/') || path.startsWith('/results/'))) {
			void goto('/play', { replaceState: true });
		}
	}

	onMount(() => {
		initTheme();
		(async () => {
			await bootSession();
			if (appState.user) await refreshState();
			enforce(page.url.pathname);
		})();
	});

	// The layout survives the login page, so socket setup must react to auth
	// rather than run only once on mount. Disconnecting on logout prevents the
	// next account in the same tab from inheriting the old socket identity.
	$effect(() => {
		if (!appState.booted) return;
		if (appState.user) {
			wireGlobalSocket();
			connectSocket().catch(() => {});
		} else {
			disconnectSocket();
		}
	});

	afterNavigate(({ to }) => {
		if (to?.url) enforce(to.url.pathname);
	});

	$effect(() => {
		// Track navigation and authoritative state changes.
		const path = page.url.pathname;
		const _state = appState.state?.state;
		if (appState.booted) enforce(path);
	});

	async function accept(inviteID: number) {
		try {
			await acceptInvite(inviteID);
			appState.invites = appState.invites.filter((i) => i.id !== inviteID);
			await refreshState();
			void goto(targetForCurrentState());
		} catch (err) {
			appState.notice = err instanceof Error ? err.message : 'Could not accept invite';
		}
	}

	async function decline(inviteID: number) {
		try {
			await declineInvite(inviteID);
		} finally {
			appState.invites = appState.invites.filter((i) => i.id !== inviteID);
		}
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<meta property="og:image" content="{page.url.origin}/og.png" />
	<meta property="og:url" content={page.url.href} />
	<meta name="twitter:image" content="{page.url.origin}/og.png" />
	<title>OSDTYP</title>
</svelte:head>

{#if !appState.booted}
	<div class="flex min-h-screen items-center justify-center bg-(--bg)">
		<div class="title-text text-6xl text-(--silver)/40">OSDTYP</div>
	</div>
{:else}
	{@render children()}
{/if}

{#if appState.invites.length > 0}
	<div class="fixed right-4 bottom-4 z-50 flex w-80 flex-col gap-2">
		{#each appState.invites as invite (invite.id)}
			<div class="border border-(--border) bg-(--fbg) p-4 shadow-none">
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<div class="text-[10px] tracking-[0.16em] text-(--mer) uppercase">room invite</div>
						<div class="mt-1 truncate text-sm text-(--silver)">
							{invite.room_name || invite.room_code}
						</div>
						<div class="mt-1 font-mono text-[11px] text-(--silver)/45">
							from {invite.from} · room {invite.room_code}
						</div>
					</div>
					<button
						class="cursor-pointer font-mono text-[10px] text-(--silver)/35 hover:text-(--silver)"
						onclick={() => decline(invite.id)}
						title="decline"
					>✕</button>
				</div>
				<div class="mt-3 flex gap-2">
					<button
						class="flex-1 cursor-pointer border border-(--mer) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase hover:bg-(--mer)/10"
						onclick={() => accept(invite.id)}
					>join</button>
					<button
						class="flex-1 cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:text-(--silver)"
						onclick={() => decline(invite.id)}
					>ignore</button>
				</div>
			</div>
		{/each}
	</div>
{/if}

{#if appState.notice}
	<button
		class="fixed bottom-4 left-4 z-50 max-w-md cursor-pointer border border-(--border) bg-(--fbg) px-4 py-2 text-left font-mono text-[11px] text-(--silver)/60"
		onclick={() => (appState.notice = '')}
	>
		{appState.notice}
	</button>
{/if}
