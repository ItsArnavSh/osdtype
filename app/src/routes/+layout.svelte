<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import AuthGate from '$lib/components/AuthGate.svelte';
	import MobileLock from '$lib/components/MobileLock.svelte';
	import { initTheme } from '$lib/core/theme.svelte';
	import { MOBILE_QUERY, mobileAllowed } from '$lib/core/mobile';
	import { browser } from '$app/environment';
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

	// These pages work without an account (and without the Go backend).
	// Everything else stays renderable but is covered by a login gate for
	// guests, so they can still see what unlocking an account adds.
	const GUEST_PATHS = [
		'/',
		'/login',
		'/play',
		'/play/me',
		'/play/zen',
		'/contests',
		'/docs',
		'/bugs',
		'/admin'
	];
	const GUEST_PREFIXES = ['/replay/', '/contests/'];

	function guestAllowed(path: string): boolean {
		// Public contest pages are shareable. Hosting and editing still need
		// an account.
		if (path === '/contests/new' || path.endsWith('/edit')) return false;
		return GUEST_PATHS.includes(path) || GUEST_PREFIXES.some((prefix) => path.startsWith(prefix));
	}

	let mobile = $state(browser ? window.matchMedia(MOBILE_QUERY).matches : false);
	let mobileBlocked = $derived(
		appState.booted && mobile && page.status < 400 && !mobileAllowed(page.url.pathname)
	);

	let gated = $derived(
		appState.booted &&
			!mobileBlocked &&
			page.status < 400 &&
			!appState.user &&
			!guestAllowed(page.url.pathname)
	);

	function targetForCurrentState(): string {
		return currentActive();
	}

	function enforce(path: string): void {
		if (!appState.booted) return;
		if (mobile) {
			// Phones are a read-only companion. Do not let an active desktop
			// game or queue pull a phone into a locked gameplay route.
			if (path === '/') void goto('/play', { replaceState: true });
			return;
		}
		if (!appState.user) {
			// Guests may browse the hub and play offline modes. Protected
			// paths are blurred behind an AuthGate rather than redirected, so
			// an invite link can carry the login "next" destination.
			return;
		}
		if (path.startsWith('/admin')) return;
		if (path === '/') {
			void goto(targetForCurrentState(), { replaceState: true });
			return;
		}
		if (path === '/login') {
			// The login page owns redirects that carry a `next` destination;
			// redirecting here first would discard the gated route the player
			// came from.
			if (page.url.searchParams.get('next')) return;
			void goto(targetForCurrentState(), { replaceState: true });
			return;
		}
		// Shared recordings are public, even for signed-in users. In
		// particular, opening a replay from the results screen must not
		// bounce the viewer back to their unfinished results state.
		if (path.startsWith('/replay/')) return;
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
		const media = window.matchMedia(MOBILE_QUERY);
		const onChange = (event: MediaQueryListEvent) => {
			mobile = event.matches;
		};
		media.addEventListener('change', onChange);
		return () => media.removeEventListener('change', onChange);
	});

	onMount(() => {
		if (!import.meta.env.PROD || !('serviceWorker' in navigator)) return;
		void navigator.serviceWorker.register('/sw.js').catch(() => {
			// PWA support is progressive; registration failures must not break the app.
		});
	});

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
{:else if mobileBlocked}
	<MobileLock path={page.url.pathname} username={appState.user?.username ?? ''} />
{:else}
	<div
		class={`relative min-h-screen ${gated ? 'pointer-events-none opacity-40 blur-[6px] select-none' : ''}`}
		aria-hidden={gated}
		inert={gated}
	>
		{@render children()}
	</div>
	{#if gated}
		<AuthGate next={page.url.pathname + page.url.search} />
	{/if}
{/if}

{#if !mobile && appState.invites.length > 0}
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
						title="decline">✕</button
					>
				</div>
				<div class="mt-3 flex gap-2">
					<button
						class="flex-1 cursor-pointer border border-(--mer) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase hover:bg-(--mer)/10"
						onclick={() => accept(invite.id)}>join</button
					>
					<button
						class="flex-1 cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase hover:text-(--silver)"
						onclick={() => decline(invite.id)}>ignore</button
					>
				</div>
			</div>
		{/each}
	</div>
{/if}

{#if appState.notice}
	<button
		class="fixed bottom-4 left-1/2 z-50 max-w-md -translate-x-1/2 cursor-pointer border border-(--border) bg-(--fbg) px-4 py-2 text-left font-mono text-[11px] text-(--silver)/60"
		onclick={() => (appState.notice = '')}
	>
		{appState.notice}
	</button>
{/if}
