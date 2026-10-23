<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { cycleTheme, themeState } from '$lib/core/theme.svelte';
	import { errorMessage } from '$lib/core/api/client';
	import {
		adminActivity,
		adminAudits,
		adminBugs,
		adminEvents,
		adminGrant,
		adminHealth,
		adminLogs,
		adminOverview,
		adminRevoke,
		adminUpdateBug,
		adminUsers,
		type ActivityHour,
		type AdminBug,
		type AdminHealth,
		type AdminOverview,
		type AdminUser,
		type AuditLog,
		type LogEntry,
		type PageMeta,
		type SystemEvent
	} from '$lib/core/api/admin';
	import AdminPager from '$lib/components/AdminPager.svelte';
	import type { BugReportStatus } from '$lib/core/api/types';

	const PAGE = 50;
	const LOG_PAGE = 100;

	function emptyPage(limit = PAGE): PageMeta {
		return { total: 0, limit, offset: 0, next: null, has_more: false };
	}

	let { data } = $props();
	let liveOverview = $state<AdminOverview | null>(null);
	let overview = $derived(liveOverview ?? data.overview);
	let bugs = $state<AdminBug[]>([]);
	let users = $state<AdminUser[]>([]);
	let activity = $state<ActivityHour[]>([]);
	let health = $state<AdminHealth | null>(null);
	let logs = $state<LogEntry[]>([]);
	let events = $state<SystemEvent[]>([]);
	let audits = $state<AuditLog[]>([]);

	let tab = $state<'overview' | 'bugs' | 'users' | 'activity' | 'health' | 'logs'>('overview');
	let bugFilter = $state<BugReportStatus | ''>('');
	let bugQuery = $state('');
	let bugSeverity = $state('');
	let bugPage = $state<PageMeta>(emptyPage());
	let userQuery = $state('');
	let userRole = $state('');
	let userPage = $state<PageMeta>(emptyPage());
	let activityWindow = $state(24);
	let activityPage = $state<PageMeta>(emptyPage(24));
	let healthPage = $state<PageMeta>(emptyPage(100));
	let logQuery = $state('');
	let logLevel = $state('');
	let logUser = $state('');
	let logPage = $state<PageMeta>(emptyPage(LOG_PAGE));
	let eventQuery = $state('');
	let eventLevel = $state('');
	let eventKind = $state('');
	let eventPage = $state<PageMeta>(emptyPage());
	let auditQuery = $state('');
	let auditActor = $state('');
	let auditAction = $state('');
	let auditPage = $state<PageMeta>(emptyPage());
	let busy = $state(false);
	let savingBug = $state('');
	let adminBusy = $state(0);
	let error = $state('');
	let notice = $state('');

	const tabs = [
		{ id: 'overview', label: 'overview' },
		{ id: 'bugs', label: 'bug reports' },
		{ id: 'users', label: 'users' },
		{ id: 'activity', label: 'activity' },
		{ id: 'health', label: 'health' },
		{ id: 'logs', label: 'logs' }
	] as const;

	const bugStatuses: BugReportStatus[] = [
		'new',
		'triaged',
		'in_progress',
		'resolved',
		'closed',
		'wontfix'
	];

	let maxActivity = $derived(Math.max(1, ...activity.map((row) => row.count)));
	let primary = $derived(overview.me.is_primary);
	let uptimeLabel = $derived(
		overview.uptime_24h == null ? '—' : `${overview.uptime_24h.toFixed(2)}%`
	);

	let cards = $derived([
		{ label: 'active this hour', value: overview.active_hour, hint: 'distinct accounts' },
		{ label: 'active · 24h', value: overview.active_24h, hint: 'distinct accounts' },
		{ label: 'online now', value: overview.online_now, hint: 'open sockets' },
		{ label: 'in queue', value: overview.queue_size, hint: 'ranked players' },
		{ label: 'open reports', value: overview.bugs_open, hint: `${overview.bugs_total} total` },
		{ label: 'uptime · 24h', value: uptimeLabel, hint: `${overview.failed_checks_24h} failed checks` },
		{ label: 'accounts', value: overview.users_total, hint: 'excluding bots' },
		{ label: 'games', value: overview.games_total, hint: 'all time' }
	]);

	onMount(() => {
		void refreshAll();
	});

	async function refreshAll() {
		busy = true;
		error = '';
		notice = '';
		await Promise.all([
			loadOverview(),
			loadBugs(),
			loadUsers(),
			loadActivity(),
			loadHealth(),
			loadLogs(),
			loadEvents(),
			loadAudits()
		]);
		busy = false;
	}

	async function loadOverview() {
		try {
			liveOverview = await adminOverview();
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function loadBugs(offset = bugPage.offset) {
		try {
			const res = await adminBugs({
				status: bugFilter,
				severity: bugSeverity,
				q: bugQuery,
				limit: PAGE,
				offset
			});
			bugs = res.bugs;
			bugPage = res.page;
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function loadUsers(offset = userPage.offset) {
		try {
			const res = await adminUsers({
				q: userQuery,
				role: userRole,
				limit: PAGE,
				offset
			});
			users = res.users;
			userPage = res.page;
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function loadActivity() {
		try {
			const res = await adminActivity(activityWindow, activityWindow, 0);
			activity = res.hours;
			activityPage = res.page;
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function loadHealth(offset = healthPage.offset) {
		try {
			health = await adminHealth(100, offset);
			healthPage = health.page;
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function loadLogs(offset = logPage.offset) {
		try {
			const res = await adminLogs({
				q: logQuery,
				level: logLevel,
				user: logUser,
				limit: LOG_PAGE,
				offset
			});
			logs = res.logs;
			logPage = res.page;
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function loadEvents(offset = eventPage.offset) {
		try {
			const res = await adminEvents({
				q: eventQuery,
				level: eventLevel,
				kind: eventKind,
				limit: PAGE,
				offset
			});
			events = res.events;
			eventPage = res.page;
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function loadAudits(offset = auditPage.offset) {
		try {
			const res = await adminAudits({
				q: auditQuery,
				actor: auditActor,
				action: auditAction,
				limit: PAGE,
				offset
			});
			audits = res.audits;
			auditPage = res.page;
		} catch (err) {
			error = errorMessage(err);
		}
	}

	async function saveBug(bug: AdminBug) {
		savingBug = bug.public_id;
		error = '';
		notice = '';
		try {
			const updated = await adminUpdateBug(bug.public_id, bug.status, bug.admin_notes ?? '');
			bugs = bugs.map((item) => (item.public_id === updated.public_id ? updated : item));
			notice = `${updated.public_id} saved as ${updated.status}.`;
			await loadOverview();
		} catch (err) {
			error = errorMessage(err);
		} finally {
			savingBug = '';
		}
	}

	async function toggleAdmin(user: AdminUser) {
		const verb = user.is_admin ? 'revoke admin access from' : 'grant admin access to';
		if (!window.confirm(`${verb} ${user.username}?`)) return;
		adminBusy = user.id;
		error = '';
		notice = '';
		try {
			if (user.is_admin) await adminRevoke(user.id);
			else await adminGrant(user.id);
			notice = `${user.username} ${user.is_admin ? 'is no longer an admin.' : 'is now an admin.'}`;
			await loadUsers();
			await loadOverview();
		} catch (err) {
			error = errorMessage(err);
		} finally {
			adminBusy = 0;
		}
	}

	function timeOnly(value: string): string {
		return new Date(value).toISOString().slice(11, 16);
	}

	function dateTime(value: string): string {
		return `${new Date(value).toISOString().replace('T', ' ').slice(0, 16)} UTC`;
	}

	function hourLabel(value: string): string {
		return `${new Date(value).toISOString().slice(11, 13)}:00`;
	}

	function levelClass(level: string): string {
		if (level === 'error' || level === 'fatal' || level === 'panic') return 'text-(--red)';
		if (level === 'warn' || level === 'warning') return 'text-(--syntax-literal)';
		if (level === 'info') return 'text-(--mer)';
		return 'text-(--silver)/45';
	}

	function statusClass(status: string): string {
		if (status === 'resolved') return 'border-(--mer)/40 text-(--mer)';
		if (status === 'closed' || status === 'wontfix') return 'border-(--border) text-(--silver)/35';
		if (status === 'in_progress') return 'border-(--mer)/40 text-(--mer)';
		if (status === 'triaged') return 'border-(--silver)/40 text-(--silver)/70';
		return 'border-(--red)/40 text-(--red)';
	}

	function fieldsText(fields: Record<string, unknown> | undefined): string {
		if (!fields || Object.keys(fields).length === 0) return '';
		try {
			return JSON.stringify(fields);
		} catch {
			return String(fields);
		}
	}

	function metadataText(value: string | undefined): string {
		if (!value) return '';
		try {
			return JSON.stringify(JSON.parse(value), null, 2);
		} catch {
			return value;
		}
	}
</script>

<svelte:head>
	<title>admin · OSDTYP</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="min-h-screen bg-(--bg) p-4 md:p-8">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-(--border) pb-3">
		<div class="flex flex-wrap items-center gap-3">
			<a href={resolve('/play')} class="title-text text-3xl leading-none text-(--silver)">OSDTYP</a>
			<span class="font-mono text-[10px] tracking-[0.14em] text-(--red) uppercase">admin</span>
			<span class="font-mono text-[10px] text-(--silver)/30">{overview.me.username}</span>
		</div>
		<nav class="flex flex-wrap items-center gap-2 font-mono text-[10px] tracking-[0.1em] uppercase">
			<a
				href={resolve('/play')}
				class="border border-(--border) px-3 py-2 text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
				>← hub</a
			>
			<a
				href={resolve('/bugs')}
				class="border border-(--border) px-3 py-2 text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
				>report</a
			>
			<button
				class="cursor-pointer border border-(--border) px-3 py-2 text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
				onclick={() => void refreshAll()}
				disabled={busy}
			>{busy ? 'loading…' : 'refresh'}</button
			>
			<button
				class="cursor-pointer border border-(--border) px-3 py-2 text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
				onclick={cycleTheme}
				title="cycle theme"
			>{themeState.current}</button>
		</nav>
	</header>

	{#if error}
		<div class="mt-4 border border-(--red)/40 bg-(--red)/5 p-3 font-mono text-xs text-(--red)">
			{error}
		</div>
	{/if}
	{#if notice}
		<div class="mt-4 border border-(--mer)/40 bg-(--mer)/5 p-3 font-mono text-xs text-(--mer)">
			{notice}
		</div>
	{/if}

	<div class="mt-4 flex flex-wrap gap-2 border-b border-(--border) pb-3">
		{#each tabs as item (item.id)}
			<button
				class={`cursor-pointer border px-3 py-2 font-mono text-[10px] tracking-[0.12em] uppercase transition-colors ${
					tab === item.id
						? 'border-(--mer)/50 bg-(--mer)/10 text-(--mer)'
						: 'border-(--border) text-(--silver)/45 hover:border-(--silver)/30 hover:text-(--silver)/75'
				}`}
				onclick={() => (tab = item.id)}
			>{item.label}</button>
		{/each}
	</div>

	<main class="mt-5">
		{#if tab === 'overview'}
			<section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
				{#each cards as card (card.label)}
					<div class="border border-(--border) bg-(--fbg) p-4">
						<div class="font-mono text-[10px] tracking-[0.14em] text-(--silver)/35 uppercase">
							{card.label}
						</div>
						<div class="mt-2 font-mono text-3xl leading-none text-(--silver)">{card.value}</div>
						<div class="mt-2 font-mono text-[10px] text-(--silver)/30">{card.hint}</div>
					</div>
				{/each}
			</section>

			<section class="mt-4 grid gap-3 lg:grid-cols-2">
				<div class="border border-(--border) bg-(--fbg) p-5">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer) uppercase">
						service pulse
					</div>
					<div class="mt-3 flex items-center justify-between gap-3">
						<span class="text-sm text-(--silver)/60">uptime · last 24h</span>
						<span class="font-mono text-2xl text-(--silver)">{uptimeLabel}</span>
					</div>
					<div class="mt-3 h-2 w-full bg-(--border)/40">
						<div
							class="h-full bg-(--mer)"
							style={`width:${overview.uptime_24h == null ? 0 : Math.max(2, overview.uptime_24h)}%`}
						></div>
					</div>
					<div class="mt-2 flex justify-between font-mono text-[10px] text-(--silver)/30">
						<span>{overview.checks_24h} checks</span>
						<span>{overview.failed_checks_24h} failed</span>
					</div>
				</div>
				<div class="border border-(--border) bg-(--fbg) p-5">
					<div class="font-mono text-[10px] tracking-[0.14em] text-(--mer) uppercase">
						operator
					</div>
					<div class="mt-3 grid gap-2 font-mono text-xs text-(--silver)/55">
						<div class="flex justify-between gap-3">
							<span class="text-(--silver)/30">primary</span>
							<span>{overview.primary_admin || 'not configured'}</span>
						</div>
						<div class="flex justify-between gap-3">
							<span class="text-(--silver)/30">signed in</span>
							<span>{overview.me.username}{primary ? ' · primary' : ''}</span>
						</div>
						<div class="flex justify-between gap-3">
							<span class="text-(--silver)/30">server time</span>
							<span>{dateTime(overview.now)}</span>
						</div>
					</div>
					{#if !primary}
						<p class="mt-4 text-xs leading-relaxed text-(--silver)/35">
							Only the primary admin can grant or revoke admin access.
						</p>
					{/if}
				</div>
			</section>
		{:else if tab === 'bugs'}
			<div class="flex flex-wrap items-end justify-between gap-3">
				<div>
					<h2 class="text-2xl text-(--silver)">Bug reports</h2>
					<p class="mt-1 font-mono text-[10px] text-(--silver)/35">
						{overview.bugs_open} open · {overview.bugs_total} total
					</p>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<form
						class="flex items-center gap-2"
						onsubmit={(e) => {
							e.preventDefault();
							void loadBugs(0);
						}}
					>
						<input
							class="w-52 border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver)/70"
							bind:value={bugQuery}
							placeholder="search id, title, text"
						/>
						<button
							class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
							>search</button
						>
					</form>
					<select
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-[10px] text-(--silver)/70 uppercase"
						bind:value={bugSeverity}
						onchange={() => void loadBugs(0)}
					>
						<option value="">all severities</option>
						<option value="low">low</option>
						<option value="medium">medium</option>
						<option value="high">high</option>
						<option value="critical">critical</option>
					</select>
					<select
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-[10px] text-(--silver)/70 uppercase"
						bind:value={bugFilter}
						onchange={() => void loadBugs(0)}
					>
						<option value="">all statuses</option>
						{#each bugStatuses as status (status)}
							<option value={status}>{status}</option>
						{/each}
					</select>
					<button
						class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
						onclick={() => void loadBugs(0)}>reload</button
					>
				</div>
			</div>

			<div class="mt-4 grid gap-3">
				{#each bugs as bug (bug.public_id)}
					<article class="border border-(--border) bg-(--fbg)">
						<div class="flex flex-wrap items-start justify-between gap-3 border-b border-(--border) p-4">
							<div class="min-w-0">
								<div class="flex flex-wrap items-center gap-2">
									<span
										class={`border px-2 py-0.5 font-mono text-[9px] uppercase ${statusClass(
											bug.status
										)}`}>{bug.status}</span
									>
									<span class="font-mono text-[9px] uppercase text-(--silver)/30"
										>{bug.severity} · {bug.category}</span
									>
								</div>
								<h3 class="mt-2 text-lg text-(--silver)">{bug.title}</h3>
								<div class="mt-1 font-mono text-[10px] text-(--silver)/35">
									{bug.public_id} · {bug.username || bug.contact_email || 'guest'} · {dateTime(
										bug.created_at
									)}
								</div>
							</div>
							<a
								class="font-mono text-[10px] text-(--mer) hover:underline"
								href={bug.page_url || resolve('/play')}
								target="_blank"
								rel="noreferrer">{bug.page_url ? 'open page ↗' : 'hub'}</a
							>
						</div>
						<div class="grid gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
							<div class="grid gap-3 text-sm leading-relaxed text-(--silver)/55">
								<p class="whitespace-pre-wrap">{bug.description}</p>
								{#if bug.steps}
									<div>
										<div class="font-mono text-[9px] uppercase text-(--silver)/30">steps</div>
										<p class="whitespace-pre-wrap">{bug.steps}</p>
									</div>
								{/if}
								{#if bug.expected}
									<div>
										<div class="font-mono text-[9px] uppercase text-(--silver)/30">expected</div>
										<p class="whitespace-pre-wrap">{bug.expected}</p>
									</div>
								{/if}
								{#if bug.actual}
									<div>
										<div class="font-mono text-[9px] uppercase text-(--silver)/30">actual</div>
										<p class="whitespace-pre-wrap">{bug.actual}</p>
									</div>
								{/if}
								<div class="flex flex-wrap gap-x-4 gap-y-1 font-mono text-[10px] text-(--silver)/30">
									{#if bug.contact_email}<span>contact: {bug.contact_email}</span>{/if}
									{#if bug.request_id}<span>request: {bug.request_id}</span>{/if}
									<span>version: {bug.app_version || '—'}</span>
								</div>
							</div>
							<div class="grid content-start gap-2 border-t border-(--border) pt-3 lg:border-t-0 lg:border-l lg:pl-4 lg:pt-0">
								<label class="grid gap-1">
									<span class="font-mono text-[9px] uppercase text-(--silver)/30">status</span>
									<select
										class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
										bind:value={bug.status}
									>
										{#each bugStatuses as status (status)}
											<option value={status}>{status}</option>
										{/each}
									</select>
								</label>
								<label class="grid gap-1">
									<span class="font-mono text-[9px] uppercase text-(--silver)/30">notes</span>
									<textarea
										class="min-h-20 resize-y border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
										bind:value={bug.admin_notes}
										placeholder="internal triage notes"
									></textarea>
								</label>
								<button
									class="cursor-pointer border border-(--mer)/40 px-3 py-2 font-mono text-[10px] uppercase text-(--mer) hover:bg-(--mer)/10 disabled:opacity-40"
									onclick={() => void saveBug(bug)}
									disabled={savingBug === bug.public_id}
								>{savingBug === bug.public_id ? 'saving…' : 'save triage'}</button>
							</div>
						</div>
					</article>
				{:else}
					<div class="border border-(--border) bg-(--fbg) p-6 font-mono text-xs text-(--silver)/35">
						No reports match this filter.
					</div>
				{/each}
			</div>
			<AdminPager page={bugPage} noun="reports" onpage={(offset) => void loadBugs(offset)} />
		{:else if tab === 'users'}
			<div class="flex flex-wrap items-end justify-between gap-3">
				<div>
					<h2 class="text-2xl text-(--silver)">Users</h2>
					<p class="mt-1 font-mono text-[10px] text-(--silver)/35">
						{primary ? 'primary admin can grant and revoke access' : 'read-only admin access'}
					</p>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<form
						class="flex items-center gap-2"
						onsubmit={(e) => {
							e.preventDefault();
							void loadUsers(0);
						}}
					>
						<input
							class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver)/70"
							bind:value={userQuery}
							placeholder="search username or id"
						/>
						<button
							class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
							>search</button
						>
					</form>
					<select
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-[10px] text-(--silver)/70 uppercase"
						bind:value={userRole}
						onchange={() => void loadUsers(0)}
					>
						<option value="">all roles</option>
						<option value="admin">admins</option>
						<option value="player">players</option>
					</select>
				</div>
			</div>
			<div class="mt-4 overflow-x-auto border border-(--border) bg-(--fbg)">
				<table class="w-full min-w-[44rem] text-left">
					<thead class="border-b border-(--border) font-mono text-[9px] uppercase text-(--silver)/30">
						<tr>
							<th class="p-3">user</th>
							<th class="p-3">rating</th>
							<th class="p-3">games</th>
							<th class="p-3">role</th>
							<th class="p-3 text-right">action</th>
						</tr>
					</thead>
					<tbody>
						{#each users as user (user.id)}
							<tr class="border-b border-(--border)/50 last:border-0">
								<td class="p-3">
									<div class="font-mono text-xs text-(--silver)/75">{user.username}</div>
									<div class="font-mono text-[9px] text-(--silver)/25">id {user.id}</div>
								</td>
								<td class="p-3 font-mono text-xs text-(--silver)/55">{user.rating}</td>
								<td class="p-3 font-mono text-xs text-(--silver)/55">{user.games_played}</td>
								<td class="p-3">
									{#if user.is_primary}
										<span class="border border-(--red)/40 px-2 py-0.5 font-mono text-[9px] uppercase text-(--red)"
											>primary</span
										>
									{:else if user.is_admin}
										<span class="border border-(--mer)/40 px-2 py-0.5 font-mono text-[9px] uppercase text-(--mer)"
											>admin</span
										>
									{:else}
										<span class="font-mono text-[9px] uppercase text-(--silver)/25">player</span>
									{/if}
								</td>
								<td class="p-3 text-right">
									{#if primary && !user.is_primary}
										<button
											class={`cursor-pointer border px-2.5 py-1 font-mono text-[9px] uppercase ${
												user.is_admin
													? 'border-(--red)/40 text-(--red) hover:bg-(--red)/10'
													: 'border-(--mer)/40 text-(--mer) hover:bg-(--mer)/10'
											} disabled:opacity-40`}
											onclick={() => void toggleAdmin(user)}
											disabled={adminBusy === user.id}
										>{adminBusy === user.id ? '…' : user.is_admin ? 'revoke' : 'grant'}</button
										>
									{/if}
								</td>
							</tr>
						{:else}
							<tr>
								<td colspan="5" class="p-6 font-mono text-xs text-(--silver)/35">No users found.</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<AdminPager page={userPage} noun="users" onpage={(offset) => void loadUsers(offset)} />
		{:else if tab === 'activity'}
			<div class="flex flex-wrap items-end justify-between gap-3">
				<div>
					<h2 class="text-2xl text-(--silver)">Active users per hour</h2>
					<p class="mt-1 font-mono text-[10px] text-(--silver)/35">
						UTC · distinct accounts · window {activityPage.limit}h
					</p>
				</div>
				<div class="flex items-center gap-2">
					<select
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-[10px] text-(--silver)/70 uppercase"
						bind:value={activityWindow}
						onchange={() => void loadActivity()}
					>
						<option value={24}>last 24h</option>
						<option value={72}>last 72h</option>
						<option value={168}>last 7d</option>
					</select>
					<button
						class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
						onclick={() => void loadActivity()}>reload</button
					>
				</div>
			</div>
			<div class="mt-4 border border-(--border) bg-(--fbg) p-4">
				<div class="flex h-64 items-end gap-1">
					{#each activity as row (row.hour)}
						<div
							class="group flex h-full flex-1 flex-col justify-end"
							title={`${dateTime(row.hour)} · ${row.count} active`}
						>
							<div
								class="w-full bg-(--mer)/70 transition-colors group-hover:bg-(--mer)"
								style={`height:${Math.max(row.count > 0 ? 3 : 1, (row.count / maxActivity) * 100)}%`}
							></div>
						</div>
					{/each}
				</div>
				<div class="mt-2 flex justify-between font-mono text-[9px] text-(--silver)/25">
					{#each activity as row, i (row.hour)}
						{#if i % Math.max(1, Math.ceil(activity.length / 8)) === 0}<span
								>{hourLabel(row.hour)}</span
							>{/if}
					{/each}
				</div>
			</div>
			<div class="mt-3 grid gap-2 sm:grid-cols-3">
				<div class="border border-(--border) bg-(--fbg) p-4">
					<div class="font-mono text-[9px] uppercase text-(--silver)/30">peak · {activityWindow}h</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">{maxActivity}</div>
				</div>
				<div class="border border-(--border) bg-(--fbg) p-4">
					<div class="font-mono text-[9px] uppercase text-(--silver)/30">active · this hour</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">{overview.active_hour}</div>
				</div>
				<div class="border border-(--border) bg-(--fbg) p-4">
					<div class="font-mono text-[9px] uppercase text-(--silver)/30">active · 24h</div>
					<div class="mt-1 font-mono text-2xl text-(--silver)">{overview.active_24h}</div>
				</div>
			</div>
		{:else if tab === 'health'}
			<div class="flex flex-wrap items-end justify-between gap-3">
				<div>
					<h2 class="text-2xl text-(--silver)">Health and downtime</h2>
					<p class="mt-1 font-mono text-[10px] text-(--silver)/35">
						self-check every minute · last 24h
					</p>
				</div>
				<button
					class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					onclick={() => void loadHealth()}>check now</button
				>
			</div>
			{#if health}
				<section class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
					<div class="border border-(--border) bg-(--fbg) p-4">
						<div class="font-mono text-[9px] uppercase text-(--silver)/30">postgres</div>
						<div class={`mt-2 font-mono text-xl ${health.current.postgres ? 'text-(--mer)' : 'text-(--red)'}`}>
							{health.current.postgres ? 'up' : 'down'}
						</div>
					</div>
					<div class="border border-(--border) bg-(--fbg) p-4">
						<div class="font-mono text-[9px] uppercase text-(--silver)/30">redis</div>
						<div class={`mt-2 font-mono text-xl ${health.current.redis ? 'text-(--mer)' : 'text-(--red)'}`}>
							{health.current.redis ? 'up' : 'down'}
						</div>
					</div>
					<div class="border border-(--border) bg-(--fbg) p-4">
						<div class="font-mono text-[9px] uppercase text-(--silver)/30">uptime · 24h</div>
						<div class="mt-2 font-mono text-xl text-(--silver)">
							{health.uptime_24h == null ? '—' : `${health.uptime_24h.toFixed(2)}%`}
						</div>
					</div>
					<div class="border border-(--border) bg-(--fbg) p-4">
						<div class="font-mono text-[9px] uppercase text-(--silver)/30">failed checks</div>
						<div class="mt-2 font-mono text-xl text-(--silver)">{health.failed_checks_24h}</div>
					</div>
				</section>
				<div class="mt-4 overflow-x-auto border border-(--border) bg-(--fbg)">
					<table class="w-full min-w-[36rem] text-left">
						<thead class="border-b border-(--border) font-mono text-[9px] uppercase text-(--silver)/30">
							<tr>
								<th class="p-3">checked</th>
								<th class="p-3">postgres</th>
								<th class="p-3">redis</th>
								<th class="p-3">latency</th>
								<th class="p-3">error</th>
							</tr>
						</thead>
						<tbody>
							{#each health.checks as check, i (`${check.checked_at}-${i}`)}
								<tr class="border-b border-(--border)/50 last:border-0">
									<td class="p-3 font-mono text-[10px] text-(--silver)/55">{dateTime(check.checked_at)}</td>
									<td class={`p-3 font-mono text-[10px] ${check.postgres_ok ? 'text-(--mer)' : 'text-(--red)'}`}>
										{check.postgres_ok ? 'up' : 'down'}
									</td>
									<td class={`p-3 font-mono text-[10px] ${check.redis_ok ? 'text-(--mer)' : 'text-(--red)'}`}>
										{check.redis_ok ? 'up' : 'down'}
									</td>
									<td class="p-3 font-mono text-[10px] text-(--silver)/55">{check.latency_ms} ms</td>
									<td class="p-3 font-mono text-[10px] text-(--red)">{check.error || '—'}</td>
								</tr>
							{:else}
								<tr>
									<td colspan="5" class="p-6 font-mono text-xs text-(--silver)/35">
										No checks recorded yet.
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<AdminPager page={healthPage} noun="checks" onpage={(offset) => void loadHealth(offset)} />
			{:else}
				<div class="mt-4 border border-(--border) bg-(--fbg) p-6 font-mono text-xs text-(--silver)/35">
					Health data unavailable.
				</div>
			{/if}
		{:else if tab === 'logs'}
			<div class="flex flex-wrap items-end justify-between gap-3">
				<div>
					<h2 class="text-2xl text-(--silver)">Logs and events</h2>
					<p class="mt-1 font-mono text-[10px] text-(--silver)/35">
						recent process logs · durable system events · admin audit trail
					</p>
				</div>
				<button
					class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					onclick={() => void Promise.all([loadLogs(), loadEvents(), loadAudits()])}>reload</button
				>
			</div>
			<form
				class="mt-4 flex flex-wrap items-end gap-2 border border-(--border) bg-(--fbg) p-3"
				onsubmit={(e) => {
					e.preventDefault();
					void loadLogs(0);
				}}
			>
				<label class="grid gap-1">
					<span class="font-mono text-[9px] uppercase text-(--silver)/30">search</span>
					<input
						class="w-64 border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver)/70"
						bind:value={logQuery}
						placeholder="message, request id, path"
					/>
				</label>
				<label class="grid gap-1">
					<span class="font-mono text-[9px] uppercase text-(--silver)/30">user</span>
					<input
						class="w-44 border border-(--border) bg-(--bg) px-3 py-2 font-mono text-xs text-(--silver)/70"
						bind:value={logUser}
						placeholder="username or id"
					/>
				</label>
				<label class="grid gap-1">
					<span class="font-mono text-[9px] uppercase text-(--silver)/30">level</span>
					<select
						class="border border-(--border) bg-(--bg) px-3 py-2 font-mono text-[10px] text-(--silver)/70 uppercase"
						bind:value={logLevel}
					>
						<option value="">all levels</option>
						<option value="debug">debug</option>
						<option value="info">info</option>
						<option value="warn">warn</option>
						<option value="error">error</option>
					</select>
				</label>
				<button
					class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					>search</button
				>
				<button
					type="button"
					class="cursor-pointer border border-(--border) px-3 py-2 font-mono text-[10px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
					onclick={() => {
						logQuery = '';
						logLevel = '';
						logUser = '';
						void loadLogs(0);
					}}>reset</button
				>
			</form>
			<div class="mt-4 grid gap-4 xl:grid-cols-2">
				<section class="border border-(--border) bg-(--fbg)">
					<div class="border-b border-(--border) p-3 font-mono text-[10px] uppercase text-(--silver)/35">
						process logs · newest first
					</div>
					<div class="max-h-[34rem] overflow-auto p-3">
						{#each logs as entry, i (`${entry.time}-${i}`)}
							<div class="border-b border-(--border)/40 py-2 last:border-0">
								<div class="flex flex-wrap items-center gap-2 font-mono text-[9px]">
									<span class="text-(--silver)/25">{timeOnly(entry.time)}</span>
									<span class={`uppercase ${levelClass(entry.level)}`}>{entry.level}</span>
									<span class="text-(--silver)/60">{entry.message}</span>
								</div>
								{#if fieldsText(entry.fields)}
									<pre class="mt-1 overflow-x-auto font-mono text-[9px] text-(--silver)/30">{fieldsText(
										entry.fields
									)}</pre>
								{/if}
							</div>
						{:else}
							<div class="font-mono text-xs text-(--silver)/35">No logs in the buffer.</div>
						{/each}
					</div>
					<AdminPager page={logPage} noun="logs" onpage={(offset) => void loadLogs(offset)} />
				</section>

				<div class="grid gap-4">
					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) p-3 font-mono text-[10px] uppercase text-(--silver)/35">
							system events
						</div>
						<form
							class="flex flex-wrap items-center gap-2 border-b border-(--border) p-3"
							onsubmit={(e) => {
								e.preventDefault();
								void loadEvents(0);
							}}
						>
							<input
								class="min-w-0 flex-1 border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
								bind:value={eventQuery}
								placeholder="search message or metadata"
							/>
							<select
								class="border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
								bind:value={eventLevel}
							>
								<option value="">level</option>
								<option value="info">info</option>
								<option value="warn">warn</option>
								<option value="error">error</option>
							</select>
							<input
								class="w-24 border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
								bind:value={eventKind}
								placeholder="kind"
							/>
							<button
								class="cursor-pointer border border-(--border) px-2.5 py-1.5 font-mono text-[9px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
								>filter</button
							>
						</form>
						<div class="max-h-64 overflow-auto p-3">
							{#each events as event (event.id)}
								<div class="border-b border-(--border)/40 py-2 last:border-0">
									<div class="flex flex-wrap items-center gap-2 font-mono text-[9px]">
										<span class="text-(--silver)/25">{dateTime(event.created_at)}</span>
										<span class={`uppercase ${levelClass(event.level)}`}>{event.level}</span>
										<span class="text-(--mer)/70">{event.kind}</span>
										<span class="text-(--silver)/60">{event.message}</span>
									</div>
									{#if metadataText(event.metadata)}
										<pre class="mt-1 overflow-x-auto font-mono text-[9px] text-(--silver)/30">{metadataText(
											event.metadata
										)}</pre>
									{/if}
								</div>
							{:else}
								<div class="font-mono text-xs text-(--silver)/35">No events recorded.</div>
							{/each}
						</div>
						<AdminPager page={eventPage} noun="events" onpage={(offset) => void loadEvents(offset)} />
					</section>

					<section class="border border-(--border) bg-(--fbg)">
						<div class="border-b border-(--border) p-3 font-mono text-[10px] uppercase text-(--silver)/35">
							admin audit trail
						</div>
						<form
							class="flex flex-wrap items-center gap-2 border-b border-(--border) p-3"
							onsubmit={(e) => {
								e.preventDefault();
								void loadAudits(0);
							}}
						>
							<input
								class="min-w-0 flex-1 border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
								bind:value={auditQuery}
								placeholder="search target or detail"
							/>
							<input
								class="w-28 border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
								bind:value={auditActor}
								placeholder="actor"
							/>
							<input
								class="w-28 border border-(--border) bg-(--bg) px-2 py-1.5 font-mono text-[10px] text-(--silver)/70"
								bind:value={auditAction}
								placeholder="action"
							/>
							<button
								class="cursor-pointer border border-(--border) px-2.5 py-1.5 font-mono text-[9px] uppercase text-(--silver)/55 hover:border-(--mer) hover:text-(--mer)"
								>filter</button
							>
						</form>
						<div class="max-h-64 overflow-auto p-3">
							{#each audits as audit (audit.id)}
								<div class="border-b border-(--border)/40 py-2 last:border-0 font-mono text-[9px]">
									<div class="flex flex-wrap items-center gap-2">
										<span class="text-(--silver)/25">{dateTime(audit.created_at)}</span>
										<span class="text-(--mer)/70">{audit.actor}</span>
										<span class="text-(--silver)/60">{audit.action}</span>
										{#if audit.target}<span class="text-(--silver)/35">→ {audit.target}</span>{/if}
									</div>
									{#if audit.detail}<div class="mt-1 text-(--silver)/30">{audit.detail}</div>{/if}
								</div>
							{:else}
								<div class="font-mono text-xs text-(--silver)/35">No privileged actions yet.</div>
							{/each}
						</div>
						<AdminPager page={auditPage} noun="actions" onpage={(offset) => void loadAudits(offset)} />
					</section>
				</div>
			</div>
		{/if}
	</main>
</div>
