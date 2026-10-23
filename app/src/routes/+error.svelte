<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let status = $derived(page.status || 404);
	let digits = $derived(String(status).split(''));
	let title = $derived(status === 404 ? 'route not found' : 'something broke');
	let detail = $derived(
		status === 404
			? 'That page is not on the keyboard. Check the address or head back to the hub.'
			: page.error?.message || 'Something hit an unexpected key. Try again in a moment.'
	);

	function goBack() {
		if (typeof window !== 'undefined' && window.history.length > 1) {
			window.history.back();
			return;
		}
		void goto('/play');
	}
</script>

<svelte:head>
	<title>{status} · OSDTYP</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-(--bg) p-5 md:p-8">
	<div class="w-full max-w-3xl border border-(--border) bg-(--fbg)">
		<div
			class="flex flex-wrap items-center justify-between gap-2 border-b border-(--border) px-4 py-3 font-mono text-[10px] uppercase tracking-[0.16em] text-(--silver)/35"
		>
			<span>osdtyp · error</span>
			<span class="text-(--red)">status {status}</span>
		</div>

		<div class="p-6 md:p-10">
			<div class="flex flex-wrap gap-3" aria-hidden="true">
				{#each digits as digit, i (i)}
					<span
						class="keycap inline-flex h-20 w-16 items-center justify-center border border-(--border) bg-(--bg) font-mono text-5xl text-(--silver) md:h-28 md:w-24 md:text-7xl"
						style={`transform: rotate(${i % 2 === 0 ? -2 : 2}deg)`}>{digit}</span
					>
				{/each}
			</div>

			<div class="mt-8 font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
				key not bound
			</div>
			<h1 class="mt-2 text-4xl leading-none text-(--silver) md:text-6xl">{title}.</h1>
			<p class="mt-4 max-w-xl text-sm leading-relaxed text-(--silver)/50">{detail}</p>

			<div class="mt-8 flex flex-wrap gap-2">
				<button
					class="keycap cursor-pointer border border-(--mer)/50 bg-(--mer)/10 px-4 py-2.5 font-mono text-[11px] uppercase tracking-[0.12em] text-(--mer)"
					onclick={goBack}>← back</button
				>
				<a
					href={resolve('/play')}
					class="border border-(--border) px-4 py-2.5 font-mono text-[11px] uppercase tracking-[0.12em] text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					>hub</a
				>
				<a
					href={resolve('/docs')}
					class="border border-(--border) px-4 py-2.5 font-mono text-[11px] uppercase tracking-[0.12em] text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					>docs</a
				>
				<a
					href={resolve('/bugs')}
					class="border border-(--border) px-4 py-2.5 font-mono text-[11px] uppercase tracking-[0.12em] text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					>report</a
				>
			</div>

			<div
				class="mt-8 flex flex-wrap items-center justify-between gap-2 border-t border-(--border) pt-3 font-mono text-[10px] text-(--silver)/25"
			>
				<div>
					try <span class="text-(--silver)/45">/play</span> ·
					<span class="text-(--silver)/45">/docs</span> ·
					<span class="text-(--silver)/45">/contests</span>
				</div>
				<div class="text-(--red)/50">
					v0.0.1 experimental · <a href={resolve('/bugs')} class="underline">report bugs</a>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.keycap {
		box-shadow:
			0 4px 0 color-mix(in srgb, var(--border) 70%, transparent),
			inset 0 -2px 0 color-mix(in srgb, var(--border) 35%, transparent);
	}
</style>
