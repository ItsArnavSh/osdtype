<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { errorMessage } from '$lib/core/api/client';
	import { getRoom, joinRoom } from '$lib/core/api/rooms';
	import type { RoomView } from '$lib/core/api/types';
	import { LANGUAGE_LABELS } from '$lib/core/api/types';
	import { appState, refreshState } from '$lib/core/state/state.svelte';

	let room = $state<RoomView | null>(null);
	let loading = $state(true);
	let joining = $state(false);
	let password = $state('');
	let error = $state('');

	const code = $derived((page.params.code ?? '').toUpperCase());

	onMount(async () => {
		try {
			if (appState.state?.state === 'room' && appState.state.room?.code === code) {
				await goto(resolve(`/room/${code}`));
				return;
			}
			room = await getRoom(code);
			if (room.my_role) {
				await refreshState();
				await goto(resolve(`/room/${code}`));
			}
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	});

	async function join() {
		if (!room || joining) return;
		joining = true;
		error = '';
		try {
			await joinRoom(code, room.has_password ? password : '');
			await refreshState();
			await goto(resolve(`/room/${code}`));
		} catch (err) {
			error = errorMessage(err);
		} finally {
			joining = false;
		}
	}
</script>

<svelte:head><title>join {code} · OSDTYP</title></svelte:head>

<div class="flex min-h-screen items-center justify-center bg-(--bg) p-6">
	<div class="w-full max-w-md border border-(--border) bg-(--fbg)">
		<div class="border-b border-(--border) px-6 py-4">
			<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">room invite</div>
			<div class="mt-2 text-2xl text-(--silver)">{room?.name ?? 'Resolving room…'}</div>
			<div class="mt-1 font-mono text-xs text-(--silver)/40">code {code}</div>
		</div>

		<div class="flex flex-col gap-4 p-6">
			{#if loading}
				<div class="py-8 text-center font-mono text-xs text-(--silver)/35">reading invitation…</div>
			{:else if !room}
				<div class="border border-(--red)/40 px-3 py-2 font-mono text-xs text-(--red)">{error}</div>
				<a
					href={resolve('/play/rooms')}
					class="border border-(--border) px-4 py-2 text-center font-mono text-xs text-(--silver)/55 hover:text-(--silver)"
					>back to rooms</a
				>
			{:else}
				<div class="grid grid-cols-2 gap-2 font-mono text-[11px]">
					<div class="border border-(--border) p-3">
						<div class="text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">players</div>
						<div class="mt-1 text-(--silver)">{room.member_count}/{room.max_players}</div>
					</div>
					<div class="border border-(--border) p-3">
						<div class="text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">format</div>
						<div class="mt-1 text-(--silver)"
							>{room.duration}s · {LANGUAGE_LABELS[room.language] ?? room.language}</div
						>
					</div>
				</div>
				{#if room.description}
					<p class="font-mono text-xs leading-relaxed text-(--silver)/45">{room.description}</p>
				{/if}
				<div class="flex items-center gap-2 font-mono text-[11px] text-(--silver)/40">
					<span class="h-1.5 w-1.5 bg-(--mer)"></span>
					{room.has_password ? 'this room is password protected' : 'open room'}
				</div>

				{#if room.has_password}
					<label class="flex flex-col gap-1">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase"
							>password</span
						>
						<input
							type="password"
							class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
							bind:value={password}
							onkeydown={(e) => e.key === 'Enter' && join()}
						/>
					</label>
				{/if}

				{#if error}
					<div class="border border-(--red)/40 px-3 py-2 font-mono text-xs text-(--red)">{error}</div>
				{/if}

				<button
					class="cursor-pointer border border-(--mer) px-4 py-2.5 font-mono text-sm text-(--mer) hover:bg-(--mer)/10 disabled:opacity-40"
					onclick={join}
					disabled={joining}
					>{joining ? 'joining…' : 'join room =>'}</button
				>
				<a
					href={resolve('/play')}
					class="text-center font-mono text-[10px] text-(--silver)/30 hover:text-(--silver)/60"
					>not now</a
				>
			{/if}
		</div>
	</div>
</div>
