<script lang="ts">
	import { resolve } from '$app/paths';

	const sections = [
		{ id: 'start', label: 'Getting started' },
		{ id: 'modes', label: 'Game modes' },
		{ id: 'ranked', label: 'Ranked and Elo' },
		{ id: 'practice', label: 'Practice and Zen' },
		{ id: 'rooms', label: 'Rooms and friends' },
		{ id: 'contests', label: 'Contests' },
		{ id: 'hosting', label: 'Hosting an event' },
		{ id: 'replays', label: 'Replays and graphs' },
		{ id: 'resrap', label: 'Resrap snippets' },
		{ id: 'how', label: 'How it works' },
		{ id: 'troubleshooting', label: 'Troubleshooting' },
		{ id: 'about', label: 'About' }
	];

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

	// A small, hand-written index keeps docs search usable without shipping a
	// full-text index. Keywords cover the words people actually look for.
	const searchIndex = [
		{ id: 'start', label: 'Getting started', keywords: 'install setup run local dev prod make account guest' },
		{ id: 'modes', label: 'Game modes', keywords: 'ranked practice zen rooms contest replay mode' },
		{ id: 'ranked', label: 'Ranked and Elo', keywords: 'elo rating queue ladder bot 30 90 300 rating delta' },
		{ id: 'practice', label: 'Practice and Zen', keywords: 'practice zen offline bots hotlap warmup' },
		{ id: 'rooms', label: 'Rooms and friends', keywords: 'room invite friend follow private lobby password' },
		{ id: 'contests', label: 'Contests', keywords: 'contest qualifier hotlaps practice stage funnel live timing top 10 lookup' },
		{ id: 'hosting', label: 'Hosting an event', keywords: 'host contest create stage advance percent custom css banner' },
		{ id: 'replays', label: 'Replays and graphs', keywords: 'replay graph speed keyboard share link result' },
		{ id: 'resrap', label: 'Resrap snippets', keywords: 'resrap codegen snippet seed language rust wasm' },
		{ id: 'how', label: 'How it works', keywords: 'stack architecture server websocket postgres redis svelte go' },
		{ id: 'troubleshooting', label: 'Troubleshooting', keywords: 'error broken stuck queue contest attempt css' },
		{ id: 'about', label: 'About', keywords: 'about arnav osdc github author project principles stack' }
	];

	let docQuery = $state('');

	let searchResults = $derived.by(() => {
		const needle = docQuery.trim().toLowerCase();
		if (!needle) return [];
		return searchIndex.filter(
			(section) =>
				section.label.toLowerCase().includes(needle) || section.keywords.includes(needle)
		);
	});

	const modes = [
		{
			name: 'Ranked',
			href: '/play/ranked',
			copy: 'Queue for 30s, 90s or 300s. Each clock has its own Elo ladder. A bot fills the match when nobody else is waiting.',
			tag: 'account'
		},
		{
			name: 'Practice',
			href: '/play',
			copy: 'A normal timed run with generated code. Run it solo or race adaptive bots. Practice never changes ranked rating.',
			tag: 'works offline'
		},
		{
			name: 'Zen',
			href: '/play/zen',
			copy: 'No clock and no pressure. Type through generated code with the same engine used by every other mode.',
			tag: 'works offline'
		},
		{
			name: 'Rooms',
			href: '/play/rooms',
			copy: 'Create a public or private room, share the code, and race up to 16 players with invite links and optional moderation.',
			tag: 'account'
		},
		{
			name: 'Contests',
			href: '/contests',
			copy: 'Scheduled standalone events with enrollment, public or private pages, multi-stage funnels, and host-controlled rules.',
			tag: 'public pages'
		}
	];

	const stageTypes = [
		{
			name: 'Qualifier',
			copy: 'Players submit attempts during a stage window. A minimum WPM line decides who is eligible. The host can fix the language or let each player choose. Try limits can be finite or unlimited.'
		},
		{
			name: 'HotLaps',
			copy: 'Everyone gets the same snippet and seed. The host sets a try limit (0 means unlimited). The fastest successful run rises to the top and the host decides how many advance.'
		},
		{
			name: 'Practice / warm-up',
			copy: 'A shared-seed warm-up inside the event. Anyone eligible can post times on the board, but nobody is eliminated: every player advances to the next stage.'
		},
		{
			name: 'Live multiplayer',
			copy: 'A scheduled live race. If more players advance than one heat can hold, the server splits the field into balanced heats of at most 10. Opponent cursors stay visible but blurred.'
		},
		{
			name: 'Ranked-style final',
			copy: 'The last stage behaves like a ranked match. It uses the same live race rules and produces the final event leaderboard. Its result cut can classify everyone, a fixed top N, or a percentage.'
		}
	];

	const languages = ['C', 'C++', 'Go', 'Haskell', 'HTML', 'Java', 'JavaScript', 'Python', 'Rust', 'SQL'];

	const cssExample = `.contest-title { letter-spacing: 0.18em; }
.contest-stage.is-active { border-color: var(--mer); }
.contest-cta { background: var(--mer); color: var(--bg); }`;
</script>

