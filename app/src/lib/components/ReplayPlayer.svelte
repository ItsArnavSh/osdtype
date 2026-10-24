<script lang="ts">
	import SpeedGraph from '$lib/components/SpeedGraph.svelte';
	import { KeypressAction, type ClientKeypress, type ReplayRun } from '$lib/core/api/types';
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
	const events = $derived([...run.events].sort((a, b) => (a.time_ms ?? 0) - (b.time_ms ?? 0)));
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

	type KeyCap = {
		id: string;
		label: string;
		shiftLabel?: string;
		weight?: number;
		mod?: boolean;
	};

	// A compact keyboard, not a photo-real one. Everything here is a key the
	// browser can record; media/volume/browser keys are intentionally absent.
	const KEY_ROWS: KeyCap[][] = [
		[
			{ id: '`', label: '`', shiftLabel: '~' },
			{ id: '1', label: '1', shiftLabel: '!' },
			{ id: '2', label: '2', shiftLabel: '@' },
			{ id: '3', label: '3', shiftLabel: '#' },
			{ id: '4', label: '4', shiftLabel: '$' },
			{ id: '5', label: '5', shiftLabel: '%' },
			{ id: '6', label: '6', shiftLabel: '^' },
			{ id: '7', label: '7', shiftLabel: '&' },
			{ id: '8', label: '8', shiftLabel: '*' },
			{ id: '9', label: '9', shiftLabel: '(' },
			{ id: '0', label: '0', shiftLabel: ')' },
			{ id: '-', label: '-', shiftLabel: '_' },
			{ id: '=', label: '=', shiftLabel: '+' },
			{ id: 'Backspace', label: '⌫', weight: 2, mod: true }
		],
		[
			{ id: 'Tab', label: 'tab', weight: 1.4, mod: true },
			{ id: 'Q', label: 'Q' },
			{ id: 'W', label: 'W' },
			{ id: 'E', label: 'E' },
			{ id: 'R', label: 'R' },
			{ id: 'T', label: 'T' },
			{ id: 'Y', label: 'Y' },
			{ id: 'U', label: 'U' },
			{ id: 'I', label: 'I' },
			{ id: 'O', label: 'O' },
			{ id: 'P', label: 'P' },
			{ id: '[', label: '[', shiftLabel: '{' },
			{ id: ']', label: ']', shiftLabel: '}' },
			{ id: '\\', label: '\\', shiftLabel: '|', weight: 1.2 }
		],
		[
			{ id: 'CapsLock', label: 'caps', weight: 1.7, mod: true },
			{ id: 'A', label: 'A' },
			{ id: 'S', label: 'S' },
			{ id: 'D', label: 'D' },
			{ id: 'F', label: 'F' },
			{ id: 'G', label: 'G' },
			{ id: 'H', label: 'H' },
			{ id: 'J', label: 'J' },
			{ id: 'K', label: 'K' },
			{ id: 'L', label: 'L' },
			{ id: ';', label: ';', shiftLabel: ':' },
			{ id: "'", label: "'", shiftLabel: '"' },
			{ id: 'Enter', label: 'enter', weight: 2.2, mod: true }
		],
		[
			{ id: 'ShiftLeft', label: 'shift', weight: 2.3, mod: true },
			{ id: 'Z', label: 'Z' },
			{ id: 'X', label: 'X' },
			{ id: 'C', label: 'C' },
			{ id: 'V', label: 'V' },
			{ id: 'B', label: 'B' },
			{ id: 'N', label: 'N' },
			{ id: 'M', label: 'M' },
			{ id: ',', label: ',', shiftLabel: '<' },
			{ id: '.', label: '.', shiftLabel: '>' },
			{ id: '/', label: '/', shiftLabel: '?' },
			{ id: 'ShiftRight', label: 'shift', weight: 2.3, mod: true }
		],
		[{ id: 'Space', label: 'space', weight: 10, mod: true }]
	];

	const SHIFTED_CHARS: Record<string, string> = {
		'!': '1',
		'@': '2',
		'#': '3',
		$: '4',
		'%': '5',
		'^': '6',
		'&': '7',
		'*': '8',
		'(': '9',
		')': '0',
		_: '-',
		'+': '=',
		'{': '[',
		'}': ']',
		'|': '\\',
		':': ';',
		'"': "'",
		'<': ',',
		'>': '.',
		'?': '/',
		'~': '`'
	};
	const DIRECT_KEYS = new Set(Array.from("`1234567890-=[]\\;',./"));
	const KEY_FLASH_MS = 180;

	function keyForEvent(event: ClientKeypress): { id: string; shift: boolean } | null {
		if (event.action === KeypressAction.BACKSPACE) {
			return { id: 'Backspace', shift: false };
		}
		const value = event.value ?? '';
		const char = Array.from(value)[0];
		if (!char || char === '\n' || char === '\r' || char === '\t') return null;
		if (char === ' ') return { id: 'Space', shift: false };
		const shifted = SHIFTED_CHARS[char];
		if (shifted) return { id: shifted, shift: true };
		if (/[A-Z]/.test(char)) return { id: char, shift: true };
		if (/[a-z]/.test(char)) return { id: char.toUpperCase(), shift: false };
		if (/[0-9]/.test(char) || DIRECT_KEYS.has(char)) return { id: char, shift: false };
		return null;
	}

	const activeKeys = $derived.by(() => {
		const pressed = new Map<string, 'primary' | 'secondary'>();
		const cutoff = currentMs - KEY_FLASH_MS;
		let last: { id: string; shift: boolean } | null = null;
		for (const event of events) {
			const at = event.time_ms ?? 0;
			if (at > currentMs) break;
			if (at < cutoff) continue;
			const key = keyForEvent(event);
			if (!key) continue;
			pressed.set(key.id, 'secondary');
			last = key;
		}
		if (last) {
			pressed.set(last.id, 'primary');
			if (last.shift) pressed.set('ShiftLeft', 'secondary');
		}
		return pressed;
	});

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
			<div
				class="flex items-center gap-2 font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase"
			>
				<span class={`inline-block h-1.5 w-1.5 ${playing ? 'bg-(--mer)' : 'bg-(--red)'}`}></span>
				simulated replay
			</div>
			<div class="mt-1 flex flex-wrap items-baseline gap-3">
				<span class="text-xl text-(--silver)">{run.username}</span>
				<span class="font-mono text-[10px] text-(--silver)/35">
					{Math.round(run.accuracy * 100)}% acc · {Math.round(run.wpm)} wpm
				</span>
				{#if run.is_bot}
					<span
						class="border border-(--border) px-1.5 py-0.5 font-mono text-[9px] text-(--silver)/35 uppercase"
						>bot</span
					>
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
			<div
				class="flex items-center justify-between border-b border-(--border) px-4 py-2 font-mono text-[10px] text-(--silver)/35"
			>
				<span>typing simulation · {Math.round(currentMs / 1000)}s / {Math.round(durationSec)}s</span
				>
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
						<div class="mt-1 font-mono text-[9px] tracking-widest text-(--silver)/35 uppercase">
							wpm
						</div>
					</div>
					<div>
						<div class="font-mono text-2xl text-(--silver)">
							{Math.round(stats.accuracy * 100)}%
						</div>
						<div class="mt-1 font-mono text-[9px] tracking-widest text-(--silver)/35 uppercase">
							acc
						</div>
					</div>
					<div>
						<div class="font-mono text-2xl text-(--silver)">
							{Math.round((currentMs / durationMs) * 100)}%
						</div>
						<div class="mt-1 font-mono text-[9px] tracking-widest text-(--silver)/35 uppercase">
							through
						</div>
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
					<SpeedGraph
						{series}
						duration={durationSec}
						showLegend={false}
						emptyLabel="waiting for keystrokes"
					/>
				</div>
			</div>
		</div>
	</div>

	<div class="border border-(--border) bg-(--fbg) p-3 md:p-4">
		<div class="flex flex-wrap items-center justify-between gap-2">
			<div>
				<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/70 uppercase">
					keyboard
				</div>
				<div class="mt-0.5 font-mono text-[10px] text-(--silver)/25">
					keys pressed at the replay cursor
				</div>
			</div>
			<div class="flex items-center gap-3 font-mono text-[10px] text-(--silver)/30">
				<span>{run.events.length} keystrokes</span>
				<span class="hidden sm:inline">backspace flashes ⌫</span>
			</div>
		</div>

		<div class="replay-keyboard-scroll mt-3 overflow-x-auto pb-1">
			<div
				class="replay-keyboard mx-auto flex w-full max-w-3xl min-w-[620px] flex-col gap-1.5 max-md:min-w-0 max-md:gap-1"
			>
				{#each KEY_ROWS as row}
					<div class="flex gap-1.5 max-md:gap-1">
						{#each row as key}
							{@const state = activeKeys.get(key.id)}
							<div
								class={`replay-key${key.mod ? ' is-mod' : ''}${state === 'primary' ? ' is-primary' : state === 'secondary' ? ' is-secondary' : ''}`}
								style={`flex:${key.weight ?? 1} 1 0%`}
								title={key.shiftLabel ? `${key.label} / ${key.shiftLabel}` : key.label}
							>
								{#if key.shiftLabel}
									<span class="replay-key-alt">{key.shiftLabel}</span>
								{/if}
								<span>{key.label}</span>
							</div>
						{/each}
					</div>
				{/each}
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

	.replay-key {
		position: relative;
		display: flex;
		align-items: flex-end;
		justify-content: center;
		min-height: 2.15rem;
		border: 1px solid var(--border);
		background: color-mix(in srgb, var(--bg) 82%, var(--fbg));
		padding: 0.25rem 0.35rem;
		font-family: 'Fira Code', monospace;
		font-size: 10px;
		line-height: 1;
		color: color-mix(in srgb, var(--silver) 48%, transparent);
		transition:
			background 0.08s ease,
			border-color 0.08s ease,
			color 0.08s ease,
			transform 0.08s ease;
	}

	.replay-key.is-mod {
		font-size: 9px;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: color-mix(in srgb, var(--silver) 30%, transparent);
	}

	.replay-key-alt {
		position: absolute;
		top: 0.22rem;
		left: 0.35rem;
		font-size: 8px;
		color: color-mix(in srgb, var(--silver) 22%, transparent);
	}

	.replay-key.is-secondary {
		border-color: color-mix(in srgb, var(--mer) 34%, var(--border));
		background: color-mix(in srgb, var(--mer) 7%, transparent);
		color: color-mix(in srgb, var(--silver) 68%, transparent);
	}

	.replay-key.is-primary {
		border-color: var(--mer);
		background: color-mix(in srgb, var(--mer) 22%, transparent);
		color: var(--mer);
		transform: translateY(1px);
	}

	.replay-key.is-primary .replay-key-alt {
		color: color-mix(in srgb, var(--mer) 70%, transparent);
	}

	@media (max-width: 767px) {
		.replay-key {
			min-height: 1.8rem;
			padding: 0.2rem 0.1rem;
			font-size: 8px;
		}
		.replay-key.is-mod {
			font-size: 7px;
			letter-spacing: 0;
		}
		.replay-key-alt {
			top: 0.12rem;
			left: 0.15rem;
			font-size: 6px;
		}
	}
</style>
