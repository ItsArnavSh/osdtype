<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import ModeIcon from '$lib/components/ModeIcon.svelte';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';

	// Only singleplayer (Practice) is live right now. Everything else is
	// rendered but disabled, blurred and tagged as coming soon.
	const SOON = 'coming soon';
	const SOON_MODES = [
		{
			name: 'Sprint',
			icon: 'sprint',
			desc: 'Fast rounds. Instant action. No waiting.',
			time: '30s'
		},
		{
			name: 'Standard',
			icon: 'standard',
			desc: 'Balanced gameplay. Steady pace. Pure fun.',
			time: '90s'
		},
		{
			name: 'Marathon',
			icon: 'marathon',
			desc: 'Endurance mode. Only the focused survive.',
			time: '300s'
		}
	] as const;
</script>

<div class="flex h-screen flex-col bg-(--bg) p-8 pb-16">
	<!-- Header -->
	<header class="flex flex-row items-start justify-between">
		<div class="flex items-end gap-4">
			<div class="title-text text-9xl leading-none text-(--silver)">OSDTYP</div>
			<div class="mb-3 flex flex-col gap-1">
				<button
					class="cursor-pointer rounded border border-(--border) px-2.5 py-1 font-mono text-[11px] text-(--silver)/55 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
					onclick={cycleTheme}
					title="cycle theme (ctrl+shift+T)"
				>
					{themeState.current}
				</button>
			</div>
		</div>
		<div class="flex flex-col items-end gap-1">
			<span class="font-mono text-xs tracking-widest text-(--red) uppercase">hub</span>
			<div class="title-text text-7xl leading-none text-(--silver)">PICK A MODE</div>
		</div>
	</header>

	<!-- Main grid -->
	<div
		class="min-h-0 flex-1 pt-4"
		style="display: grid; grid-template-columns: 1fr 2fr; grid-template-rows: 1fr; gap: 0.75rem;"
	>
		<!-- LEFT: Ranked card + sub-mode buttons -->
		<div
			style="display: grid; grid-template-rows: 1fr auto auto auto; gap: 0.75rem; min-height: 0;"
		>
			<div
				class="relative flex flex-col items-center justify-center gap-6 overflow-hidden rounded bg-(--fbg) p-6"
				style="border: 1px solid var(--border); border-top: 1px solid var(--silver);"
			>
				<div
					class="flex flex-col items-center justify-center gap-6 blur-[2px] select-none"
					aria-hidden="true"
				>
					<div class="text-4xl text-(--silver)">Ranked</div>
					<ModeIcon name="ranked" class="h-24 w-24 text-(--silver)/60" />
					<div class="text-center text-(--silver)">May the fastest fingers win</div>
				</div>
				<span class="soon-badge absolute top-3 right-3">{SOON}</span>
			</div>

			{#each SOON_MODES as mode, index (index)}
				<button
					disabled
					class="relative flex cursor-not-allowed flex-row items-center justify-between overflow-hidden rounded bg-(--fbg) px-4 py-3 opacity-60"
					style="border: 1px solid var(--border);"
				>
					<div class="flex items-center gap-3 blur-[1px] select-none" aria-hidden="true">
						<ModeIcon name={mode.icon} class="h-7 w-7 shrink-0 text-(--silver)/45" />
						<div class="flex flex-col items-start gap-1">
							<span class="text-base text-(--red)">{mode.name}</span>
							<span class="text-left text-xs text-(--silver)">{mode.desc}</span>
						</div>
					</div>
					<span
						class="ml-4 flex-shrink-0 font-mono text-[10px] tracking-widest text-(--silver)/55 uppercase"
					>
						soon
					</span>
				</button>
			{/each}
		</div>

		<!-- RIGHT: Rooms + Practice + Zen -->
		<div style="display: grid; grid-template-rows: 1fr auto; gap: 0.75rem; min-height: 0;">
			<!-- Rooms + Practice -->
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; min-height: 0;">
				<!-- Rooms (coming soon) -->
				<div
					class="relative flex flex-col overflow-hidden rounded bg-(--fbg) p-6"
					style="border: 1px solid var(--border); border-top: 1px solid var(--silver);"
				>
					<div
						class="flex flex-1 flex-col items-center justify-center gap-6 blur-[2px] select-none"
						aria-hidden="true"
					>
						<div class="text-4xl text-(--silver)">Rooms</div>
						<ModeIcon name="rooms" class="h-24 w-24 text-(--silver)/60" />
						<div class="text-center text-(--silver)">
							Time to show the gang<br />who is the boss
						</div>
					</div>
					<div class="flex justify-center pt-4 blur-[2px] select-none" aria-hidden="true">
						<span class="text-4xl text-(--mer)">Create =></span>
					</div>
					<span class="soon-badge absolute top-3 right-3">{SOON}</span>
				</div>

				<!-- Practice (live) -->
				<div
					class="flex flex-col rounded bg-(--fbg) p-6"
					style="border: 1px solid var(--border); border-top: 1px solid var(--mer);"
				>
					<div class="flex flex-1 flex-col items-center justify-center gap-6">
						<div class="text-4xl text-(--silver)">Practice</div>
						<ModeIcon name="practice" class="h-24 w-24 text-(--mer)" />
						<div class="text-center text-(--silver)">The<br />No Judgment Zone</div>
					</div>
					<div class="flex justify-center pt-4">
						<button
							class="cursor-pointer border border-(--mer) px-6 py-2 text-2xl text-(--mer) transition-colors hover:bg-(--mer)/10"
							onclick={() => {
								goto(resolve('/play/me'));
							}}
						>
							Start =>
						</button>
					</div>
				</div>
			</div>

			<!-- Zen (live) -->
			<div
				class="flex w-full items-center justify-between rounded bg-(--fbg) px-8 py-6"
				style="border: 1px solid var(--border); border-top: 1px solid var(--mer);"
			>
				<div class="flex items-center gap-5">
					<ModeIcon name="zen" class="h-14 w-14 shrink-0 text-(--mer)" />
					<div class="flex flex-col gap-2">
						<div class="text-4xl text-(--silver)">Zen Mode</div>
						<div class="text-base text-(--silver)">
							No timers. No stats. Just you, the keys, and the flow.
						</div>
					</div>
				</div>
				<a
					href={resolve('/play/zen')}
					class="shrink-0 border border-(--mer) px-6 py-2 text-2xl text-(--mer) transition-colors hover:bg-(--mer)/10"
				>
					Start =>
				</a>
			</div>
		</div>
	</div>
</div>

<style>
	.soon-badge {
		display: inline-block;
		border: 1px solid var(--border);
		background: color-mix(in srgb, var(--bg) 85%, transparent);
		padding: 2px 8px;
		font-family: 'Fira Code', monospace;
		font-size: 10px;
		letter-spacing: 0.15em;
		text-transform: uppercase;
		color: color-mix(in srgb, var(--silver) 60%, transparent);
	}
</style>
