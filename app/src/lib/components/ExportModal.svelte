<script lang="ts">
	import { toPng } from 'html-to-image';
	import SpeedGraph from '$lib/components/SpeedGraph.svelte';
	import {
		THEMES,
		THEME_PALETTES,
		paletteStyle,
		themeState,
		type Theme
	} from '$lib/core/theme.svelte';
	import type { ExportStat, SpeedSeries } from '$lib/core/stats';

	let {
		title,
		subtitle,
		series,
		duration,
		stats = [],
		filename = 'osdtype-run',
		onclose
	}: {
		title: string;
		subtitle?: string;
		series: SpeedSeries[];
		duration: number;
		stats?: ExportStat[];
		filename?: string;
		onclose: () => void;
	} = $props();

	let theme = $state<Theme>(themeState.current);
	let scale = $state(2);
	let cardRef = $state<HTMLDivElement | null>(null);
	let exporting = $state(false);
	let error = $state('');

	const palette = $derived(THEME_PALETTES[theme]);

	async function exportImage() {
		if (!cardRef || exporting) return;
		exporting = true;
		error = '';
		try {
			const dataUrl = await toPng(cardRef, {
				backgroundColor: palette.bg,
				pixelRatio: scale,
				cacheBust: true
			});
			const link = document.createElement('a');
			const safe = filename.replace(/[^a-z0-9-]+/gi, '-').replace(/^-+|-+$/g, '') || 'osdtype-run';
			link.download = `${safe}-${theme}-${scale}x.png`;
			link.href = dataUrl;
			link.click();
		} catch (err) {
			console.error('Could not export result', err);
			error = 'Could not render the image. Try again.';
		} finally {
			exporting = false;
		}
	}
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-(--bg)/90 p-3 md:p-6"
	role="presentation"
	onclick={onclose}
>
	<div
		class="max-h-full w-full max-w-5xl overflow-y-auto border border-(--border) bg-(--bg) p-4 md:p-6"
		role="dialog"
		aria-modal="true"
		aria-label="export run image"
		onclick={(event) => event.stopPropagation()}
		onkeydown={(event) => event.key === 'Escape' && onclose()}
		tabindex="-1"
	>
		<div class="flex flex-wrap items-start justify-between gap-3 border-b border-(--border) pb-3">
			<div>
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">export image</div>
				<div class="mt-1 text-xl text-(--silver)">{title}</div>
				{#if subtitle}<div class="mt-1 font-mono text-[10px] text-(--silver)/35">{subtitle}</div>{/if}
			</div>
			<button
				class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--silver)/50 uppercase hover:text-(--silver)"
				onclick={onclose}>close ✕</button
			>
		</div>

		<div class="mt-4 grid gap-4 lg:grid-cols-[1fr_260px]">
			<div class="overflow-x-auto border border-(--border) bg-(--bg) p-3">
				<div class="mx-auto w-[820px] max-w-none" bind:this={cardRef} style={paletteStyle(palette)}>
					<div class="border border-(--border) bg-(--bg) p-8">
						<div class="flex items-start justify-between gap-4 border-b border-(--border) pb-5">
							<div>
								<div class="title-text text-4xl leading-none text-(--silver)">OSDTYP</div>
								<div class="mt-2 font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
									{title}
								</div>
							</div>
							<div class="text-right font-mono text-[10px] text-(--silver)/35">
								<div>{duration}s · {theme}</div>
								<div class="mt-1">osdtype.local</div>
							</div>
						</div>

						{#if stats.length > 0}
							<div class="mt-6 grid grid-cols-4 gap-2">
								{#each stats as stat (stat.label)}
									<div class="border border-(--border) px-3 py-2">
										<div class={`font-mono text-2xl ${stat.accent ? 'text-(--mer)' : 'text-(--silver)'}`}>
											{stat.value}
										</div>
										<div class="mt-1 font-mono text-[9px] tracking-widest text-(--silver)/35 uppercase">
											{stat.label}
										</div>
									</div>
								{/each}
							</div>
						{/if}

						<div class="mt-6 border border-(--border) bg-(--fbg) p-4">
							<SpeedGraph {series} {duration} />
						</div>

						<div class="mt-4 flex items-center justify-between font-mono text-[9px] tracking-[0.14em] text-(--silver)/25 uppercase">
							<span>recorded runs · simulated replay</span>
							<span>ctrl+shift+t to change theme</span>
						</div>
					</div>
				</div>
			</div>

			<aside class="flex flex-col gap-3">
				<div class="border border-(--border) bg-(--fbg) p-4">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">theme</div>
					<div class="mt-3 grid grid-cols-2 gap-1.5">
						{#each THEMES as option (option)}
							<button
								class={`flex cursor-pointer items-center gap-2 border px-2 py-1.5 text-left font-mono text-[10px] capitalize transition-colors ${
									theme === option
										? 'border-(--mer) text-(--mer)'
										: 'border-(--border) text-(--silver)/45 hover:text-(--silver)'
								}`}
								onclick={() => (theme = option)}
							>
								<span
									class="h-2.5 w-2.5 shrink-0 border border-(--border)"
									style={`background:${THEME_PALETTES[option].mer}`}
								></span>
								{option}
							</button>
						{/each}
					</div>
				</div>

				<div class="border border-(--border) bg-(--fbg) p-4">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
						resolution
					</div>
					<div class="mt-3 grid grid-cols-3 gap-1.5">
						{#each [1, 2, 3] as option (option)}
							<button
								class={`cursor-pointer border px-2 py-1.5 font-mono text-[10px] ${
									scale === option
										? 'border-(--mer) text-(--mer)'
										: 'border-(--border) text-(--silver)/45 hover:text-(--silver)'
								}`}
								onclick={() => (scale = option)}
							>
								{option}x
							</button>
						{/each}
					</div>
					<div class="mt-2 font-mono text-[9px] text-(--silver)/30">
						{Math.round(820 * scale)} × auto px
					</div>
				</div>

				{#if error}
					<div class="border border-(--red)/40 px-3 py-2 font-mono text-[10px] text-(--red)">{error}</div>
				{/if}

				<button
					class="cursor-pointer border border-(--mer) px-4 py-2.5 font-mono text-sm text-(--mer) hover:bg-(--mer)/10 disabled:opacity-40"
					onclick={exportImage}
					disabled={exporting}
				>{exporting ? 'rendering…' : 'download png =>'}</button
				>
				<p class="font-mono text-[9px] leading-relaxed text-(--silver)/30">
					The captured card uses the selected theme. Resolution is a pixel-density multiplier.
				</p>
			</aside>
		</div>
	</div>
</div>
