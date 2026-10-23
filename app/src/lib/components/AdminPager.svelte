<script lang="ts">
	import type { PageMeta } from '$lib/core/api/admin';

	let {
		page,
		noun = 'rows',
		onpage
	}: { page: PageMeta; noun?: string; onpage: (offset: number) => void } = $props();

	let start = $derived(page.total === 0 ? 0 : page.offset + 1);
	let end = $derived(Math.min(page.offset + page.limit, page.total));
</script>

<div class="flex flex-wrap items-center justify-between gap-3 border-t border-(--border) px-3 py-2 font-mono text-[10px] text-(--silver)/35">
	<span>{start}–{end} of {page.total} {noun}</span>
	<div class="flex items-center gap-2">
		<button
			class="cursor-pointer border border-(--border) px-2.5 py-1 uppercase hover:border-(--mer) hover:text-(--mer) disabled:cursor-default disabled:opacity-25"
			disabled={page.offset === 0}
			onclick={() => onpage(Math.max(0, page.offset - page.limit))}>← prev</button
		>
		<button
			class="cursor-pointer border border-(--border) px-2.5 py-1 uppercase hover:border-(--mer) hover:text-(--mer) disabled:cursor-default disabled:opacity-25"
			disabled={!page.has_more}
			onclick={() => onpage(page.next ?? page.offset + page.limit)}>next →</button
		>
	</div>
</div>
