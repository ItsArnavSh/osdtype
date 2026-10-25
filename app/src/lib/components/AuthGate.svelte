<script lang="ts">
	import { resolve } from '$app/paths';

	let { next = '/play' }: { next?: string } = $props();

	const reasons: [string, string][] = [
		['/play/ranked', 'Ranked matches'],
		['/play/rooms', 'Rooms'],
		['/friend', 'Friends'],
		['/leaderboard', 'The leaderboard'],
		['/results', 'Saved results'],
		['/replay', 'Saved replays'],
		['/profile', 'Player profiles'],
		['/invites', 'Invites'],
		['/contests/new', 'Hosting a contest'],
		['/contests/', 'Contest hosting'],
		['/room/', 'Rooms'],
		['/game/', 'Live games']
	];

	let path = $derived(next.split('?')[0] ?? next);
	let reason = $derived(
		reasons.find(([prefix]) => path === prefix || path.startsWith(prefix))?.[1] ?? ''
	);
</script>

<div
	class="fixed inset-0 z-[100] flex items-center justify-center overflow-y-auto bg-(--bg)/75 p-5 backdrop-blur-md"
	role="dialog"
	aria-modal="true"
	aria-labelledby="auth-gate-title"
>
	<div class="ui-card relative w-full max-w-md p-7" style="border-top-color: var(--mer);">
		<a
			href={resolve('/play')}
			class="ui-btn ui-btn-sm absolute top-3 right-3 px-2 py-1 text-base leading-none"
			aria-label="Keep browsing as a guest"
			title="Keep browsing as a guest">×</a
		>

		<div class="ui-eyebrow">account needed</div>
		<h1 id="auth-gate-title" class="mt-2 text-3xl leading-tight text-balance text-(--silver)">
			Sign in to keep going.
		</h1>
		<p class="mt-3 font-mono text-xs leading-relaxed text-(--silver)/55">
			{#if reason}{reason} live behind an account.{:else}This page is behind an account.{/if}
			Practice and Zen stay free — you can keep typing without one.
		</p>

		<div class="mt-6 flex flex-col gap-2">
			<a
				href={resolve(`/login?next=${encodeURIComponent(next)}` as '/login')}
				class="ui-btn ui-btn-primary py-3 text-[12px]"
			>
				Continue with GitHub <span aria-hidden="true">↗</span>
			</a>
			<a href={resolve('/play')} class="ui-btn">Keep browsing as a guest</a>
		</div>

		<p class="mt-5 font-mono text-[10px] leading-relaxed text-(--silver)/40">
			Signing in with GitHub creates your account automatically. No password to remember.
		</p>
	</div>
</div>
