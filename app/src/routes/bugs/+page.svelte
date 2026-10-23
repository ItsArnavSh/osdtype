<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import { appState } from '$lib/core/state/state.svelte';
	import { ApiError, errorMessage, newIdempotencyKey } from '$lib/core/api/client';
	import { myBugReports, submitBugReport } from '$lib/core/api/bugs';
	import type { BugReport } from '$lib/core/api/types';

	const categories = [
		{ value: 'bug', label: 'General bug' },
		{ value: 'ui', label: 'Interface / layout' },
		{ value: 'gameplay', label: 'Gameplay / scoring' },
		{ value: 'performance', label: 'Performance / connection' },
		{ value: 'account', label: 'Account / login' },
		{ value: 'contest', label: 'Contest / event' },
		{ value: 'other', label: 'Something else' }
	] as const;

	const severities = [
		{ value: 'low', label: 'Low — cosmetic or minor' },
		{ value: 'medium', label: 'Medium — annoying but usable' },
		{ value: 'high', label: 'High — blocks a feature' },
		{ value: 'critical', label: 'Critical — data loss or outage' }
	] as const;

	let user = $derived(appState.user);
	let title = $state('');
	let category = $state('bug');
	let severity = $state('medium');
	let description = $state('');
	let steps = $state('');
	let expected = $state('');
	let actual = $state('');
	let pageUrl = $state('');
	let userAgent = $state('');
	let appVersion = $state('web');
	let viewport = $state('');
	let requestId = $state('');
	let contactEmail = $state('');
	let honeypot = $state('');
	let submissionId = $state('');
	let busy = $state(false);
	let error = $state('');
	let reportsError = $state('');
	let submitted = $state<BugReport | null>(null);
	let reports = $state<BugReport[]>([]);
	let copied = $state(false);
	let submittedDuplicate = $state(false);

	onMount(() => {
		pageUrl = document.referrer && document.referrer.startsWith(window.location.origin)
			? document.referrer
			: window.location.href;
		userAgent = navigator.userAgent;
		viewport = `${window.innerWidth}×${window.innerHeight}`;
		appVersion = (import.meta.env.VITE_APP_VERSION as string | undefined) ?? 'web';
		submissionId = newIdempotencyKey();
	});

	$effect(() => {
		// The layout boots the session asynchronously, so the child page may
		// mount before appState.user is available.
		if (appState.user) void loadReports();
	});

	async function loadReports() {
		reportsError = '';
		try {
			reports = await myBugReports();
		} catch (err) {
			reportsError = errorMessage(err);
		}
	}

	async function submit() {
		if (busy) return;
		error = '';
		submitted = null;
		if (title.trim().length < 3) {
			error = 'Give the report a short title.';
			return;
		}
		if (description.trim().length < 10) {
			error = 'Describe what happened in at least 10 characters.';
			return;
		}
		busy = true;
		copied = false;
		try {
			const result = await submitBugReport({
				title,
				category,
				severity,
				description,
				steps,
				expected,
				actual,
				page_url: pageUrl,
				user_agent: `${userAgent} · viewport ${viewport}`.slice(0, 1024),
				app_version: appVersion,
				request_id: requestId,
				contact_email: contactEmail,
				submission_id: submissionId,
				website: honeypot
			});
			submitted = result.report;
			submittedDuplicate = result.duplicate;
			title = '';
			description = '';
			steps = '';
			expected = '';
			actual = '';
			contactEmail = '';
			requestId = '';
			submissionId = newIdempotencyKey();
			if (user) await loadReports();
		} catch (err) {
			error = errorMessage(err);
			if (err instanceof ApiError && err.requestId) requestId = err.requestId;
		} finally {
			busy = false;
		}
	}

	async function copyId() {
		if (!submitted) return;
		try {
			await navigator.clipboard.writeText(submitted.public_id);
			copied = true;
			setTimeout(() => (copied = false), 1600);
		} catch {
			copied = false;
		}
	}
</script>

<svelte:head>
	<title>report a bug · OSDTYP</title>
	<meta
		name="description"
		content="Report a bug, broken game, or interface issue in OSDTYP."
	/>
</svelte:head>

