<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import {
		cancelContest,
		contestCalendarURL,
		enrollContest,
		getContest,
		leaveContest,
		startContestAttempt
	} from '$lib/core/api/contests';
	import { errorMessage } from '$lib/core/api/client';
	import { onSocketEvent } from '$lib/core/api/socket';
	import type { ContestStageView, ContestView } from '$lib/core/api/types';
	import { LANGUAGE_LABELS } from '$lib/core/api/types';
	import { LANGUAGES } from '$lib/core/entity/languages';
	import { renderMarkdown } from '$lib/core/markdown';
	import { appState } from '$lib/core/state/state.svelte';

	const code = $derived((page.params.code ?? '').toUpperCase());

	let contest = $state<ContestView | null>(null);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');
	let notice = $state('');
	let password = $state('');
	let copied = $state(false);
	let now = $state(Date.now());
	let clockTimer: ReturnType<typeof setInterval> | null = null;
	let refreshTimer: ReturnType<typeof setInterval> | null = null;
	let unsubscribe: (() => void) | null = null;
	let customStyle: HTMLStyleElement | null = null;

	let descriptionHTML = $derived(renderMarkdown(contest?.description ?? ''));
	let participants = $derived(contest?.participants ?? []);
	let startsAt = $derived(contest ? new Date(contest.starts_at).getTime() : 0);
	let remaining = $derived(Math.max(0, startsAt - now));
	let activeStage = $derived(contest?.stages?.find((stage) => stage.status === 'active') ?? null);
	let viewerGameID = $derived(
		activeStage?.viewer_game_id ||
			(activeStage?.id === 0 ? (contest?.game_id ?? 0) : 0)
	);
	let attemptBusy = $state(0);
	let stageLanguage = $state<Record<number, string>>({});
	let countdown = $derived.by(() => {
		if (!contest) return '-';
		if (contest.status === 'started') {
			return activeStage ? `stage ${activeStage.position} live` : 'event live';
		}
		if (contest.status === 'ended') return 'event finished';
		if (contest.status === 'cancelled') return 'cancelled';
		const seconds = Math.floor(remaining / 1000);
		const days = Math.floor(seconds / 86400);
		const hours = Math.floor((seconds % 86400) / 3600);
		const minutes = Math.floor((seconds % 3600) / 60);
		const secs = seconds % 60;
		if (days > 0) return `${days}d ${hours}h ${minutes}m`;
		if (hours > 0) return `${hours}h ${minutes}m ${String(secs).padStart(2, '0')}s`;
		return `${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
	});
	let startLabel = $derived(
		contest
			? new Date(contest.starts_at).toLocaleString(undefined, {
					weekday: 'short',
					month: 'short',
					day: 'numeric',
					hour: '2-digit',
					minute: '2-digit'
				})
			: ''
	);
	let enrolled = $derived(Boolean(contest?.is_enrolled || contest?.is_host));
	let full = $derived(Boolean(contest && contest.entry_count >= contest.max_players));
	let canEnroll = $derived(
		Boolean(contest && contest.status === 'upcoming' && !enrolled && !full)
	);
	let needsPassword = $derived(Boolean(contest?.requires_password && !enrolled));
	let seatsLeft = $derived(contest ? Math.max(0, contest.max_players - contest.entry_count) : 0);
	let dashboardVisible = $derived(Boolean(contest && (contest.visibility === 'public' || enrolled)));
	let totalRuns = $derived(
		contest?.stages?.reduce((sum, stage) => sum + (stage.run_count ?? 0), 0) ?? 0
	);
	let survivors = $derived.by(() => {
		if (!contest) return 0;
		if (contest.status === 'upcoming') return contest.entry_count;
		if (contest.status === 'cancelled') return 0;
		if (contest.status === 'ended') return contest.leaderboard?.length ? 1 : 0;
		const active = contest.stages?.find((stage) => stage.status === 'active');
		if (active) return active.entry_count;
		const next = contest.stages?.find(
			(stage) => stage.status === 'pending' || stage.status === 'active'
		);
		return next?.entry_count ?? contest.entry_count;
	});
	let eliminatedCount = $derived(
		contest ? Math.max(0, contest.entry_count - survivors) : 0
	);
	let funnelRows = $derived.by(() => {
		if (!contest) return [];
		const stages = contest.stages ?? [];
		const initial = Math.max(1, contest.entry_count);
		return stages.map((stage, index) => {
			let count = stage.entry_count;
			let projected = false;
			if (index === 0 && count <= 0) count = contest.entry_count;
			if (index > 0 && count <= 0) {
				const previous = stages[index - 1];
				count =
					previous.advance_count > 0
						? previous.advance_count
						: previous.run_count > 0
							? previous.entry_count
							: 0;
				projected = stage.status === 'pending';
			}
			const finalStage = index === stages.length - 1;
			const left =
				stage.status === 'completed'
					? finalStage
						? count > 0
							? 1
							: 0
						: stage.advance_count
					: count;
			const advancing =
				stage.status === 'completed'
					? left
					: stage.status === 'active'
						? stage.advance_count
						: 0;
			return {
				stage,
				index,
				count,
				projected,
				left,
				advancing,
				finalStage,
				countPct: count > 0 ? Math.max(3, Math.min(100, (count / initial) * 100)) : 0,
				advancePct: advancing > 0 ? Math.max(2, Math.min(100, (advancing / initial) * 100)) : 0
			};
		});
	});

	const STAGE_FORMAT_LABELS: Record<string, string> = {
		qualifier: 'qualifier',
		hotlaps: 'HotLaps',
		multiplayer: 'live race',
		ranked: 'ranked final'
	};

	function isOpenStage(stage: ContestStageView): boolean {
		return stage.format === 'qualifier' || stage.format === 'hotlaps';
	}

	function formatLabel(stage: ContestStageView): string {
		return STAGE_FORMAT_LABELS[stage.format] ?? stage.format;
	}

	function stageDate(value: string | null): string {
		if (!value) return '-';
		return new Date(value).toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function stageStatusLabel(stage: ContestStageView): string {
		if (stage.status === 'active') return isOpenStage(stage) ? 'open' : 'live';
		if (stage.status === 'completed') return 'complete';
		if (stage.status === 'cancelled') return 'cancelled';
		return 'queued';
	}

	function stageAdvanceLabel(stage: ContestStageView, final: boolean): string {
		if (final) return 'final ranking';
		if (stage.advance_type === 'count') return `${stage.advance_value} advance`;
		if (stage.advance_type === 'percent') return `top ${stage.advance_value}% advance`;
		if (stage.format === 'qualifier') return `all above ${stage.min_wpm} wpm`;
		return 'all finishers advance';
	}

	function stageClockLabel(stage: ContestStageView): string {
		if (stage.status === 'completed') return `closed ${stageDate(stage.ends_at)}`;
		if (stage.status === 'active') {
			if (isOpenStage(stage)) return `attempts close ${stageDate(stage.ends_at)}`;
			return `heat in progress · closes ${stageDate(stage.ends_at)}`;
		}
		if (stage.status === 'cancelled') return 'cancelled';
		return `starts ${stageDate(stage.starts_at)}`;
	}

	function selectedLanguage(stage: ContestStageView): string {
		return stageLanguage[stage.id] ?? stage.language;
	}

	async function attempt(stage: ContestStageView) {
		if (attemptBusy != 0 || !contest) return;
		attemptBusy = stage.position;
		error = '';
		notice = '';
		try {
			const language = stage.language_mode === 'choice' ? selectedLanguage(stage) : '';
			const game = await startContestAttempt(contest.code, stage.position, language);
			await goto(`/game/${game.id}`);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			attemptBusy = 0;
		}
	}

	async function load() {
		error = '';
		try {
			contest = await getContest(code);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	}

	async function enroll() {
		if (busy || !contest) return;
		if (!appState.user) {
			await goto(`/login?next=${encodeURIComponent(`/contests/${code}`)}`);
			return;
		}
		busy = true;
		error = '';
		notice = '';
		try {
			contest = await enrollContest(code, password);
			password = '';
			notice = 'You are in. The race starts on schedule.';
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function leave() {
		if (busy || !contest) return;
		busy = true;
		error = '';
		notice = '';
		try {
			contest = await leaveContest(code);
			notice = 'Enrollment removed.';
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function cancel() {
		if (busy || !contest) return;
		if (!confirm('Cancel this contest? Enrolled players will be notified.')) return;
		busy = true;
		error = '';
		try {
			contest = await cancelContest(code);
			notice = 'Contest cancelled.';
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function copyLink() {
		if (!contest) return;
		try {
			await navigator.clipboard.writeText(contest.share_link);
			copied = true;
			setTimeout(() => (copied = false), 1400);
		} catch {
			notice = contest.share_link;
		}
	}

	async function share() {
		if (!contest) return;
		if (navigator.share) {
			try {
				await navigator.share({ title: contest.title, url: contest.share_link });
				return;
			} catch {
				// The user closed the sheet; fall through to copy.
			}
		}
		await copyLink();
	}

	onMount(() => {
		void load();
		clockTimer = setInterval(() => (now = Date.now()), 1000);
		refreshTimer = setInterval(() => {
			if (contest && (contest.status === 'upcoming' || contest.status === 'started')) {
				void load();
			}
		}, 20_000);
		unsubscribe = onSocketEvent((event) => {
			if (
				event.type === 'contest_started' ||
				event.type === 'contest_enrollment' ||
				event.type === 'contest_cancelled' ||
				event.type === 'contest_stage_started' ||
				event.type === 'contest_stage_completed' ||
				event.type === 'contest_ended'
			) {
				void load();
			}
		});
		return () => {
			if (clockTimer) clearInterval(clockTimer);
			if (refreshTimer) clearInterval(refreshTimer);
			unsubscribe?.();
		};
	});

	$effect(() => {
		const css = contest?.custom_css?.trim() ?? '';
		if (typeof document === 'undefined') return;
		if (!css) {
			customStyle?.remove();
			customStyle = null;
			return;
		}
		if (!customStyle) {
			customStyle = document.createElement('style');
			customStyle.dataset.contest = code;
			document.head.appendChild(customStyle);
		}
		// @scope keeps host CSS on the event page instead of leaking into the
		// rest of the app. The server rejects imports, url() and raw tags.
		customStyle.textContent = `@scope (.contest-custom) {\n${css}\n}`;
		return () => {
			customStyle?.remove();
			customStyle = null;
		};
	});
</script>

<svelte:head>
	<title>{contest ? `${contest.title} · OSDTYP contests` : 'contest · OSDTYP'}</title>
	{#if contest}
		<meta property="og:title" content={contest.title} />
		<meta property="og:description" content={contest.description.slice(0, 180)} />
		<meta property="og:url" content={contest.share_link} />
		{#if contest.banner_url}
			<meta property="og:image" content={contest.banner_url} />
		{/if}
	{/if}
</svelte:head>

<div
	class="contest-custom contest-page min-h-screen bg-(--bg) p-5 md:p-8"
	style={`--event:${contest?.accent ?? '#00a19c'}`}
>
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<div class="flex items-center gap-3">
			<a
				href={resolve('/contests')}
				class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← all contests</a
			>
			{#if contest}
				<span class="border border-(--border) px-2 py-1 font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase"
					>{contest.code}</span
				>
				<span
					class="font-mono text-[9px] tracking-[0.14em] uppercase"
					style={`color:${contest.accent}`}>{contest.status}</span
				>
			{/if}
		</div>
		<div class="flex items-center gap-2">
			{#if contest?.is_host && contest.status === 'upcoming'}
				<a
					href={resolve(`/contests/${contest.code}/edit`)}
					class="border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--silver)/45 uppercase hover:border-(--event) hover:text-(--silver)"
					>edit</a
				>
				<button
					class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--silver)/35 uppercase hover:border-(--red) hover:text-(--red)"
					onclick={cancel}
					disabled={busy}>cancel event</button
				>
			{/if}
			{#if appState.user}
				<a
					href={resolve('/contests/new')}
					class="border border-(--mer) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
					>host yours</a
				>
			{/if}
		</div>
	</header>

	{#if loading}
		<div class="mt-5 border border-(--border) bg-(--fbg) p-10 font-mono text-xs text-(--silver)/35">
			opening event…
		</div>
	{:else if !contest}
		<div class="mt-5 border border-(--red)/40 p-8">
			<div class="font-mono text-sm text-(--red)">{error || 'contest not found'}</div>
			<a href={resolve('/contests')} class="mt-3 inline-block font-mono text-xs text-(--mer)"
				>← return to the contest board</a
			>
		</div>
	{:else}
		<section class="contest-hero relative mt-4 overflow-hidden border border-(--border) bg-(--fbg)">
			{#if contest.banner_url}
				<img
					src={contest.banner_url}
					alt=""
					class="absolute inset-0 h-full w-full object-cover opacity-25"
				/>
				<div class="absolute inset-0 bg-gradient-to-t from-(--fbg) via-(--fbg)/80 to-transparent"></div>
			{:else}
				<div
					class="absolute inset-0"
					style={`background: radial-gradient(ellipse at 15% 0%, color-mix(in srgb, ${contest.accent} 24%, transparent), transparent 62%)`}
				></div>
			{/if}
			<div class="relative p-6 md:p-9">
				<div class="flex flex-wrap items-center gap-2 font-mono text-[9px] tracking-[0.16em] uppercase">
					<span style={`color:${contest.accent}`}>{contest.visibility} event</span>
					<span class="text-(--silver)/20">/</span>
					<span class="text-(--silver)/40"
						>{contest.stage_count} {contest.stage_count === 1 ? 'stage' : 'stages'} · {contest.duration}s
						{LANGUAGE_LABELS[contest.language] ?? contest.language}</span
					>
					{#if contest.visibility === 'private'}
						<span class="border border-(--border) px-1.5 py-0.5 text-(--silver)/30">invite link only</span>
					{/if}
				</div>
				<h1 class="contest-title mt-3 max-w-4xl text-4xl leading-[1.02] text-(--silver) md:text-6xl">
					{contest.title}
				</h1>
				<div class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 font-mono text-[11px]">
					<a
						href={resolve(`/profile/${contest.host.username}`)}
						class="flex items-center gap-2 text-(--silver)/50 hover:text-(--silver)"
					>
						<Avatar
							src={contest.host.avatar_url}
							name={contest.host.username}
							class="h-5 w-5"
							text="text-[8px]"
						/>
						hosted by {contest.host.username}
					</a>
					<span class="text-(--silver)/30">{startLabel}</span>
					<span class="text-(--silver)/30">{contest.entry_count}/{contest.max_players} enrolled</span>
				</div>
				<div class="mt-7 flex flex-wrap items-end justify-between gap-5">
					<div>
						<div class="font-mono text-[9px] tracking-[0.16em] text-(--silver)/30 uppercase">
							{contest.status === 'upcoming' ? 'starts in' : 'clock'}
						</div>
						<div
							class="mt-1 font-mono text-5xl leading-none md:text-6xl"
							style={`color:${contest.accent}`}>{countdown}</div>
					</div>
					<div class="flex flex-wrap gap-2">
						<button
							class="cursor-pointer border border-(--event) px-4 py-2 font-mono text-[10px] tracking-[0.1em] uppercase"
							style={`color:${contest.accent}`}
							onclick={share}>{copied ? 'link copied' : 'share event'}</button
						>
						<a
							href={contestCalendarURL(contest.code)}
							class="border border-(--border) px-4 py-2 font-mono text-[10px] tracking-[0.1em] text-(--silver)/40 uppercase hover:text-(--silver)"
							>add to calendar</a
						>
						{#if contest.status === 'started' && enrolled && viewerGameID}
							<a
								href={resolve(`/game/${viewerGameID}`)}
								class="contest-cta border border-(--mer) bg-(--mer)/10 px-4 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/20"
								>enter your heat →</a
							>
						{/if}
					</div>
				</div>
			</div>
		</section>

		{#if error}
			<div class="mt-3 border border-(--red)/40 px-4 py-3 font-mono text-xs text-(--red)">{error}</div>
		{:else if notice}
			<div class="mt-3 border border-(--mer)/40 px-4 py-3 font-mono text-xs text-(--mer)">{notice}</div>
		{/if}

		{#if dashboardVisible}
			<section class="contest-panel mt-4 border border-(--border) bg-(--fbg)">
				<div class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-5 py-3">
					<div class="flex items-center gap-3">
						<span class="font-mono text-[10px] tracking-[0.16em] uppercase" style={`color:${contest.accent}`}
							>event dashboard</span
						>
						<span class="font-mono text-[9px] text-(--silver)/25">
							{contest.visibility === 'public' ? 'public board · open to everyone' : 'private event board'}
						</span>
					</div>
					{#if contest.status === 'started' && activeStage}
						<span class="flex items-center gap-2 border border-(--mer)/40 px-2 py-1 font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">
							<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-(--mer)"></span>
							stage {activeStage.position} live
						</span>
					{:else if contest.status === 'ended'}
						<span class="border border-(--border) px-2 py-1 font-mono text-[9px] tracking-[0.12em] text-(--silver)/35 uppercase"
							>final board</span
						>
					{:else}
						<span class="border border-(--border) px-2 py-1 font-mono text-[9px] tracking-[0.12em] text-(--silver)/35 uppercase"
							>{contest.status}</span
						>
					{/if}
				</div>

				<div class="grid grid-cols-2 gap-px bg-(--border) md:grid-cols-4">
					<div class="bg-(--fbg) px-5 py-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/25 uppercase">entrants</div>
						<div class="mt-1 font-mono text-2xl text-(--silver)">{contest.entry_count}</div>
						<div class="mt-0.5 font-mono text-[9px] text-(--silver)/25">of {contest.max_players} seats</div>
					</div>
					<div class="bg-(--fbg) px-5 py-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/25 uppercase">still in</div>
						<div class="mt-1 font-mono text-2xl" style={`color:${contest.accent}`}>{survivors}</div>
						<div class="mt-0.5 font-mono text-[9px] text-(--silver)/25">
							{eliminatedCount} eliminated
						</div>
					</div>
					<div class="bg-(--fbg) px-5 py-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/25 uppercase">runs recorded</div>
						<div class="mt-1 font-mono text-2xl text-(--silver)">{totalRuns}</div>
						<div class="mt-0.5 font-mono text-[9px] text-(--silver)/25">across the funnel</div>
					</div>
					<div class="bg-(--fbg) px-5 py-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/25 uppercase">stage</div>
						<div class="mt-1 font-mono text-2xl text-(--silver)">
							{contest.current_stage || (contest.status === 'ended' ? contest.stage_count : 0)}<span
								class="text-sm text-(--silver)/25">/{contest.stage_count}</span
							>
						</div>
						<div class="mt-0.5 font-mono text-[9px] text-(--silver)/25">
							{activeStage ? activeStage.name : contest.status === 'ended' ? 'complete' : 'waiting'}
						</div>
					</div>
				</div>

				<div class="grid gap-5 p-5 lg:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)]">
					<div>
						<div class="flex items-center justify-between font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase">
							<span>funnel progression</span>
							<span>entrants → survivors</span>
						</div>
						<div class="mt-3 flex flex-col gap-3">
							{#each funnelRows as row (row.stage.id || row.stage.position)}
								<div class="grid grid-cols-[82px_minmax(0,1fr)_78px] items-center gap-3 md:grid-cols-[140px_minmax(0,1fr)_104px]">
									<div class="min-w-0">
										<div class="truncate font-mono text-[10px] text-(--silver)/65">
											{String(row.stage.position).padStart(2, '0')} {row.stage.name}
										</div>
										<div class="mt-0.5 truncate font-mono text-[8px] tracking-[0.08em] text-(--silver)/25 uppercase">
											{formatLabel(row.stage)}
											{#if row.stage.status === 'active'}<span style={`color:${contest.accent}`}>
													· live</span
												>{/if}
										</div>
									</div>
									<div class={`relative h-7 overflow-hidden border bg-(--bg) ${row.projected ? 'border-dashed border-(--silver)/20' : 'border-(--border)'}`}>
										{#if row.count > 0}
											<div
												class="absolute inset-y-0 left-0 border-r border-(--silver)/10"
												style={`width:${row.countPct}%; background: color-mix(in srgb, ${contest.accent} 14%, transparent);`}
											></div>
										{/if}
										{#if row.advancePct > 0}
											<div
												class="absolute inset-y-0 left-0 border-r-2"
												style={`width:${row.advancePct}%; background: color-mix(in srgb, ${contest.accent} 52%, transparent); border-color:${contest.accent};`}
											></div>
										{/if}
										<div class="absolute inset-0 flex items-center justify-between px-2 font-mono text-[8px] text-(--silver)/45">
											<span>{row.count > 0 ? `${row.count} in` : 'no entrants'}</span>
											<span>{row.count > 0 ? `${row.left} left` : ''}</span>
										</div>
									</div>
									<div class="text-right font-mono text-[8px] leading-tight text-(--silver)/30">
										{#if row.stage.status === 'completed'}
											<div class="text-(--silver)/50">
												{row.finalStage ? 'champion' : `${row.left} advanced`}
											</div>
											<div>{row.advancing > 0 ? `${row.advancing} survive` : 'no survivors'}</div>
										{:else if row.stage.status === 'active'}
											<div class="text-(--mer)">in progress</div>
											<div>{row.stage.qualified_count} qualified</div>
										{:else if row.projected}
											<div>projected</div>
											<div>{row.stage.status === 'pending' ? 'waiting' : ''}</div>
										{:else}
											<div>{row.stage.status === 'cancelled' ? 'cancelled' : 'queued'}</div>
											<div>{row.stage.advance_type === 'all' ? 'all advance' : `${row.stage.advance_value} advance`}</div>
										{/if}
									</div>
								</div>
							{/each}
						</div>
						<div class="mt-3 font-mono text-[8px] leading-relaxed text-(--silver)/20">
							{funnelRows.some((row) => row.projected)
								? 'dashed-stage counts are projected from the previous cut until the stage begins.'
								: 'bars show the entrants who reached each stage and the players who survived it.'}
						</div>
					</div>

					<div>
						<div class="font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase">
							stage leaders
						</div>
						<div class="mt-3 flex flex-col gap-2">
							{#each contest.stages as stage (stage.id || stage.position)}
								<div class="border border-(--border)/70 bg-(--bg)/35 p-3">
									<div class="flex items-center justify-between gap-2 font-mono text-[9px]">
										<span class="truncate text-(--silver)/55">{stage.name}</span>
										<span class="shrink-0 text-(--silver)/25">
											{stage.status === 'active' ? 'live' : stage.status === 'completed' ? 'final' : 'queued'}
										</span>
									</div>
									{#if stage.standings && stage.standings.length > 0}
										<div class="mt-2 flex flex-col gap-1.5">
											{#each stage.standings.slice(0, 3) as row, index (row.user.id)}
												<div class="flex items-center gap-2 font-mono text-[10px]">
													<span class="w-4 text-(--silver)/25">{index + 1}</span>
													<Avatar src={row.user.avatar_url} name={row.user.username} class="h-5 w-5" text="text-[8px]" />
													<a
														href={row.game_id ? resolve(`/replay/${row.game_id}`) : resolve(`/profile/${row.user.username}`)}
														class="min-w-0 flex-1 truncate text-(--silver)/60 hover:text-(--silver)"
														>{row.user.username}</a
													>
													<span class="text-(--silver)">{Math.round(row.wpm)}</span>
													<span class="text-(--silver)/25">wpm</span>
												</div>
											{/each}
										</div>
									{:else}
										<p class="mt-2 font-mono text-[9px] text-(--silver)/20">No runs yet.</p>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				</div>

				<div class="border-t border-(--border) p-5">
					<div class="flex flex-wrap items-center justify-between gap-2">
						<div class="font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase">
							top runs by category
						</div>
						<div class="font-mono text-[8px] text-(--silver)/20">attempt-level board · replays where available</div>
					</div>
					<div class="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
						{#each contest.stages as stage (stage.id || stage.position)}
							<article class="border border-(--border)/70 bg-(--bg)/35 p-3">
								<div class="flex items-center justify-between gap-2 font-mono text-[9px]">
									<span class="truncate text-(--silver)/55">{stage.name}</span>
									<span class="shrink-0 text-(--silver)/25">{stage.run_count ?? 0} runs</span>
								</div>
								{#if stage.top_runs && stage.top_runs.length > 0}
									<div class="mt-2 flex flex-col gap-1">
										{#each stage.top_runs.slice(0, 4) as run (`${run.game_id}-${run.user.id}`)}
											<div class="flex items-center gap-2 font-mono text-[9px]">
												<span class="w-4 text-(--silver)/25">#{run.rank}</span>
												<a
													href={resolve(`/profile/${run.user.username}`)}
													class="min-w-0 flex-1 truncate text-(--silver)/55 hover:text-(--silver)">{run.user.username}</a
												>
												{#if run.flagged}<span class="text-(--red)/70">flag</span>{/if}
												<span class="text-(--silver)/75">{Math.round(run.wpm)}</span>
												<a
													href={resolve(`/replay/${run.game_id}`)}
													class="border border-(--border) px-1 text-(--silver)/25 hover:border-(--mer) hover:text-(--mer)"
													title="watch replay"
													>replay</a
												>
											</div>
										{/each}
									</div>
								{:else}
									<p class="mt-2 font-mono text-[9px] text-(--silver)/20">
										{stage.status === 'pending' ? 'Waiting for this stage to begin.' : 'No attempts recorded yet.'}
									</p>
								{/if}
							</article>
						{/each}
					</div>
				</div>
			</section>
		{/if}

		<main class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
			<div class="flex min-w-0 flex-col gap-4">
				<section class="contest-panel border border-(--border) bg-(--fbg)">
					<div class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-5 py-3">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							competition funnel
						</div>
						<span class="font-mono text-[10px] text-(--silver)/25">
							{contest.stage_count} {contest.stage_count === 1 ? 'stage' : 'stages'}
						</span>
					</div>

					<div class="flex flex-col gap-px bg-(--border)">
						{#each contest.stages as stage, index (stage.id)}
							<article
								id={`stage-${stage.position}`}
								class={`contest-stage bg-(--fbg) p-4 md:p-5 ${stage.status === 'active' ? 'is-active' : ''} ${stage.status === 'completed' ? 'is-complete' : ''} ${stage.status === 'cancelled' ? 'is-cancelled' : ''}`}
							>
								<div class="flex flex-wrap items-start justify-between gap-3">
									<div class="flex min-w-0 items-start gap-3">
										<div
											class="flex h-9 w-9 shrink-0 items-center justify-center border border-(--border) font-mono text-sm"
											style={stage.status === 'active' ? `color:${contest.accent};border-color:${contest.accent}` : ''}
										>
											{String(stage.position).padStart(2, '0')}
										</div>
										<div class="min-w-0">
											<div class="flex flex-wrap items-center gap-2">
												<h2 class="text-lg leading-tight text-(--silver)">{stage.name}</h2>
												<span
													class="contest-badge border border-(--border) px-1.5 py-0.5 font-mono text-[8px] tracking-[0.12em] text-(--silver)/40 uppercase"
													>{formatLabel(stage)}</span
												>
												{#if index === contest.stages.length - 1}
													<span
														class="contest-badge border px-1.5 py-0.5 font-mono text-[8px] tracking-[0.12em] uppercase"
														style={`color:${contest.accent};border-color:${contest.accent}`}>final</span
													>
												{/if}
												<span
													class={`contest-badge px-1.5 py-0.5 font-mono text-[8px] tracking-[0.12em] uppercase ${stage.status === 'active' ? 'text-(--mer)' : stage.status === 'cancelled' ? 'text-(--red)' : 'text-(--silver)/30'}`}
												>
													{stageStatusLabel(stage)}
												</span>
											</div>
											<div class="mt-1 font-mono text-[9px] text-(--silver)/30">
												{stageClockLabel(stage)}
												{#if index > 0}
													· starts {stage.delay_minutes}m after the previous stage closes
												{/if}
											</div>
										</div>
									</div>
									<div class="text-right font-mono text-[9px] text-(--silver)/30">
										<div>{stage.entry_count} eligible</div>
										<div class="mt-0.5">{stageAdvanceLabel(stage, index === contest.stages.length - 1)}</div>
									</div>
								</div>

								<div class="mt-3 flex flex-wrap gap-1.5 font-mono text-[9px] text-(--silver)/35">
									<span class="border border-(--border)/70 px-2 py-0.5">{stage.duration}s run</span>
									{#if isOpenStage(stage)}
										<span class="border border-(--border)/70 px-2 py-0.5">{stage.window_minutes}m window</span>
										<span class="border border-(--border)/70 px-2 py-0.5">
											{stage.max_tries > 0 ? `${stage.max_tries} tries` : 'unlimited tries'}
										</span>
										{#if stage.format === 'qualifier'}
											<span class="border border-(--border)/70 px-2 py-0.5">min {stage.min_wpm} wpm</span>
										{:else}
											<span class="border border-(--border)/70 px-2 py-0.5">shared snippet</span>
										{/if}
									{:else}
										<span class="border border-(--border)/70 px-2 py-0.5"
											>heats up to {stage.max_players}</span
										>
										<span class="border border-(--border)/70 px-2 py-0.5">opponent cursors blurred</span>
									{/if}
									<span class="border border-(--border)/70 px-2 py-0.5">
										{stage.language_mode === 'choice'
											? 'player picks language'
											: LANGUAGE_LABELS[stage.language] ?? stage.language}
									</span>
								</div>

								{#if stage.status === 'active' && enrolled && isOpenStage(stage)}
									<div class="mt-4 border-t border-(--border)/60 pt-3">
										{#if stage.viewer_game_id}
											<a
												href={resolve(`/game/${stage.viewer_game_id}`)}
												class="contest-cta inline-flex border border-(--mer) px-4 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
												>resume attempt →</a
											>
										{:else if stage.viewer_can_attempt}
											<div class="flex flex-wrap items-end gap-2">
												{#if stage.language_mode === 'choice'}
													<label class="flex flex-col gap-1">
														<span class="font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase"
															>language for this attempt</span
														>
														<select
															class="border border-(--border) bg-(--bg) px-2.5 py-1.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
															value={selectedLanguage(stage)}
															onchange={(event) =>
																(stageLanguage[stage.id] = (event.currentTarget as HTMLSelectElement).value)}
														>
															{#each LANGUAGES as language (language)}
																<option value={language}>{LANGUAGE_LABELS[language]}</option>
															{/each}
														</select>
													</label>
												{/if}
												<button
													class="contest-cta cursor-pointer border border-(--mer) px-4 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10 disabled:opacity-40"
													onclick={() => attempt(stage)}
													disabled={attemptBusy != 0}
												>
													{attemptBusy === stage.position ? 'starting…' : 'start attempt →'}
												</button>
												<span class="pb-2 font-mono text-[9px] text-(--silver)/30">
													{stage.viewer_attempts} used{#if stage.max_tries > 0}
														· {stage.viewer_attempts_left} left
													{:else}
														· unlimited
													{/if}
												</span>
											</div>
										{:else}
											<p class="font-mono text-[10px] leading-relaxed text-(--silver)/30">
												{stage.max_tries > 0 && stage.viewer_attempts_left === 0
													? 'You have used every attempt for this stage.'
													: 'Attempts are not open right now. A run needs enough time left in the stage window.'}
											</p>
										{/if}
									</div>
								{:else if stage.status === 'active' && enrolled && !isOpenStage(stage) && stage.viewer_game_id}
									<div class="mt-4 border-t border-(--border)/60 pt-3">
										<a
											href={resolve(`/game/${stage.viewer_game_id}`)}
											class="contest-cta inline-flex border border-(--mer) px-4 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
											>enter your heat →</a
										>
									</div>
								{:else if stage.status === 'active' && !enrolled}
									<p class="mt-4 border-t border-(--border)/60 pt-3 font-mono text-[10px] text-(--silver)/30">
										Only enrolled players can take part in this stage.
									</p>
								{/if}

								{#if stage.standings && stage.standings.length > 0}
									<div class="contest-standings mt-4 border-t border-(--border)/60 pt-3">
										<div class="mb-2 flex items-center justify-between font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase">
											<span>{stage.status === 'active' ? 'provisional standings' : 'stage standings'}</span>
											<span>{stage.qualified_count} qualified · {stage.advance_count} advance</span>
										</div>
										<div class="overflow-x-auto">
											<table class="w-full border-collapse font-mono text-[10px]">
												<thead>
													<tr class="border-b border-(--border) text-left text-[8px] tracking-[0.12em] text-(--silver)/25 uppercase">
														<th class="px-2 py-1.5">#</th>
														<th class="px-2 py-1.5">player</th>
														<th class="px-2 py-1.5 text-right">best wpm</th>
														<th class="hidden px-2 py-1.5 text-right sm:table-cell">acc</th>
														<th class="px-2 py-1.5 text-right">tries</th>
														<th class="px-2 py-1.5 text-right">status</th>
													</tr>
												</thead>
												<tbody>
													{#each stage.standings as row (row.user.id)}
														<tr class="contest-standings-row border-b border-(--border)/50">
															<td class="px-2 py-1.5 text-(--silver)/30">{row.rank}</td>
															<td class="max-w-[180px] truncate px-2 py-1.5">
																<a
																	href={resolve(`/profile/${row.user.username}`)}
																	class="text-(--silver)/65 hover:text-(--silver)">{row.user.username}</a
																>
															</td>
															<td class="px-2 py-1.5 text-right text-(--silver)">{Math.round(row.wpm)}</td>
															<td class="hidden px-2 py-1.5 text-right text-(--silver)/40 sm:table-cell"
																>{Math.round(row.accuracy)}%</td
															>
															<td class="px-2 py-1.5 text-right text-(--silver)/30">{row.attempts}</td>
															<td class="px-2 py-1.5 text-right">
																{#if row.advanced}
																	<span class="text-(--mer)">advanced</span>
																{:else if row.eliminated}
																	<span class="text-(--red)/70">out</span>
																{:else if row.flagged}
																	<span class="text-(--red)/70">flagged</span>
																{:else}
																	<span class="text-(--silver)/25">-</span>
																{/if}
															</td>
														</tr>
													{/each}
												</tbody>
											</table>
										</div>
									</div>
								{:else if stage.status === 'active' && isOpenStage(stage)}
									<p class="mt-4 border-t border-(--border)/60 pt-3 font-mono text-[10px] text-(--silver)/25">
										No attempts yet. The board will fill as players submit runs.
									</p>
								{/if}
							</article>
						{/each}
					</div>
				</section>

				<section class="contest-panel border border-(--border) bg-(--fbg) p-5 md:p-7">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/30 uppercase">
						about this event
					</div>
					{#if descriptionHTML}
						<div class="contest-md mt-3 max-w-3xl text-sm leading-relaxed text-(--silver)/60">
							{@html descriptionHTML}
						</div>
					{:else}
						<p class="mt-3 font-mono text-xs text-(--silver)/25">The host has not added a description.</p>
					{/if}
				</section>

				<section class="border border-(--border) bg-(--fbg)">
					<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							participants
						</div>
						<span class="font-mono text-[10px] text-(--silver)/25">
							{contest.entry_count}/{contest.max_players}
						</span>
					</div>
					{#if contest.visibility === 'private' && !enrolled}
						<p class="px-5 py-6 font-mono text-[11px] leading-relaxed text-(--silver)/30">
							The roster is private. Enroll to see who else is racing.
						</p>
					{:else if participants.length === 0}
						<p class="px-5 py-6 font-mono text-[11px] text-(--silver)/25">
							No one has enrolled yet.
						</p>
					{:else}
						<div class="grid gap-px bg-(--border) sm:grid-cols-2">
							{#each participants as participant (participant.user.id)}
								<a
									href={resolve(`/profile/${participant.user.username}`)}
									class="flex items-center gap-3 bg-(--fbg) px-5 py-3 transition-colors hover:bg-(--mer)/5"
								>
									<Avatar
										src={participant.user.avatar_url}
										name={participant.user.username}
										class="h-7 w-7 shrink-0"
										text="text-[9px]"
									/>
									<span class="min-w-0 flex-1">
										<span class="block truncate font-mono text-[11px] text-(--silver)/65">
											{participant.user.username}
											{#if participant.user.id === contest.host.id}
												<span class="ml-1 text-[8px] tracking-[0.1em] text-(--mer)/60 uppercase">host</span>
											{/if}
										</span>
										<span class="mt-0.5 block font-mono text-[9px] text-(--silver)/25">
											{participant.user.rating} elo
										</span>
									</span>
								</a>
							{/each}
							{#each Array.from({ length: Math.min(seatsLeft, 3) }) as _, index (index)}
								<div class="flex items-center gap-3 bg-(--fbg) px-5 py-3 opacity-35">
									<div class="h-7 w-7 shrink-0 border border-(--border)"></div>
									<span class="font-mono text-[10px] text-(--silver)/25">open seat</span>
								</div>
							{/each}
						</div>
					{/if}
				</section>

				{#if contest.leaderboard && contest.leaderboard.length > 0}
					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) px-5 py-3 font-mono text-[10px] tracking-[0.14em] uppercase" style={`color:${contest.accent}`}>
							final leaderboard
						</div>
						<div class="divide-y divide-(--border)">
							{#each contest.leaderboard as result (result.user_id)}
								<div class="grid grid-cols-[34px_minmax(0,1fr)_64px_64px] items-center gap-2 px-5 py-2.5 md:grid-cols-[40px_minmax(0,1fr)_80px_80px_90px]">
									<span class="font-mono text-sm text-(--silver)/30">#{result.rank}</span>
									<span class="flex min-w-0 items-center gap-2">
										<Avatar src={result.avatar_url} name={result.username} class="h-5 w-5" text="text-[8px]" />
										<a
											href={resolve(`/profile/${result.username}`)}
											class="truncate font-mono text-[11px] text-(--silver)/65 hover:text-(--silver)">{result.username}</a
										>
									</span>
									<span class="text-right font-mono text-[11px] text-(--silver)">{Math.round(result.wpm)} wpm</span>
									<span class="hidden text-right font-mono text-[10px] text-(--silver)/35 md:block"
										>{Math.round(result.accuracy)}%</span
									>
									<span class="text-right font-mono text-[10px] text-(--silver)/30">
										{Math.round(result.raw)} raw
									</span>
								</div>
							{/each}
						</div>
					</section>
				{/if}
			</div>

			<aside class="flex flex-col gap-3">
				<div class="border border-(--event) bg-(--fbg) p-5" style={`border-top-width:3px`}>
					<div class="font-mono text-[10px] tracking-[0.16em] uppercase" style={`color:${contest.accent}`}>
						{contest.status === 'upcoming'
							? enrolled
								? "you're in"
								: 'registration'
							: contest.status}
					</div>
					{#if contest.status === 'upcoming'}
						{#if enrolled}
							<div class="mt-2 text-2xl text-(--silver)">Seat confirmed.</div>
							<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/35">
								The race starts at {startLabel}. Keep this page open or return to it from the hub.
							</p>
							{#if !contest.is_host}
								<button
									class="mt-4 w-full cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] tracking-[0.1em] text-(--silver)/35 uppercase hover:border-(--red) hover:text-(--red) disabled:opacity-40"
									onclick={leave}
									disabled={busy}>leave contest</button
								>
							{/if}
						{:else if !appState.user}
							<a
								href={`/login?next=${encodeURIComponent(`/contests/${code}`)}`}
								class="mt-4 flex justify-center border border-(--event) px-3 py-2.5 font-mono text-[10px] tracking-[0.1em] uppercase"
								style={`color:${contest.accent}`}>sign in to enroll →</a
							>
						{:else}
							<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/35">
								{seatsLeft} of {contest.max_players} seats remain. The host is already enrolled.
							</p>
							{#if needsPassword}
								<label class="mt-3 flex flex-col gap-1.5">
									<span class="font-mono text-[9px] tracking-[0.12em] text-(--silver)/30 uppercase"
										>private password</span
									>
									<input
										type="password"
										class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--event)"
										bind:value={password}
										placeholder="required"
									/>
								</label>
							{/if}
							<button
								class="mt-3 w-full cursor-pointer border border-(--event) px-3 py-2.5 font-mono text-[10px] tracking-[0.1em] uppercase disabled:opacity-35"
								style={`color:${contest.accent}`}
								onclick={enroll}
								disabled={busy || !canEnroll || (needsPassword && !password)}>{busy ? 'enrolling…' : full ? 'contest full' : 'enroll →'}</button
							>
						{/if}
					{:else if contest.status === 'started'}
						<div class="mt-2 text-2xl text-(--silver)">
							{activeStage ? `Stage ${activeStage.position} is ${isOpenStage(activeStage) ? 'open' : 'live'}.` : 'The clock is running.'}
						</div>
						{#if activeStage}
							<p class="mt-1 font-mono text-[11px]" style={`color:${contest.accent}`}>{activeStage.name}</p>
							<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/35">
								{isOpenStage(activeStage)
									? `Submit as many runs as the stage allows before ${stageDate(activeStage.ends_at)}. Your best run counts.`
									: 'Your heat is racing now. Results advance when the heat closes.'}
							</p>
						{:else}
							<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/35">
								The next stage is being prepared. Keep this page open.
							</p>
						{/if}
						{#if enrolled && viewerGameID}
							<a
								href={resolve(`/game/${viewerGameID}`)}
								class="contest-cta mt-4 flex justify-center border border-(--mer) px-3 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
								>enter your heat →</a
							>
						{:else if enrolled && activeStage && isOpenStage(activeStage) && activeStage.viewer_can_attempt}
							<button
								class="contest-cta mt-4 w-full cursor-pointer border border-(--mer) px-3 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10 disabled:opacity-40"
								onclick={() => attempt(activeStage)}
								disabled={attemptBusy != 0}
								>{attemptBusy ? 'starting…' : 'start attempt →'}</button
							>
						{/if}
					{:else if contest.status === 'ended'}
						<div class="mt-2 text-2xl text-(--silver)">That's a wrap.</div>
						<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/35">
							The final board is on this page. Open a replay from any result to revisit the run.
						</p>
					{:else}
						<div class="mt-2 text-2xl text-(--silver)">This event was cancelled.</div>
						<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/35">
							The host can no longer start this contest.
						</p>
					{/if}
				</div>

				<div class="border border-(--border) bg-(--fbg) p-5">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
						event sheet
					</div>
					<dl class="mt-3 grid grid-cols-2 gap-x-3 gap-y-3 font-mono text-[10px]">
						<div>
							<dt class="text-(--silver)/25">start</dt>
							<dd class="mt-0.5 text-(--silver)/60">{startLabel}</dd>
						</div>
						<div>
							<dt class="text-(--silver)/25">funnel</dt>
							<dd class="mt-0.5 text-(--silver)/60">{contest.stage_count} stages</dd>
						</div>
						<div>
							<dt class="text-(--silver)/25">first stage</dt>
							<dd class="mt-0.5 text-(--silver)/60">
								{contest.stages?.[0] ? formatLabel(contest.stages[0]) : 'live race'}
							</dd>
						</div>
						<div>
							<dt class="text-(--silver)/25">capacity</dt>
							<dd class="mt-0.5 text-(--silver)/60">{contest.max_players} players</dd>
						</div>
						<div>
							<dt class="text-(--silver)/25">entered</dt>
							<dd class="mt-0.5 text-(--silver)/60">{contest.entry_count} players</dd>
						</div>
					</dl>
					<div class="mt-4 flex gap-2">
						<button
							class="flex-1 cursor-pointer border border-(--border) px-3 py-2 font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase hover:text-(--silver)"
							onclick={copyLink}>{copied ? 'copied' : 'copy link'}</button
						>
						<a
							href={contestCalendarURL(contest.code)}
							class="flex-1 border border-(--border) px-3 py-2 text-center font-mono text-[9px] tracking-[0.1em] text-(--silver)/35 uppercase hover:text-(--silver)"
							>calendar</a
						>
					</div>
					<p class="mt-2 break-all font-mono text-[9px] leading-relaxed text-(--silver)/20">
						{contest.share_link}
					</p>
				</div>

				<div class="border border-(--border) bg-(--fbg) p-5">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
						related
					</div>
					<div class="mt-2 flex flex-col gap-1.5 font-mono text-[10px]">
						<a href={resolve('/contests')} class="text-(--silver)/45 hover:text-(--silver)">→ contest board</a>
						<a href="/docs#contests" class="text-(--silver)/45 hover:text-(--silver)">→ contest docs</a>
						<a href={resolve('/leaderboard')} class="text-(--silver)/45 hover:text-(--silver)">→ ranked leaderboard</a>
						<a href={resolve(`/profile/${contest.host.username}`)} class="text-(--silver)/45 hover:text-(--silver)">→ host profile</a>
					</div>
				</div>
			</aside>
		</main>
	{/if}
</div>

<style>
	.contest-md :global(h1),
	.contest-md :global(h2),
	.contest-md :global(h3),
	.contest-md :global(h4) {
		margin: 1.2rem 0 0.45rem;
		color: var(--silver);
		font-family: 'Fira Code', monospace;
		font-size: 0.95rem;
		letter-spacing: 0.04em;
	}
	.contest-md :global(p) {
		margin: 0.65rem 0;
	}
	.contest-md :global(a) {
		color: var(--event);
		text-decoration: underline;
	}
	.contest-md :global(code) {
		border: 1px solid var(--border);
		background: var(--bg);
		padding: 0.05rem 0.3rem;
		font-size: 0.75rem;
	}
	.contest-md :global(ul),
	.contest-md :global(ol) {
		margin: 0.6rem 0;
		padding-left: 1.25rem;
	}
	.contest-md :global(li) {
		margin: 0.25rem 0;
	}
	.contest-md :global(blockquote) {
		margin: 0.7rem 0;
		border-left: 2px solid var(--event);
		padding-left: 0.8rem;
		color: var(--silver);
	}
	.contest-md :global(pre) {
		overflow-x: auto;
		border: 1px solid var(--border);
		background: var(--bg);
		padding: 0.75rem;
		font-size: 0.72rem;
	}
</style>
