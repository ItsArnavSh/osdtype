import { redirect } from '@sveltejs/kit';

// The layout bootstraps the session and lets the backend decide where the
// player belongs. The home route goes straight into practice; the hub is one
// click away from the test header.
export function load() {
	redirect(307, '/play/me');
}