<div class="min-h-screen bg-(--bg) p-5 md:p-8">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<div class="flex flex-wrap items-center gap-3">
			<a href={resolve('/play')} class="title-text text-3xl leading-none text-(--silver)">OSDTYP</a>
			<span class="font-mono text-[10px] tracking-[0.14em] text-(--mer) uppercase">bugs</span>
			<button
				class="cursor-pointer border border-(--border) px-2 py-1 font-mono text-[10px] text-(--silver)/55 transition-colors hover:border-(--silver)/40 hover:text-(--silver)"
				onclick={cycleTheme}
				title="cycle theme"
			>{themeState.current}</button>
		</div>
		<nav class="flex flex-wrap items-center gap-2 font-mono text-[10px] tracking-[0.1em] uppercase">
			<a
				href={resolve('/play')}
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>play</a
			>
			<a
				href={resolve('/docs')}
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>docs</a
			>
			<a
				href={resolve('/contests')}
				class="border border-(--border) px-3 py-1.5 text-(--silver)/45 hover:border-(--mer) hover:text-(--mer)"
				>contests</a
			>
		</nav>
	</header>

	<main class="mx-auto mt-8 grid max-w-6xl gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
		<section class="border border-(--border) bg-(--fbg)">
			<div class="border-b border-(--border) p-5 md:p-7">
				<div class="font-mono text-[10px] tracking-[0.18em] text-(--mer) uppercase">
					tell us what broke
				</div>
				<h1 class="mt-3 text-4xl leading-none text-(--silver) md:text-6xl">Report a bug.</h1>
				<p class="mt-4 max-w-2xl text-sm leading-relaxed text-(--silver)/50">
					Include the steps that led to the problem and the page where it happened. Reports are
					reviewed by a human. Do not paste passwords, cookies, API keys, or other secrets.
				</p>
			</div>

			{#if submitted}
				<div class="border-b border-(--mer)/40 bg-(--mer)/5 p-5 md:p-7">
					<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
						report received
					</div>
					<div class="mt-2 flex flex-wrap items-center gap-3">
						<span class="font-mono text-xl text-(--silver)">{submitted.public_id}</span>
						<button
							class="cursor-pointer border border-(--border) px-2.5 py-1 font-mono text-[10px] text-(--silver)/55 uppercase hover:border-(--mer) hover:text-(--mer)"
							onclick={copyId}
						>{copied ? 'copied' : 'copy id'}</button>
					</div>
					<p class="mt-2 text-sm leading-relaxed text-(--silver)/45">
						Keep this ID if you need to follow up. {submittedDuplicate
							? 'This was already submitted, so we reused the existing report.'
							: 'The report is now in the triage queue.'}
					</p>
				</div>
			{/if}

			<form
				class="flex flex-col gap-5 p-5 md:p-7"
				onsubmit={(e) => {
					e.preventDefault();
					void submit();
				}}
			>
				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
						>title</span
					>
					<input
						class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
						bind:value={title}
						maxlength="160"
						placeholder="Ranked queue gets stuck after a bot match"
						required
					/>
				</label>

				<div class="grid gap-4 sm:grid-cols-2">
					<label class="flex flex-col gap-1.5">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
							>category</span
						>
						<select
							class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
							bind:value={category}
						>
							{#each categories as item}
								<option value={item.value}>{item.label}</option>
							{/each}
						</select>
					</label>
					<label class="flex flex-col gap-1.5">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
							>severity</span
						>
						<select
							class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
							bind:value={severity}
						>
							{#each severities as item}
								<option value={item.value}>{item.label}</option>
							{/each}
						</select>
					</label>
				</div>

				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
						>what happened</span
					>
					<textarea
						class="min-h-36 w-full resize-y border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
						bind:value={description}
						maxlength="8000"
						placeholder="Describe the problem, what you expected, and what happened instead."
						required
					></textarea>
				</label>

				<label class="flex flex-col gap-1.5">
					<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
						>steps to reproduce <span class="normal-case text-(--silver)/25">(optional)</span></span
					>
					<textarea
						class="min-h-24 w-full resize-y border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
						bind:value={steps}
						maxlength="4000"
						placeholder="1. Open ranked&#10;2. Queue for 30s&#10;3. ..."
					></textarea>
				</label>

				<div class="grid gap-4 sm:grid-cols-2">
					<label class="flex flex-col gap-1.5">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
							>expected <span class="normal-case text-(--silver)/25">(optional)</span></span
						>
						<textarea
							class="min-h-20 w-full resize-y border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
							bind:value={expected}
							maxlength="4000"
						></textarea>
					</label>
					<label class="flex flex-col gap-1.5">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
							>actual <span class="normal-case text-(--silver)/25">(optional)</span></span
						>
						<textarea
							class="min-h-20 w-full resize-y border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm leading-relaxed text-(--silver) outline-none focus:border-(--mer)"
							bind:value={actual}
							maxlength="4000"
						></textarea>
					</label>
				</div>

				<details class="border border-(--border) bg-(--bg)/40 p-3">
					<summary
						class="cursor-pointer font-mono text-[10px] tracking-[0.14em] text-(--silver)/45 uppercase"
					>
						technical context
					</summary>
					<div class="mt-4 grid gap-4">
						<label class="flex flex-col gap-1.5">
							<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
								>affected page</span
							>
							<input
								class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
								bind:value={pageUrl}
								maxlength="2048"
							/>
						</label>
						<label class="flex flex-col gap-1.5">
							<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
								>browser / device</span
							>
							<input
								class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-xs text-(--silver) outline-none focus:border-(--mer)"
								bind:value={userAgent}
								maxlength="1024"
							/>
						</label>
						<label class="flex flex-col gap-1.5">
							<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
								>request id <span class="normal-case text-(--silver)/25">(optional)</span></span
							>
							<input
								class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
								bind:value={requestId}
								maxlength="64"
								placeholder="from an error message"
							/>
						</label>
					</div>
				</details>

				{#if !user}
					<label class="flex flex-col gap-1.5">
						<span class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/40 uppercase"
							>contact email <span class="normal-case text-(--silver)/25">(optional)</span></span
						>
						<input
							class="w-full border border-(--border) bg-(--bg) px-3 py-2.5 font-mono text-sm text-(--silver) outline-none focus:border-(--mer)"
							bind:value={contactEmail}
							type="email"
							maxlength="254"
							placeholder="you@example.com"
						/>
					</label>
				{:else}
					<p class="font-mono text-[10px] leading-relaxed text-(--silver)/30">
						Submitting as {user.username}. Your account is attached to this report.
					</p>
				{/if}

				<input
					class="hidden"
					tabindex="-1"
					autocomplete="off"
					bind:value={honeypot}
					aria-hidden="true"
				/>

				{#if error}
					<div class="border border-(--red)/40 px-3 py-2 font-mono text-xs text-(--red)">
						{error}
						{#if requestId}
							<div class="mt-1 text-[9px] text-(--silver)/35">request {requestId}</div>
						{/if}
					</div>
				{/if}

				<div class="flex flex-wrap items-center gap-3">
					<button
						type="submit"
						disabled={busy}
						class="cursor-pointer border border-(--mer) px-4 py-2.5 font-mono text-sm text-(--mer) transition-colors hover:bg-(--mer)/10 disabled:cursor-not-allowed disabled:opacity-40"
					>
						{busy ? 'sending…' : 'send report =>'}
					</button>
					<a
						href={resolve('/docs')}
						class="font-mono text-[10px] tracking-[0.12em] text-(--silver)/35 uppercase hover:text-(--mer)"
						>read the docs first</a
					>
				</div>
			</form>
		</section>

		<aside class="flex flex-col gap-4">
			{#if user}
				<section class="border border-(--border) bg-(--fbg) p-4">
					<div class="flex items-center justify-between gap-3">
						<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
							your reports
						</div>
						<button
							class="cursor-pointer font-mono text-[10px] text-(--silver)/35 uppercase hover:text-(--mer)"
							onclick={() => void loadReports()}>refresh</button
						>
					</div>
					{#if reportsError}
						<p class="mt-3 font-mono text-xs text-(--red)">{reportsError}</p>
					{:else if reports.length === 0}
						<p class="mt-3 text-sm leading-relaxed text-(--silver)/40">
							No reports yet. When you submit one, it will show up here.
						</p>
					{:else}
						<ul class="mt-3 flex flex-col gap-2">
							{#each reports as report}
								<li class="border border-(--border) px-3 py-2">
									<div class="flex items-center justify-between gap-2">
										<span class="font-mono text-[10px] text-(--silver)/35"
											>{report.public_id}</span
										>
										<span class="font-mono text-[9px] tracking-[0.1em] text-(--mer) uppercase"
											>{report.status.replace('_', ' ')}</span
										>
									</div>
									<div class="mt-1 text-sm leading-snug text-(--silver)/65">{report.title}</div>
									<div class="mt-1 font-mono text-[9px] text-(--silver)/25">
										{new Date(report.created_at).toLocaleString()}
									</div>
								</li>
							{/each}
						</ul>
					{/if}
				</section>
			{/if}

			<section class="border border-(--border) bg-(--fbg) p-4">
				<div class="font-mono text-[10px] tracking-[0.16em] text-(--mer) uppercase">
					before you send
				</div>
				<ul class="mt-3 flex list-disc flex-col gap-2 pl-4 text-sm leading-relaxed text-(--silver)/45">
					<li>Try a hard refresh and check whether the problem still happens.</li>
					<li>Include the exact page and, if possible, a contest or room code.</li>
					<li>Do not include passwords, session cookies, or API keys.</li>
					<li>For account recovery, use the contact email field so we can reply.</li>
				</ul>
			</section>
		</aside>
	</main>
</div>
