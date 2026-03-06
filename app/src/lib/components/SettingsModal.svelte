<script lang="ts">
	import type { Language, Time } from '$lib/core/entity/languages';
	import { LANGUAGES, TIMES } from '$lib/core/entity/languages';

	let {
		lang = $bindable<Language>('Golang'),
		timer = $bindable<Time>(300),
		open = $bindable(false),
		onchange
	}: {
		lang?: Language;
		timer?: Time;
		open?: boolean;
		onchange?: (lang: Language, timer: Time) => void;
	} = $props();

	function confirm() {
		onchange?.(lang, timer);
		open = false;
	}
</script>

{#if open}
	<!-- Backdrop -->
	<button
		class="fixed inset-0 z-40 cursor-default bg-black/60"
		onclick={() => (open = false)}
		aria-label="Close modal"
	/>

	<!-- Modal box -->
	<div
		class="fixed top-1/2 left-1/2 z-50 flex min-w-72 -translate-x-1/2 -translate-y-1/2 flex-col gap-6 border border-(--silver) bg-(--bg) p-6"
	>
		<!-- Language -->
		<div class="flex flex-col gap-2">
			<div class="text-sm tracking-widest text-(--silver) uppercase">Language</div>
			<div class="flex flex-row gap-2">
				{#each LANGUAGES as l}
					<button
						onclick={() => (lang = l)}
						class="flex-1 cursor-pointer border px-4 py-2 text-sm transition-colors
							{lang === l
							? 'border-(--mer) text-(--mer)'
							: 'border-(--border) text-(--silver) hover:border-(--mer)/50'}"
					>
						{l}
					</button>
				{/each}
			</div>
		</div>

		<!-- Timer -->
		<div class="flex flex-col gap-2">
			<div class="text-sm tracking-widest text-(--silver) uppercase">Timer</div>
			<div class="flex flex-row gap-2">
				{#each TIMES as t}
					<button
						onclick={() => (timer = t)}
						class="flex-1 cursor-pointer border px-4 py-2 text-sm transition-colors
							{timer === t
							? 'border-(--mer) text-(--mer)'
							: 'border-(--border) text-(--silver) hover:border-(--mer)/50'}"
					>
						{t}s
					</button>
				{/each}
			</div>
		</div>

		<!-- Confirm -->
		<button
			onclick={confirm}
			class="cursor-pointer bg-(--mer) px-4 py-2 text-sm font-bold text-white transition-opacity hover:opacity-80"
		>
			Confirm
		</button>
	</div>
{/if}
