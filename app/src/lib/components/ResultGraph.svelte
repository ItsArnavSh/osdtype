<script lang="ts">
	import type { WpmSample } from '$lib/core/stats';

	let { samples, duration }: { samples: WpmSample[]; duration: number } = $props();

	const W = 720;
	const H = 212;
	const padL = 42;
	const padR = 16;
	const padT = 18;
	const padB = 30;
	const plotW = W - padL - padR;
	const plotH = H - padT - padB;

	const maxWpm = $derived(
		Math.max(20, Math.ceil(Math.max(10, ...samples.map((s) => Math.max(s.raw, s.net))) / 20) * 20)
	);

	const x = $derived((t: number) => padL + (duration > 0 ? Math.min(1, t / duration) : 0) * plotW);
	const y = $derived((v: number) => padT + plotH - (Math.min(v, maxWpm) / maxWpm) * plotH);

	const netPoints = $derived(
		samples.map((s) => `${x(s.t).toFixed(1)},${y(s.net).toFixed(1)}`).join(' ')
	);
	const rawPoints = $derived(
		samples.map((s) => `${x(s.t).toFixed(1)},${y(s.raw).toFixed(1)}`).join(' ')
	);

	const errorMarks = $derived.by(() => {
		const marks: { x: number; y: number; n: number }[] = [];
		let prev = 0;
		for (const s of samples) {
			if (s.errors > prev) {
				marks.push({ x: x(s.t), y: y(s.net), n: s.errors - prev });
			}
			prev = s.errors;
		}
		return marks;
	});

	const ticks = $derived([0, maxWpm / 2, maxWpm]);
	const half = $derived(duration > 0 ? duration / 2 : 0);
</script>

<div class="result-graph">
	<svg viewBox="0 0 {W} {H}" role="img" aria-label="words per minute over time">
		<!-- horizontal grid -->
		{#each ticks as tick (tick)}
			<line
				x1={padL}
				x2={W - padR}
				y1={y(tick)}
				y2={y(tick)}
				stroke="var(--border)"
				stroke-width="1"
			/>
			<text x={padL - 8} y={y(tick) + 3} text-anchor="end" class="axis-label">
				{Math.round(tick)}
			</text>
		{/each}

		<!-- x axis -->
		<line
			x1={padL}
			x2={W - padR}
			y1={padT + plotH}
			y2={padT + plotH}
			stroke="var(--border)"
			stroke-width="1"
		/>
		<text x={padL} y={H - 10} text-anchor="start" class="axis-label">0s</text>
		<text x={padL + plotW / 2} y={H - 10} text-anchor="middle" class="axis-label">
			{Math.round(half)}s
		</text>
		<text x={W - padR} y={H - 10} text-anchor="end" class="axis-label">
			{Math.round(duration)}s
		</text>

		{#if samples.length > 1}
			<!-- raw wpm -->
			<polyline
				points={rawPoints}
				fill="none"
				stroke="color-mix(in srgb, var(--silver) 42%, transparent)"
				stroke-width="1.5"
				stroke-dasharray="4 3"
				stroke-linejoin="round"
			/>
			<!-- net wpm -->
			<polyline
				points={netPoints}
				fill="none"
				stroke="var(--mer)"
				stroke-width="2"
				stroke-linejoin="round"
				stroke-linecap="round"
			/>
		{/if}

		<!-- mistakes -->
		{#each errorMarks as mark, i (i)}
			<circle cx={mark.x} cy={mark.y} r="3" fill="var(--red)" opacity="0.85" />
		{/each}

		{#if samples.length <= 1}
			<text x={padL + plotW / 2} y={padT + plotH / 2} text-anchor="middle" class="empty-label">
				not enough data
			</text>
		{/if}
	</svg>

	<div class="legend">
		<span class="legend-item"><span class="swatch net"></span>actual wpm</span>
		<span class="legend-item"><span class="swatch raw"></span>raw wpm</span>
		<span class="legend-item"><span class="swatch err"></span>mistakes</span>
	</div>
</div>

<style>
	.result-graph {
		width: 100%;
	}
	svg {
		display: block;
		width: 100%;
		height: auto;
	}
	.axis-label {
		fill: color-mix(in srgb, var(--silver) 38%, transparent);
		font-family: 'Fira Code', monospace;
		font-size: 10px;
	}
	.empty-label {
		fill: color-mix(in srgb, var(--silver) 30%, transparent);
		font-family: 'Fira Code', monospace;
		font-size: 11px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1.25rem;
		justify-content: center;
		margin-top: 0.25rem;
		font-family: 'Fira Code', monospace;
		font-size: 10px;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: color-mix(in srgb, var(--silver) 45%, transparent);
	}
	.legend-item {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}
	.swatch {
		display: inline-block;
		width: 14px;
		height: 2px;
	}
	.swatch.net {
		background: var(--mer);
	}
	.swatch.raw {
		background: color-mix(in srgb, var(--silver) 42%, transparent);
	}
	.swatch.err {
		width: 6px;
		height: 6px;
		border-radius: 999px;
		background: var(--red);
	}
	@media (max-width: 767px) {
		.axis-label {
			font-size: 18px;
		}
		.empty-label {
			font-size: 18px;
		}
	}
</style>
