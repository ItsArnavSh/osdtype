<script lang="ts">
	import ReplayPlayer from '$lib/components/ReplayPlayer.svelte';
	import type { ReplayRun } from '$lib/core/api/types';

	let {
		run,
		snippet,
		startOffset,
		duration,
		language,
		shareURL = '',
		onclose
	}: {
		run: ReplayRun;
		snippet: string;
		startOffset: number;
		duration: number;
		language: string;
		shareURL?: string;
		onclose: () => void;
	} = $props();
</script>

<div
	class="fixed inset-0 z-40 flex items-center justify-center bg-(--bg)/85 p-3 md:p-6"
	role="presentation"
	onclick={onclose}
>
	<div
		class="max-h-full w-full max-w-6xl overflow-y-auto border border-(--border) bg-(--bg) p-4 shadow-none md:p-6"
		role="dialog"
		aria-modal="true"
		aria-label={`${run.username} replay`}
		onclick={(event) => event.stopPropagation()}
		onkeydown={(event) => event.key === 'Escape' && onclose()}
		tabindex="-1"
	>
		<ReplayPlayer {run} {snippet} {startOffset} {duration} {language} {shareURL} {onclose} />
	</div>
</div>
