import { error } from '@sveltejs/kit';
import { ApiError } from '$lib/core/api/client';
import { adminOverview } from '$lib/core/api/admin';

export async function load() {
	try {
		return { overview: await adminOverview() };
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) {
			throw error(404, 'Not found');
		}
		throw err;
	}
}
