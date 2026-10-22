import { api } from './client';
import { ENDPOINTS } from './config';
import type { FriendView, LeaderboardRow, ProfileView, UserView } from './types';

export async function leaderboard(offset = 0, limit = 50): Promise<{
	leaderboard: LeaderboardRow[];
	my_position: number;
}> {
	return api(`${ENDPOINTS.LEADERBOARD}?offset=${offset}&limit=${limit}`);
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
