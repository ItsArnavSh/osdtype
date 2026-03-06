import { BASE_URL, WS_BASE_URL } from '$lib/config';
import type { User } from './entity/user';

export async function getUserData(user: string): Promise<User> {
	const res = await fetch(`${BASE_URL}/get-user?user=${encodeURI(user)}`, {
		method: 'GET',
		credentials: 'include'
	});

	if (!res.ok) {
		throw new Error('Failed to fetch user');
	}
	return await res.json();
}
export async function whoami(): Promise<User> {
	const res = await fetch(`${BASE_URL}/user/whoami`, {
		method: 'GET',
		credentials: 'include'
	});
	if (!res.ok) {
		throw new Error('Failed to get session info');
	}
	return await res.json();
}

export let ws: WebSocket | null = null;

export async function imonline(useWs = true): Promise<WebSocket | any> {
	if (useWs) {
		// Create or reuse existing WS connection
		if (!ws || ws.readyState !== WebSocket.OPEN) {
			ws = new WebSocket(`${WS_BASE_URL}/user/imonline`);

			ws.onopen = () => console.log('Online via WebSocket');
			ws.onclose = () => console.log('Disconnected from WebSocket');
			ws.onerror = (err) => console.error('WS Error:', err);
		}
		return ws;
	} else {
		// HTTP fallback for local/dev
		const res = await fetch(`${BASE_URL}/user/imonline`, {
			method: 'GET',
			credentials: 'include'
		});
		if (!res.ok) throw new Error('Session Not Marked Online');
		return await res.json();
	}
}
