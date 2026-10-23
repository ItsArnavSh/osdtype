<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';

	let {
		compact = false,
		dismissible = true
	}: { compact?: boolean; dismissible?: boolean } = $props();

	const storageKey = 'osdtype.notice.v0.0.1-dismissed';
	let dismissed = $state(false);
	let ready = $state(false);

	onMount(() => {
		ready = true;
		if (!dismissible) return;
		try {
			dismissed = localStorage.getItem(storageKey) === '1';
		} catch {
			dismissed = false;
		}
	});

	function dismiss() {
		dismissed = true;
		try {
			localStorage.setItem(storageKey, '1');
		} catch {
			// Storage can be unavailable in private/incognito contexts.
		}
	}
</script>

{#if !dismissible || !dismissed}
	<div
		class={`flex flex-wrap items-center gap-x-3 gap-y-1.5 border border-(--red)/40 bg-(--red)/5 px-3 py-2 font-mono ${compact ? 'text-[9px]' : 'text-[10px]'}`}
	>
		<span
			class="border border-(--red)/50 px-2 py-0.5 tracking-[0.14em] text-(--red) uppercase"
			>v0.0.1 · experimental</span
		>
		<span class="leading-relaxed text-(--silver)/55">
			First public build. It is early, it is rough, and bugs will happen.
		</span>
		<a
			href={resolve('/bugs')}
			class="tracking-[0.08em] text-(--mer) uppercase underline decoration-(--mer)/40 underline-offset-4 hover:decoration-(--mer)"
			>report a bug →</a
		>
		{#if dismissible && ready}
			<button
				type="button"
				class="ml-auto cursor-pointer border border-transparent px-2 py-0.5 text-sm leading-none text-(--silver)/35 hover:border-(--border) hover:text-(--silver)"
				onclick={dismiss}
				aria-label="Dismiss the experimental warning"
				title="Dismiss">×</button
			>
		{/if}
	</div>
{/if}
