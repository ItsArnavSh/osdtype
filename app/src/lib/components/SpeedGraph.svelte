<script lang="ts">
	import type { SpeedSeries } from '$lib/core/stats';

	let {
		series,
		duration,
		showLegend = true,
		emptyLabel = 'not enough data'
	}: {
		series: SpeedSeries[];
		duration: number;
		showLegend?: boolean;
		emptyLabel?: string;
	} = $props();

	const W = 760;
	const H = 220;
	const padL = 42;
	const padR = 18;
	const padT = 18;
	const padB = 30;
	const plotW = W - padL - padR;
	const plotH = H - padT - padB;

	const palette = [
		'var(--mer)',
		'var(--syntax-keyword)',
		'var(--syntax-literal)',
		'var(--syntax-meta)',
		'var(--red)',
		'var(--syntax-type)'
	];

	const visible = $derived(series.filter((item) => item.samples.length > 0));
	const maxWpm = $derived(
		Math.max(
			20,
			Math.ceil(
				Math.max(10, ...visible.flatMap((item) => item.samples.map((sample) => sample.net))) / 20
			) * 20
		)
	);
	const x = $derived((t: number) => padL + (duration > 0 ? Math.min(1, t / duration) : 0) * plotW);
	const y = $derived((v: number) => padT + plotH - (Math.min(v, maxWpm) / maxWpm) * plotH);
	const ticks = $derived([0, maxWpm / 2, maxWpm]);
	const half = $derived(duration > 0 ? duration / 2 : 0);

	function colorFor(item: SpeedSeries, index: number): string {
		return item.color ?? palette[index % palette.length];
	}

	function pointsFor(item: SpeedSeries): string {
		return item.samples
			.map((sample) => `${x(sample.t).toFixed(1)},${y(sample.net).toFixed(1)}`)
			.join(' ');
	}

	function finalWpm(item: SpeedSeries): number {
		const last = item.samples[item.samples.length - 1];
		return last ? Math.round(last.net) : 0;
	}
</script>

<div class="speed-graph">
	<svg viewBox="0 0 {W} {H}" role="img" aria-label="words per minute over time">
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

		{#each visible as item, index (item.label)}
			<polyline
				points={pointsFor(item)}
				fill="none"
				stroke={colorFor(item, index)}
				stroke-width={item.isBot ? 1.8 : 2.2}
				stroke-dasharray={item.isBot ? '5 3' : undefined}
				stroke-linejoin="round"
				stroke-linecap="round"
				opacity={item.isBot ? '0.82' : '1'}
			/>
			{#if item.samples.length > 1}
				{@const last = item.samples[item.samples.length - 1]}
				<circle cx={x(last.t)} cy={y(last.net)} r="2.6" fill={colorFor(item, index)} />
			{/if}
		{/each}

		{#if visible.length === 0}
			<text x={padL + plotW / 2} y={padT + plotH / 2} text-anchor="middle" class="empty-label">
				{emptyLabel}
			</text>
		{/if}
	</svg>

	{#if showLegend && visible.length > 0}
		<div class="legend">
			{#each visible as item, index (item.label)}
				<span class="legend-item">
					<span
						class="swatch"
						class:dashed={item.isBot}
						style={`background:${colorFor(item, index)}`}
					></span>
					<span>{item.label}</span>
					<span class="legend-wpm">{finalWpm(item)}</span>
				</span>
			{/each}
		</div>
	{/if}
</div>

<style>
	.speed-graph {
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
		gap: 0.55rem 1.15rem;
		justify-content: center;
		margin-top: 0.25rem;
		font-family: 'Fira Code', monospace;
		font-size: 10px;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: color-mix(in srgb, var(--silver) 48%, transparent);
	}
	.legend-item {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
	}
	.legend-wpm {
		color: var(--mer);
	}
	.swatch {
		display: inline-block;
		width: 14px;
		height: 2px;
	}
	.swatch.dashed {
		opacity: 0.75;
		background-image: repeating-linear-gradient(90deg, currentColor 0 4px, transparent 4px 7px);
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
