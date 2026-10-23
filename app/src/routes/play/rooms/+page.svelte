<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount, tick } from 'svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { createRoom, joinRoom, listRooms } from '$lib/core/api/rooms';
	import type { LobbyDuration, RoomView } from '$lib/core/api/types';
	import { LOBBY_DURATIONS } from '$lib/core/api/types';
	import { LANGUAGES, LANGUAGE_LABELS, type Language } from '$lib/core/entity/languages';
	import { refreshState, appState } from '$lib/core/state/state.svelte';

	let rooms = $state<RoomView[]>([]);
	let loading = $state(true);
	let error = $state('');
	let createError = $state('');
	let showCreate = $state(false);
	let busy = $state(false);
	let joinCode = $state('');
	let nameInput = $state<HTMLInputElement | null>(null);

	let form = $state({
		name: '',
		description: '',
		is_public: true,
		password: '',
		max_players: 8,
		duration: 90 as LobbyDuration,
		language: 'go' as Language
	});

	async function load() {
		loading = true;
		error = '';
		try {
			rooms = await listRooms();
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	}

	async function openCreate() {
		createError = '';
		showCreate = true;
		await tick();
		nameInput?.focus();
	}

	function closeCreate() {
		showCreate = false;
		createError = '';
	}

	async function create() {
		if (busy) return;
		busy = true;
		createError = '';
		try {
			const room = await createRoom({
				name: form.name.trim(),
				description: form.description.trim(),
				is_public: form.is_public,
				password: form.password || undefined,
				max_players: Math.min(16, Math.max(2, Number(form.max_players) || 8)),
				duration: form.duration,
				language: form.language
			});
			await refreshState();
			await goto(resolve(`/room/${room.code}`));
		} catch (err) {
			createError = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function quickJoin() {
		const code = joinCode.trim().toUpperCase();
		if (!code) return;
		error = '';
		try {
			await joinRoom(code);
			await refreshState();
			await goto(resolve(`/room/${code}`));
		} catch (err) {
			error = errorMessage(err);
			await goto(resolve(`/join/${code}`));
		}
	}

	onMount(load);
</script>

<svelte:head><title>rooms · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-6 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex items-end gap-4">
			<a
				href={resolve('/play')}
				class="mb-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← hub</a
			>
			<div class="title-text text-7xl leading-none text-(--silver) md:text-8xl">ROOMS</div>
		</div>
		<div class="flex items-center gap-3">
			<div class="hidden text-right font-mono text-[10px] text-(--silver)/30 sm:block">
				<div>{appState.user?.username}</div>
				<div class="text-(--mer)">{appState.user?.rating ?? '—'} elo</div>
			</div>
			<button
				class="cursor-pointer border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
				onclick={openCreate}
			>+ create room</button
			>
		</div>
	</header>

	<main class="mt-5 grid flex-1 gap-3 lg:grid-cols-[1fr_360px]">
		<section class="flex flex-col gap-3">
			<div class="flex flex-wrap items-center gap-2 border border-(--border) bg-(--fbg) p-3">
				<input
					class="min-w-0 flex-1 border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs tracking-[0.14em] text-(--silver) uppercase outline-none focus:border-(--mer)"
					placeholder="join by room code"
					maxlength="6"
					bind:value={joinCode}
					onkeydown={(e) => e.key === 'Enter' && quickJoin()}
				/>
				<button
					class="cursor-pointer border border-(--border) px-4 py-2 font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					onclick={quickJoin}>join =></button
				>
				<button
					class="cursor-pointer border border-(--border) px-4 py-2 font-mono text-xs text-(--silver)/55 hover:border-(--silver)/40 hover:text-(--silver)"
					onclick={load}>refresh</button
				>
			</div>

			{#if error}
				<div class="border border-(--red)/40 px-4 py-3 font-mono text-xs text-(--red)">{error}</div>
			{/if}

			{#if loading}
				<div class="flex flex-1 items-center justify-center border border-(--border) bg-(--fbg)">
					<span class="font-mono text-xs text-(--silver)/35">loading rooms…</span>
				</div>
			{:else if rooms.length === 0}
				<div class="flex flex-1 flex-col items-center justify-center gap-3 border border-(--border) bg-(--fbg) py-20">
					<div class="text-xl text-(--silver)/60">No public rooms open.</div>
					<button
						class="cursor-pointer border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer)"
						onclick={openCreate}>create the first one</button
					>
				</div>
			{:else}
				<div class="grid gap-3 md:grid-cols-2">
					{#each rooms as room (room.id)}
						<a
							href={resolve(`/room/${room.code}`)}
							class="group flex flex-col justify-between border border-(--border) bg-(--fbg) p-5 transition-colors hover:border-(--mer)/60"
						>
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									<div class="truncate text-xl text-(--silver)">{room.name}</div>
									<div class="mt-1 truncate font-mono text-[11px] text-(--silver)/40">
										{room.description || 'no description'}
									</div>
								</div>
								<span
									class="shrink-0 border border-(--border) px-2 py-1 font-mono text-[10px] tracking-[0.12em] text-(--silver)/45 uppercase"
									>{room.has_password ? 'locked' : room.is_public ? 'public' : 'private'}</span
								>
							</div>
							<div class="mt-5 flex items-end justify-between font-mono text-[10px] text-(--silver)/35">
								<div class="flex gap-4">
									<span>{room.member_count}/{room.max_players} players</span>
									<span>{room.duration}s</span>
									<span>{LANGUAGE_LABELS[room.language as Language] ?? room.language}</span>
								</div>
								<span class="text-(--mer) opacity-0 transition-opacity group-hover:opacity-100"
									>open =></span
								>
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</section>

		<aside class="flex flex-col gap-3">
			<div class="border border-(--border) bg-(--fbg) p-5">
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--silver)/35 uppercase">
					how rooms work
				</div>
				<ul class="mt-3 flex flex-col gap-2 font-mono text-[11px] leading-relaxed text-(--silver)/45">
					<li>· 6-character code and optional password</li>
					<li>· host and mods start the clock</li>
					<li>· invite friends by username or link</li>
					<li>· up to 16 players, bots allowed</li>
					<li>· leaving is always explicit</li>
				</ul>
			</div>

			<button
				class="cursor-pointer border border-(--border) px-4 py-2.5 font-mono text-xs text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
				onclick={openCreate}>+ create room =></button
			>
		</aside>
	</main>
</div>

<svelte:window
	onkeydown={(e) => {
		if (showCreate && e.key === 'Escape') closeCreate();
	}}
/>

{#if showCreate}
	<div class="fixed inset-0 z-[80] overflow-y-auto p-4 sm:p-6">
		<button
			type="button"
			class="fixed inset-0 cursor-default bg-(--bg)/80 backdrop-blur-sm"
			aria-label="close create room form"
			onclick={closeCreate}
		></button>

		<div
			class="relative z-10 mx-auto my-4 w-full max-w-lg border border-(--border) bg-(--fbg) shadow-2xl sm:my-10"
			style="border-top-color: var(--mer);"
			role="dialog"
			aria-modal="true"
			aria-labelledby="create-room-title"
		>
			<div class="flex items-start justify-between border-b border-(--border) px-5 py-4">
				<div>
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
						new room
					</div>
					<div id="create-room-title" class="mt-1 text-xl text-(--silver)">Set the table.</div>
				</div>
				<button
					type="button"
					class="cursor-pointer px-2 py-1 font-mono text-sm text-(--silver)/35 hover:text-(--silver)"
					onclick={closeCreate}
					aria-label="close create room form">✕</button
				>
			</div>

			<form
				class="flex flex-col gap-3 p-5"
				onsubmit={(e) => {
					e.preventDefault();
					void create();
				}}
			>
				<label class="flex flex-col gap-1">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
						>name</span
					>
					<input
						bind:this={nameInput}
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.name}
						maxlength="64"
						required
						placeholder="friday night sprint"
					/>
				</label>
				<label class="flex flex-col gap-1">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
						>description</span
					>
					<input
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.description}
						maxlength="280"
						placeholder="optional"
					/>
				</label>
				<div class="grid grid-cols-2 gap-2">
					<label class="flex flex-col gap-1">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
							>players</span
						>
						<input
							type="number"
							min="2"
							max="16"
							class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
							bind:value={form.max_players}
						/>
					</label>
					<label class="flex flex-col gap-1">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
							>duration</span
						>
						<select
							class="border border-(--border) bg-(--bg) px-2 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
							bind:value={form.duration}
						>
							{#each LOBBY_DURATIONS as d (d)}
								<option value={d}>{d}s</option>
							{/each}
						</select>
					</label>
				</div>
				<label class="flex flex-col gap-1">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
						>language</span
					>
					<select
						class="border border-(--border) bg-(--bg) px-2 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.language}
					>
						{#each LANGUAGES as lang (lang)}
							<option value={lang}>{LANGUAGE_LABELS[lang]}</option>
						{/each}
					</select>
				</label>
				<label class="flex items-center gap-2 font-mono text-[11px] text-(--silver)/55">
					<input type="checkbox" bind:checked={form.is_public} /> public room
				</label>
				<label class="flex flex-col gap-1">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
						>password (optional)</span
					>
					<input
						type="password"
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
						bind:value={form.password}
						placeholder="leave blank for none"
					/>
				</label>

				{#if createError}
					<div class="border border-(--red)/40 px-3 py-2 font-mono text-xs text-(--red)">
						{createError}
					</div>
				{/if}

				<button
					type="submit"
					disabled={busy || !form.name.trim()}
					class="mt-1 cursor-pointer border border-(--mer) px-4 py-2.5 font-mono text-xs text-(--mer) hover:bg-(--mer)/10 disabled:cursor-not-allowed disabled:opacity-40"
				>{busy ? 'creating…' : 'create room =>'}</button
				>
			</form>
		</div>
	</div>
{/if}
