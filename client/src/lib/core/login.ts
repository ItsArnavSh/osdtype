import { BASE_URL } from '$lib/config';

export async function login(username: string) {
	try {
		const res = await fetch(
			`${BASE_URL}/login/github/fake?username=${encodeURIComponent(username)}`,
			{
				method: 'GET',
				credentials: 'include'
			}
		);

		if (!res.ok) {
			// Try to read backend error message if available
			let errorText;
			try {
				const data = await res.json();
				errorText = data.message || JSON.stringify(data);
			} catch {
				errorText = await res.text();
			}

			console.error('Login failed:', res.status, errorText);
			throw new Error(`Login failed: ${errorText}`);
		}

		// Parse JSON on success
		const data = await res.json();
		console.log('Login successful:', data);
		return data;
	} catch (err) {
		console.error('Network or server error:', err);
		throw err;
	}
}
