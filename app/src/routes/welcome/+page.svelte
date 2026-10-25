<script lang="ts">
	import { resolve } from '$app/paths';
	import ModeIcon from '$lib/components/ModeIcon.svelte';

	const modes = [
		{
			name: 'Practice',
			icon: 'practice',
			href: '/play/me',
			tag: 'no account',
			copy: 'A timed run with generated code. Go solo or race adaptive bots. Practice never changes your rating.',
			meta: '30 / 90 / 300s · works offline',
			cta: 'start practice'
		},
		{
			name: 'Zen',
			icon: 'zen',
			href: '/play/zen',
			tag: 'no account',
			copy: 'No clock, no score. The same snippet engine, with nothing counting down.',
			meta: 'untimed · works offline',
			cta: 'open zen'
		},
		{
			name: 'Ranked',
			icon: 'ranked',
			href: '/play/ranked',
			tag: 'account',
			copy: 'Queue for 30, 90, or 300 seconds. Each clock has its own Elo ladder.',
			meta: '3 clocks · bot fallback',
			cta: 'enter queue'
		},
		{
			name: 'Rooms',
			icon: 'rooms',
			href: '/play/rooms',
			tag: 'account',
			copy: 'Public or private lobbies for 2–16 players. Share a six-character code or invite link.',
			meta: '2–16 players · 6-character code',
			cta: 'browse rooms'
		},
		{
			name: 'Contests',
			icon: 'sprint',
			href: '/contests',
			tag: 'public pages',
			copy: 'Scheduled events with enrollment, stage cuts, and a live timing tower.',
			meta: 'up to 10 stages',
			cta: 'contest board'
		}
	] as const;

	const features = [
		{
			label: 'snippets',
			copy: 'Generated code in C, C++, Go, Haskell, HTML, Java, JavaScript, Python, Rust, and SQL.'
		},
		{
			label: 'engine',
			copy: 'The same Rust/WASM snippet generator runs in the browser and the Go service from a seed and token target.'
		},
		{
			label: 'clocks',
			copy: '30, 90, and 300 seconds for ranked and practice; no clock in Zen.'
		},
		{
			label: 'bots',
			copy: 'Adaptive bots fill ranked queues and can be added to rooms; they make readable mistakes and corrections.'
		},
		{
			label: 'anti-cheat',
			copy: 'Scoring is server-authoritative. Flagged runs do not change Elo.'
		},
		{
			label: 'replays',
			copy: 'Scored runs can keep their keystrokes. Watch the original timing with a keyboard overlay, corrections, and probable switch sounds.'
		},
		{
			label: 'graphs',
			copy: 'A race result combines every player on one labelled speed graph; export it as an image.'
		},
		{
			label: 'rooms',
			copy: '2–16 players, six-character codes, invite links, optional passwords, owner/mod controls, kicks, and bans.'
		},
		{
			label: 'contests',
			copy: 'Up to 10 stages, public or private pages, count/percent/all cuts, balanced live heats, and persisted standings.'
		},
		{
			label: 'contest tools',
			copy: 'Banner, accent colour, scoped custom CSS, share link, and calendar file.'
		},
		{
			label: 'leaderboard',
			copy: 'Top five, five places around you, username search, and followed players.'
		},
		{
			label: 'friends',
			copy: 'Follow players and see who is online. Room invites arrive in the hub.'
		},
		{
			label: 'profiles',
			copy: 'Display name, bio, public, anonymous, or private visibility, and per-mode run sharing.'
		},
		{
			label: 'data control',
			copy: 'Export everything as JSON, delete unpinned history, or save no run data. Retention keeps the newest 30 unpinned runs plus pinned and best runs.'
		},
		{
			label: 'themes',
			copy: 'Seven palettes. Mono is the default; cycle the rest from the practice header.'
		},
		{
			label: 'offline',
			copy: 'Practice and Zen run from the WASM build when the API is down.'
		},
		{
			label: 'companion',
			copy: 'Installable Android PWA: hub, contests, leaderboard, friends, profiles, replays, settings, and admin. Typing stays on desktop.'
		},
		{
			label: 'admin',
			copy: 'Hidden from non-admins: users, bugs, activity, health, logs, events, and audit records.'
		},
		{
			label: 'reports',
			copy: 'In-app bug form with the page and steps; admins can triage from the same surface.'
		},
		{
			label: 'docs',
			copy: 'Searchable manual covering modes, Elo, contest hosting, snippets, and troubleshooting.'
		},
		{
			label: 'access',
			copy: 'Practice, Zen, public contest pages, docs, and this page work without an account. Elo, rooms, friends, and admin need sign-in.'
		}
	];
