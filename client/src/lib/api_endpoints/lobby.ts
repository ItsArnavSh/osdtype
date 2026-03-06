import { BASE_URL } from '$lib/config';
export async function getUserData(user: string): Promise<User> {
	const res = await fetch(`${BASE_URL}/join-lobby`, {
		method: 'GET',
		credentials: 'include'
	});

	if (!res.ok) {
		throw new Error('Failed to fetch user');
	}
	return await res.json();
}
