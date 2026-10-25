import { api } from './client';
import { ENDPOINTS } from './config';
import type {
	FriendView,
	LeaderboardSearchView,
	LeaderboardView,
	ProfileSettings,
	ProfileSettingsInput,
	ProfileView,
	RunMetaInput,
	UserView
} from './types';

export function leaderboard(
	duration = 90,
	offset = 0,
	limit = 5
): Promise<LeaderboardView> {
	return api<LeaderboardView>(
		`${ENDPOINTS.LEADERBOARD}?duration=${duration}&offset=${offset}&limit=${limit}`
	);
}

export function searchLeaderboard(
	duration: number,
	query: string,
	limit = 10
): Promise<LeaderboardSearchView> {
	return api<LeaderboardSearchView>(
		`${ENDPOINTS.LEADERBOARD_SEARCH}?duration=${duration}&q=${encodeURIComponent(query)}&limit=${limit}`
	);
}

export function updateRunMeta(gameID: number, input: RunMetaInput): Promise<{ ok: boolean }> {
	return api(ENDPOINTS.RUN_UPDATE(gameID), { method: 'PATCH', body: input });
}

export function profile(username: string): Promise<ProfileView> {
	return api<ProfileView>(ENDPOINTS.PROFILE(username));
}

export function follow(username: string): Promise<{ ok: boolean }> {
	return api(ENDPOINTS.FOLLOW, { method: 'POST', body: { user: username } });
}

export function unfollow(username: string): Promise<{ ok: boolean }> {
	return api(ENDPOINTS.UNFOLLOW, { method: 'POST', body: { user: username } });
}

export async function friends(): Promise<FriendView[]> {
	const res = await api<{ friends: FriendView[] }>(ENDPOINTS.FRIENDS);
	return res.friends ?? [];
}

export async function searchUsers(q: string): Promise<UserView[]> {
	const res = await api<{ users: UserView[] }>(
		`${ENDPOINTS.SEARCH_USERS}?q=${encodeURIComponent(q)}&limit=10`
	);
	return res.users ?? [];
}

export function profileSettings(): Promise<ProfileSettings> {
	return api<ProfileSettings>(ENDPOINTS.USER_SETTINGS);
}

export function updateProfileSettings(input: ProfileSettingsInput): Promise<ProfileSettings> {
	return api<ProfileSettings>(ENDPOINTS.USER_SETTINGS, { method: 'PATCH', body: input });
}

export function exportAccountData(): Promise<Record<string, unknown>> {
	return api<Record<string, unknown>>(ENDPOINTS.USER_EXPORT);
}

export function deleteRunHistory(): Promise<{
	deleted: number;
	pinned_kept: boolean;
	best_kept: boolean;
}> {
	return api<{ deleted: number; pinned_kept: boolean; best_kept: boolean }>(ENDPOINTS.USER_RUNS, {
		method: 'DELETE'
	});
}
