<script lang="ts">
	import type { Language, Time } from '$lib/core/entity/languages';
	import { LANGUAGES, LANGUAGE_LABELS, TIMES } from '$lib/core/entity/languages';
	import { THEMES, applyTheme, themeState } from '$lib/core/theme.svelte';
	import type { Theme } from '$lib/core/theme.svelte';

	const THEME_SWATCH: Record<Theme, { bg: string; mer: string; silver: string }> = {
		gruvbox: { bg: '#282828', mer: '#b8bb26', silver: '#ebdbb2' },
		meridian: { bg: '#0b0c0e', mer: '#00a19c', silver: '#c0c0c0' },
		amber: { bg: '#1d1509', mer: '#ffb000', silver: '#e8d5a3' },
		paper: { bg: '#fffdf7', mer: '#0f766e', silver: '#2b2924' },
		crt: { bg: '#040806', mer: '#6bff9e', silver: '#b9ffcf' }
	};

	let {
		lang = $bindable<Language>('go'),
		timer = $bindable<Time>(30),
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
		aria-label="Close settings"
	></button>

	<!-- Panel -->
	<div
		class="fixed top-1/2 left-1/2 z-50 flex w-[min(30rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col gap-6 border border-(--border) bg-(--bg) p-6 font-mono"
	>
		<div class="flex items-center justify-between">
			<span class="text-[11px] tracking-widest text-(--silver)/50 uppercase">settings</span>
			<button
				class="cursor-pointer text-xs text-(--silver)/45 transition-colors hover:text-(--silver)"
				onclick={() => (open = false)}
			>
				close
			</button>
		</div>

		<!-- Language -->
		<div class="flex flex-col gap-2">
			<div class="text-[11px] tracking-widest text-(--silver)/50 uppercase">language</div>
			<div class="grid grid-cols-3 gap-2">
				{#each LANGUAGES as l (l)}
					<button
						onclick={() => (lang = l)}
						class="cursor-pointer border px-3 py-2 text-xs transition-colors
							{lang === l
							? 'border-(--mer) text-(--mer)'
							: 'border-(--border) text-(--silver)/70 hover:border-(--silver)/40 hover:text-(--silver)'}"
					>
						{LANGUAGE_LABELS[l]}
					</button>
				{/each}
			</div>
		</div>

		<!-- Timer -->
		<div class="flex flex-col gap-2">
			<div class="text-[11px] tracking-widest text-(--silver)/50 uppercase">timer</div>
			<div class="grid grid-cols-3 gap-2">
				{#each TIMES as t (t)}
					<button
						onclick={() => (timer = t)}
						class="cursor-pointer border px-3 py-2 text-xs transition-colors
							{timer === t
							? 'border-(--mer) text-(--mer)'
							: 'border-(--border) text-(--silver)/70 hover:border-(--silver)/40 hover:text-(--silver)'}"
					>
						{t}s
					</button>
				{/each}
			</div>
		</div>

		<!-- Theme -->
		<div class="flex flex-col gap-2">
			<div class="text-[11px] tracking-widest text-(--silver)/50 uppercase">theme</div>
			<div class="grid grid-cols-2 gap-2">
				{#each THEMES as theme (theme)}
					<button
						onclick={() => applyTheme(theme)}
						class="flex cursor-pointer items-center gap-3 border px-3 py-2 text-xs transition-colors
							{themeState.current === theme
							? 'border-(--mer) text-(--mer)'
							: 'border-(--border) text-(--silver)/70 hover:border-(--silver)/40 hover:text-(--silver)'}"
					>
						<span class="flex overflow-hidden rounded-sm border border-(--border)">
							<span class="h-4 w-2" style="background-color: {THEME_SWATCH[theme].bg}"></span>
							<span class="h-4 w-2" style="background-color: {THEME_SWATCH[theme].silver}"></span>
							<span class="h-4 w-2" style="background-color: {THEME_SWATCH[theme].mer}"></span>
						</span>
						{theme}
					</button>
				{/each}
			</div>
		</div>

		<button
			onclick={confirm}
			class="cursor-pointer border border-(--mer) px-4 py-2 text-xs tracking-widest text-(--mer) uppercase transition-colors hover:bg-(--mer)/10"
		>
			done
		</button>
	</div>
{/if}