<svelte:head>
	<title>docs · OSDTYP</title>
	<meta
		name="description"
		content="How OSDTYP works: ranked ladders, practice, rooms, contest funnels, replays, resrap snippets and the stack behind the typing arena."
	/>
</svelte:head>

<div class="min-h-screen bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<div class="flex flex-wrap items-center gap-3">
			<a href={resolve('/play')} class="title-text text-3xl leading-none text-(--silver)">OSDTYP</a>
			<span class="font-mono text-[10px] tracking-[0.14em] text-(--mer) uppercase">docs</span>
		</div>
		<nav class="flex flex-wrap items-center gap-2 font-mono text-[10px] tracking-[0.1em] uppercase">
			<a
				href={resolve('/docs')}
				class="border border-(--mer) px-3 py-1.5 text-(--mer)"
				aria-current="page">docs</a
			>
			<a
				href="#about"
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>about</a
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

	<section class="mx-auto mt-8 max-w-5xl border-b border-(--border) pb-8">
		<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">the manual</div>
		<h1 class="mt-3 text-5xl leading-none text-(--silver) md:text-7xl">How OSDTYP works</h1>
		<p class="mt-4 max-w-3xl text-sm leading-relaxed text-(--silver)/50">
			OSDTYP is a speedtyping arena for source code. It has ranked ladders, offline practice,
			multiplayer rooms, and standalone contest events with multi-stage funnels. This page covers
			every mode, the contest format, replays, the resrap snippet generator, and the parts behind
			the browser.
		</p>
		<div class="mt-5 flex flex-wrap gap-2 font-mono text-[9px] tracking-[0.1em] uppercase">
			<a
				href="#start"
				class="border border-(--mer) px-3 py-1.5 text-(--mer) hover:bg-(--mer)/10">start here</a
			>
			<a
				href="#contests"
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>contest funnels</a
			>
			<a
				href="#resrap"
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>resrap</a
			>
		</div>

		<div class="mt-5 max-w-md">
			<label class="flex flex-col gap-1">
				<span class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase"
					>search the docs</span
				>
				<input
					class="w-full border border-(--border) bg-(--fbg) px-3 py-2 font-mono text-xs text-(--silver) outline-none placeholder:text-(--silver)/20 focus:border-(--mer)"
					bind:value={docQuery}
					placeholder="ranked, contests, replays, hosting…"
					maxlength="60"
				/>
			</label>
			{#if docQuery.trim()}
				<div class="mt-2 flex flex-col gap-1 border border-(--border) bg-(--fbg) p-2">
					{#if searchResults.length === 0}
						<p class="px-2 py-1 font-mono text-[10px] text-(--silver)/25">
							No section matches “{docQuery.trim()}”.
						</p>
					{:else}
						{#each searchResults as result (result.id)}
							<a
								href={`#${result.id}`}
								class="border border-(--border)/60 px-2 py-1.5 font-mono text-[10px] text-(--silver)/50 hover:border-(--mer)/50 hover:text-(--mer)"
								onclick={() => (docQuery = '')}
								>{result.label}</a
							>
						{/each}
					{/if}
				</div>
			{/if}
		</div>
	</section>

	<div class="mx-auto mt-6 grid max-w-5xl gap-6 lg:grid-cols-[210px_minmax(0,1fr)]">
		<aside class="hidden lg:block">
			<div class="sticky top-5 border border-(--border) bg-(--fbg) p-4">
				<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/30 uppercase">
					on this page
				</div>
				<nav class="mt-3 flex flex-col gap-1.5 font-mono text-[10px]">
					{#each sections as section (section.id)}
						<a href={`#${section.id}`} class="text-(--silver)/40 hover:text-(--mer)"
							>{section.label}</a
						>
					{/each}
				</nav>
			</div>
		</aside>

		<main class="flex min-w-0 flex-col gap-6">
			<section id="start" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Getting started</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					Open the play hub. Practice and Zen work without an account and keep working when the Go
					backend is offline. Ranked, rooms, contests, profiles, friends and saved results need an
					account.
				</p>
				<div class="mt-4 grid gap-2 sm:grid-cols-2">
					<a
						href={resolve('/play')}
						class="border border-(--border) px-4 py-3 font-mono text-xs text-(--silver)/65 hover:border-(--mer) hover:text-(--mer)"
						>-> open the play hub</a
					>
					<a
						href={resolve('/play/zen')}
						class="border border-(--border) px-4 py-3 font-mono text-xs text-(--silver)/65 hover:border-(--mer) hover:text-(--mer)"
						>-> try Zen without an account</a
					>
					<a
						href={resolve('/login')}
						class="border border-(--border) px-4 py-3 font-mono text-xs text-(--silver)/65 hover:border-(--mer) hover:text-(--mer)"
						>-> sign in</a
					>
					<a
						href={resolve('/contests')}
						class="border border-(--border) px-4 py-3 font-mono text-xs text-(--silver)/65 hover:border-(--mer) hover:text-(--mer)"
						>-> browse contests</a
					>
				</div>
				<p class="mt-4 font-mono text-[10px] leading-relaxed text-(--silver)/30">
					The app is keyboard-first. Use the on-screen back links and the play hub to move between
					modes. Ranked and contest games take over the tab until the run is acknowledged.
				</p>
			</section>

			<section id="modes" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Game modes</h2>
				<div class="mt-4 grid gap-3">
					{#each modes as mode (mode.name)}
						<a
							href={mode.href}
							class="group border border-(--border) p-4 hover:border-(--mer)"
						>
							<div class="flex flex-wrap items-center justify-between gap-2">
								<h3 class="text-lg text-(--silver) group-hover:text-(--mer)">{mode.name}</h3>
								<span class="font-mono text-[8px] tracking-[0.12em] text-(--silver)/25 uppercase"
									>{mode.tag}</span
								>
							</div>
							<p class="mt-2 text-sm leading-relaxed text-(--silver)/45">{mode.copy}</p>
						</a>
					{/each}
				</div>
			</section>

			<section id="ranked" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Ranked and Elo</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					The three ranked clocks are separate ladders. Your 30s, 90s and 300s ratings all start at
					400 and move independently. The hub shows your highest ladder. There are no tiers,
					promotion games or progress bars to grind.
				</p>
				<div class="mt-4 grid gap-3 sm:grid-cols-3">
					<div class="border border-(--border) p-4">
						<div class="font-mono text-2xl text-(--mer)">400</div>
						<div class="mt-1 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase">
							starting Elo on every clock
						</div>
					</div>
					<div class="border border-(--border) p-4">
						<div class="font-mono text-2xl text-(--mer)">3</div>
						<div class="mt-1 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase">
							independent ladders
						</div>
					</div>
					<div class="border border-(--border) p-4">
						<div class="font-mono text-2xl text-(--mer)">5</div>
						<div class="mt-1 font-mono text-[9px] tracking-[0.1em] text-(--silver)/30 uppercase">
							places around you on the board
						</div>
					</div>
				</div>
				<p class="mt-4 text-sm leading-relaxed text-(--silver)/45">
					Matchmaking uses the clock you picked. If nobody joins in time, an adaptive bot fills the
					race. A ranked bot makes readable mistakes and corrects them like a player. When a
					mistake is recorded, the wrong key and the correction land together, so the scored run
					never contains a half-typed bad state.
				</p>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/45">
					The leaderboard shows the top five, five places around your current position, username
					search, and followed players. Ranked games can be replayed from the result screen.
				</p>
				<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/30">
					Flagged runs do not change rating. Ranked updates are final after you acknowledge the
					result.
				</p>
			</section>

			<section id="practice" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Practice and Zen</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					Practice is a normal timed run with generated code. You can race adaptive bots or just
					take the run yourself. Practice does not touch ranked Elo, but logged-in runs can be
					kept in your profile history and replayed.
				</p>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/45">
					Zen removes the clock. The snippet still comes from the same generator, so movement and
					text feel match the competitive modes. Practice and Zen use the browser WASM build of
					resrap, so they can run even if the API is down.
				</p>
			</section>

			<section id="rooms" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Rooms and friends</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					Rooms are short-lived lobbies with six-character codes. A host can make a room public or
					private, set a password, choose the clock and language, and start when enough players are
					ready. Room owners can moderate members and send invites.
				</p>
				<ul class="mt-3 list-disc space-y-2 pl-5 text-sm leading-relaxed text-(--silver)/45">
					<li>Up to 16 players in a room.</li>
					<li>Public rooms can be joined from the room board.</li>
					<li>Private rooms use invite links and may also use a password.</li>
					<li>Room results show every player on one speed graph and support replays.</li>
					<li>
						Following a player adds them to your friends panel. Friends show online state and can
						be searched from the ranked leaderboard.
					</li>
				</ul>
			</section>

			<section id="contests" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Contests</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					Contests are standalone events. They are not rooms and do not need a lobby. Every event
					has a public page with its own countdown, rules, participants, schedule and final board.
					Hosts enroll automatically. Events can be public or invite-only.
				</p>
				<div class="mt-4 grid gap-3 sm:grid-cols-2">
					<div class="border border-(--border) p-4">
						<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">public</div>
						<p class="mt-2 text-sm leading-relaxed text-(--silver)/45">
							Listed on the contest board. Anyone with an account can enroll. Public events never
							use a password.
						</p>
					</div>
					<div class="border border-(--border) p-4">
						<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">private</div>
						<p class="mt-2 text-sm leading-relaxed text-(--silver)/45">
							Hidden from public listings. The invite link is the entry point. A host may add an
							optional password on top of the link.
						</p>
					</div>
				</div>
				<p class="mt-4 text-sm leading-relaxed text-(--silver)/45">
					A contest is an ordered funnel. The first stage starts with the event. Later stages start
					a host-set delay after the previous stage closes. Each stage defines its own game type,
					clock, rules and advancement target.
				</p>
				<div class="mt-4 grid gap-3">
					{#each stageTypes as stage (stage.name)}
						<div class="border border-(--border) p-4">
							<h3 class="font-mono text-xs tracking-[0.1em] text-(--mer) uppercase">{stage.name}</h3>
							<p class="mt-2 text-sm leading-relaxed text-(--silver)/45">{stage.copy}</p>
						</div>
					{/each}
				</div>
				<p class="mt-4 text-sm leading-relaxed text-(--silver)/45">
					Advancement can be a fixed player count, a percentage, or every player who meets the
					stage rule. Quick presets cover top 1, top 3, top 10, top 10%, top 25% and top 50%, while
					free count and percent values are still available. Percentages round up so a small field
					still moves someone on. A stage window closes at a fixed time, and all attempts inside it
					are aggregated. Live stages use balanced heats, so a field of 27 with a heat size of 10
					becomes three races.
				</p>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/45">
					Live events expose a public timing tower. It shows the current top 10 with animated
					position changes every few seconds, and the lookup box finds any enrolled player's exact
					position even when they are outside the top 10.
				</p>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/45">
					The final stage produces the event leaderboard. Its funnel-result cut decides whether
					everyone is classified, or only the top count/percentage appears in the final result.
					The contest page keeps the full funnel visible after the event, including each stage's
					standings and who advanced.
				</p>
			</section>

			<section id="hosting" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Hosting an event</h2>
				<ol class="mt-3 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-(--silver)/50">
					<li>Open the contest board and choose host contest.</li>
					<li>Add a title, description and optional banner. Markdown is supported in the description.</li>
					<li>Set the event start, total capacity, visibility and optional private password.</li>
					<li>
						Build the funnel. Add up to 10 stages and order them. The centred + between stages
						opens the stage picker, so a stage can be inserted before, between or after the
						existing ones.
					</li>
					<li>
						For each stage, pick the game type, run clock, stage window or heat size, language,
						tries and minimum WPM where relevant.
					</li>
					<li>
						Choose how many players advance. The last stage also has a result cut: all
						finishers, a fixed top N, or a percentage.
					</li>
					<li>Publish, share the link, and download the calendar file if you want a local reminder.</li>
				</ol>
				<p class="mt-4 text-sm leading-relaxed text-(--silver)/45">
					Hosts can edit an event while it is upcoming. Editing replaces the funnel and reschedules
					the first stage. Once an event starts, results and advancement are server-controlled.
				</p>
				<div class="mt-4 border border-(--border) p-4">
					<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">
						custom CSS
					</div>
					<p class="mt-2 text-sm leading-relaxed text-(--silver)/45">
						Each event can carry its own CSS. It is scoped to the event page and cannot leak into
						the rest of OSDTYP. The editor includes a guide button with the available classes.
						Imports and external URLs are blocked.
					</p>
					<pre
						class="mt-3 overflow-x-auto border border-(--border) bg-(--bg) p-3 font-mono text-[10px] leading-relaxed text-(--silver)/50">{cssExample}</pre>
				</div>
			</section>

			<section id="replays" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Replays and graphs</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					Every scored run can keep its input recording. The result page combines all players on a
					labelled speed graph and lets you export the image. Open a replay to step through the
					recorded run at the original timing.
				</p>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/45">
					The replay player shows a compact keyboard below the text. Keys light up when they are
					pressed in the recording, including corrections and backspaces. Replays can be shared
					with a public link.
				</p>
				<p class="mt-3 font-mono text-[10px] leading-relaxed text-(--silver)/30">
					Replay pages are public when you have the link. They do not expose account settings or
					the rest of your profile.
				</p>
			</section>

			<section id="resrap" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Resrap snippets</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					Resrap is the Rust snippet generator behind every run. The project ships
					<code class="border border-(--border) bg-(--bg) px-1">resrap-rs 1.0.1</code> in the HTTP
					service and in the browser WASM wrapper. Both builds get the same language, seed and token
					count, so the server and the browser produce byte-identical text.
				</p>
				<div class="mt-4 flex flex-wrap gap-1.5">
					{#each languages as language (language)}
						<span class="border border-(--border) px-2 py-1 font-mono text-[9px] text-(--silver)/40"
							>{language}</span
						>
					{/each}
				</div>
				<p class="mt-4 text-sm leading-relaxed text-(--silver)/45">
					A snippet request has three inputs: a language template, a 63-bit seed, and a token
					target. Resrap builds a complete program from the template. The seed drives generated
					identifiers, branches and expression choices. The server generates the seed, stores it on
					the game, and sends the snippet to every player.
				</p>
				<div class="mt-4 border border-(--border) p-4">
					<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">
						why do generated snippets repeat?
					</div>
					<p class="mt-2 text-sm leading-relaxed text-(--silver)/45">
						The opening of a template is mostly imports, helpers and the same program skeleton. The
						seed changes the generated body, not the language boilerplate. A start offset also
						skips about 15 lines and can begin inside that shared preamble, which makes two runs
						look more alike than they are. Different seeds still produce different visible code
						after the skeleton. In a 400-seed check, every visible output was unique.
					</p>
				</div>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/45">
					Random snippets use a fresh cryptographic seed. HotLaps fixes one seed for the stage, so
					every attempt sees the same text and the fastest run is a fair comparison. Practice and
					Zen use browser randomness so they still work offline.
				</p>
			</section>

			<section id="how" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">How it works</h2>
				<p class="mt-3 text-sm leading-relaxed text-(--silver)/55">
					The frontend is SvelteKit and TypeScript. The API and game runtime are Go. PostgreSQL
					stores durable data and Redis handles live queue and state. Websockets keep games,
					rooms, contests and invites in sync. The server owns the clock, the start offset, scoring,
					anti-cheat flags and the authoritative game state.
				</p>
				<div class="mt-4 grid gap-2 font-mono text-[10px] text-(--silver)/45 sm:grid-cols-2">
					<div class="border border-(--border) p-3">app/ - SvelteKit client and WASM generator</div>
					<div class="border border-(--border) p-3">
						server/ - Go API, matchmaking, rooms, contests, runtime
					</div>
					<div class="border border-(--border) p-3">codegen/ - resrap-rs HTTP service</div>
					<div class="border border-(--border) p-3">PostgreSQL + Redis - durable and live state</div>
				</div>
				<p class="mt-4 text-sm leading-relaxed text-(--silver)/45">
					Most pages call the API on load and then subscribe to websocket events. When a contest
					stage starts or ends, the event page reloads its funnel. A contest attempt creates a
					normal game with a stage link. When it finishes, the result screen returns you to the
					contest, where the stage board updates after the window closes.
				</p>
			</section>

			<section id="troubleshooting" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<h2 class="text-2xl text-(--silver)">Troubleshooting</h2>
				<div class="mt-3 flex flex-col gap-4">
					<div>
						<h3 class="font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase">
							Practice or Zen will not load
						</h3>
						<p class="mt-1 text-sm leading-relaxed text-(--silver)/45">
							These modes are local. If the page is stale, reload it. They do not need the Go
							server or codegen service.
						</p>
					</div>
					<div>
						<h3 class="font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase">
							Ranked queue is taking a while
						</h3>
						<p class="mt-1 text-sm leading-relaxed text-(--silver)/45">
							Queues are split by clock. The server uses a bot when it cannot find a human
							opponent in time. Cancel and rejoin if the page lost its socket.
						</p>
					</div>
					<div>
						<h3 class="font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase">
							A contest stage will not start an attempt
						</h3>
						<p class="mt-1 text-sm leading-relaxed text-(--silver)/45">
							Check that you are enrolled, that you advanced to the current stage, that tries
							remain, and that there is enough time left in the window for a full run. A live
							stage launches its own heats and cannot be started from the attempt button.
						</p>
					</div>
					<div>
						<h3 class="font-mono text-[10px] tracking-[0.12em] text-(--mer) uppercase">
							Custom CSS did not apply
						</h3>
						<p class="mt-1 text-sm leading-relaxed text-(--silver)/45">
							Only event-page classes are in scope. Imports, raw tags and external URLs are
							blocked. Open the CSS guide in the editor for the current class list.
						</p>
					</div>
				</div>
			</section>

			<footer id="about" class="border border-(--border) bg-(--fbg) p-5 md:p-7">
				<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
					about the project
				</div>
				<h2 class="mt-3 text-3xl text-(--silver)">A typing arena for people who write code.</h2>
				<p class="mt-3 max-w-3xl text-sm leading-relaxed text-(--silver)/50">
					OSDTYP is a speedtyping game built around real source code. It has independent ranked
					ladders, offline practice, multiplayer rooms, and contest events that can run as full
					multi-stage competitions. It is made for the fun of typing fast without hiding the parts
					that make code hard.
				</p>

				<div class="mt-5 grid gap-3 md:grid-cols-3">
					{#each principles as principle (principle.title)}
						<div class="border border-(--border) p-4">
							<div class="font-mono text-[9px] tracking-[0.12em] text-(--mer) uppercase">
								{principle.title}
							</div>
							<p class="mt-2 text-xs leading-relaxed text-(--silver)/45">{principle.copy}</p>
						</div>
					{/each}
				</div>

				<div class="mt-4 grid gap-3 md:grid-cols-2">
					<div class="border border-(--border) p-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/35 uppercase">
							what is inside
						</div>
						<ul class="mt-3 space-y-1.5 text-xs leading-relaxed text-(--silver)/45">
							<li>Ranked queues at 30s, 90s and 300s with separate Elo ladders.</li>
							<li>Practice, Zen, adaptive bots and local snippet generation.</li>
							<li>Public and private rooms with invites, moderation and replayable results.</li>
							<li>Contests with qualifiers, practice stages, HotLaps and live finals.</li>
							<li>Profiles, follows, friends and a searchable leaderboard.</li>
							<li>Speed graphs, input replays and share links.</li>
						</ul>
					</div>
					<div class="border border-(--border) p-4">
						<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/35 uppercase">
							stack
						</div>
						<div class="mt-3 flex flex-wrap gap-x-4 gap-y-1 font-mono text-[10px] text-(--silver)/45">
							<span>SvelteKit + TypeScript</span>
							<span>Go + Gin</span>
							<span>PostgreSQL + Redis</span>
							<span>Rust + WebAssembly</span>
							<span>resrap-rs</span>
						</div>
						<p class="mt-3 text-xs leading-relaxed text-(--silver)/30">
							The server is authoritative for clocks, scoring and match state. The browser owns
							input, rendering and offline practice.
						</p>
					</div>
				</div>

				<div class="mt-4 border border-(--border) p-4">
					<div class="flex flex-wrap items-start justify-between gap-4">
						<div>
							<div class="font-mono text-[9px] tracking-[0.14em] text-(--silver)/35 uppercase">
								built by
							</div>
							<div class="mt-1 text-lg text-(--silver)">Arnav Sharma</div>
							<p class="mt-1 max-w-xl text-xs leading-relaxed text-(--silver)/40">
								The project lives on GitHub and is maintained by Arnav. The links below cover the
								developer community and the personal site.
							</p>
						</div>
						<div class="flex flex-wrap gap-2 font-mono text-[9px] uppercase">
							<a
								href="https://osdc.dev/"
								target="_blank"
								rel="noreferrer"
								class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
								>osdc.dev</a
							>
							<a
								href="https://itsarnavsh.github.io/"
								target="_blank"
								rel="noreferrer"
								class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
								>author site</a
							>
							<a
								href="https://github.com/ItsArnavSh/osdtype"
								target="_blank"
								rel="noreferrer"
								class="border border-(--mer) px-3 py-1.5 text-(--mer) hover:bg-(--mer)/10"
								>github repository</a
							>
						</div>
					</div>
				</div>

				<div
					class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-(--border) pt-4"
				>
					<p class="max-w-xl text-xs leading-relaxed text-(--silver)/40">
						If you want to get faster at typing code, open Practice. If you want a fair match,
						queue Ranked. If you want to run an event, build a contest funnel.
					</p>
					<a
						href={resolve('/play')}
						class="border border-(--mer) px-3 py-1.5 font-mono text-[10px] tracking-[0.1em] text-(--mer) uppercase hover:bg-(--mer)/10"
						>start typing -></a
					>
				</div>
			</footer>
		</main>
	</div>
</div>
