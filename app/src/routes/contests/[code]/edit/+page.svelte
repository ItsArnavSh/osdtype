<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import ContestForm from '$lib/components/ContestForm.svelte';
	import {
		getContest,
		updateContest,
		type ContestInput
	} from '$lib/core/api/contests';
	import { errorMessage } from '$lib/core/api/client';
	import type { ContestView } from '$lib/core/api/types';

	const code = $derived((page.params.code ?? '').toUpperCase());

	let contest = $state<ContestView | null>(null);
	let loading = $state(true);
	let busy = $state(false);
	let error = $state('');

	onMount(async () => {
		try {
			contest = await getContest(code);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			loading = false;
		}
	});

	async function submit(input: ContestInput) {
		if (busy) return;
		busy = true;
		error = '';
		try {
			await updateContest(code, input);
			await goto(resolve(`/contests/${code}`));
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>edit contest · OSDTYP</title></svelte:head>

<div class="min-h-screen bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4 border-b border-(--border) pb-4">
		<div>
			<a
				href={resolve(`/contests/${code}`)}
				class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← contest</a
			>
			<div class="title-text mt-2 text-6xl leading-none text-(--silver) md:text-7xl">EDIT EVENT</div>
		</div>
	</header>

	<main class="mt-5">
		{#if loading}
			<div class="border border-(--border) bg-(--fbg) p-8 font-mono text-xs text-(--silver)/35">
				loading contest…
			</div>
		{:else if !contest}
			<div class="border border-(--red)/40 p-5 font-mono text-xs text-(--red)">
				{error || 'contest not found'}
			</div>
		{:else if !contest.is_host}
			<div class="border border-(--border) bg-(--fbg) p-6 font-mono text-xs text-(--silver)/45">
				Only the host can edit this contest.
			</div>
		{:else if contest.status !== 'upcoming'}
			<div class="border border-(--border) bg-(--fbg) p-6 font-mono text-xs leading-relaxed text-(--silver)/45">
				This contest can no longer be edited because it is {contest.status}.
				<a href={resolve(`/contests/${code}`)} class="text-(--mer)">Back to the event →</a>
			</div>
		{:else}
			<ContestForm contest={contest} {busy} {error} submitLabel="save contest" onsubmit={submit} />
		{/if}
	</main>
</div>
