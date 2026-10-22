<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onDestroy, onMount } from 'svelte';
	import Avatar from '$lib/components/Avatar.svelte';
	import { errorMessage } from '$lib/core/api/client';
	import {
		addRoomBot,
		banFromRoom,
		closeRoom,
		createContest,
		demoteInRoom,
		getRoom,
		inviteToRoom,
		joinRoom,
		kickFromRoom,
		leaveRoom,
		listContests,
		promoteInRoom,
		startRoomGame,
		updateRoom
	} from '$lib/core/api/rooms';
	import { onSocketEvent } from '$lib/core/api/socket';
	import type { ContestView, LobbyDuration, RoomView } from '$lib/core/api/types';
	import { LANGUAGE_LABELS, LOBBY_DURATIONS } from '$lib/core/api/types';
	import { LANGUAGES, type Language } from '$lib/core/entity/languages';
	import { appState, refreshState } from '$lib/core/state/state.svelte';

	const code = $derived((page.params.code ?? '').toUpperCase());

	let room = $state<RoomView | null>(null);
	let contests = $state<ContestView[]>([]);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');
	let notice = $state('');
	let inviteName = $state('');
	let password = $state('');
	let settingsOpen = $state(false);
	let contestOpen = $state(false);
	let unsubscribe: (() => void) | null = null;
	let clockTimer: ReturnType<typeof setInterval> | null = null;

	let settings = $state({
		name: '',
		description: '',
		is_public: true,
		max_players: 8,
		duration: 90 as LobbyDuration,
		language: 'go' as Language,
		password: ''
	});
	let contestForm = $state({
		title: '',
		description: '',
		minutes: 5,
		duration: 90 as LobbyDuration,
		language: 'go' as Language
	});

	let isMember = $derived(Boolean(room?.my_role));
	let isOwner = $derived(room?.my_role === 'owner');
	let canManage = $derived(room?.my_role === 'owner' || room?.my_role === 'mod');
	let canStart = $derived(canManage && (room?.member_count ?? 0) >= 2);
	let myID = $derived(appState.user?.id ?? 0);

	function fillSettings(value: RoomView) {
		settings = {
			name: value.name,
			description: value.description,
			is_public: value.is_public,
			max_players: value.max_players,
			duration: value.duration as LobbyDuration,
			language: value.language as Language,
			password: ''
		};
	}

	async function load(silent = false) {
		if (!silent) loading = true;
		try {
			const next = await getRoom(code);
			room = next;
			fillSettings(next);
			if (next.my_role) contests = await listContests(code);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			if (!silent) loading = false;
		}
	}

	async function refreshContests() {
		try {
			contests = await listContests(code);
		} catch {
			// Keep the last known list; the next room update will retry.
		}
	}

	async function join() {
		if (busy || !room) return;
		busy = true;
		error = '';
		try {
			room = await joinRoom(code, room.has_password ? password : '');
			await refreshState();
			await load(true);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function leave() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			await leaveRoom(code);
			await refreshState();
			await goto(resolve('/play'));
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function close() {
		if (busy || !confirm('Close this room for everyone?')) return;
		busy = true;
		try {
			await closeRoom(code);
			await refreshState();
			await goto(resolve('/play'));
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function start() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			const game = await startRoomGame(code);
			await refreshState();
			await goto(`/game/${game.id}`);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function invite() {
		const username = inviteName.trim();
		if (!username) return;
		busy = true;
		error = '';
		notice = '';
		try {
			await inviteToRoom(code, username);
			inviteName = '';
			notice = `Invite sent to ${username}.`;
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function addBot(persona: 'LEWIS' | 'MAX' | 'LANCE') {
		busy = true;
		error = '';
		try {
			room = await addRoomBot(code, persona);
			fillSettings(room);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function memberAction(action: 'kick' | 'ban' | 'promote' | 'demote', userID: number) {
		busy = true;
		error = '';
		try {
			if (action === 'kick') await kickFromRoom(code, userID);
			else if (action === 'ban') await banFromRoom(code, userID);
			else if (action === 'promote') await promoteInRoom(code, userID);
			else await demoteInRoom(code, userID);
			await load(true);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function saveSettings() {
		busy = true;
		error = '';
		notice = '';
		try {
			room = await updateRoom(code, {
				name: settings.name.trim(),
				description: settings.description.trim(),
				is_public: settings.is_public,
				max_players: Number(settings.max_players),
				duration: settings.duration,
				language: settings.language,
				...(settings.password ? { password: settings.password } : {})
			});
			fillSettings(room);
			settingsOpen = false;
			notice = 'Room settings saved.';
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function makeContest() {
		busy = true;
		error = '';
		notice = '';
		try {
			const starts = new Date(Date.now() + contestForm.minutes * 60_000);
			await createContest(code, {
				title: contestForm.title.trim(),
				description: contestForm.description.trim(),
				starts_at: starts.toISOString(),
				duration: contestForm.duration,
				language: contestForm.language
			});
			contests = await listContests(code);
			contestOpen = false;
			notice = 'Contest scheduled.';
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function copyLink() {
		const link = room?.invite_link || `${location.origin}/join/${code}`;
		try {
			await navigator.clipboard.writeText(link);
			notice = 'Invite link copied.';
		} catch {
			notice = link;
		}
	}

	function roleLabel(role: string): string {
		return role === 'owner' ? 'host' : role;
	}

	function fmtDate(value: string): string {
		const date = new Date(value);
		return date.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
	}

	async function backToHub() {
		// Let authoritative state settle before navigating. Navigating first can
		// bounce off the stale room guard and strand the user on a dead join page.
		await refreshState();
		await goto(resolve('/play'));
	}

	async function enterGame(id: number) {
		await refreshState();
		await goto(`/game/${id}`);
	}

	onMount(() => {
		clockTimer = setInterval(() => {}, 60_000);
		unsubscribe = onSocketEvent((event) => {
			if (event.type === 'room_update' && event.data) {
				const next = event.data as RoomView;
				if (next.code === code) {
					room = next;
					if (isOwner) fillSettings(next);
					if (next.my_role) void refreshContests();
				}
			}
			if (event.type === 'game_start') {
				const game = event.data as { id?: number } | null;
				if (game?.id) void enterGame(game.id);
			}
			if (event.type === 'room_closed') {
				const data = event.data as { room_code?: string } | null;
				if (!data?.room_code || data.room_code === code) void backToHub();
			}
			if (event.type === 'room_left') {
				const data = event.data as { room_code?: string } | null;
				if (!data?.room_code || data.room_code === code) void backToHub();
			}
		});
		void load();
		return () => {
			unsubscribe?.();
			if (clockTimer) clearInterval(clockTimer);
		};
	});

	onDestroy(() => {
		unsubscribe?.();
		if (clockTimer) clearInterval(clockTimer);
	});
</script>

<svelte:head><title>{room?.name ?? code} · room · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-6 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex items-end gap-4">
			<a
				href={resolve('/play/rooms')}
				class="mb-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← rooms</a
			>
			<div>
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
					{room?.status === 'playing' ? 'in game' : 'lobby'} · {code}
				</div>
				<div class="title-text text-6xl leading-none text-(--silver) md:text-7xl">
					{room?.name ?? 'ROOM'}
				</div>
			</div>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#if isMember}
				<button
					class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[11px] text-(--silver)/55 hover:border-(--silver)/40 hover:text-(--silver)"
					onclick={copyLink}>copy invite link</button
				>
				{#if isOwner}
					<button
						class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[11px] text-(--silver)/55 hover:border-(--silver)/40 hover:text-(--silver)"
						onclick={() => (settingsOpen = !settingsOpen)}>settings</button
					>
					<button
						class="cursor-pointer border border-(--red)/40 px-3 py-1.5 font-mono text-[11px] text-(--red)/70 hover:text-(--red)"
						onclick={close}>close room</button
					>
				{/if}
				<button
					class="cursor-pointer border border-(--border) px-3 py-1.5 font-mono text-[11px] text-(--silver)/55 hover:border-(--red) hover:text-(--red) disabled:opacity-40"
					onclick={leave}
					disabled={busy}>leave</button
				>
			{/if}
		</div>
	</header>

	{#if loading}
		<div class="mt-6 flex flex-1 items-center justify-center border border-(--border) bg-(--fbg)">
			<span class="font-mono text-xs text-(--silver)/35">loading room…</span>
		</div>
	{:else if !room}
		<div class="mt-6 border border-(--red)/40 p-6 font-mono text-xs text-(--red)">{error}</div>
	{:else if !isMember}
		<div class="mx-auto mt-12 w-full max-w-md border border-(--border) bg-(--fbg)">
			<div class="border-b border-(--border) px-6 py-4">
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">join room</div>
				<div class="mt-2 text-2xl text-(--silver)">{room.name}</div>
				<div class="mt-1 font-mono text-xs text-(--silver)/40">
					{room.member_count}/{room.max_players} players · {room.duration}s ·
					{LANGUAGE_LABELS[room.language] ?? room.language}
				</div>
			</div>
			<div class="flex flex-col gap-3 p-6">
				{#if room.has_password}
					<input
						type="password"
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
						placeholder="room password"
						bind:value={password}
					/>
				{/if}
				{#if error}
					<div class="border border-(--red)/40 px-3 py-2 font-mono text-xs text-(--red)">{error}</div>
				{/if}
				<button
					class="cursor-pointer border border-(--mer) px-4 py-2.5 font-mono text-sm text-(--mer) hover:bg-(--mer)/10 disabled:opacity-40"
					onclick={join}
					disabled={busy}
					>{busy ? 'joining…' : 'join room =>'}</button
				>
			</div>
		</div>
	{:else}
		<div class="mt-4 flex flex-wrap items-center justify-between gap-3 border-y border-(--border) py-2 font-mono text-[10px] text-(--silver)/35">
			<div class="flex flex-wrap items-center gap-4">
				<span>{room.member_count}/{room.max_players} players</span>
				<span>{room.duration}s</span>
				<span>{LANGUAGE_LABELS[room.language] ?? room.language}</span>
				<span>{room.is_public ? 'public' : 'private'}{room.has_password ? ' · locked' : ''}</span>
			</div>
			<div class="flex items-center gap-3">
				<span class="text-(--mer)">{room.my_role === 'owner' ? 'host' : room.my_role}</span>
				<span>code {code}</span>
			</div>
		</div>

		{#if error}
			<div class="mt-3 border border-(--red)/40 px-4 py-2 font-mono text-xs text-(--red)">{error}</div>
		{/if}
		{#if notice}
			<button
				class="mt-3 w-full cursor-pointer border border-(--mer)/40 px-4 py-2 text-left font-mono text-xs text-(--mer)"
				onclick={() => (notice = '')}>{notice}</button
			>
		{/if}

		<main class="mt-4 grid flex-1 gap-3 lg:grid-cols-[1fr_340px]">
			<section class="border border-(--border) bg-(--fbg)">
				<div class="flex items-center justify-between border-b border-(--border) px-5 py-3">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/40 uppercase">
						members
					</div>
					<div class="flex items-center gap-2">
						{#if canManage}
							{#each ['LEWIS', 'MAX', 'LANCE'] as persona (persona)}
								<button
									class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[9px] tracking-[0.1em] text-(--silver)/45 uppercase hover:border-(--mer) hover:text-(--mer) disabled:opacity-40"
									onclick={() => addBot(persona as 'LEWIS' | 'MAX' | 'LANCE')}
									disabled={busy}>+ {persona}</button
								>
							{/each}
						{/if}
					</div>
				</div>

				<div class="divide-y divide-(--border)">
					{#each room.members as member (member.user_id)}
						<div class="flex flex-wrap items-center gap-3 px-5 py-3">
							<Avatar
								src={member.avatar_url}
								name={member.username}
								class="h-8 w-8"
								text="text-xs"
							/>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<a
										href={resolve(`/profile/${member.username}`)}
										class="truncate text-sm text-(--silver) hover:text-(--mer)">{member.username}</a
									>
									{#if member.user_id === room.host_id}
										<span class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase"
											>host</span
										>
									{:else if member.role !== 'member'}
										<span class="font-mono text-[9px] tracking-[0.12em] text-(--silver)/35 uppercase"
											>{roleLabel(member.role)}</span
										>
									{/if}
									{#if member.is_bot}
										<span class="font-mono text-[9px] text-(--silver)/25">bot</span>
									{/if}
								</div>
								<div class="mt-0.5 font-mono text-[10px] text-(--silver)/35">
									{member.rank.label} · {member.rating} · {member.online ? 'online' : 'away'}
								</div>
							</div>
							{#if canManage && member.user_id !== myID && member.user_id !== room.host_id}
								<div class="flex gap-1">
									{#if isOwner}
										<button
											class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[9px] text-(--silver)/40 hover:text-(--mer)"
											onclick={() => memberAction(member.role === 'mod' ? 'demote' : 'promote', member.user_id)}
											>{member.role === 'mod' ? 'demote' : 'mod'}</button
										>
									{/if}
									<button
										class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[9px] text-(--silver)/40 hover:text-(--red)"
										onclick={() => memberAction('kick', member.user_id)}>kick</button
									>
									<button
										class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[9px] text-(--silver)/40 hover:text-(--red)"
										onclick={() => memberAction('ban', member.user_id)}>ban</button
									>
								</div>
							{/if}
						</div>
					{/each}
				</div>

				<div class="border-t border-(--border) px-5 py-4">
					{#if canManage}
						<button
							class="w-full cursor-pointer border border-(--mer) px-4 py-3 font-mono text-sm text-(--mer) hover:bg-(--mer)/10 disabled:cursor-not-allowed disabled:opacity-35"
							onclick={start}
							disabled={busy || !canStart}
						>
							{busy ? 'starting…' : canStart ? 'start clock =>' : 'need at least 2 players'}
						</button>
					{:else}
						<div class="text-center font-mono text-[11px] text-(--silver)/30">
							waiting for the host to start the clock
						</div>
					{/if}
				</div>
			</section>

			<aside class="flex flex-col gap-3">
				<div class="border border-(--border) bg-(--fbg) p-5">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
						invite a player
					</div>
					<div class="mt-3 flex gap-2">
						<input
							class="min-w-0 flex-1 border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
							placeholder="username"
							bind:value={inviteName}
							onkeydown={(e) => e.key === 'Enter' && invite()}
						/>
						<button
							class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
							onclick={invite}>invite</button
						>
					</div>
					<button
						class="mt-2 w-full cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] text-(--silver)/40 hover:text-(--silver)"
						onclick={copyLink}>copy /join/{code}</button
					>
				</div>

				{#if settingsOpen && isOwner}
					<div class="border border-(--border) bg-(--fbg) p-5">
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
							room settings
						</div>
						<div class="mt-3 flex flex-col gap-2">
							<input
								class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
								bind:value={settings.name}
								maxlength="64"
							/>
							<input
								class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
								bind:value={settings.description}
								maxlength="280"
							/>
							<div class="grid grid-cols-2 gap-2">
								<select
									class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
									bind:value={settings.duration}
								>
									{#each LOBBY_DURATIONS as duration (duration)}
										<option value={duration}>{duration}s</option>
									{/each}
								</select>
								<select
									class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
									bind:value={settings.language}
								>
									{#each LANGUAGES as lang (lang)}
										<option value={lang}>{LANGUAGE_LABELS[lang]}</option>
									{/each}
								</select>
							</div>
							<div class="grid grid-cols-2 gap-2">
								<input
									type="number"
									min="2"
									max="16"
									class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
									bind:value={settings.max_players}
								/>
								<input
									type="password"
									class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
									placeholder="new password"
									bind:value={settings.password}
								/>
							</div>
							<label class="flex items-center gap-2 font-mono text-[11px] text-(--silver)/55">
								<input type="checkbox" bind:checked={settings.is_public} /> public
							</label>
							<button
								class="cursor-pointer border border-(--mer) px-3 py-2 font-mono text-xs text-(--mer) disabled:opacity-40"
								onclick={saveSettings}
								disabled={busy}>save settings</button
							>
						</div>
					</div>
				{/if}

				<div class="border border-(--border) bg-(--fbg) p-5">
					<div class="flex items-center justify-between">
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
							contests
						</div>
						{#if isOwner}
							<button
								class="cursor-pointer font-mono text-[10px] text-(--silver)/40 hover:text-(--mer)"
								onclick={() => (contestOpen = !contestOpen)}>{contestOpen ? 'close' : '+ schedule'}</button
							>
						{/if}
					</div>
					{#if contestOpen && isOwner}
						<div class="mt-3 flex flex-col gap-2">
							<input
								class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
								placeholder="contest title"
								bind:value={contestForm.title}
								maxlength="120"
							/>
							<div class="grid grid-cols-2 gap-2">
								<input
									type="number"
									min="1"
									class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
									bind:value={contestForm.minutes}
									title="minutes from now"
								/>
								<select
									class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-xs text-(--silver)"
									bind:value={contestForm.duration}
								>
									{#each LOBBY_DURATIONS as duration (duration)}
										<option value={duration}>{duration}s</option>
									{/each}
								</select>
							</div>
							<button
								class="cursor-pointer border border-(--mer) px-3 py-2 font-mono text-xs text-(--mer) disabled:opacity-40"
								onclick={makeContest}
								disabled={busy || !contestForm.title.trim()}>schedule</button
							>
						</div>
					{/if}
					{#if contests.length === 0}
						<p class="mt-3 font-mono text-[11px] text-(--silver)/30">No contests scheduled.</p>
					{:else}
						<div class="mt-3 flex flex-col gap-2">
							{#each contests.slice(0, 5) as contest (contest.id)}
								<div class="border-b border-(--border)/50 pb-2">
									<div class="flex items-center justify-between gap-2">
										<span class="truncate font-mono text-[11px] text-(--silver)/70">{contest.title}</span>
										<span class="shrink-0 font-mono text-[9px] text-(--mer) uppercase">{contest.status}</span>
									</div>
									<div class="mt-1 font-mono text-[9px] text-(--silver)/30">
										{fmtDate(contest.starts_at)} · {contest.duration}s
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<div class="border border-(--border) p-5">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/30 uppercase">
						room code
					</div>
					<button
						class="mt-2 font-mono text-2xl tracking-[0.2em] text-(--silver) hover:text-(--mer)"
						onclick={copyLink}>{code}</button
					>
					<p class="mt-2 font-mono text-[10px] leading-relaxed text-(--silver)/25">
						Share the link. Invites arrive instantly for online players.
					</p>
				</div>
			</aside>
		</main>
	{/if}
</div>
