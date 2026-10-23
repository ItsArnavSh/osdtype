<script lang="ts">
	import SpeedGraph from '$lib/components/SpeedGraph.svelte';
	import type { ReplayRun } from '$lib/core/api/types';
	import type { Language } from '$lib/core/entity/languages';
	import type { SpeedSeries } from '$lib/core/stats';
	import {
		charsOf,
		deriveSamples,
		mapTypedToDisplay,
		statsAt,
		typedAt,
		byteToChar
	} from '$lib/core/replay';
	import { highlightCode } from '$lib/core/syntax';
	import { onMount } from 'svelte';

	let {
		run,
		snippet,
		startOffset,
		duration,
		language,
		shareURL = '',
		onclose
	}: {
		run: ReplayRun;
		snippet: string;
		startOffset: number;
		duration: number;
		language: string;
		shareURL?: string;
		onclose?: () => void;
	} = $props();

	const chars = $derived(charsOf(snippet));
	const startChar = $derived(byteToChar(snippet, startOffset));
	const syntaxKinds = $derived(highlightCode(snippet, language as Language));
	const durationSec = $derived(Math.max(1, duration));
	const durationMs = $derived(durationSec * 1000);
	const events = $derived(
		[...run.events].sort((a, b) => (a.time_ms ?? 0) - (b.time_ms ?? 0))
	);
	const lines = $derived.by(() => {
		const out: number[][] = [];
		let current: number[] = [];
		for (let index = 0; index < chars.length; index++) {
			if (chars[index] === '\n') {
				out.push(current);
				current = [];
			} else {
				current.push(index);
			}
		}
		out.push(current);
		return out;
	});

	let currentMs = $state(0);
	let playing = $state(true);
	let speed = $state(1);
	let copied = $state(false);
	let codeEl = $state<HTMLDivElement | null>(null);
	let lineRefs: (HTMLDivElement | undefined)[] = [];
	let raf = 0;
	let lastFrame = 0;

	const typedChars = $derived(typedAt(events, currentMs));
	const display = $derived(mapTypedToDisplay(snippet, startOffset, typedChars));
	const currentLine = $derived.by(() => {
		let line = 0;
		for (let index = 0; index < display.cursor && index < chars.length; index++) {
			if (chars[index] === '\n') line++;
		}
		return Math.min(line, Math.max(0, lines.length - 1));
	});
	const stats = $derived(statsAt(events, snippet, startOffset, currentMs));
	const fullSamples = $derived(deriveSamples(events, snippet, startOffset, durationSec));
	const liveSamples = $derived(fullSamples.filter((sample) => sample.t * 1000 <= currentMs + 1));
	const series = $derived<SpeedSeries[]>([
		{ label: run.username, samples: liveSamples, color: 'var(--mer)' }
	]);
	const elapsedLabel = $derived(formatTime(currentMs));
	const totalLabel = $derived(formatTime(durationMs));

	function formatTime(ms: number): string {
		const safe = Math.max(0, Math.round(ms / 1000));
		return `${Math.floor(safe / 60)}:${(safe % 60).toString().padStart(2, '0')}`;
	}

	function togglePlay() {
		if (currentMs >= durationMs) currentMs = 0;
		playing = !playing;
		lastFrame = 0;
	}

	function restart() {
		currentMs = 0;
		playing = true;
		lastFrame = 0;
	}

	function cycleSpeed() {
		speed = speed === 1 ? 2 : speed === 2 ? 4 : 1;
	}

	function seek(event: Event) {
		const value = Number((event.currentTarget as HTMLInputElement).value);
		currentMs = Math.min(durationMs, Math.max(0, value));
		lastFrame = 0;
	}

	async function copyShare() {
		if (!shareURL) return;
		try {
			await navigator.clipboard.writeText(shareURL);
			copied = true;
			setTimeout(() => (copied = false), 1600);
		} catch {
			copied = false;
		}
	}

	function charClass(index: number): string {
		if (index < startChar) return 'text-(--silver)/15';
		const typed = display.typed[index];
		if (typed != null) {
			if (typed === chars[index]) {
				const kind = syntaxKinds[index] ?? 'plain';
				return kind === 'plain' || kind === 'ws' ? 'text-(--silver)' : `syn-${kind}`;
			}
			return 'bg-(--wrong-bg) text-(--red)';
		}
		return 'text-(--silver)/35';
	}

	/** Registers rendered lines so the viewport can follow the active one. */
	function trackLine(node: HTMLDivElement, index: number) {
		lineRefs[index] = node;
		return {
			destroy() {
				lineRefs[index] = undefined;
			}
		};
	}

	onMount(() => {
		const tick = (now: number) => {
			if (playing) {
				if (!lastFrame) lastFrame = now;
				const delta = Math.min(100, now - lastFrame);
				lastFrame = now;
				currentMs = Math.min(durationMs, currentMs + delta * speed);
				if (currentMs >= durationMs) playing = false;
			} else {
				lastFrame = 0;
			}
			raf = requestAnimationFrame(tick);
		};
		raf = requestAnimationFrame(tick);
		return () => cancelAnimationFrame(raf);
	});

	$effect(() => {
		const line = currentLine;
		const code = codeEl;
		const active = lineRefs[line];
		if (!code || !active) return;
		const lineHeight = Number.parseFloat(getComputedStyle(code).lineHeight) || 28;
		// Keep three lines above the active line and four below it. Clamping
		// at the start/end is intentional; the viewport cannot invent lines.
		code.scrollTop = Math.max(0, active.offsetTop - lineHeight * 3);
	});
