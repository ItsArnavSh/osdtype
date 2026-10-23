<script lang="ts">
	import type { ContestInput, ContestStageInput } from '$lib/core/api/contests';
	import type {
		ContestStageFormat,
		ContestView,
		ContestVisibility,
		LobbyDuration
	} from '$lib/core/api/types';
	import { LOBBY_DURATIONS } from '$lib/core/api/types';
	import { LANGUAGES, LANGUAGE_LABELS, type Language } from '$lib/core/entity/languages';
	import { renderMarkdown } from '$lib/core/markdown';

	let {
		contest = null,
		submitLabel = 'publish contest',
		busy = false,
		error = '',
		onsubmit
	}: {
		contest?: ContestView | null;
		submitLabel?: string;
		busy?: boolean;
		error?: string;
		onsubmit: (input: ContestInput) => void | Promise<void>;
	} = $props();

	type StageDraft = ContestStageInput;

	const STAGE_FORMATS: {
		value: ContestStageFormat;
		label: string;
		short: string;
		description: string;
	}[] = [
		{
			value: 'qualifier',
			label: 'qualifier',
			short: 'qualifier',
			description: 'A WPM cutoff, host-fixed or player-chosen language, and an optional try limit.'
		},
		{
			value: 'hotlaps',
			label: 'HotLaps',
			short: 'hotlaps',
			description: 'One shared snippet. Unlimited tries inside the stage window. Fastest run advances.'
		},
		{
			value: 'multiplayer',
			label: 'live multiplayer',
			short: 'live race',
			description: 'A scheduled live race. Larger fields are split into balanced heats of up to 10.'
		},
		{
			value: 'ranked',
			label: 'ranked-style final',
			short: 'ranked final',
			description: 'The final behaves like a ranked match, with the same clock and live race rules.'
		}
	];

	const CSS_GUIDE = `/* Everything is scoped to the event page. */
.contest-hero {
  border-color: var(--mer);
}

.contest-title {
  letter-spacing: 0.18em;
}

.contest-panel {
  background: #0b0d0f;
}

.contest-stage.is-active {
  border-color: var(--mer);
}

.contest-badge {
  letter-spacing: 0.14em;
}

.contest-cta {
  background: var(--mer);
  color: var(--bg);
}

.contest-standings-row:nth-child(2) {
  background: rgba(255, 255, 255, 0.04);
}`;

	function localDateTime(date: Date): string {
		const pad = (value: number) => String(value).padStart(2, '0');
		return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
	}

	function isOpen(format: ContestStageFormat): boolean {
		return format === 'qualifier' || format === 'hotlaps';
	}

	function stageDefaults(index: number, format: ContestStageFormat = 'multiplayer'): StageDraft {
		const open = isOpen(format);
		return {
			name: '',
			format,
			delay_minutes: index === 0 ? 0 : 10,
			window_minutes: open ? 60 : 0,
			duration: 90,
			language_mode: 'fixed',
			language: 'go',
			max_tries: format === 'qualifier' ? 3 : 0,
			min_wpm: format === 'qualifier' ? 80 : 0,
			advance_type: 'all',
			advance_value: 10,
			max_players: 8
		};
	}

	function blankForm() {
		return {
			title: '',
			description: '',
			banner_url: '',
			accent: '#00a19c',
			custom_css: '',
			visibility: 'public' as ContestVisibility,
			password: '',
			starts_at: localDateTime(new Date(Date.now() + 15 * 60_000)),
			max_players: 16,
			stages: [stageDefaults(0, 'multiplayer')] as StageDraft[]
		};
	}

	function formFrom(source: ContestView) {
		const stages =
			source.stages && source.stages.length > 0
				? source.stages.map((stage, index) => ({
						name: stage.name,
						format: stage.format,
						delay_minutes: index === 0 ? 0 : stage.delay_minutes,
						window_minutes: stage.window_minutes,
						duration: stage.duration as LobbyDuration,
						language_mode: stage.language_mode,
						language: stage.language,
						max_tries: stage.max_tries,
						min_wpm: stage.min_wpm,
						advance_type: stage.advance_type,
						advance_value: stage.advance_value || 10,
						max_players: stage.max_players
					}))
				: [stageDefaults(0, 'multiplayer')];
		return {
			title: source.title,
			description: source.description,
			banner_url: source.banner_url,
			accent: source.accent,
			custom_css: source.custom_css,
			visibility: source.visibility,
			password: '',
			starts_at: localDateTime(new Date(source.starts_at)),
			max_players: source.max_players,
			stages
		};
	}

	let form = $state(blankForm());
	let seeded = $state(false);
	let formError = $state('');
	let preview = $state(false);
	let cssGuide = $state(false);

	$effect(() => {
		if (seeded || !contest) return;
		form = formFrom(contest);
		seeded = true;
	});

	let renderedDescription = $derived(renderMarkdown(form.description));

	function setFormat(stage: StageDraft, format: ContestStageFormat) {
		stage.format = format;
		if (isOpen(format)) {
			if (stage.window_minutes <= 0) stage.window_minutes = 60;
			if (format === 'hotlaps') {
				stage.max_tries = 0;
				stage.language_mode = 'fixed';
				stage.min_wpm = 0;
			}
			if (format === 'qualifier') {
				stage.min_wpm = stage.min_wpm || 80;
				if (stage.max_tries < 0) stage.max_tries = 0;
			}
		} else {
			stage.window_minutes = 0;
			stage.language_mode = 'fixed';
			stage.min_wpm = 0;
			stage.max_tries = 0;
			stage.max_players = Math.min(10, Math.max(2, Number(stage.max_players) || 8));
		}
	}

	function addStage(format: ContestStageFormat) {
		if (form.stages.length >= 10) return;
		const index = form.stages.length;
		const stage = stageDefaults(index, format);
		stage.delay_minutes = 10;
		stage.language = form.stages[0]?.language ?? 'go';
		stage.duration = form.stages[0]?.duration ?? 90;
		form.stages.push(stage);
	}

	function removeStage(index: number) {
		if (form.stages.length <= 1) return;
		form.stages.splice(index, 1);
		if (form.stages[0]) form.stages[0].delay_minutes = 0;
	}

	function moveStage(index: number, delta: number) {
		const target = index + delta;
		if (target < 0 || target >= form.stages.length) return;
		const [stage] = form.stages.splice(index, 1);
		form.stages.splice(target, 0, stage);
		form.stages.forEach((item, i) => {
			if (i === 0) item.delay_minutes = 0;
			else if (item.delay_minutes <= 0) item.delay_minutes = 10;
		});
	}

	function openWindowSeconds(stage: StageDraft): number {
		return isOpen(stage.format) ? Number(stage.window_minutes) * 60 : Number(stage.duration);
	}

	function projectedStart(index: number): Date {
		let cursor = new Date(form.starts_at).getTime();
		for (let i = 0; i < index; i++) {
			const previous = form.stages[i];
			cursor += openWindowSeconds(previous) * 1000;
			if (!isOpen(previous.format)) cursor += 15_000;
			cursor += Math.max(0, Number(form.stages[i + 1]?.delay_minutes) || 0) * 60_000;
		}
		return new Date(cursor);
	}

	function projectedLabel(index: number): string {
		const at = projectedStart(index);
		if (Number.isNaN(at.getTime())) return 'after start';
		return at.toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function formatLabel(format: ContestStageFormat): string {
		return STAGE_FORMATS.find((item) => item.value === format)?.short ?? format;
	}

	function clamp(value: number, low: number, high: number): number {
		return Math.min(high, Math.max(low, Number(value) || low));
	}

	function submit(event: SubmitEvent) {
		event.preventDefault();
		formError = '';
		const starts = new Date(form.starts_at);
		if (!form.title.trim()) {
			formError = 'Give the contest a title.';
			return;
		}
		if (Number.isNaN(starts.getTime())) {
			formError = 'Choose a valid start date and time.';
			return;
		}
		if (starts.getTime() < Date.now() + 30_000) {
			formError = 'The start time must be at least 30 seconds from now.';
			return;
		}
		if (form.visibility === 'public' && form.password) {
			formError = 'Public contests cannot have a password.';
			return;
		}
		if (form.stages.length === 0) {
			formError = 'Add at least one stage to the funnel.';
			return;
		}
		for (let i = 0; i < form.stages.length; i++) {
			const stage = form.stages[i];
			if (isOpen(stage.format)) {
				if (Number(stage.window_minutes) * 60 < Number(stage.duration) + 15) {
					formError = `Stage ${i + 1} needs a longer window to fit a ${stage.duration}s run.`;
					return;
				}
			}
			if (!isOpen(stage.format) && (Number(stage.max_players) < 2 || Number(stage.max_players) > 10)) {
				formError = `Stage ${i + 1} heats must hold between 2 and 10 players.`;
				return;
			}
			if (stage.advance_type !== 'all' && Number(stage.advance_value) < 1) {
				formError = `Stage ${i + 1} needs a positive advancement target.`;
				return;
			}
		}
		const stages = form.stages.map((stage, index) => ({
			name: stage.name.trim(),
			format: stage.format,
			delay_minutes: index === 0 ? 0 : clamp(Number(stage.delay_minutes), 0, 10080),
			window_minutes: isOpen(stage.format) ? clamp(Number(stage.window_minutes), 1, 10080) : 0,
			duration: Number(stage.duration) as LobbyDuration,
			language_mode: stage.format === 'qualifier' ? stage.language_mode : 'fixed',
			language: stage.language,
			max_tries: stage.format === 'qualifier' ? clamp(Number(stage.max_tries), 0, 1000) : 0,
			min_wpm: stage.format === 'qualifier' ? clamp(Number(stage.min_wpm), 1, 300) : 0,
			advance_type: stage.advance_type,
			advance_value: stage.advance_type === 'all' ? 0 : clamp(Number(stage.advance_value), 1, 512),
			max_players: isOpen(stage.format)
				? 2
				: clamp(Number(stage.max_players), 2, 10)
		}));
		void onsubmit({
			title: form.title.trim(),
			description: form.description.trim(),
			banner_url: form.banner_url.trim(),
			accent: form.accent,
			custom_css: form.custom_css,
			visibility: form.visibility,
			password: form.visibility === 'private' && form.password ? form.password : undefined,
			starts_at: starts.toISOString(),
			duration: stages[0].duration,
			language: stages[0].language,
			max_players: clamp(Number(form.max_players), 2, 512),
			stages
		});
	}
</script>

<form class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_340px]" onsubmit={submit}>
	<div class="flex flex-col gap-3">
		<div class="border border-(--border) bg-(--fbg) p-5">
			<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
				event identity
			</div>
			<div class="mt-3 flex flex-col gap-3">
				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase">title</span>
					<input
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.title}
						maxlength="120"
						placeholder="Friday night Go sprint"
					/>
				</label>
				<label class="flex flex-col gap-1.5">
					<span class="flex items-center justify-between font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase">
						<span>description · markdown</span>
						<button
							type="button"
							class="cursor-pointer font-mono text-[9px] tracking-[0.1em] text-(--mer)/70 normal-case hover:text-(--mer)"
							onclick={() => (preview = !preview)}>{preview ? 'edit' : 'preview'}</button
						>
					</span>
					{#if preview}
						<div class="contest-md min-h-40 border border-(--border) bg-(--bg) px-4 py-3 text-sm text-(--silver)/65">
							{@html renderedDescription || '<p class="text-(--silver)/25">Nothing to preview yet.</p>'}
						</div>
					{:else}
						<textarea
							class="min-h-40 resize-y border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
							bind:value={form.description}
							maxlength="20000"
							placeholder="What is this contest? Rules, format, a short welcome…"
						></textarea>
					{/if}
				</label>
			</div>
		</div>

		<div class="border border-(--border) bg-(--fbg) p-5">
			<div class="flex items-start justify-between gap-3">
				<div>
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
						competition funnel
					</div>
					<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/30">
						Every stage advances players into the next. The last stage produces the final standings.
					</p>
				</div>
				<span class="shrink-0 border border-(--border) px-2 py-1 font-mono text-[9px] text-(--silver)/35">
					{form.stages.length} {form.stages.length === 1 ? 'stage' : 'stages'}
				</span>
			</div>

			<div class="mt-4 flex flex-col gap-3">
				{#each form.stages as stage, index (stage)}
					<article
						class={`border bg-(--bg)/60 ${stage.format === 'ranked' ? 'border-(--mer)/40' : 'border-(--border)'}`}
					>
						<header class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-3 py-2">
							<div class="flex items-center gap-2">
								<span class="font-mono text-[10px] text-(--mer)/70"
									>{String(index + 1).padStart(2, '0')}</span
								>
								<span class="font-mono text-[10px] tracking-[0.1em] text-(--silver)/55 uppercase"
									>{formatLabel(stage.format)}</span
								>
								<span class="font-mono text-[9px] text-(--silver)/25">
									{index === 0 ? 'starts with the event' : `+${stage.delay_minutes || 0}m after previous`}
								</span>
							</div>
							<div class="flex items-center gap-1">
								<button
									type="button"
									class="cursor-pointer px-1.5 py-0.5 font-mono text-[10px] text-(--silver)/30 hover:text-(--silver) disabled:opacity-20"
									disabled={index === 0}
									onclick={() => moveStage(index, -1)}
									aria-label="move stage up">↑</button
								>
								<button
									type="button"
									class="cursor-pointer px-1.5 py-0.5 font-mono text-[10px] text-(--silver)/30 hover:text-(--silver) disabled:opacity-20"
									disabled={index === form.stages.length - 1}
									onclick={() => moveStage(index, 1)}
									aria-label="move stage down">↓</button
								>
								<button
									type="button"
									class="cursor-pointer px-1.5 py-0.5 font-mono text-[10px] text-(--red)/50 hover:text-(--red) disabled:opacity-20"
									disabled={form.stages.length === 1}
									onclick={() => removeStage(index)}
									aria-label="remove stage">×</button
								>
							</div>
						</header>

						<div class="grid gap-3 p-3 sm:grid-cols-2">
							<label class="flex flex-col gap-1">
								<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase">stage name</span>
								<input
									class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
									bind:value={stage.name}
									maxlength="80"
									placeholder={formatLabel(stage.format) + ' ' + (index + 1)}
								/>
							</label>
							<label class="flex flex-col gap-1">
								<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase">game type</span>
								<select
									class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
									value={stage.format}
									onchange={(event) =>
										setFormat(stage, (event.currentTarget as HTMLSelectElement).value as ContestStageFormat)}
								>
									{#each STAGE_FORMATS as format (format.value)}
										<option value={format.value}>{format.label}</option>
									{/each}
								</select>
							</label>
						</div>
						<p class="-mt-1 px-3 pb-2 font-mono text-[9px] leading-relaxed text-(--silver)/25">
							{STAGE_FORMATS.find((item) => item.value === stage.format)?.description}
						</p>

						<div class="grid gap-3 border-t border-(--border)/60 p-3 sm:grid-cols-3">
							<label class="flex flex-col gap-1">
								<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase">run clock</span>
								<select
									class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
									bind:value={stage.duration}
								>
									{#each LOBBY_DURATIONS as duration (duration)}
										<option value={duration}>{duration}s</option>
									{/each}
								</select>
							</label>
							{#if isOpen(stage.format)}
								<label class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>stage window · minutes</span
									>
									<input
										type="number"
										min="1"
										max="10080"
										class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
										bind:value={stage.window_minutes}
									/>
								</label>
								<label class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>tries · 0 = unlimited</span
									>
									{#if stage.format === 'hotlaps'}
										<div
											class="border border-(--border)/60 px-2.5 py-1.5 font-mono text-xs text-(--silver)/30"
										>
											unlimited
										</div>
									{:else}
										<input
											type="number"
											min="0"
											max="1000"
											class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
											bind:value={stage.max_tries}
										/>
									{/if}
								</label>
							{:else}
								<label class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>heat size · max 10</span
									>
									<input
										type="number"
										min="2"
										max="10"
										class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
										bind:value={stage.max_players}
									/>
								</label>
								<div class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase">start</span>
									<div class="border border-(--border)/60 px-2.5 py-1.5 font-mono text-[10px] text-(--silver)/35">
										scheduled live
									</div>
								</div>
							{/if}
						</div>

						<div class="grid gap-3 border-t border-(--border)/60 p-3 sm:grid-cols-3">
							<label class="flex flex-col gap-1">
								<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase">language</span>
								<select
									class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
									bind:value={stage.language}
								>
									{#each LANGUAGES as language (language)}
										<option value={language}>{LANGUAGE_LABELS[language]}</option>
									{/each}
								</select>
							</label>
							{#if stage.format === 'qualifier'}
								<label class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>language choice</span
									>
									<select
										class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
										bind:value={stage.language_mode}
									>
										<option value="fixed">host-fixed</option>
										<option value="choice">player picks</option>
									</select>
								</label>
								<label class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>minimum WPM</span
									>
									<input
										type="number"
										min="1"
										max="300"
										class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
										bind:value={stage.min_wpm}
									/>
								</label>
							{:else if stage.format === 'hotlaps'}
								<div class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>snippet</span
									>
									<div
										class="border border-(--border)/60 px-2.5 py-1.5 font-mono text-[10px] text-(--silver)/35"
									>
										shared seed · everyone races the same text
									</div>
								</div>
							{:else}
								<div class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>opponent cursors</span
									>
									<div
										class="border border-(--border)/60 px-2.5 py-1.5 font-mono text-[10px] text-(--silver)/35"
									>
										visible but blurred
									</div>
								</div>
							{/if}
							<div class="flex flex-col gap-1">
								<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
									>projected start</span
								>
								<div
									class="border border-(--border)/60 px-2.5 py-1.5 font-mono text-[10px] text-(--silver)/45"
								>
									{projectedLabel(index)}
								</div>
							</div>
						</div>

						<div class="grid gap-3 border-t border-(--border)/60 p-3 sm:grid-cols-[1fr_120px_120px]">
							<label class="flex flex-col gap-1">
								<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
									>{index === form.stages.length - 1 ? 'final ranking' : 'players advancing'}</span
								>
								<select
									class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
									bind:value={stage.advance_type}
									disabled={index === form.stages.length - 1}
								>
									<option value="all">{stage.format === 'qualifier' ? 'everyone above the cut' : 'all finishers'}</option>
									<option value="count">fixed number</option>
									<option value="percent">percentage</option>
								</select>
							</label>
							{#if stage.advance_type !== 'all' && index !== form.stages.length - 1}
								<label class="flex flex-col gap-1">
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
										>{stage.advance_type === 'percent' ? 'percent' : 'count'}</span
									>
									<input
										type="number"
										min="1"
										max={stage.advance_type === 'percent' ? 100 : 512}
										class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
										bind:value={stage.advance_value}
									/>
								</label>
							{:else}
								<div></div>
							{/if}
							<label class="flex flex-col gap-1">
								<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
									>{index === 0 ? 'starts with event' : 'delay after prev · min'}</span
								>
								<input
									type="number"
									min="0"
									max="10080"
									class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
									bind:value={stage.delay_minutes}
									disabled={index === 0}
								/>
							</label>
						</div>
					</article>
				{/each}
			</div>

			{#if form.stages.length < 10}
				<div class="mt-3 flex flex-wrap items-center gap-2 border-t border-(--border) pt-3">
					<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/25 uppercase">add stage</span>
					{#each STAGE_FORMATS as format (format.value)}
						<button
							type="button"
							class="cursor-pointer border border-(--border) px-2.5 py-1 font-mono text-[9px] tracking-[0.08em] text-(--silver)/45 uppercase hover:border-(--mer)/50 hover:text-(--mer)"
							onclick={() => addStage(format.value)}>+ {format.short}</button
						>
					{/each}
				</div>
			{/if}
		</div>

		<div class="border border-(--border) bg-(--fbg) p-5">
			<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
				theme & presentation
			</div>
			<div class="mt-3 grid gap-3 sm:grid-cols-[1fr_120px]">
				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
						>banner image URL</span
					>
					<input
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.banner_url}
						placeholder="https://…/banner.jpg"
					/>
				</label>
				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase">accent</span>
					<span class="flex h-[38px] items-center gap-2 border border-(--border) bg-(--bg) px-2">
						<input
							type="color"
							class="h-6 w-8 cursor-pointer border-0 bg-transparent p-0"
							bind:value={form.accent}
						/>
						<input
							class="min-w-0 flex-1 bg-transparent font-mono text-[11px] text-(--silver) outline-none"
							bind:value={form.accent}
							maxlength="7"
						/>
					</span>
				</label>
			</div>
			{#if form.banner_url}
				<div class="mt-3 overflow-hidden border border-(--border) bg-(--bg)">
					<img
						src={form.banner_url}
						alt=""
						class="h-32 w-full object-cover opacity-75"
						onerror={(event) => ((event.currentTarget as HTMLImageElement).style.display = 'none')}
					/>
				</div>
			{/if}
			<div class="mt-3 flex flex-col gap-1.5">
				<div class="flex items-center justify-between">
					<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
						>custom CSS <span class="normal-case text-(--silver)/20">· scoped to the event page</span></span
					>
					<button
						type="button"
						class="cursor-pointer border border-(--border) px-2 py-0.5 font-mono text-[9px] tracking-[0.08em] text-(--silver)/40 uppercase hover:border-(--mer)/50 hover:text-(--mer)"
						aria-expanded={cssGuide}
						onclick={() => (cssGuide = !cssGuide)}>{cssGuide ? 'close guide' : 'css guide'}</button
					>
				</div>
				{#if cssGuide}
					<div class="border border-(--mer)/30 bg-(--bg) p-3">
						<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer)/70 uppercase">
							event page classes
						</div>
						<p class="mt-1 font-mono text-[9px] leading-relaxed text-(--silver)/35">
							Styles apply inside <span class="text-(--silver)/55">.contest-custom</span>. Use normal
							classes such as <span class="text-(--silver)/55">.contest-hero</span>,
							<span class="text-(--silver)/55">.contest-title</span>,
							<span class="text-(--silver)/55">.contest-panel</span>,
							<span class="text-(--silver)/55">.contest-stage</span>,
							<span class="text-(--silver)/55">.contest-badge</span>,
							<span class="text-(--silver)/55">.contest-cta</span>, and
							<span class="text-(--silver)/55">.contest-standings-row</span>. <span
								class="text-(--red)/60">@import</span
							> and <span class="text-(--red)/60">url()</span> are blocked.
						</p>
						<pre
							class="mt-2 max-h-48 overflow-auto border border-(--border) bg-(--fbg) p-2 font-mono text-[9px] leading-relaxed text-(--silver)/55">{CSS_GUIDE}</pre>
					</div>
				{/if}
				<textarea
					class="min-h-24 resize-y border border-(--border) bg-(--bg) px-3 py-2 font-mono text-[11px] leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
					bind:value={form.custom_css}
					maxlength="20000"
					placeholder={'.contest-title { letter-spacing: 0.2em; }'}
				></textarea>
			</div>
		</div>
	</div>

	<aside class="flex flex-col gap-3">
		<div class="border border-(--border) bg-(--fbg) p-5">
			<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">schedule</div>
			<div class="mt-3 flex flex-col gap-3">
				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
						>event starts · your local time</span
					>
					<input
						type="datetime-local"
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.starts_at}
					/>
				</label>
				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
						>contest capacity · total entrants</span
					>
					<input
						type="number"
						min="2"
						max="512"
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.max_players}
					/>
				</label>
				<div class="border border-(--border)/60 bg-(--bg)/40 p-3">
					<div class="font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase">funnel preview</div>
					<div class="mt-2 flex flex-col gap-1.5">
						{#each form.stages as stage, index (stage)}
							<div class="flex items-start justify-between gap-2 font-mono text-[9px]">
								<span class="text-(--silver)/55">{index + 1}. {formatLabel(stage.format)}</span>
								<span class="text-right text-(--silver)/25">{projectedLabel(index)}</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>

		<div class="border border-(--border) bg-(--fbg) p-5">
			<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">access</div>
			<div class="mt-3 grid grid-cols-2 gap-2">
				<button
					type="button"
					class={`cursor-pointer border px-3 py-2 font-mono text-[10px] tracking-[0.1em] uppercase ${form.visibility === 'public' ? 'border-(--mer) text-(--mer)' : 'border-(--border) text-(--silver)/45'}`}
					onclick={() => {
						form.visibility = 'public';
						form.password = '';
					}}>public</button
				>
				<button
					type="button"
					class={`cursor-pointer border px-3 py-2 font-mono text-[10px] tracking-[0.1em] uppercase ${form.visibility === 'private' ? 'border-(--mer) text-(--mer)' : 'border-(--border) text-(--silver)/45'}`}
					onclick={() => (form.visibility = 'private')}>private</button
				>
			</div>
			<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/30">
				{form.visibility === 'public'
					? 'Listed on the contest board and open to anyone with an account.'
					: 'Hidden from public listings. Share the invite link directly.'}
			</p>
			{#if form.visibility === 'private'}
				<label class="mt-3 flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
						>password <span class="normal-case text-(--silver)/20">· optional</span></span
					>
					<input
						type="password"
						autocomplete="new-password"
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.password}
						placeholder={contest?.requires_password ? 'leave blank to keep current' : 'optional gate'}
					/>
				</label>
			{/if}
		</div>

		{#if formError || error}
			<div class="border border-(--red)/40 px-4 py-3 font-mono text-[11px] leading-relaxed text-(--red)">
				{formError || error}
			</div>
		{/if}
		<button
			type="submit"
			class="cursor-pointer border border-(--mer) px-5 py-3 font-mono text-xs tracking-[0.12em] text-(--mer) uppercase hover:bg-(--mer)/10 disabled:opacity-40"
			disabled={busy}
		>{busy ? 'working…' : submitLabel}</button
		>
		<p class="font-mono text-[9px] leading-relaxed text-(--silver)/20">
			Hosts are enrolled automatically. The first stage needs two human entrants at start time or the
			event is cancelled.
		</p>
	</aside>
</form>

<style>
	.contest-md :global(h1),
	.contest-md :global(h2),
	.contest-md :global(h3) {
		margin: 1rem 0 0.4rem;
		color: var(--silver);
		font-family: 'Fira Code', monospace;
		font-size: 0.9rem;
		letter-spacing: 0.04em;
	}
	.contest-md :global(p) {
		margin: 0.55rem 0;
	}
	.contest-md :global(a) {
		color: var(--mer);
		text-decoration: underline;
	}
	.contest-md :global(code) {
		border: 1px solid var(--border);
		background: var(--fbg);
		padding: 0.05rem 0.3rem;
		font-size: 0.75rem;
	}
	.contest-md :global(ul),
	.contest-md :global(ol) {
		margin: 0.5rem 0;
		padding-left: 1.2rem;
	}
	.contest-md :global(li) {
		margin: 0.2rem 0;
	}
	.contest-md :global(blockquote) {
		margin: 0.6rem 0;
		border-left: 2px solid var(--mer);
		padding-left: 0.75rem;
		color: var(--silver);
	}
	.contest-md :global(pre) {
		overflow-x: auto;
		border: 1px solid var(--border);
		background: var(--fbg);
		padding: 0.7rem;
		font-size: 0.72rem;
	}
</style>
