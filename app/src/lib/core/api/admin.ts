import { api } from './client';
import type { BugReport, BugReportStatus } from './types';

export interface AdminUser {
	id: number;
	username: string;
	avatar_url: string;
	rating: number;
	games_played: number;
	is_admin: boolean;
	is_primary: boolean;
	created_at: string;
}

export interface AdminOverview {
	now: string;
	primary_admin: string;
	me: AdminUser;
	active_hour: number;
	active_24h: number;
	online_now: number;
	queue_size: number;
	users_total: number;
	games_total: number;
	bugs_open: number;
	bugs_total: number;
	uptime_24h: number | null;
	checks_24h: number;
	failed_checks_24h: number;
}

export interface AdminBug extends BugReport {
	user_id?: number;
	username?: string;
	admin_notes?: string;
}

export interface ActivityHour {
	hour: string;
	count: number;
}

export interface HealthCheck {
	id: number;
	checked_at: string;
	postgres_ok: boolean;
	redis_ok: boolean;
	latency_ms: number;
	error?: string;
}

export interface AdminHealth {
	current: { postgres: boolean; redis: boolean };
	uptime_24h: number | null;
	checks_24h: number;
	failed_checks_24h: number;
	checks: HealthCheck[];
	page: PageMeta;
}

export interface PageMeta {
	total: number;
	limit: number;
	offset: number;
	next: number | null;
	has_more: boolean;
}

export interface AdminBugQuery {
	status?: string;
	severity?: string;
	category?: string;
	q?: string;
	limit?: number;
	offset?: number;
}

export interface AdminUserQuery {
	q?: string;
	role?: string;
	limit?: number;
	offset?: number;
}

export interface AdminLogQuery {
	q?: string;
	level?: string;
	user?: string;
	limit?: number;
	offset?: number;
}

export interface AdminEventQuery {
	q?: string;
	level?: string;
	kind?: string;
	limit?: number;
	offset?: number;
}

export interface AdminAuditQuery {
	q?: string;
	actor?: string;
	action?: string;
	limit?: number;
	offset?: number;
}

export interface AdminBugPage {
	bugs: AdminBug[];
	page: PageMeta;
}

export interface AdminUserPage {
	users: AdminUser[];
	page: PageMeta;
}

export interface AdminActivityPage {
	hours: ActivityHour[];
	page: PageMeta;
}

export interface AdminLogPage {
	logs: LogEntry[];
	page: PageMeta;
}

export interface AdminEventPage {
	events: SystemEvent[];
	page: PageMeta;
}

export interface AdminAuditPage {
	audits: AuditLog[];
	page: PageMeta;
}

export interface LogEntry {
	time: string;
	level: string;
	message: string;
	fields?: Record<string, unknown>;
}

export interface SystemEvent {
	id: number;
	level: string;
	kind: string;
	message: string;
	metadata?: string;
	created_at: string;
}

export interface AuditLog {
	id: number;
	actor_id: number;
	actor: string;
	action: string;
	target?: string;
	detail?: string;
	created_at: string;
}

export function adminOverview(): Promise<AdminOverview> {
	return api<AdminOverview>('/admin/overview');
}

function pageQuery(params: object): URLSearchParams {
	const values = params as Record<string, string | number | undefined>;
	const query = new URLSearchParams({
		limit: String(values.limit ?? 50),
		offset: String(values.offset ?? 0)
	});
	for (const [key, value] of Object.entries(values)) {
		if (key === 'limit' || key === 'offset' || !value) continue;
		query.set(key, String(value));
	}
	return query;
}

export function adminBugs(params: AdminBugQuery = {}): Promise<AdminBugPage> {
	return api<AdminBugPage>(`/admin/bugs?${pageQuery(params)}`);
}

export async function adminUpdateBug(
	publicID: string,
	status: BugReportStatus,
	adminNotes: string
): Promise<AdminBug> {
	const res = await api<{ bug: AdminBug }>(`/admin/bugs/${encodeURIComponent(publicID)}`, {
		method: 'PATCH',
		body: { status, admin_notes: adminNotes }
	});
	return res.bug;
}

export function adminUsers(params: AdminUserQuery = {}): Promise<AdminUserPage> {
	return api<AdminUserPage>(`/admin/users?${pageQuery(params)}`);
}

export function adminGrant(userID: number): Promise<{ ok: boolean }> {
	return api<{ ok: boolean }>(`/admin/users/${userID}/admin`, { method: 'POST' });
}

export function adminRevoke(userID: number): Promise<{ ok: boolean }> {
	return api<{ ok: boolean }>(`/admin/users/${userID}/admin`, { method: 'DELETE' });
}

export function adminActivity(hours = 24, limit = 24, offset = 0): Promise<AdminActivityPage> {
	return api<AdminActivityPage>(`/admin/activity?${pageQuery({ hours, limit, offset })}`);
}

export function adminHealth(limit = 100, offset = 0): Promise<AdminHealth> {
	return api<AdminHealth>(`/admin/health?${pageQuery({ limit, offset })}`);
}

export function adminLogs(params: AdminLogQuery = {}): Promise<AdminLogPage> {
	return api<AdminLogPage>(`/admin/logs?${pageQuery({ ...params, limit: params.limit ?? 100 })}`);
}

export function adminEvents(params: AdminEventQuery = {}): Promise<AdminEventPage> {
	return api<AdminEventPage>(`/admin/events?${pageQuery(params)}`);
}

export function adminAudits(params: AdminAuditQuery = {}): Promise<AdminAuditPage> {
	return api<AdminAuditPage>(`/admin/audits?${pageQuery(params)}`);
}
