import { redirect } from '@sveltejs/kit';

// The layout bootstraps the session and lets the backend decide where the
// player belongs. From here, idle users land on the hub.
export function load() {
	redirect(307, '/play');
}
