<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { promptState } from '$lib/core/prompts.svelte';

	const ONBOARDED_COOKIE = 'osdtype_onboarded';
	const ONBOARDED_CACHE = 'osdtype.onboarded.v1';
	const ONBOARDED_VALUE = 'v1';
	const CONSENT_COOKIE = 'osdtype_cookie_consent';
	const CONSENT_CACHE = 'osdtype.cookieConsent.v1';
	const ONE_YEAR = 60 * 60 * 24 * 365;

	let ready = $state(false);
	let needsOnboarding = $state(false);
	let hasConsent = $state(false);
	let primaryButton = $state<HTMLButtonElement | null>(null);

	// The welcome note belongs to the first solo practice visit only. The
	// cookie banner can appear elsewhere, but never over a race and never at
	// the same time as the welcome note.
	let onPracticePage = $derived(page.url.pathname.replace(/\/+$/, '') === '/play/me');
	let promptsAllowed = $derived(
		!page.url.pathname.startsWith('/game/') && !page.url.pathname.startsWith('/room/')
	);
	let showOnboarding = $derived(ready && needsOnboarding && onPracticePage);
	let showCookies = $derived(ready && !showOnboarding && !hasConsent && promptsAllowed);

	function readCookie(name: string): string | null {
		if (typeof document === 'undefined') return null;
		const prefix = `${name}=`;
		const found = document.cookie.split('; ').find((part) => part.startsWith(prefix));
		return found ? decodeURIComponent(found.slice(prefix.length)) : null;
	}

	function writeCookie(name: string, value: string) {
		if (typeof document === 'undefined') return;
		const secure = window.location.protocol === 'https:' ? '; Secure' : '';
		document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${ONE_YEAR}; SameSite=Lax${secure}`;
	}

	function readCache(key: string): string | null {
		try {
			return localStorage.getItem(key);
		} catch {
			return null;
		}
	}

	function writeCache(key: string, value: string) {
		try {
			localStorage.setItem(key, value);
		} catch {
			// Cookies still remember the choice if storage is unavailable.
		}
	}

	function markOnboarded() {
		writeCookie(ONBOARDED_COOKIE, ONBOARDED_VALUE);
		writeCache(ONBOARDED_CACHE, '1');
	}

	function finishOnboarding() {
		if (!showOnboarding) return;
		markOnboarded();
		needsOnboarding = false;
	}

	function chooseConsent(choice: 'essential' | 'all') {
		writeCookie(CONSENT_COOKIE, choice);
		writeCache(CONSENT_CACHE, choice);
		hasConsent = true;
	}

	function stopDialogKeys(event: KeyboardEvent) {
		// The typing test listens on window, so keep every key inside the card.
		event.stopPropagation();
		if (event.key === 'Escape') {
			event.preventDefault();
			finishOnboarding();
		}
	}

	$effect(() => {
		promptState.blocking = showOnboarding;
	});

	$effect(() => {
		if (typeof document === 'undefined' || !showOnboarding) return;
		const previous = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = previous;
		};
	});

	$effect(() => {
		if (!showOnboarding || !primaryButton) return;
		// Wait for the typing surface to finish its own mount-time focus call.
		const timer = window.setTimeout(() => primaryButton?.focus(), 100);
		return () => window.clearTimeout(timer);
	});

	onMount(() => {
		const onboarded =
			readCookie(ONBOARDED_COOKIE) === ONBOARDED_VALUE || readCache(ONBOARDED_CACHE) === '1';
		hasConsent = Boolean(readCookie(CONSENT_COOKIE) || readCache(CONSENT_CACHE));
		needsOnboarding = !onboarded;
		ready = true;
	});

	onDestroy(() => {
		promptState.blocking = false;
	});
</script>

{#if showOnboarding}
	<div
		class="fixed inset-0 z-[90] flex items-end justify-center overflow-y-auto bg-(--bg)/85 p-3 backdrop-blur-sm sm:items-center md:p-6"
		role="presentation"
	>
		<div
			class="ui-card max-h-[calc(100vh-1.5rem)] w-full max-w-2xl overflow-y-auto p-5 md:p-7"
			role="dialog"
			aria-modal="true"
			aria-labelledby="first-visit-title"
			tabindex="-1"
			onkeydown={stopDialogKeys}
		>
			<div class="flex flex-wrap items-center justify-between gap-2">
				<span class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
					first visit
				</span>
				<span class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
					one-time note
				</span>
			</div>

			<h2
				id="first-visit-title"
				class="title-text mt-3 text-5xl leading-[0.9] text-(--silver) md:text-6xl"
			>
				start typing
			</h2>
			<p class="mt-3 max-w-xl font-mono text-[11px] leading-relaxed text-(--silver)/50">
				Start on line 1. The clock starts on your first key.
			</p>

			<div class="mt-5 grid gap-4 border-y border-(--border) py-4 sm:grid-cols-3">
				<div>
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/70 uppercase">
						language + timer
					</div>
					<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/40">
						Both are just above the code. Change them before you start.
					</p>
				</div>
				<div>
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/70 uppercase">
						space + enter
					</div>
					<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/40">
						Adaptive is the default: missed spaces pass, extra spaces count, and enters are manual.
						Indentation always auto-tabs. Switch it above the code.
					</p>
				</div>
				<div>
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/70 uppercase">
						sign in for more
					</div>
					<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/40">
						Save runs, race in ranked, join rooms, and play contests.
					</p>
				</div>
			</div>

			<div class="mt-5 flex flex-col gap-3 sm:flex-row sm:items-center">
				<button
					bind:this={primaryButton}
					class="ui-btn ui-btn-primary flex-1 justify-center py-3"
					onclick={finishOnboarding}
				>
					got it · start typing
				</button>
				<a
					href={resolve(`/login?next=${encodeURIComponent(page.url.pathname)}` as '/login')}
					class="text-center font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase underline decoration-(--border) underline-offset-4 hover:text-(--mer)"
					onclick={finishOnboarding}>sign in →</a
				>
			</div>
		</div>
	</div>
{/if}

{#if showCookies}
	<div class="fixed inset-x-0 bottom-0 z-[80] p-3 md:p-5" role="region" aria-label="Cookie choice">
		<div
			class="mx-auto flex w-full max-w-5xl flex-col gap-3 border border-(--border) bg-(--fbg)/95 p-4 shadow-[0_0_0_1px_var(--bg)] backdrop-blur md:flex-row md:items-center md:gap-5"
		>
			<div class="min-w-0 flex-1">
				<div class="flex flex-wrap items-baseline gap-2">
					<span class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
						cookies
					</span>
					<span class="font-mono text-[9px] text-(--silver)/25"> essential + preferences </span>
				</div>
				<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/45 md:text-[11px]">
					OSDTYP uses essential cookies to keep you signed in and remember this browser. No ads, no
					third-party tracking.
				</p>
			</div>
			<div class="flex shrink-0 flex-wrap gap-2">
				<button class="ui-btn ui-btn-sm" onclick={() => chooseConsent('essential')}>
					essential only
				</button>
				<button class="ui-btn ui-btn-sm ui-btn-primary" onclick={() => chooseConsent('all')}>
					accept
				</button>
			</div>
		</div>
	</div>
{/if}
