<script lang="ts">
	import { resolve } from '$app/paths';

	let { path, username = '' }: { path: string; username?: string } = $props();

	const loginHref = $derived(`/login?next=${encodeURIComponent('/play')}`);
	const profileHref = $derived(username ? resolve(`/profile/${username}`) : loginHref);
	const friendsHref = $derived(
		username ? resolve('/friends') : `/login?next=${encodeURIComponent('/friends')}`
	);
	const ladderHref = $derived(
		username ? resolve('/leaderboard') : `/login?next=${encodeURIComponent('/leaderboard')}`
	);
</script>

<svelte:head><title>desktop only · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-5 pb-8 md:p-8">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<a href={resolve('/play')} class="flex items-baseline gap-3">
			<span class="title-text text-3xl leading-none text-(--silver)">OSDTYP</span>
			<span class="font-mono text-[9px] tracking-[0.16em] text-(--mer) uppercase">phone build</span>
		</a>
		<a
			href={resolve('/play')}
			class="border border-(--border) px-3 py-2 font-mono text-[10px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
			>← mobile hub</a
		>
	</header>

	<main class="flex flex-1 items-center justify-center py-8">
		<section class="w-full max-w-xl border border-(--border) bg-(--fbg) p-6 md:p-8">
			<div class="font-mono text-[10px] tracking-[0.18em] text-(--red) uppercase">desktop only</div>
			<h1 class="title-text mt-2 text-4xl leading-none text-(--silver) md:text-5xl">
				THIS SCREEN NEEDS A KEYBOARD.
			</h1>
			<p class="mt-4 font-mono text-xs leading-relaxed text-(--silver)/45">
				Ranked, rooms, practice, Zen and live games are locked on phones for now. The phone build
				keeps profiles, shared replays, friends, leaderboards, docs and bug reports open.
			</p>
			<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/25">
				requested: <span class="text-(--silver)/40">{path}</span> · open the desktop build to continue.
			</p>

			<nav class="mt-6 grid gap-2 sm:grid-cols-2">
				<a
					href={resolve('/play')}
					class="border border-(--mer) px-4 py-2.5 text-center font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
					>← mobile hub</a
				>
				<a
					href={profileHref}
					class="border border-(--border) px-4 py-2.5 text-center font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					>{username ? 'your profile' : 'sign in'}</a
				>
				{#if username}
					<a
						href={friendsHref}
						class="border border-(--border) px-4 py-2.5 text-center font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
						>friends</a
					>
					<a
						href={ladderHref}
						class="border border-(--border) px-4 py-2.5 text-center font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
						>leaderboard</a
					>
				{/if}
				<a
					href={resolve('/docs')}
					class="border border-(--border) px-4 py-2.5 text-center font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					>docs</a
				>
				<a
					href={resolve('/bugs')}
					class="border border-(--border) px-4 py-2.5 text-center font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					>report a bug</a
				>
			</nav>
		</section>
	</main>

	<footer class="font-mono text-[9px] tracking-[0.08em] text-(--silver)/25">
		OSDTYP v0.0.1 · desktop gameplay · mobile companion
	</footer>
</div>