</script>

<svelte:head>
	<title>features · OSDTYP</title>
	<meta
		name="description"
		content="OSDTYP is a typing arena for programmers: generated code, 30/90/300s ranked ladders, rooms, multi-stage contests, replays, and an installable Android companion."
	/>
	<meta property="og:title" content="features · OSDTYP" />
	<meta
		property="og:description"
		content="Generated code, three ranked clocks, 2–16 player rooms, multi-stage contests, replays, and a phone companion."
	/>
</svelte:head>

<div class="flex min-h-dvh flex-col scroll-smooth bg-(--bg) text-(--silver)">
	<header class="sticky top-0 z-40 border-b border-(--border) bg-(--bg)">
		<div
			class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-5 py-3 md:px-8"
		>
			<a href={resolve('/play')} class="flex items-baseline gap-3" aria-label="OSDTYP hub">
				<span class="title-text text-3xl leading-none text-(--silver)">OSDTYP</span>
				<span
					class="hidden font-mono text-[9px] tracking-[0.14em] text-(--silver)/40 uppercase sm:inline"
					>code typing arena</span
				>
			</a>
			<nav
				class="flex flex-wrap items-center gap-1.5 font-mono text-[10px] tracking-[0.08em] uppercase"
				aria-label="Page links"
			>
				<a href="#modes" class="ui-btn ui-btn-sm">modes</a>
				<a href="#features" class="ui-btn ui-btn-sm wide-only">features</a>
				<a href={resolve('/docs')} class="ui-btn ui-btn-sm wide-only">docs</a>
				<a href={resolve('/play')} class="ui-btn ui-btn-sm ui-btn-primary narrow-only">hub</a>
				<a href={resolve('/play/me')} class="ui-btn ui-btn-sm ui-btn-primary wide-only"
					>start typing →</a
				>
			</nav>
		</div>
	</header>

	<main class="mx-auto w-full max-w-6xl flex-1 px-5 md:px-8">
		<section class="grid gap-8 py-10 md:grid-cols-[0.95fr_1.05fr] md:items-center md:py-16">
			<div>
				<p class="ui-eyebrow">v0.0.1 · first public build</p>
				<h1
					class="title-text mt-3 text-6xl leading-[0.9] tracking-tight text-(--silver) sm:text-7xl lg:text-8xl"
				>
					Type real code.<br />Race the clock.
				</h1>
				<p class="mt-5 max-w-xl font-mono text-sm leading-7 text-(--silver)/55">
					OSDTYP is a typing arena for programmers. Generated code, server-authoritative scoring,
					ranked ladders, rooms, contests, and replays. Practice is free without an account.
				</p>
				<div class="mt-6 flex flex-wrap gap-2">
					<a href={resolve('/play/me')} class="ui-btn ui-btn-primary desktop-only"
						>start practice →</a
					>
					<a href={resolve('/play')} class="ui-btn ui-btn-primary mobile-only"
						>open the companion hub →</a
					>
					<a href="#modes" class="ui-btn">see the modes</a>
				</div>
				<p class="mt-4 max-w-md font-mono text-[10px] leading-5 text-(--silver)/30">
					Desktop for typing. Android install for the hub, leaderboard, profiles, friends, and
					admin.
				</p>
			</div>

			<figure class="min-w-0">
				<div class="border border-(--border) bg-(--fbg) p-1.5">
					<img
						src="/landing/practice.webp"
						alt="OSDTYP practice screen with a Go snippet and a 30-second timer"
						width="1440"
						height="900"
						class="block h-auto w-full"
						loading="eager"
						fetchpriority="high"
						decoding="async"
					/>
				</div>
				<figcaption
					class="mt-2 flex items-center justify-between gap-3 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase"
				>
					<span>practice · go · 30s</span>
					<span>mono theme</span>
				</figcaption>
			</figure>
		</section>

		<div class="grid grid-cols-2 gap-px border-y border-(--border) bg-(--border) md:grid-cols-4">
			<div class="bg-(--bg) p-4">
				<div class="ui-eyebrow">ranked clocks</div>
				<div class="mt-1 font-mono text-xl text-(--silver)">30 / 90 / 300s</div>
				<p class="mt-1.5 font-mono text-[10px] leading-5 text-(--silver)/35">
					three independent Elos
				</p>
			</div>
			<div class="bg-(--bg) p-4">
				<div class="ui-eyebrow">languages</div>
				<div class="mt-1 font-mono text-xl text-(--silver)">10</div>
				<p class="mt-1.5 font-mono text-[10px] leading-5 text-(--silver)/35">
					C, C++, Go, Haskell, HTML, Java, JavaScript, Python, Rust, SQL
				</p>
			</div>
			<div class="bg-(--bg) p-4">
				<div class="ui-eyebrow">rooms</div>
				<div class="mt-1 font-mono text-xl text-(--silver)">2–16</div>
				<p class="mt-1.5 font-mono text-[10px] leading-5 text-(--silver)/35">players per lobby</p>
			</div>
			<div class="bg-(--bg) p-4">
				<div class="ui-eyebrow">themes</div>
				<div class="mt-1 font-mono text-xl text-(--silver)">7</div>
				<p class="mt-1.5 font-mono text-[10px] leading-5 text-(--silver)/35">
					mono ships as the default
				</p>
			</div>
		</div>

		<section id="modes" class="scroll-mt-20 py-12 md:py-16">
			<div class="flex flex-wrap items-end justify-between gap-3 border-b border-(--border) pb-4">
				<div>
					<p class="ui-eyebrow">modes</p>
					<h2 class="title-text mt-1 text-5xl leading-none text-(--silver) md:text-6xl">
						Five ways in
					</h2>
				</div>
				<p class="max-w-sm font-mono text-[11px] leading-6 text-(--silver)/40">
					Same snippet engine. Same keyboard. Different stakes.
				</p>
			</div>

			<div class="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
				{#each modes as mode, index (mode.name)}
					<a
						href={resolve(mode.href)}
						class="ui-card group flex min-h-56 flex-col justify-between p-5 transition-colors hover:border-(--mer)/60"
					>
						<div>
							<div class="flex items-start justify-between gap-3">
								<div class="flex items-center gap-3">
									<ModeIcon name={mode.icon} class="h-10 w-10 shrink-0 text-(--mer)/70" />
									<div>
										<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/25">
											0{index + 1}
										</div>
										<h3 class="text-2xl leading-tight text-(--silver) group-hover:text-(--mer)">
											{mode.name}
										</h3>
									</div>
								</div>
								<span
									class="shrink-0 font-mono text-[8px] tracking-[0.1em] text-(--silver)/25 uppercase"
									>{mode.tag}</span
								>
							</div>
							<p class="mt-4 max-w-md font-mono text-xs leading-6 text-(--silver)/45">
								{mode.copy}
							</p>
						</div>
						<div
							class="mt-6 flex items-center justify-between gap-3 border-t border-(--border) pt-3 font-mono text-[9px] tracking-[0.08em] uppercase"
						>
							<span class="text-(--silver)/30">{mode.meta}</span>
							<span class="text-(--mer)">{mode.cta} →</span>
						</div>
					</a>
				{/each}
			</div>
		</section>

		<section class="border-t border-(--border) py-12 md:py-16">
			<div class="grid items-center gap-6 md:grid-cols-[1.05fr_0.95fr] md:gap-10">
				<figure class="min-w-0">
					<div class="border border-(--border) bg-(--fbg) p-1.5">
						<img
							src="/landing/ranked.webp"
							alt="OSDTYP ranked screen showing separate 30, 90, and 300 second Elo ladders"
							width="1440"
							height="900"
							class="block h-auto w-full"
							loading="lazy"
							decoding="async"
						/>
					</div>
					<figcaption
						class="mt-2 flex items-center justify-between gap-3 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase"
					>
						<span>ranked · three clocks</span>
						<span>independent Elo</span>
					</figcaption>
				</figure>
				<div>
					<p class="ui-eyebrow">ranked</p>
					<h2 class="title-text mt-2 text-5xl leading-none text-(--silver)">
						Three clocks.<br />Three Elos.
					</h2>
					<p class="mt-4 font-mono text-sm leading-7 text-(--silver)/55">
						Your 30, 90, and 300-second ratings are separate. Everyone starts at 400, and the hub
						shows your highest. There are no tiers or promotion games to grind.
					</p>
					<ul class="mt-5 space-y-2 font-mono text-[11px] leading-6 text-(--silver)/45">
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Bots fill an empty queue after 20 seconds and make readable
							corrections.
						</li>
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Flagged runs do not move Elo. Updates are final after
							you acknowledge the result.
						</li>
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Leaderboard: top five, five places around you, username
							search, followed players.
						</li>
					</ul>
					<a href={resolve('/play/ranked')} class="ui-btn ui-btn-primary mt-6"
						>enter the ranked queue →</a
					>
				</div>
			</div>
		</section>

		<section class="border-t border-(--border) py-12 md:py-16">
			<div class="grid items-center gap-6 md:grid-cols-[0.95fr_1.05fr] md:gap-10">
				<div class="md:order-2">
					<figure class="min-w-0">
						<div class="border border-(--border) bg-(--fbg) p-1.5">
							<img
								src="/landing/rooms.webp"
								alt="OSDTYP room lobby with a host, three bots, a room code, and an invite panel"
								width="1440"
								height="900"
								class="block h-auto w-full"
								loading="lazy"
								decoding="async"
							/>
						</div>
						<figcaption
							class="mt-2 flex items-center justify-between gap-3 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase"
						>
							<span>rooms · lobby</span>
							<span>bots + humans</span>
						</figcaption>
					</figure>
				</div>
				<div class="md:order-1">
					<p class="ui-eyebrow">rooms</p>
					<h2 class="title-text mt-2 text-5xl leading-none text-(--silver)">
						A room code<br />and a clock.
					</h2>
					<p class="mt-4 font-mono text-sm leading-7 text-(--silver)/55">
						Open a public room or keep it behind a link. Pick the language and clock, then start
						when the lobby is ready. Bots can take the empty seats.
					</p>
					<ul class="mt-5 space-y-2 font-mono text-[11px] leading-6 text-(--silver)/45">
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> 2–16 players, six-character codes, invite links, and
							an optional password.
						</li>
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Owner and mod controls: promote, kick, ban, close.
						</li>
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Room results share one speed graph and keep replay links.
						</li>
					</ul>
					<a href={resolve('/play/rooms')} class="ui-btn ui-btn-primary mt-6"
						>open the room board →</a
					>
				</div>
			</div>
		</section>

		<section class="border-t border-(--border) py-12 md:py-16">
			<div class="grid items-center gap-6 md:grid-cols-[1.05fr_0.95fr] md:gap-10">
				<figure class="min-w-0">
					<div class="border border-(--border) bg-(--fbg) p-1.5">
						<img
							src="/landing/contests.webp"
							alt="OSDTYP contest page with a countdown, entrants, funnel, and stage board"
							width="1440"
							height="900"
							class="block h-auto w-full"
							loading="lazy"
							decoding="async"
						/>
					</div>
					<figcaption
						class="mt-2 flex items-center justify-between gap-3 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase"
					>
						<span>contests · event page</span>
						<span>funnel + timing tower</span>
					</figcaption>
				</figure>
				<div>
					<p class="ui-eyebrow">contests</p>
					<h2 class="title-text mt-2 text-5xl leading-none text-(--silver)">
						Events with<br />more than one round.
					</h2>
					<p class="mt-4 font-mono text-sm leading-7 text-(--silver)/55">
						Contests are scheduled pages, not rooms. Enroll, wait for the stage window, then type.
						The event page keeps the whole funnel after the final.
					</p>
					<ul class="mt-5 space-y-2 font-mono text-[11px] leading-6 text-(--silver)/45">
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Qualifiers, practice stages, HotLaps, and live finals.
							Up to 10 stages.
						</li>
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Advance a fixed count, a percentage, or everyone who
							clears the rule. Live stages use balanced heats.
						</li>
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Public timing tower: top 10, exact position lookup,
							animated changes.
						</li>
						<li class="flex gap-2">
							<span class="text-(--mer)">›</span> Scoped custom CSS, banner and accent, calendar file.
							Results and standings persist.
						</li>
					</ul>
					<a href={resolve('/contests')} class="ui-btn ui-btn-primary mt-6"
						>see the contest board →</a
					>
				</div>
			</div>
		</section>

		<section class="border-t border-(--border) py-12 md:py-16">
			<div class="grid gap-6 md:grid-cols-[0.8fr_1.2fr] md:gap-10">
				<div>
					<p class="ui-eyebrow">replays + graphs</p>
					<h2 class="title-text mt-2 text-5xl leading-none text-(--silver)">
						The run<br />doesn't disappear.
					</h2>
					<p class="mt-4 max-w-md font-mono text-sm leading-7 text-(--silver)/55">
						Scored runs can keep their keystrokes. Open a replay to watch the original timing, key
						by key, or put every player on one speed graph.
					</p>
					<a href={resolve('/docs')} class="ui-btn mt-6">how replays work →</a>
				</div>
				<div class="grid gap-3 sm:grid-cols-2">
					<div class="ui-card p-4">
						<div class="ui-eyebrow">01 · keystrokes</div>
						<p class="mt-2 font-mono text-[11px] leading-6 text-(--silver)/45">
							The replay player steps through the recorded run at the original timing.
						</p>
					</div>
					<div class="ui-card p-4">
						<div class="ui-eyebrow">02 · keyboard</div>
						<p class="mt-2 font-mono text-[11px] leading-6 text-(--silver)/45">
							A compact keyboard lights up pressed keys, corrections, and backspaces.
						</p>
					</div>
					<div class="ui-card p-4">
						<div class="ui-eyebrow">03 · share</div>
						<p class="mt-2 font-mono text-[11px] leading-6 text-(--silver)/45">
							Public replay links. They do not expose account settings or the rest of your profile.
						</p>
					</div>
					<div class="ui-card p-4">
						<div class="ui-eyebrow">04 · export</div>
						<p class="mt-2 font-mono text-[11px] leading-6 text-(--silver)/45">
							The result graph compares all players and can be exported as an image.
						</p>
					</div>
				</div>
			</div>
		</section>

		<section class="border-t border-(--border) py-12 md:py-16">
			<div class="flex flex-wrap items-end justify-between gap-3 border-b border-(--border) pb-4">
				<div>
					<p class="ui-eyebrow">desktop + android</p>
					<h2 class="title-text mt-1 text-5xl leading-none text-(--silver) md:text-6xl">
						Desktop for typing.<br class="sm:hidden" /> Phone for the rest.
					</h2>
				</div>
				<a href={resolve('/play')} class="ui-btn">open the hub →</a>
			</div>

			<div class="mt-5 grid gap-4 lg:grid-cols-[1.45fr_0.55fr]">
				<figure class="min-w-0">
					<div class="border border-(--border) bg-(--fbg) p-1.5">
						<img
							src="/landing/hub.webp"
							alt="OSDTYP desktop hub with practice, ranked, rooms, Zen, contests, and account panels"
							width="1440"
							height="900"
							class="block h-auto w-full"
							loading="lazy"
							decoding="async"
						/>
					</div>
					<figcaption
						class="mt-2 flex items-center justify-between gap-3 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase"
					>
						<span>hub · desktop</span>
						<span>practice · ranked · rooms · contests</span>
					</figcaption>
				</figure>

				<figure class="flex min-w-0 flex-col items-center border border-(--border) bg-(--fbg) p-4">
					<div class="w-full max-w-60 border border-(--border) bg-(--bg) p-1.5">
						<img
							src="/landing/companion.webp"
							alt="OSDTYP hub on an Android phone with the install app button visible"
							width="390"
							height="844"
							class="block h-auto w-full"
							loading="lazy"
							decoding="async"
						/>
					</div>
					<figcaption
						class="mt-3 w-full max-w-60 font-mono text-[9px] leading-5 tracking-[0.08em] text-(--silver)/30 uppercase"
					>
						installable PWA · starts at the hub
					</figcaption>
				</figure>
			</div>

			<div class="mt-4 grid gap-3 md:grid-cols-3">
				<div class="border border-(--border) p-4">
					<div class="ui-eyebrow">companion</div>
					<p class="mt-2 font-mono text-[11px] leading-6 text-(--silver)/45">
						Install from Chrome on Android. It opens the hub in a standalone window, not a typing
						screen.
					</p>
				</div>
				<div class="border border-(--border) p-4">
					<div class="ui-eyebrow">offline shell</div>
					<p class="mt-2 font-mono text-[11px] leading-6 text-(--silver)/45">
						The app shell and static assets cache. Live games, admin data, and invites stay
						network-only.
					</p>
				</div>
				<div class="border border-(--border) p-4">
					<div class="ui-eyebrow">scope</div>
					<p class="mt-2 font-mono text-[11px] leading-6 text-(--silver)/45">
						Admin is inside the PWA for admins. Typing surfaces still send phones to the desktop
						gate. The desktop web layout is unchanged.
					</p>
				</div>
			</div>
		</section>

		<section id="features" class="scroll-mt-20 border-t border-(--border) py-12 md:py-16">
			<div class="flex flex-wrap items-end justify-between gap-3 border-b border-(--border) pb-4">
				<div>
					<p class="ui-eyebrow">features</p>
					<h2 class="title-text mt-1 text-5xl leading-none text-(--silver) md:text-6xl">
						Everything else
					</h2>
				</div>
				<p class="max-w-sm font-mono text-[11px] leading-6 text-(--silver)/40">
					The full index, without the launch-day adjectives.
				</p>
			</div>

			<div class="grid gap-x-10 md:grid-cols-2">
				{#each features as feature (feature.label)}
					<div
						class="grid grid-cols-[88px_1fr] gap-3 border-b border-(--border) py-3.5 font-mono text-[11px] leading-6"
					>
						<div class="text-(--mer)/70 uppercase">{feature.label}</div>
						<p class="text-(--silver)/45">{feature.copy}</p>
					</div>
				{/each}
			</div>
		</section>

		<section class="border-t border-(--border) py-12 md:py-16">
			<div class="flex flex-wrap items-end justify-between gap-4">
				<div>
					<p class="ui-eyebrow">ready</p>
					<h2 class="title-text mt-1 text-6xl leading-none text-(--silver) md:text-7xl">
						Open the arena.
					</h2>
				</div>
				<div class="flex flex-wrap gap-2">
					<a href={resolve('/play/me')} class="ui-btn ui-btn-primary desktop-only"
						>start practice →</a
					>
					<a href={resolve('/play')} class="ui-btn ui-btn-primary mobile-only">open the hub →</a>
					<a href={resolve('/contests')} class="ui-btn">contest board</a>
					<a href={resolve('/docs')} class="ui-btn">docs</a>
				</div>
			</div>
		</section>
	</main>

	<footer class="mt-auto border-t border-(--border)">
		<div
			class="mx-auto flex max-w-6xl flex-col gap-4 px-5 py-6 md:flex-row md:items-center md:justify-between md:px-8"
		>
			<div>
				<div class="title-text text-3xl leading-none text-(--silver)">OSDTYP</div>
				<p class="mt-1 font-mono text-[10px] leading-5 text-(--silver)/35">
					v0.0.1 · experimental · a typing arena for people who write code.
				</p>
				<p class="font-mono text-[10px] leading-5 text-(--red)/65">
					First public build — bugs expected.
					<a href={resolve('/bugs')} class="underline">Report them.</a>
				</p>
			</div>
			<nav class="flex flex-wrap gap-2 font-mono text-[9px] uppercase" aria-label="Footer links">
				<a href={resolve('/play')} class="ui-btn ui-btn-sm">hub</a>
				<a href={resolve('/docs')} class="ui-btn ui-btn-sm">docs</a>
				<a href={resolve('/bugs')} class="ui-btn ui-btn-sm">report bug</a>
				<a
					href="https://github.com/ItsArnavSh/osdtype"
					target="_blank"
					rel="noreferrer"
					class="ui-btn ui-btn-sm border-(--mer)/40 text-(--mer)">github</a
				>
			</nav>
		</div>
	</footer>
</div>

<style>
	/* `.ui-btn` is declared after Tailwind's utilities, so responsive
	   `hidden` classes on the button itself get overridden by its
	   `display: inline-flex`. Keep the breakpoint decisions here. */
	@media (max-width: 639px) {
		:global(.wide-only) {
			display: none !important;
		}
	}

	@media (min-width: 640px) {
		:global(.narrow-only) {
			display: none !important;
		}
	}

	@media (max-width: 767px) {
		:global(.desktop-only) {
			display: none !important;
		}
	}

	@media (min-width: 768px) {
		:global(.mobile-only) {
			display: none !important;
		}
	}
</style>
