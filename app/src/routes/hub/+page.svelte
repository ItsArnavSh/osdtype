<script lang="ts">
	import {
		createContest,
		listContests,
		type Contest,
		type LeaderboardEntry,
		joinControlledLobby,
		ContestStatus,
		LobbyDuration
	} from '$lib/core/api';
	import { LANGUAGES, type Language } from '$lib/core/entity/languages';
	import { onMount } from 'svelte';

	let contests: Contest[] = $state([]);
	let loading = $state(true);
	let showCreateModal = $state(false);
	let joiningLobby = $state<string | null>(null);

	// expanded contest for leaderboard view
	let expandedID = $state<string | null>(null);

	let form = $state<{
		title: string;
		about: string;
		lang: Language;
		duration: LobbyDuration;
		scheduledDate: string;
		scheduledTime: string;
	}>({
		title: '',
		about: '',
		lang: 'CPP',
		duration: 90,
		scheduledDate: '',
		scheduledTime: ''
	});
	let creating = $state(false);
	let createError = $state('');

	const langLabels: Record<Language, string> = {
		C: 'C',
		Go: 'Go',
		CPP: 'C++',
		Java: 'Java',
		Rust: 'Rust',
		TypeScript: 'TypeScript'
	};

	const langIcons: Record<Language, string> = {
		C: '/cpp.png',
		Go: '/cpp.png',
		CPP: '/cpp.png',
		Java: '/python.png',
		Rust: '/cpp.png',
		TypeScript: '/html.png'
	};

	const durationLabels: Record<LobbyDuration, string> = {
		30: '30s Sprint',
		90: '90s Standard',
		300: '300s Marathon'
	};

	const statusLabel: Record<number, string> = {
		[ContestStatus.UPCOMING]: 'UPCOMING',
		[ContestStatus.LOBBY]: 'LOBBY OPEN',
		[ContestStatus.STARTED]: 'LIVE',
		[ContestStatus.ENDED]: 'ENDED'
	};

	const statusColor: Record<number, string> = {
		[ContestStatus.UPCOMING]: 'var(--silver)',
		[ContestStatus.LOBBY]: 'var(--mer)',
		[ContestStatus.STARTED]: 'var(--red)',
		[ContestStatus.ENDED]: 'rgba(192,192,192,0.35)'
	};

	// Data field is "title\nabout" — split on first newline
	function parseData(data: string): { title: string; about: string } {
		const [title = '', ...rest] = (data ?? '').split('\n');
		return { title: title || 'Untitled', about: rest.join('\n').trim() };
	}

	function timeUntilLobby(contest: Contest): string {
		const now = new Date();
		const contestTime = new Date(contest.time);
		const lobbyOpenAt = new Date(contestTime.getTime() - 5 * 60 * 1000);
		const diff = lobbyOpenAt.getTime() - now.getTime();
		if (diff <= 0) return 'Now';
		const mins = Math.floor(diff / 60000);
		const secs = Math.floor((diff % 60000) / 1000);
		if (mins >= 60) {
			const hrs = Math.floor(mins / 60);
			return `${hrs}h ${mins % 60}m`;
		}
		return mins > 0 ? `${mins}m ${secs}s` : `${secs}s`;
	}

	function langFromInt(n: number): Language {
		return LANGUAGES[n] ?? 'CPP';
	}

	async function load() {
		loading = true;
		try {
			contests = await listContests(0, 100);
		} catch (e) {
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function handleCreate() {
		if (!form.title.trim()) {
			createError = 'Title is required';
			return;
		}
		if (!form.scheduledDate || !form.scheduledTime) {
			createError = 'Set a date & time';
			return;
		}
		creating = true;
		createError = '';
		try {
			// Pack title + about into the data string field
			const dataStr = form.about.trim() ? `${form.title}\n${form.about}` : form.title;
			const contest = {
				id: '',
				jobID: 0,
				roomID: 0,
				time: new Date(`${form.scheduledDate}T${form.scheduledTime}`),
				data: dataStr,
				lang: LANGUAGES.indexOf(form.lang),
				duration: form.duration,
				lobbyID: 0,
				status: 0,
				leaderboard: []
			} as unknown as Contest;
			await createContest(contest);
			showCreateModal = false;
			form = {
				title: '',
				about: '',
				lang: 'CPP',
				duration: 90,
				scheduledDate: '',
				scheduledTime: ''
			};
			await load();
		} catch (e) {
			createError = String(e);
		} finally {
			creating = false;
		}
	}

	async function handleJoinLobby(contest: Contest) {
		joiningLobby = contest.id;
		try {
			await joinControlledLobby(contest.lobbyID);
		} catch (e) {
			console.error(e);
		} finally {
			joiningLobby = null;
		}
	}

	onMount(load);
</script>

<div class="flex h-screen flex-col overflow-hidden bg-(--bg) p-8 pb-20">
	<!-- Header -->
	<div class="mb-6 flex flex-row items-start justify-between">
		<div class="title-text text-9xl text-(--silver)">OSDTYP</div>
		<div class="flex flex-col items-end gap-1">
			<span class="text-sm text-(--red)">Hub</span>
			<div class="title-text text-7xl text-(--silver)">CONTESTS</div>
		</div>
	</div>

	<!-- Contests list + fab -->
	<div class="relative min-h-0 flex-1">
		{#if loading}
			<div class="flex h-full items-center justify-center">
				<span class="text-xl tracking-widest text-(--silver) opacity-50">LOADING...</span>
			</div>
		{:else if contests.length === 0}
			<div class="flex h-full items-center justify-center">
				<span class="text-xl tracking-widest text-(--silver) opacity-40">NO CONTESTS YET</span>
			</div>
		{:else}
			<div
				class="h-full overflow-y-auto pr-1"
				style="scrollbar-width: thin; scrollbar-color: var(--silver) transparent;"
			>
				<div
					style="display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 0.75rem;"
				>
					{#each contests as contest (contest.id)}
						{@const isLobby = contest.status === ContestStatus.LOBBY}
						{@const isUpcoming = contest.status === ContestStatus.UPCOMING}
						{@const isLive = contest.status === ContestStatus.STARTED}
						{@const isEnded = contest.status === ContestStatus.ENDED}
						{@const parsed = parseData(contest.data as unknown as string)}
						{@const lang = langFromInt(contest.lang as unknown as number)}
						{@const leaderboard = (contest.leaderboard ?? []) as unknown as LeaderboardEntry[]}
						{@const isExpanded = expandedID === contest.id}

						<div
							class="flex flex-col gap-4 rounded bg-(--fbg) p-6"
							style="
								border: 1px solid rgba(192,192,192,0.1);
								border-top: 2px solid {statusColor[contest.status]};
								opacity: {isEnded ? 0.6 : 1};
							"
						>
							<!-- Top row: lang icon + status badge -->
							<div class="flex items-center justify-between">
								<img src={langIcons[lang]} alt={langLabels[lang]} class="h-8 w-8 object-contain" />
								<span
									class="rounded px-2 py-1 text-xs tracking-widest"
									style="
										color: {statusColor[contest.status]};
										background: {isLobby ? 'rgba(0,200,200,0.08)' : isLive ? 'rgba(220,50,50,0.08)' : 'transparent'};
										border: 1px solid {statusColor[contest.status]};
									"
								>
									{isLive ? '● ' : ''}{statusLabel[contest.status]}
								</span>
							</div>

							<!-- Title -->
							<div class="text-xl leading-tight text-(--silver)">{parsed.title}</div>

							<!-- About (if present) -->
							{#if parsed.about}
								<div class="line-clamp-2 text-sm leading-snug text-(--silver) opacity-50">
									{parsed.about}
								</div>
							{/if}

							<!-- Meta row -->
							<div class="flex items-center gap-3 text-xs text-(--silver) opacity-50">
								<span>{langLabels[lang]}</span>
								<span>·</span>
								<span>{durationLabels[contest.duration] ?? contest.duration + 's'}</span>
								<span>·</span>
								<span
									>{new Date(contest.time).toLocaleString(undefined, {
										month: 'short',
										day: 'numeric',
										hour: '2-digit',
										minute: '2-digit'
									})}</span
								>
							</div>

							<!-- Leaderboard (ended contests, if populated) -->
							{#if isEnded && leaderboard.length > 0}
								<div style="border-top: 1px solid rgba(192,192,192,0.08);" class="pt-2">
									<button
										onclick={() => (expandedID = isExpanded ? null : contest.id)}
										class="w-full cursor-pointer text-left text-xs tracking-widest text-(--silver) opacity-40 hover:opacity-70"
									>
										{isExpanded ? '▲ HIDE' : '▼ LEADERBOARD'} ({leaderboard.length})
									</button>
									{#if isExpanded}
										<div class="mt-3 flex flex-col gap-1">
											{#each leaderboard.slice(0, 5) as entry, i (entry.id ?? i)}
												<div class="flex items-center justify-between text-xs">
													<div class="flex items-center gap-2">
														<span class="w-4 text-(--silver) opacity-30">{i + 1}</span>
														<span class="text-(--silver)">{entry.name}</span>
													</div>
													<div class="flex items-center gap-3 font-mono text-(--silver) opacity-60">
														<span>{entry.wpm.toFixed(0)} wpm</span>
														<span>{entry.accuracy.toFixed(0)}%</span>
													</div>
												</div>
											{/each}
										</div>
									{/if}
								</div>
							{/if}

							<!-- Action footer -->
							<div class="mt-auto pt-2" style="border-top: 1px solid rgba(192,192,192,0.08);">
								{#if isUpcoming}
									<div class="flex items-center justify-between">
										<div class="flex flex-col gap-0.5">
											<span class="text-xs tracking-widest text-(--silver) opacity-40"
												>LOBBY OPENS IN</span
											>
											<span class="font-mono text-base" style="color: var(--mer);"
												>{timeUntilLobby(contest)}</span
											>
										</div>
										<button
											disabled
											class="cursor-not-allowed rounded px-3 py-2 text-xs text-(--silver) opacity-25"
											style="border: 1px solid rgba(192,192,192,0.2);"
										>
											Not open yet
										</button>
									</div>
								{:else if isLobby}
									<div class="flex items-center justify-between">
										<span class="text-xs tracking-widest" style="color: var(--mer);"
											>LOBBY OPEN NOW</span
										>
										<button
											onclick={() => handleJoinLobby(contest)}
											disabled={joiningLobby === contest.id}
											class="cursor-pointer rounded px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
											style="background: var(--mer); color: var(--bg);"
										>
											{joiningLobby === contest.id ? 'Joining...' : 'Join Lobby =>'}
										</button>
									</div>
								{:else if isLive}
									<div class="flex items-center justify-between">
										<span class="text-xs tracking-widest" style="color: var(--red);"
											>IN PROGRESS</span
										>
										<button
											onclick={() => handleJoinLobby(contest)}
											class="cursor-pointer rounded px-4 py-2 text-sm transition-opacity hover:opacity-80"
											style="background: var(--red); color: var(--silver);"
										>
											Watch Live =>
										</button>
									</div>
								{:else}
									<span class="text-xs tracking-widest text-(--silver) opacity-25"
										>CONTEST OVER</span
									>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- FAB -->
		<button
			onclick={() => {
				showCreateModal = true;
				createError = '';
			}}
			class="absolute right-0 bottom-0 flex h-14 w-14 cursor-pointer items-center justify-center rounded-full text-3xl shadow-lg transition-opacity hover:opacity-80"
			style="background: var(--red); color: var(--silver);"
			aria-label="Create Contest"
		>
			+
		</button>
	</div>
</div>

<!-- Create Contest Modal -->
{#if showCreateModal}
	<button
		class="fixed inset-0 z-40 cursor-default"
		style="background: rgba(0,0,0,0.7); backdrop-filter: blur(4px);"
		onclick={(e) => {
			if (e.target === e.currentTarget) showCreateModal = false;
		}}
		aria-label="Close the create contest dialog"
	></button>
	<div
		class="fixed inset-0 z-50 flex items-center justify-center"
		role="dialog"
		aria-modal="true"
		aria-label="Create contest"
	>
		<div
			class="flex w-full max-w-md flex-col gap-5 rounded bg-(--fbg) p-8"
			style="border: 1px solid rgba(192,192,192,0.15); border-top: 2px solid var(--silver);"
		>
			<!-- Header -->
			<div class="flex items-start justify-between">
				<div>
					<div class="mb-1 text-xs tracking-widest text-(--red)">NEW</div>
					<div class="title-text text-4xl text-(--silver)">CREATE CONTEST</div>
				</div>
				<button
					onclick={() => (showCreateModal = false)}
					class="cursor-pointer text-2xl leading-none text-(--silver) opacity-40 hover:opacity-80"
					>×</button
				>
			</div>

			<!-- Title -->
			<div class="flex flex-col gap-2">
				<label for="contest-title" class="text-xs tracking-widest text-(--silver) opacity-50"
					>TITLE</label
				>
				<input
					id="contest-title"
					bind:value={form.title}
					placeholder="e.g. Friday Blitz #1"
					class="rounded bg-(--bg) px-4 py-3 text-sm text-(--silver) outline-none"
					style="border: 1px solid rgba(192,192,192,0.15); font-family: inherit;"
				/>
			</div>

			<!-- About -->
			<div class="flex flex-col gap-2">
				<label for="contest-about" class="text-xs tracking-widest text-(--silver) opacity-50"
					>ABOUT <span class="opacity-50">(optional)</span></label
				>
				<textarea
					id="contest-about"
					bind:value={form.about}
					placeholder="Short description, rules, writeup..."
					rows="2"
					class="resize-none rounded bg-(--bg) px-4 py-3 text-sm text-(--silver) outline-none"
					style="border: 1px solid rgba(192,192,192,0.15); font-family: inherit;"
				></textarea>
			</div>

			<!-- Language + Duration -->
			<div class="grid grid-cols-2 gap-4">
				<div class="flex flex-col gap-2">
					<label for="contest-lang" class="text-xs tracking-widest text-(--silver) opacity-50"
						>LANGUAGE</label
					>
					<select
						id="contest-lang"
						bind:value={form.lang}
						class="cursor-pointer rounded bg-(--bg) px-4 py-3 text-sm text-(--silver) outline-none"
						style="border: 1px solid rgba(192,192,192,0.15); font-family: inherit;"
					>
						{#each LANGUAGES as lang (lang)}
							<option value={lang}>{langLabels[lang]}</option>
						{/each}
					</select>
				</div>
				<div class="flex flex-col gap-2">
					<label for="contest-duration" class="text-xs tracking-widest text-(--silver) opacity-50"
						>DURATION</label
					>
					<select
						id="contest-duration"
						bind:value={form.duration}
						class="cursor-pointer rounded bg-(--bg) px-4 py-3 text-sm text-(--silver) outline-none"
						style="border: 1px solid rgba(192,192,192,0.15); font-family: inherit;"
					>
						<option value={30}>30s Sprint</option>
						<option value={90}>90s Standard</option>
						<option value={300}>300s Marathon</option>
					</select>
				</div>
			</div>

			<!-- Date + Time -->
			<div class="grid grid-cols-2 gap-4">
				<div class="flex flex-col gap-2">
					<label for="contest-date" class="text-xs tracking-widest text-(--silver) opacity-50"
						>DATE</label
					>
					<input
						id="contest-date"
						type="date"
						bind:value={form.scheduledDate}
						class="cursor-pointer rounded bg-(--bg) px-4 py-3 text-sm text-(--silver) outline-none"
						style="border: 1px solid rgba(192,192,192,0.15); font-family: inherit; color-scheme: dark;"
					/>
				</div>
				<div class="flex flex-col gap-2">
					<label for="contest-time" class="text-xs tracking-widest text-(--silver) opacity-50"
						>TIME</label
					>
					<input
						id="contest-time"
						type="time"
						bind:value={form.scheduledTime}
						class="cursor-pointer rounded bg-(--bg) px-4 py-3 text-sm text-(--silver) outline-none"
						style="border: 1px solid rgba(192,192,192,0.15); font-family: inherit; color-scheme: dark;"
					/>
				</div>
			</div>

			{#if createError}
				<div class="text-sm opacity-80" style="color: var(--red);">{createError}</div>
			{/if}

			<!-- Actions -->
			<div class="flex gap-3 pt-1">
				<button
					onclick={() => (showCreateModal = false)}
					class="flex-1 cursor-pointer rounded py-3 text-sm text-(--silver) transition-opacity hover:opacity-70"
					style="border: 1px solid rgba(192,192,192,0.15);"
				>
					Cancel
				</button>
				<button
					onclick={handleCreate}
					disabled={creating}
					class="flex-1 cursor-pointer rounded py-3 text-sm transition-opacity hover:opacity-80 disabled:opacity-40"
					style="background: var(--red); color: var(--silver);"
				>
					{creating ? 'Creating...' : 'Create =>'}
				</button>
			</div>
		</div>
	</div>
{/if}
