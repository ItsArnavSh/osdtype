import { BASE_URL } from '$lib/config';

export async function ping() {
	const res = await fetch(`${BASE_URL}/ping`, {
		method: 'GET'
	});

	if (!res.ok) {
		throw new Error('Failed to fetch user');
	}

	return await res.json();
}
