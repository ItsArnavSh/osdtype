<script lang="ts">
	import { resolve } from '$app/paths';
	import Avatar from '$lib/components/Avatar.svelte';
	import { errorMessage } from '$lib/core/api/client';
	import {
		deleteRunHistory,
		exportAccountData,
		profileSettings,
		updateProfileSettings
	} from '$lib/core/api/users';
	import type { ProfileSettings, ProfileSettingsInput, ProfileVisibility, RunSharing } from '$lib/core/api/types';
	import { appState, refreshState } from '$lib/core/state/state.svelte';
	import { onMount } from 'svelte';

	let draft = $state<ProfileSettings | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let exporting = $state(false);
	let deleting = $state(false);
	let confirmDelete = $state(false);
	let error = $state('');
	let notice = $state('');
	let savedAt = $state<Date | null>(null);

	const visibilityOptions: { value: ProfileVisibility; label: string; copy: string; tag: string }[] = [
		{
			value: 'public',
			label: 'public',
			copy: 'Your profile, stats and shared runs are visible to everyone.',
			tag: 'everyone'
		},
		{
			value: 'anonymous',
			label: 'anonymous',
			copy: 'Hide your name, avatar, bio and online dot behind an anonymous profile. Your handle can still appear on ladders and in rooms.',
			tag: 'masked card'
		},
		{
			value: 'private',
			label: 'private',
			copy: 'Only you can open your profile. Other players get a not-found page.',
			tag: 'only you'
		}
	];

	const sharingOptions: { value: RunSharing; label: string; copy: string }[] = [
		{ value: 'all', label: 'all runs', copy: 'Show every saved run that matches the modes below.' },
		{
			value: 'pinned',
			label: 'pinned only',
			copy: 'Only runs you pin appear on your public profile.'
		},
		{ value: 'none', label: 'none', copy: 'Hide your run history and best run from everyone.' }
	];

	const modeOptions = [
		{ key: 'share_ranked' as const, label: 'ranked', copy: 'ladder races' },
		{ key: 'share_rooms' as const, label: 'rooms', copy: 'public and private lobbies' },
		{ key: 'share_practice' as const, label: 'practice', copy: 'saved practice runs' },
		{ key: 'share_contests' as const, label: 'contests', copy: 'contest attempts' }
	];

	const displayName = $derived(draft?.display_name.trim() || draft?.username || '');
	const bio = $derived(draft?.bio.trim() || '');
	const chosenVisibility = $derived(visibilityOptions.find((o) => o.value === draft?.profile_visibility));
	const previewAnonymous = $derived(draft?.profile_visibility === 'anonymous');

	async function load() {
		loading = true;
		error = '';
		try {
			draft = await profileSettings();
		} catch (err) {
			error = errorMessage(err);
			draft = null;
		} finally {
			loading = false;
		}
	}

	async function save() {
		if (!draft || saving) return;
		saving = true;
		error = '';
		notice = '';
		try {
			const payload: ProfileSettingsInput = {
				display_name: draft.display_name.trim(),
				bio: draft.bio.trim(),
				profile_visibility: draft.profile_visibility,
				do_not_disturb: draft.do_not_disturb,
				hide_online: draft.hide_online,
				share_runs: draft.share_runs,
				share_ranked: draft.share_ranked,
				share_rooms: draft.share_rooms,
				share_practice: draft.share_practice,
				share_contests: draft.share_contests,
				hide_private_rooms: draft.hide_private_rooms,
				share_replays: draft.share_replays,
				save_run_data: draft.save_run_data
			};
			draft = await updateProfileSettings(payload);
			savedAt = new Date();
			notice = 'settings saved';
			if (appState.user && draft) {
				appState.user.display_name = draft.display_name;
				appState.user.bio = draft.bio;
				appState.user.save_run_data = draft.save_run_data;
			}
			void refreshState();
		} catch (err) {
			error = errorMessage(err);
		} finally {
			saving = false;
		}
	}

	async function downloadExport() {
		if (!draft || exporting) return;
		exporting = true;
		error = '';
		try {
			const data = await exportAccountData();
			const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = `osdtyp-${draft.username}-export.json`;
			link.click();
			URL.revokeObjectURL(url);
			notice = 'export downloaded';
		} catch (err) {
			error = errorMessage(err);
		} finally {
			exporting = false;
		}
	}

	async function removeHistory() {
		if (deleting || !confirmDelete) return;
		deleting = true;
		error = '';
		try {
			const result = await deleteRunHistory();
			confirmDelete = false;
			notice =
				result.deleted === 0
					? 'no unpinned runs to delete'
					: `deleted ${result.deleted} unpinned ${result.deleted === 1 ? 'run' : 'runs'}`;
		} catch (err) {
			error = errorMessage(err);
		} finally {
			deleting = false;
		}
	}

	function toggle(key: 'do_not_disturb' | 'hide_online' | 'hide_private_rooms' | 'share_replays' | 'save_run_data') {
		if (!draft) return;
		draft[key] = !draft[key];
		notice = '';
	}

	function toggleMode(key: 'share_ranked' | 'share_rooms' | 'share_practice' | 'share_contests') {
		if (!draft) return;
		draft[key] = !draft[key];
		notice = '';
	}

	function setVisibility(value: ProfileVisibility) {
		if (!draft) return;
		draft.profile_visibility = value;
		notice = '';
	}

	function setSharing(value: RunSharing) {
		if (!draft) return;
		draft.share_runs = value;
		notice = '';
	}

	onMount(load);
