<script lang="ts">
	import { resolve } from '$app/paths';

	const principles = [
		{
			title: 'Code first',
			copy: 'Every run uses generated source code. Not quotes, not word lists. The point is to practise the shapes and symbols you actually type.'
		},
		{
			title: 'Clean competition',
			copy: 'The server owns the clock and the snippet. Ranked games, rooms and contest stages all score the same recorded run, so the board means something.'
		},
		{
			title: 'Low noise',
			copy: 'No coins, no daily streaks, no tier grind. Just a keyboard, a snippet, and a result you can replay and learn from.'
		}
	];
</script>

<svelte:head>
	<title>about · OSDTYP</title>
	<meta
		name="description"
		content="About OSDTYP, a speedtyping arena for source code. Built by Arnav Sharma. Links to osdc.dev and the author site."
	/>
</svelte:head>

<div class="min-h-screen bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<div class="flex flex-wrap items-center gap-3">
			<a href={resolve('/play')} class="title-text text-3xl leading-none text-(--silver)">OSDTYP</a>
			<span class="font-mono text-[10px] tracking-[0.14em] text-(--mer) uppercase">about</span>
		</div>
		<nav class="flex flex-wrap items-center gap-2 font-mono text-[10px] tracking-[0.1em] uppercase">
			<a
				href={resolve('/docs')}
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>docs</a
			>
			<a
				href={resolve('/about')}
				class="border border-(--mer) px-3 py-1.5 text-(--mer)"
				aria-current="page">about</a
			>
			<a
				href={resolve('/contests')}
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>contests</a
			>
			<a
				href={resolve('/play')}
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>play</a
			>
		</nav>
	</header>

	<main class="mx-auto max-w-4xl">
		<section class="mt-10 border-b border-(--border) pb-10">
			<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">about the project</div>
			<h1 class="mt-3 max-w-3xl text-5xl leading-none text-(--silver) md:text-7xl">
				A typing arena for people who write code.
			</h1>
			<p class="mt-5 max-w-2xl text-sm leading-relaxed text-(--silver)/50">
				OSDTYP is a speedtyping game built around real source code. It has independent ranked
				ladders, offline practice, multiplayer rooms, and contest events that can run as full
				multi-stage competitions. It is made for the fun of typing fast without hiding the parts
				that make code hard.
			</p>
		</section>

		<section class="mt-8 grid gap-3 md:grid-cols-3">
			{#each principles as principle (principle.title)}
				<div class="border border-(--border) bg-(--fbg) p-5">
					<div class="font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase">
						{principle.title}
					</div>
					<p class="mt-3 text-sm leading-relaxed text-(--silver)/45">{principle.copy}</p>
				</div>
			{/each}
		</section>

		<section class="mt-8 grid gap-4 md:grid-cols-[1.1fr_0.9fr]">
			<div class="border border-(--border) bg-(--fbg) p-6">
				<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
					what is inside
				</div>
				<ul class="mt-4 space-y-2 text-sm leading-relaxed text-(--silver)/45">
					<li>Ranked queues at 30s, 90s and 300s with separate Elo ladders.</li>
					<li>Practice, Zen, adaptive bots and local snippet generation.</li>
					<li>Public and private rooms with invites, moderation and replayable results.</li>
					<li>Standalone contests with qualifiers, HotLaps, live heats and ranked finals.</li>
					<li>Profiles, follows, friends, best runs and a searchable leaderboard.</li>
					<li>Speed graphs, input replays, replay keyboards and share links.</li>
					<li>Themes and per-event custom CSS scoped to the event page.</li>
				</ul>
				<a
					href={resolve('/docs')}
					class="mt-5 inline-block border border-(--mer) px-3 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
					>read the docs -></a
				>
			</div>

			<div class="border border-(--border) bg-(--fbg) p-6">
				<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
					stack
				</div>
				<div class="mt-4 flex flex-col gap-2 font-mono text-[10px] text-(--silver)/45">
					<span>SvelteKit and TypeScript</span>
					<span>Go and Gin</span>
					<span>PostgreSQL and Redis</span>
					<span>Rust and WebAssembly</span>
					<span>resrap-rs 1.0.1</span>
				</div>
				<p class="mt-4 text-xs leading-relaxed text-(--silver)/30">
					The server is authoritative for clocks, scoring and match state. The browser owns input,
					rendering and offline practice.
				</p>
			</div>
		</section>

		<section class="mt-8 border border-(--border) bg-(--fbg) p-6">
			<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
				built by
			</div>
			<h2 class="mt-3 text-2xl text-(--silver)">Arnav Sharma</h2>
			<p class="mt-2 max-w-2xl text-sm leading-relaxed text-(--silver)/45">
				The project lives on GitHub and is maintained by Arnav. The links below cover the
				developer community and the personal site.
			</p>
			<div class="mt-4 grid gap-2 sm:grid-cols-2">
				<a
					href="https://osdc.dev/"
					target="_blank"
					rel="noreferrer"
					class="border border-(--border) p-4 hover:border-(--mer)"
				>
					<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">community</div>
					<div class="mt-1 text-sm text-(--silver)/65">osdc.dev</div>
					<div class="mt-1 font-mono text-[9px] text-(--silver)/25">the open source developer community</div>
				</a>
				<a
					href="https://itsarnavsh.github.io/"
					target="_blank"
					rel="noreferrer"
					class="border border-(--border) p-4 hover:border-(--mer)"
				>
					<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">author</div>
					<div class="mt-1 text-sm text-(--silver)/65">itsarnavsh.github.io</div>
					<div class="mt-1 font-mono text-[9px] text-(--silver)/25">projects, writing and other work</div>
				</a>
			</div>
			<div class="mt-4 flex flex-wrap gap-2 font-mono text-[10px] uppercase">
				<a
					href="https://github.com/ItsArnavSh/osdtype"
					target="_blank"
					rel="noreferrer"
					class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
					>github repository</a
				>
				<a
					href={resolve('/docs')}
					class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
					>documentation</a
				>
			</div>
		</section>

		<section class="mt-8 border border-(--border) bg-(--fbg) p-6">
			<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
				the short version
			</div>
			<p class="mt-3 max-w-2xl text-sm leading-relaxed text-(--silver)/50">
				If you want to get faster at typing code, open Practice. If you want a fair match, queue
				Ranked. If you want to run an event, build a contest funnel. The docs explain all three
				without the marketing.
			</p>
			<a
				href={resolve('/play')}
				class="mt-5 inline-block border border-(--mer) px-4 py-2 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
				>start typing -></a
			>
		</section>
	</main>
</div>
