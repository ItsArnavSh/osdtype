<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { gameState } from '$lib/core/state/state.svelte';

	// No routing yet — purely visual
</script>

<div class="flex h-screen flex-col bg-(--bg) p-8 pb-20">
	<!-- Header -->
	<div class="flex flex-row items-start justify-between">
		<div class="title-text text-9xl text-(--silver)">OSDTYP</div>
		<div class="flex flex-col items-end gap-1">
			<span class="text-sm text-(--red)">Hub</span>
			<div class="title-text text-7xl text-(--silver)">PICK A MODE</div>
		</div>
	</div>

	<!-- Main grid -->
	<div
		class="min-h-0 flex-1"
		style="display: grid; grid-template-columns: 1fr 2fr; grid-template-rows: 1fr; gap: 0.75rem;"
	>
		<!-- LEFT: Ranked card + sub-mode buttons -->
		<div
			style="display: grid; grid-template-rows: 1fr auto auto auto; gap: 0.75rem; min-height: 0;"
		>
			<div
				class="flex flex-col items-center justify-center gap-6 rounded bg-(--fbg) p-6"
				style="border: 1px solid rgba(192,192,192,0.1); border-top: 1px solid var(--silver);"
			>
				<div class="text-4xl text-(--silver)">Ranked</div>
				<img src="/cpp.png" alt="C++" class="h-24 w-24 object-contain" />
				<div class="text-center text-(--silver)">May the fastest fingers win</div>
			</div>

			{#each [{ name: 'Sprint', desc: 'Fast rounds. Instant action. No waiting.', duration: 30, time: '30s' }, { name: 'Standard', desc: 'Balanced gameplay. Steady pace. Pure fun.', duration: 90, time: '90s' }, { name: 'Marathon', desc: 'Endurance mode. Only the focused survive.', duration: 300, time: '300s' }] as mode, index (index)}
				<button
					class="flex cursor-pointer flex-row items-center justify-between rounded bg-(--fbg) px-4 py-4 transition-colors hover:brightness-110"
					style="border: 1px solid rgba(192,192,192,0.1);"
					onclick={() => {
						gameState.mode = mode.duration;
						goto(resolve('/play/ranked'));
					}}
				>
					<div class="flex flex-col items-start gap-1">
						<span class="text-base text-(--red)">{mode.name}</span>
						<span class="text-left text-sm text-(--silver)">{mode.desc}</span>
					</div>
					<span class="ml-4 flex-shrink-0 text-sm text-(--silver)">{mode.time}</span>
				</button>
			{/each}
		</div>

		<!-- RIGHT: Rooms + Practice + Zen -->
		<div style="display: grid; grid-template-rows: 1fr auto; gap: 0.75rem; min-height: 0;">
			<!-- Rooms + Practice — fills same 1fr as Ranked card -->
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; min-height: 0;">
				<!-- Rooms -->
				<div
					class="flex flex-col rounded bg-(--fbg) p-6"
					style="border: 1px solid rgba(192,192,192,0.1); border-top: 1px solid var(--silver);"
				>
					<div class="flex flex-1 flex-col items-center justify-center gap-6">
						<div class="text-4xl text-(--silver)">Rooms</div>
						<img src="/python.png" alt="Python" class="h-24 w-24 object-contain" />
						<div class="text-center text-(--silver)">
							Time to show the gang<br />who is the boss
						</div>
					</div>
					<div class="flex justify-center pt-4">
						<button
							class="cursor-pointer text-4xl text-(--mer) transition-opacity hover:opacity-70"
						>
							Create =>
						</button>
					</div>
				</div>

				<!-- Practice -->
				<div
					class="flex flex-col rounded bg-(--fbg) p-6"
					style="border: 1px solid rgba(192,192,192,0.1); border-top: 1px solid var(--silver);"
				>
					<div class="flex flex-1 flex-col items-center justify-center gap-6">
						<div class="text-4xl text-(--silver)">Practice</div>
						<img src="/html.png" alt="HTML5" class="h-24 w-24 object-contain" />
						<div class="text-center text-(--silver)">The<br />No Judgment Zone</div>
					</div>
					<div class="flex justify-center pt-4">
						<button
							class="cursor-pointer text-4xl text-(--mer) transition-opacity hover:opacity-70"
							onclick={() => {
								goto(resolve('/play/me'));
							}}
						>
							Start =>
						</button>
					</div>
				</div>
			</div>

			<!-- Zen — auto height, sits below like sub-mode buttons -->
			<div
				class="flex w-full items-stretch justify-between overflow-hidden rounded bg-(--fbg)"
				style="border: 1px solid rgba(192,192,192,0.1);"
			>
				<div class="flex flex-col gap-2 px-8 py-6">
					<div class="text-4xl text-(--silver)">Zen Mode</div>
					<div class="text-base text-(--silver)">
						No timers. No stats. Just you, the keys, and the flow.
					</div>
				</div>
				<button
					class="flex w-24 items-center justify-center px-8 text-4xl transition-opacity hover:opacity-80"
					style="background-color: var(--mer); color: var(--silver);"
				>
					=>
				</button>
			</div>
		</div>
	</div>
</div>
