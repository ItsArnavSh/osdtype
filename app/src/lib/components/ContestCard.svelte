<script lang="ts">
	import { resolve } from '$app/paths';
	import Avatar from '$lib/components/Avatar.svelte';
	import { contestCalendarURL } from '$lib/core/api/contests';
	import type { ContestView } from '$lib/core/api/types';
	import { LANGUAGE_LABELS } from '$lib/core/api/types';

	let { contest, now }: { contest: ContestView; now: number } = $props();

	let copied = $state(false);
	let starts = $derived(new Date(contest.starts_at).getTime());
	let seconds = $derived(Math.max(0, Math.floor((starts - now) / 1000)));
	let countdown = $derived.by(() => {
		if (contest.status === 'started') return 'live now';
		if (contest.status === 'ended') return 'finished';
		if (contest.status === 'cancelled') return 'cancelled';
		if (seconds <= 0) return 'starting…';
		const days = Math.floor(seconds / 86400);
		const hours = Math.floor((seconds % 86400) / 3600);
		const minutes = Math.floor((seconds % 3600) / 60);
		const secs = seconds % 60;
		if (days > 0) return `in ${days}d ${hours}h`;
		if (hours > 0) return `in ${hours}h ${minutes}m`;
		if (minutes > 0) return `in ${minutes}m ${String(secs).padStart(2, '0')}s`;
		return `in ${secs}s`;
	});
	let startLabel = $derived(
		new Date(contest.starts_at).toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		})
	);
	let preview = $derived(
		contest.description
			.replace(/```[\s\S]*?```/g, '')
			.replace(/[#>*_`[\]()!-]/g, ' ')
			.replace(/\s+/g, ' ')
			.trim()
			.slice(0, 190)
	);
	let statusTone = $derived(
		contest.status === 'started'
			? 'text-(--mer)'
			: contest.status === 'cancelled'
				? 'text-(--red)'
				: 'text-(--silver)/45'
	);
	const STAGE_LABELS: Record<string, string> = {
		qualifier: 'qualifier',
		hotlaps: 'HotLaps',
		multiplayer: 'live race',
		ranked: 'ranked final'
	};
	let firstStage = $derived(contest.stages?.[0] ?? null);
	let funnelLabel = $derived(
		`${contest.stage_count || contest.stages?.length || 1} ${(contest.stage_count || contest.stages?.length || 1) === 1 ? 'stage' : 'stages'} · ${STAGE_LABELS[firstStage?.format ?? 'multiplayer'] ?? 'live race'}`
	);

	async function copyLink() {
		try {
			await navigator.clipboard.writeText(contest.share_link);
			copied = true;
			setTimeout(() => (copied = false), 1400);
		} catch {
			// Clipboard access can be denied; the link is still visible on the
			// event page.
		}
	}
</script>

<article
	class="group flex min-w-0 flex-col border border-(--border) bg-(--fbg) transition-colors hover:border-(--event)"
	style={`--event:${contest.accent}`}
>
	<div class="h-[3px] w-full" style={`background:${contest.accent}`}></div>
	{#if contest.banner_url}
		<a href={resolve(`/contests/${contest.code}`)} class="block overflow-hidden border-b border-(--border)">
			<img
				src={contest.banner_url}
				alt=""
				class="h-28 w-full object-cover opacity-70 transition-opacity group-hover:opacity-90"
			/>
		</a>
	{/if}
	<div class="flex flex-1 flex-col p-4">
		<div class="flex flex-wrap items-center gap-2 font-mono text-[9px] tracking-[0.14em] uppercase">
			<span class={statusTone}>{contest.status}</span>
			<span class="text-(--silver)/20">/</span>
			<span class="text-(--silver)/30">{funnelLabel}</span>
			<span class="text-(--silver)/15">·</span>
			<span class="text-(--silver)/30"
				>{contest.duration}s {LANGUAGE_LABELS[contest.language] ?? contest.language}</span
			>
			{#if contest.visibility === 'private'}
				<span class="ml-auto border border-(--border) px-1.5 py-0.5 text-(--silver)/35">private</span>
			{/if}
		</div>
		<a
			href={resolve(`/contests/${contest.code}`)}
			class="mt-2 line-clamp-2 text-lg leading-tight text-(--silver) hover:text-(--event)"
		>{contest.title}</a
		>
		<p class="mt-2 line-clamp-3 min-h-[3.2rem] font-mono text-[10px] leading-relaxed text-(--silver)/35">
			{preview || 'A typing contest on OSDTYP.'}
		</p>
		<div class="mt-3 flex items-center justify-between gap-2 font-mono text-[10px]">
			<span class="text-(--silver)/45">{countdown}</span>
			<span class="text-(--silver)/25">{startLabel}</span>
		</div>
		<div class="mt-2 flex items-center justify-between gap-2 border-t border-(--border) pt-3">
			<a
				href={resolve(`/profile/${contest.host.username}`)}
				class="flex min-w-0 items-center gap-1.5 font-mono text-[10px] text-(--silver)/40 hover:text-(--silver)"
			>
				<Avatar
					src={contest.host.avatar_url}
					name={contest.host.username}
					class="h-4 w-4 shrink-0"
					text="text-[7px]"
				/>
				<span class="truncate">{contest.host.username}</span>
			</a>
			<span class="shrink-0 font-mono text-[10px] text-(--silver)/30">
				{contest.entry_count}/{contest.max_players}
			</span>
		</div>
		<div class="mt-3 flex flex-wrap items-center gap-2">
			<a
				href={resolve(`/contests/${contest.code}`)}
				class="border border-(--event) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] uppercase"
				style={`color:${contest.accent}`}
				>open event →</a
			>
			<a
				href={contestCalendarURL(contest.code)}
				class="border border-(--border) px-2.5 py-1.5 font-mono text-[10px] tracking-[0.08em] text-(--silver)/35 uppercase hover:text-(--silver)"
				title="download calendar file">calendar</a
			>
			<button
				class="ml-auto cursor-pointer border border-(--border) px-2.5 py-1.5 font-mono text-[10px] tracking-[0.08em] text-(--silver)/35 uppercase hover:text-(--silver)"
				onclick={copyLink}>{copied ? 'copied' : 'share'}</button
			>
		</div>
	</div>
</article>