</script>

<div class="flex flex-col gap-4">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<div>
			<div class="flex items-center gap-2 font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
				<span class={`inline-block h-1.5 w-1.5 ${playing ? 'bg-(--mer)' : 'bg-(--red)'}`}></span>
				simulated replay
			</div>
			<div class="mt-1 flex flex-wrap items-baseline gap-3">
				<span class="text-xl text-(--silver)">{run.username}</span>
				<span class="font-mono text-[10px] text-(--silver)/35">
					{Math.round(run.accuracy * 100)}% acc · {Math.round(run.wpm)} wpm
				</span>
				{#if run.is_bot}
					<span class="border border-(--border) px-1.5 py-0.5 font-mono text-[9px] text-(--silver)/35 uppercase">bot</span>
				{/if}
			</div>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#if shareURL}
				<button
					class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--silver)/50 uppercase hover:border-(--mer) hover:text-(--mer)"
					onclick={copyShare}>{copied ? 'link copied' : 'share link'}</button
				>
			{/if}
			{#if onclose}
				<button
					class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--silver)/50 uppercase hover:text-(--silver)"
					onclick={onclose}>close ✕</button
				>
			{/if}
		</div>
	</header>

	<div class="grid gap-3 lg:grid-cols-[1fr_300px]">
		<div class="flex min-h-0 flex-col border border-(--border) bg-(--fbg)">
			<div class="flex items-center justify-between border-b border-(--border) px-4 py-2 font-mono text-[10px] text-(--silver)/35">
				<span>typing simulation · {Math.round(currentMs / 1000)}s / {Math.round(durationSec)}s</span>
				<span class="text-(--mer)">{Math.round(stats.net)} wpm live</span>
			</div>
			<div class="px-4 py-3">
				<div
					bind:this={codeEl}
					class="replay-code relative overflow-hidden font-mono text-[15px] leading-7 md:text-base"
				>
					{#each lines as line, lineIndex (lineIndex)}
						<div
							class="code-line"
							class:code-line-active={lineIndex === currentLine}
							use:trackLine={lineIndex}
						>
							<span class="line-no">{lineIndex + 1}</span>
							<span class="line-code">
								{#each line as index (index)}
									<span class={charClass(index)} class:replay-caret={index === display.cursor}
										>{chars[index]}</span
									>
								{/each}
							</span>
						</div>
					{/each}
				</div>
			</div>
		</div>

		<div class="flex flex-col gap-3">
			<div class="border border-(--border) bg-(--fbg) p-4">
				<div class="grid grid-cols-3 gap-2">
					<div>
						<div class="font-mono text-2xl text-(--mer)">{Math.round(stats.net)}</div>
						<div class="mt-1 font-mono text-[9px] tracking-widest text-(--silver)/35 uppercase">wpm</div>
					</div>
					<div>
						<div class="font-mono text-2xl text-(--silver)">{Math.round(stats.accuracy * 100)}%</div>
						<div class="mt-1 font-mono text-[9px] tracking-widest text-(--silver)/35 uppercase">acc</div>
					</div>
					<div>
						<div class="font-mono text-2xl text-(--silver)">{Math.round((currentMs / durationMs) * 100)}%</div>
						<div class="mt-1 font-mono text-[9px] tracking-widest text-(--silver)/35 uppercase">through</div>
					</div>
				</div>
				<div class="mt-4 h-px w-full bg-(--border)">
					<div
						class="h-px bg-(--mer)"
						style={`width:${Math.min(100, (currentMs / durationMs) * 100)}%`}
					></div>
				</div>
			</div>

			<div class="border border-(--border) bg-(--fbg) p-3">
				<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
					live speed
				</div>
				<div class="mt-1">
					<SpeedGraph {series} duration={durationSec} showLegend={false} emptyLabel="waiting for keystrokes" />
				</div>
			</div>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-3 border border-(--border) bg-(--fbg) px-4 py-3">
		<button
			class="w-24 cursor-pointer border border-(--mer) px-3 py-1.5 font-mono text-[11px] text-(--mer) hover:bg-(--mer)/10"
			onclick={togglePlay}>{playing ? 'pause' : currentMs >= durationMs ? 'replay' : 'play'}</button
		>
		<button
			class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[11px] text-(--silver)/50 hover:text-(--silver)"
			onclick={restart}>restart</button
		>
		<button
			class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[11px] text-(--silver)/50 hover:text-(--mer)"
			onclick={cycleSpeed}>{speed}x</button
		>
		<input
			type="range"
			min="0"
			max={durationMs}
			step="50"
			value={currentMs}
			oninput={seek}
			class="h-1 min-w-[120px] flex-1 cursor-pointer accent-(--mer)"
			aria-label="replay position"
		/>
		<span class="font-mono text-[10px] text-(--silver)/35">{elapsedLabel} / {totalLabel}</span>
	</div>
</div>

<style>
	.replay-code {
		/* Eight code rows: three above the active line, four below. */
		height: calc(1.75rem * 8);
		scrollbar-width: none;
	}
	.replay-code :global(.code-line) {
		align-items: flex-start;
		min-height: 1.75rem;
	}
	.replay-code::-webkit-scrollbar {
		display: none;
	}
	.replay-code :global(.replay-caret) {
		background: color-mix(in srgb, var(--mer) 26%, transparent);
		box-shadow: inset 0 -2px 0 var(--mer);
	}
</style>
