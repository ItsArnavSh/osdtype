<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { acceptInvite, declineInvite, listInvites } from '$lib/core/api/rooms';
	import { appState, refreshState } from '$lib/core/state/state.svelte';
	import { errorMessage } from '$lib/core/api/client';

	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			appState.invites = await listInvites();
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	});

	async function accept(id: number) {
		error = '';
		try {
			await acceptInvite(id);
			await refreshState();
			appState.invites = appState.invites.filter((invite) => invite.id !== id);
			await goto(resolve('/play'));
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function decline(id: number) {
		error = '';
		try {
			await declineInvite(id);
			appState.invites = appState.invites.filter((invite) => invite.id !== id);
		} catch (err) {
			error = errorMessage(err);
		}
	}
</script>

<svelte:head><title>invites · OSDTYP</title></svelte:head>

<div class="flex min-h-screen flex-col bg-(--bg) p-6 md:p-8">
	<header class="flex items-end gap-4">
		<a
			href={resolve('/play')}
			class="mb-2 font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
			>← hub</a
		>
		<div class="title-text text-7xl leading-none text-(--silver) md:text-8xl">INVITES</div>
	</header>

	{#if error}
		<div class="mt-4 border border-(--red)/40 px-4 py-3 font-mono text-xs text-(--red)">{error}</div>
	{/if}

	<main class="mt-5 max-w-3xl">
		{#if loading}
			<div class="border border-(--border) bg-(--fbg) p-10 text-center font-mono text-xs text-(--silver)/35">
				loading…
			</div>
		{:else if appState.invites.length === 0}
			<div class="border border-(--border) bg-(--fbg) p-10 text-center">
				<div class="text-xl text-(--silver)/60">No pending invites.</div>
				<p class="mt-2 font-mono text-xs text-(--silver)/30">
					Room invites appear here and as a toast while you are online.
				</p>
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each appState.invites as invite (invite.id)}
					<div class="flex flex-wrap items-center justify-between gap-4 border border-(--border) bg-(--fbg) p-5">
						<div>
							<div class="text-lg text-(--silver)">{invite.room_name || invite.room_code}</div>
							<div class="mt-1 font-mono text-[11px] text-(--silver)/40">
								from {invite.from} · code {invite.room_code}
							</div>
						</div>
						<div class="flex gap-2">
							<button
								class="cursor-pointer border border-(--mer) px-4 py-2 font-mono text-xs text-(--mer) hover:bg-(--mer)/10"
								onclick={() => accept(invite.id)}>accept</button
							>
							<button
								class="cursor-pointer border border-(--border) px-4 py-2 font-mono text-xs text-(--silver)/45 hover:text-(--silver)"
								onclick={() => decline(invite.id)}>ignore</button
							>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</main>
</div>
