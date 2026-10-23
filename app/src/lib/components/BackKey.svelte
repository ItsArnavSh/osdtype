<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';

	let { label = 'back' } = $props();

	function goBack() {
		if (typeof window !== 'undefined' && window.history.length > 1) {
			window.history.back();
			return;
		}
		void goto(page.url.pathname === '/play' ? '/' : '/play');
	}
</script>

<button
	class="keycap fixed bottom-4 left-4 z-40 cursor-pointer border border-(--border) bg-(--fbg) px-3 py-2 font-mono text-[10px] tracking-[0.12em] text-(--silver)/55 uppercase transition-colors hover:border-(--mer) hover:text-(--mer)"
	onclick={goBack}
	aria-label="Go back"
	title="Go back"
>
	<span class="text-(--mer)">←</span> {label}
</button>

<style>
	.keycap {
		box-shadow:
			0 3px 0 color-mix(in srgb, var(--border) 75%, transparent),
			inset 0 -1px 0 color-mix(in srgb, var(--border) 35%, transparent);
	}
	.keycap:active {
		transform: translateY(2px);
		box-shadow: 0 1px 0 color-mix(in srgb, var(--border) 75%, transparent);
	}
</style>
