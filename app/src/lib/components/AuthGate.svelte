<script lang="ts">
	import { resolve } from '$app/paths';

	let { next = '/play' }: { next?: string } = $props();

	let loginHref = $derived(`/login?next=${encodeURIComponent(next)}`);
</script>

<div
	class="fixed inset-0 z-[100] flex items-center justify-center overflow-y-auto bg-(--bg)/60 p-6 backdrop-blur-sm"
>
	<div
		class="w-full max-w-md border border-(--border) bg-(--fbg) p-7"
		style="border-top-color: var(--mer);"
	>
		<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">members only</div>
		<h1 class="mt-2 text-3xl leading-tight text-(--silver)">Login to continue.</h1>
		<p class="mt-3 font-mono text-xs leading-relaxed text-(--silver)/45">
			<span class="md:hidden">
				Profiles, shared replays, friends and leaderboards need an account on phones. Live typing
				modes stay on the desktop build.
			</span>
			<span class="hidden md:inline">
				Ranked, rooms, contests, leaderboards and saved results need an account. Singleplayer and
				Zen stay free — no login required.
			</span>
		</p>

		<div class="mt-6 flex flex-col gap-2">
			<a
				href={loginHref}
				class="border border-(--mer) px-4 py-3 text-center font-mono text-sm text-(--mer) transition-colors hover:bg-(--mer)/10"
			>
				sign up / log in =>
			</a>
			<a
				href={resolve('/play')}
				class="border border-(--border) px-4 py-2.5 text-center font-mono text-xs text-(--silver)/45 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
			>
				← back to guest hub
			</a>
		</div>

		<p class="mt-5 font-mono text-[10px] leading-relaxed text-(--silver)/25">
			<span class="md:hidden">
				Your account keeps your profile, shared replays, friends and ladder positions.
			</span>
			<span class="hidden md:inline">
				Your account keeps your rating, room memberships and unfinished matches. Login returns you
				to exactly where you left off.
			</span>
		</p>
	</div>
</div>
