import { api } from './client';
import { BASE_URL, ENDPOINTS } from './config';
import { clearAuthToken } from './session';
import type { UserView } from './types';

export function githubLogin(): void {
	window.location.href = `${BASE_URL}${ENDPOINTS.GITHUB_LOGIN}`;
}

export async function devLogin(username: string): Promise<UserView> {
	const res = await api<{ user: UserView }>(ENDPOINTS.DEV_LOGIN, {
		method: 'POST',
		body: { username }
	});
	return res.user;
}

export async function logout(): Promise<void> {
	try {
		await api(ENDPOINTS.LOGOUT, { method: 'POST' });
	} finally {
		clearAuthToken();
	}
}

export async function me(): Promise<UserView | null> {
	try {
		return await api<UserView>(ENDPOINTS.ME);
	} catch (err) {
		if (err instanceof Error && 'status' in err && (err as { status?: number }).status === 401) {
			clearAuthToken();
			return null;
		}
		return null;
	}
}
