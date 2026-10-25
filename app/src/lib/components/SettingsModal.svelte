<script lang="ts">
	import { THEMES, applyTheme, themeState } from '$lib/core/theme.svelte';
	import type { Theme } from '$lib/core/theme.svelte';

	const THEME_SWATCH: Record<Theme, { bg: string; mer: string; silver: string }> = {
		carbon: { bg: '#040806', mer: '#00a19b', silver: '#e8eceb' },
		mono: { bg: '#070707', mer: '#f2f2f2', silver: '#ededed' },
		gruvbox: { bg: '#282828', mer: '#b8bb26', silver: '#ebdbb2' },
		meridian: { bg: '#0b0c0e', mer: '#00a19c', silver: '#c0c0c0' },
		amber: { bg: '#1d1509', mer: '#ffb000', silver: '#e8d5a3' },
		paper: { bg: '#fffdf7', mer: '#0f766e', silver: '#2b2924' },
		crt: { bg: '#040806', mer: '#6bff9e', silver: '#b9ffcf' }
	};

	const SHORTCUTS = [
		['tab', 'restart with a new snippet'],
		['esc', 'open or close settings'],
		['ctrl + shift + t', 'cycle the theme'],
		['ctrl + shift + l', 'next language'],
		['ctrl + shift + s', 'next timer']
	];

	let { open = $bindable(false) }: { open?: boolean } = $props();
</script>

{#if open}
	<button
		class="fixed inset-0 z-40 cursor-default bg-black/60 backdrop-blur-[2px]"
		onclick={() => (open = false)}
		aria-label="Close settings"
		tabindex="-1"
	></button>

	<div
		class="ui-card fixed top-1/2 left-1/2 z-50 flex max-h-[min(42rem,calc(100vh-2rem))] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-y-auto"
		role="dialog"
		aria-modal="true"
		aria-labelledby="settings-title"
		style="border-top-color: var(--mer);"
	>
		<div class="flex items-center justify-between border-b border-(--border) px-5 py-4">
			<div>
				<div class="ui-eyebrow">appearance</div>
				<h2 id="settings-title" class="mt-1 text-xl text-(--silver)">Settings</h2>
			</div>
			<button
				class="ui-btn-quiet font-mono text-lg leading-none"
				onclick={() => (open = false)}
				aria-label="Close settings">×</button
			>
		</div>

		<div class="flex flex-col gap-6 p-5">
			<section>
				<div class="ui-label">theme</div>
				<p class="ui-hint mt-1">Changes apply immediately and are saved for next time.</p>
				<div class="mt-3 grid grid-cols-2 gap-2">
					{#each THEMES as theme (theme)}
						<button
							onclick={() => applyTheme(theme)}
							class="flex cursor-pointer items-center gap-3 border px-3 py-2 text-left font-mono text-xs transition-colors
								{themeState.current === theme
								? 'border-(--mer) text-(--mer)'
								: 'border-(--border) text-(--silver)/65 hover:border-(--silver)/40 hover:text-(--silver)'}"
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
			</section>

			<section class="border-t border-(--border) pt-5">
				<div class="ui-label">keyboard</div>
				<ul class="mt-3 flex flex-col gap-2">
					{#each SHORTCUTS as [keys, description] (keys)}
						<li class="flex items-center gap-3 font-mono text-[11px]">
							<span class="kbd min-w-28 text-center">{keys}</span>
							<span class="text-(--silver)/55">{description}</span>
						</li>
					{/each}
				</ul>
			</section>

			<p
				class="border-t border-(--border) pt-4 font-mono text-[10px] leading-relaxed text-(--silver)/35"
			>
				Language and timer are on the test toolbar, right above the snippet. No need to open
				settings to change them.
			</p>
		</div>
	</div>
{/if}