</script>

<svelte:head><title>settings · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg)">
	<header class="border-b border-(--border)">
		<div
			class="mx-auto flex w-full max-w-6xl flex-wrap items-end justify-between gap-4 px-5 py-5 md:px-8"
		>
			<div>
				<a
					href={resolve('/play')}
					class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
					>← hub</a
				>
				<div class="mt-1 flex flex-wrap items-baseline gap-3">
					<h1 class="title-text text-5xl leading-none text-(--silver) md:text-6xl">SETTINGS</h1>
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/70 uppercase"
						>account · privacy · data</span
					>
				</div>
			</div>
			{#if draft}
				<a
					href={resolve(`/profile/${draft.username}`)}
					class="border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
					>view profile →</a
				>
			{/if}
		</div>
	</header>

	<main class="mx-auto w-full max-w-6xl flex-1 px-5 py-5 pb-24 md:px-8">
		{#if error}
			<div
				class="mb-4 flex items-center justify-between gap-3 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)"
			>
				<span>{error}</span>
				<button class="cursor-pointer text-(--red)/60 hover:text-(--red)" onclick={load}
					>retry</button
				>
			</div>
		{/if}
		{#if notice}
			<div
				class="mb-4 border border-(--mer)/40 bg-(--mer)/5 px-4 py-2 font-mono text-xs text-(--mer)"
			>
				{notice}
			</div>
		{/if}

		{#if loading}
			<div
				class="border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35"
			>
				loading account settings…
			</div>
		{:else if draft}
			<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
				<div class="flex min-w-0 flex-col gap-4">
					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) px-5 py-3">
							<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
								01 · identity
							</div>
							<p class="mt-1 font-mono text-[11px] text-(--silver)/40">
								How you appear when someone opens your profile.
							</p>
						</div>
						<div class="grid gap-5 p-5 md:grid-cols-[minmax(0,1fr)_16rem]">
							<div class="flex flex-col gap-4">
								<label class="block">
									<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
										>display name</span
									>
									<input
										class="mt-2 w-full border border-(--border) bg-(--bg) px-3 py-2 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
										maxlength="40"
										bind:value={draft.display_name}
										placeholder={draft.username}
										oninput={() => (notice = '')}
									/>
									<span class="mt-1 block text-right font-mono text-[9px] text-(--silver)/25"
										>{draft.display_name.length}/40 · optional</span
									>
								</label>
								<label class="block">
									<span class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase"
										>bio</span
									>
									<textarea
										class="mt-2 min-h-28 w-full resize-y border border-(--border) bg-(--bg) px-3 py-2 font-mono text-sm leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
										maxlength="280"
										bind:value={draft.bio}
										placeholder="a line about how you type, what you build, or nothing at all"
										oninput={() => (notice = '')}
									></textarea>
									<span class="mt-1 block text-right font-mono text-[9px] text-(--silver)/25"
										>{draft.bio.length}/280</span
									>
								</label>
							</div>

							<div class="border border-(--border) bg-(--bg)/55 p-4">
								<div class="flex items-center justify-between gap-2">
									<span class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase"
										>profile preview</span
									>
									<span class="border border-(--mer)/40 px-1.5 py-0.5 font-mono text-[8px] tracking-[0.08em] text-(--mer) uppercase">
										{previewAnonymous ? 'anonymous' : chosenVisibility?.tag}
									</span>
								</div>
								<div class="mt-4 flex items-center gap-3">
									<Avatar
										src={previewAnonymous ? '' : (appState.user?.avatar_url ?? '')}
										name={previewAnonymous ? 'anonymous' : displayName}
										class="h-12 w-12"
										text="text-base"
									/>
									<div class="min-w-0">
										<div class="truncate text-lg text-(--silver)">
											{previewAnonymous ? 'anonymous' : displayName}
										</div>
										{#if !previewAnonymous}
											<div class="truncate font-mono text-[10px] text-(--silver)/35">
												@{draft.username}
											</div>
										{/if}
									</div>
								</div>
								<p class="mt-3 line-clamp-4 font-mono text-[11px] leading-relaxed text-(--silver)/45">
									{previewAnonymous ? 'name, avatar, bio and online status are hidden.' : bio || 'no bio yet.'}
								</p>
							</div>
						</div>
					</section>

					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) px-5 py-3">
							<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
								02 · profile visibility
							</div>
							<p class="mt-1 font-mono text-[11px] text-(--silver)/40">
								Choose how much of your profile card other people can see.
							</p>
						</div>
						<div class="grid gap-2 p-4 md:grid-cols-3">
							{#each visibilityOptions as option (option.value)}
								<button
									type="button"
									class={`flex min-h-32 cursor-pointer flex-col border p-4 text-left transition-colors ${draft.profile_visibility === option.value ? 'border-(--mer) bg-(--mer)/5' : 'border-(--border) hover:border-(--silver)/30'}`}
									onclick={() => setVisibility(option.value)}
								>
									<div class="flex items-center justify-between gap-2">
										<span class="text-lg text-(--silver)">{option.label}</span>
										<span
											class={`flex h-5 w-5 items-center justify-center border font-mono text-[11px] ${draft.profile_visibility === option.value ? 'border-(--mer) text-(--mer)' : 'border-(--border) text-transparent'}`}
											aria-hidden="true">✓</span
										>
									</div>
									<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/45">
										{option.copy}
									</p>
									<span class="mt-auto pt-3 font-mono text-[9px] tracking-[0.1em] text-(--mer)/55 uppercase">
										{option.tag}
									</span>
								</button>
							{/each}
						</div>
						{#if draft.profile_visibility === 'anonymous'}
							<p class="border-t border-(--border) px-5 py-3 font-mono text-[10px] leading-relaxed text-(--silver)/35">
								anonymous is a profile card setting, not a full identity cloak: your handle can still
								show on leaderboards, rooms and contest results.
							</p>
						{/if}
					</section>

					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) px-5 py-3">
							<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
								03 · contact & presence
							</div>
						</div>
						<div class="flex flex-col divide-y divide-(--border)">
							<div class="flex items-start justify-between gap-4 p-5">
								<div>
									<div class="text-lg text-(--silver)">do not disturb</div>
									<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/45">
										Block room invitations from other players. You can still join rooms yourself or use
										an invite link.
									</p>
								</div>
								<button
									type="button"
									role="switch"
									aria-checked={draft.do_not_disturb}
									aria-label="Do not disturb"
									class={`relative mt-1 h-6 w-11 shrink-0 cursor-pointer border transition-colors ${draft.do_not_disturb ? 'border-(--mer) bg-(--mer)/15' : 'border-(--border) bg-(--bg)'}`}
									onclick={() => toggle('do_not_disturb')}
								>
									<span
										class={`absolute top-0.5 left-0.5 h-4 w-4 transition-transform ${draft.do_not_disturb ? 'translate-x-5 bg-(--mer)' : 'translate-x-0 bg-(--silver)/30'}`}
									></span>
								</button>
							</div>
							<div class="flex items-start justify-between gap-4 p-5">
								<div>
									<div class="text-lg text-(--silver)">appear offline</div>
									<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/45">
										Hide your online dot on your profile and friends list. You can still receive invites
										unless do not disturb is on.
									</p>
								</div>
								<button
									type="button"
									role="switch"
									aria-checked={draft.hide_online}
									aria-label="Appear offline"
									class={`relative mt-1 h-6 w-11 shrink-0 cursor-pointer border transition-colors ${draft.hide_online ? 'border-(--mer) bg-(--mer)/15' : 'border-(--border) bg-(--bg)'}`}
									onclick={() => toggle('hide_online')}
								>
									<span
										class={`absolute top-0.5 left-0.5 h-4 w-4 transition-transform ${draft.hide_online ? 'translate-x-5 bg-(--mer)' : 'translate-x-0 bg-(--silver)/30'}`}
									></span>
								</button>
							</div>
						</div>
					</section>

					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) px-5 py-3">
							<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
								04 · who sees your runs
							</div>
							<p class="mt-1 font-mono text-[11px] text-(--silver)/40">
								These switches control your public profile. You always see your own runs.
							</p>
						</div>
						<div class="grid gap-2 p-4 md:grid-cols-3">
							{#each sharingOptions as option (option.value)}
								<button
									type="button"
									class={`flex min-h-28 cursor-pointer flex-col border p-4 text-left transition-colors ${draft.share_runs === option.value ? 'border-(--mer) bg-(--mer)/5' : 'border-(--border) hover:border-(--silver)/30'}`}
									onclick={() => setSharing(option.value)}
								>
									<div class="flex items-center justify-between gap-2">
										<span class="text-lg text-(--silver)">{option.label}</span>
										<span
											class={`flex h-5 w-5 items-center justify-center border font-mono text-[11px] ${draft.share_runs === option.value ? 'border-(--mer) text-(--mer)' : 'border-(--border) text-transparent'}`}
											aria-hidden="true">✓</span
										>
									</div>
									<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/45">
										{option.copy}
									</p>
								</button>
							{/each}
						</div>

						<div class="border-t border-(--border) p-5">
							<div class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/40 uppercase">
								modes
							</div>
							<div class="mt-3 grid gap-2 sm:grid-cols-2">
								{#each modeOptions as mode (mode.key)}
									<button
										type="button"
										class={`flex cursor-pointer items-center gap-3 border px-3 py-2 text-left transition-colors ${draft[mode.key] ? 'border-(--mer)/45 text-(--silver)' : 'border-(--border) text-(--silver)/40'}`}
										onclick={() => toggleMode(mode.key)}
										aria-pressed={draft[mode.key]}
									>
										<span
											class={`flex h-5 w-5 shrink-0 items-center justify-center border font-mono text-[11px] ${draft[mode.key] ? 'border-(--mer) text-(--mer)' : 'border-(--border) text-transparent'}`}
											aria-hidden="true">✓</span
										>
										<span class="min-w-0">
											<span class="block font-mono text-xs">{mode.label}</span>
											<span class="block font-mono text-[9px] text-(--silver)/30">{mode.copy}</span>
										</span>
									</button>
								{/each}
							</div>
							<p class="mt-3 font-mono text-[9px] leading-relaxed text-(--silver)/30">
								Mode ticks apply when runs are shared. With “pinned only”, only pinned runs in the
								selected modes appear.
							</p>
						</div>

						<div class="flex flex-col divide-y divide-(--border) border-t border-(--border)">
							<div class="flex items-start justify-between gap-4 px-5 py-4">
								<div>
									<div class="font-mono text-xs text-(--silver)">hide private-room runs</div>
									<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/40">
										Keep runs from password-protected or invite-only rooms off your public profile,
										even when room sharing is on.
									</p>
								</div>
								<button
									type="button"
									role="switch"
									aria-checked={draft.hide_private_rooms}
									aria-label="Hide private-room runs"
									class={`relative mt-1 h-6 w-11 shrink-0 cursor-pointer border transition-colors ${draft.hide_private_rooms ? 'border-(--mer) bg-(--mer)/15' : 'border-(--border) bg-(--bg)'}`}
									onclick={() => toggle('hide_private_rooms')}
								>
									<span
										class={`absolute top-0.5 left-0.5 h-4 w-4 transition-transform ${draft.hide_private_rooms ? 'translate-x-5 bg-(--mer)' : 'translate-x-0 bg-(--silver)/30'}`}
									></span>
								</button>
							</div>
							<div class="flex items-start justify-between gap-4 px-5 py-4">
								<div>
									<div class="font-mono text-xs text-(--silver)">share replay links</div>
									<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/40">
										Allow anyone with a run’s replay link to watch it. Turning this off still shows
										your run cards, without replay links.
									</p>
								</div>
								<button
									type="button"
									role="switch"
									aria-checked={draft.share_replays}
									aria-label="Share replay links"
									class={`relative mt-1 h-6 w-11 shrink-0 cursor-pointer border transition-colors ${draft.share_replays ? 'border-(--mer) bg-(--mer)/15' : 'border-(--border) bg-(--bg)'}`}
									onclick={() => toggle('share_replays')}
								>
									<span
										class={`absolute top-0.5 left-0.5 h-4 w-4 transition-transform ${draft.share_replays ? 'translate-x-5 bg-(--mer)' : 'translate-x-0 bg-(--silver)/30'}`}
									></span>
								</button>
							</div>
						</div>
					</section>

					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) px-5 py-3">
							<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
								05 · data & retention
							</div>
							<p class="mt-1 font-mono text-[11px] text-(--silver)/40">
								OSDTYP keeps only what you allow. These controls apply to new runs.
							</p>
						</div>
						<div class="flex flex-col divide-y divide-(--border)">
							<div class="flex items-start justify-between gap-4 p-5">
								<div>
									<div class="flex flex-wrap items-center gap-2">
										<span class="text-lg text-(--silver)">save run data</span>
										<span
											class={`border px-1.5 py-0.5 font-mono text-[8px] tracking-[0.1em] uppercase ${draft.save_run_data ? 'border-(--mer)/35 text-(--mer)/70' : 'border-(--red)/40 text-(--red)/75'}`}
										>
											{draft.save_run_data ? 'saving' : 'save no data'}
										</span>
									</div>
									<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/45">
										When off, new runs still count for Elo and produce a result, but OSDTYP keeps no
										history entry and no replay keystrokes for them. Existing runs stay until you
										delete them below.
									</p>
								</div>
								<button
									type="button"
									role="switch"
									aria-checked={draft.save_run_data}
									aria-label="Save run data"
									class={`relative mt-1 h-6 w-11 shrink-0 cursor-pointer border transition-colors ${draft.save_run_data ? 'border-(--mer) bg-(--mer)/15' : 'border-(--red)/50 bg-(--red)/10'}`}
									onclick={() => toggle('save_run_data')}
								>
									<span
										class={`absolute top-0.5 left-0.5 h-4 w-4 transition-transform ${draft.save_run_data ? 'translate-x-5 bg-(--mer)' : 'translate-x-0 bg-(--red)/70'}`}
									></span>
								</button>
							</div>

							<div class="p-5">
								<div class="flex flex-wrap items-baseline justify-between gap-2">
									<div class="font-mono text-xs text-(--silver)">retention</div>
									<span class="font-mono text-[9px] tracking-[0.1em] text-(--mer)/65 uppercase"
										>automatic</span
									>
								</div>
								<p class="mt-1 font-mono text-[10px] leading-relaxed text-(--silver)/45">
									Keep the latest <span class="text-(--silver)">{draft.retention_limit}</span> unpinned runs,
									plus your <span class="text-(--silver)">best run</span> and every pinned run. Everything
									else is pruned regularly with its replay data.
								</p>
							</div>

							<div class="grid gap-3 p-5 sm:grid-cols-2">
								<div class="border border-(--border) bg-(--bg)/45 p-4">
									<div class="font-mono text-[10px] tracking-[0.1em] text-(--silver)/40 uppercase">
										export
									</div>
									<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/40">
										Download your profile, settings, retained run history and saved replay events as
										JSON.
									</p>
									<button
										type="button"
										class="mt-4 w-full cursor-pointer border border-(--mer) px-3 py-2 font-mono text-[11px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10 disabled:opacity-40"
										onclick={downloadExport}
										disabled={exporting}
									>
										{exporting ? 'preparing…' : 'export my data ↓'}
									</button>
								</div>
								<div class="border border-(--red)/30 bg-(--red)/5 p-4">
									<div class="font-mono text-[10px] tracking-[0.1em] text-(--red)/75 uppercase">
										delete
									</div>
									<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/40">
										Remove every unpinned run except your best run and its replay data. Pinned runs
										are always kept.
									</p>
									{#if confirmDelete}
										<div class="mt-4 grid gap-2 sm:grid-cols-2">
											<button
												type="button"
												class="cursor-pointer border border-(--red) px-3 py-2 font-mono text-[10px] tracking-[0.08em] text-(--red) uppercase hover:bg-(--red)/10 disabled:opacity-40"
												onclick={removeHistory}
												disabled={deleting}>{deleting ? 'deleting…' : 'yes, delete'}</button
											>
											<button
												type="button"
												class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] tracking-[0.08em] text-(--silver)/45 uppercase hover:text-(--silver)"
												onclick={() => (confirmDelete = false)}>cancel</button
											>
										</div>
									{:else}
										<button
											type="button"
											class="mt-4 w-full cursor-pointer border border-(--red)/50 px-3 py-2 font-mono text-[11px] tracking-[0.1em] text-(--red)/80 uppercase hover:bg-(--red)/10"
											onclick={() => (confirmDelete = true)}>delete unpinned runs</button
										>
									{/if}
								</div>
							</div>
						</div>
					</section>
				</div>

				<aside class="flex flex-col gap-3">
					<section class="border border-(--border) bg-(--fbg) p-5">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer)/65 uppercase">
							at a glance
						</div>
						<dl class="mt-4 flex flex-col gap-3 font-mono text-[11px]">
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-(--silver)/35">profile</dt>
								<dd class="text-right text-(--silver)">{draft.profile_visibility}</dd>
							</div>
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-(--silver)/35">runs</dt>
								<dd class="text-right text-(--silver)">{draft.share_runs}</dd>
							</div>
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-(--silver)/35">replay links</dt>
								<dd class="text-right text-(--silver)">{draft.share_replays ? 'on' : 'off'}</dd>
							</div>
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-(--silver)/35">invites</dt>
								<dd class="text-right text-(--silver)">{draft.do_not_disturb ? 'blocked' : 'open'}</dd>
							</div>
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-(--silver)/35">new run data</dt>
								<dd class={`text-right ${draft.save_run_data ? 'text-(--silver)' : 'text-(--red)/80'}`}>
									{draft.save_run_data ? 'saved' : 'not saved'}
								</dd>
							</div>
						</dl>
					</section>

					<section class="border border-(--border) bg-(--fbg) p-5">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							privacy notes
						</div>
						<ul class="mt-3 flex flex-col gap-2 font-mono text-[10px] leading-relaxed text-(--silver)/40">
							<li class="flex gap-2"><span class="text-(--mer)">·</span>Private profiles return not found to everyone else.</li>
							<li class="flex gap-2"><span class="text-(--mer)">·</span>Anonymous masks the profile card; your handle can still appear on ladders.</li>
							<li class="flex gap-2"><span class="text-(--mer)">·</span>No-save keeps rating integrity but drops history and replay keystrokes.</li>
							<li class="flex gap-2"><span class="text-(--mer)">·</span>Your best run and pinned runs are exempt from automatic retention.</li>
						</ul>
					</section>

					{#if savedAt}
						<p class="font-mono text-[9px] text-(--silver)/25">
							last saved {savedAt.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })}
						</p>
					{/if}
				</aside>
			</div>
		{:else}
			<div
				class="border border-(--border) bg-(--fbg) p-12 text-center font-mono text-xs text-(--silver)/35"
			>
				Could not load settings.
			</div>
		{/if}
	</main>

	{#if draft}
		<div
			class="sticky bottom-0 z-20 border-t border-(--border) bg-(--bg)/95 px-5 py-3 backdrop-blur md:px-8"
		>
			<div class="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-3">
				<p class="font-mono text-[10px] leading-relaxed text-(--silver)/30">
					{draft.save_run_data
						? 'saving new run history · pinned runs always kept'
						: 'save no data · new run history and replay keystrokes are dropped'}
				</p>
				<button
					type="button"
					class="cursor-pointer border border-(--mer) bg-(--mer)/10 px-5 py-2.5 font-mono text-[11px] tracking-[0.12em] text-(--mer) uppercase hover:bg-(--mer) hover:text-(--bg) disabled:opacity-40"
					onclick={save}
					disabled={saving}
				>
					{saving ? 'saving…' : 'save settings'}
				</button>
			</div>
		</div>
	{/if}
</div>
