<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import ContestForm from '$lib/components/ContestForm.svelte';
	import { createContest, type ContestInput } from '$lib/core/api/contests';
	import { errorMessage } from '$lib/core/api/client';

	let busy = $state(false);
	let error = $state('');

	async function submit(input: ContestInput) {
		if (busy) return;
		busy = true;
		error = '';
		try {
			const contest = await createContest(input);
			await goto(resolve(`/contests/${contest.code}`));
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>host a contest · OSDTYP</title></svelte:head>

<div class="min-h-screen bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-end justify-between gap-4 border-b border-(--border) pb-4">
		<div>
			<a
				href={resolve('/contests')}
				class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase hover:text-(--silver)"
				>← contests</a
			>
			<div class="title-text mt-2 text-6xl leading-none text-(--silver) md:text-7xl">HOST AN EVENT</div>
			<p class="mt-2 max-w-2xl font-mono text-[11px] leading-relaxed text-(--silver)/40">
				Build a standalone typing event. Public contests are listed for everyone; private contests
				stay behind their invite link.
			</p>
		</div>
		<a
			href={resolve('/play')}
			class="border border-(--border) px-3 py-2 font-mono text-[11px] tracking-[0.12em] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
			>hub</a
		>
	</header>

	<main class="mt-5">
		<ContestForm {busy} {error} onsubmit={submit} />
	</main>
</div>
