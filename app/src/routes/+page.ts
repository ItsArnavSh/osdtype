import { browser } from '$app/environment';
import { redirect } from '@sveltejs/kit';
import { MOBILE_QUERY } from '$lib/core/mobile';

// The layout bootstraps the session and lets the backend decide where the
// player belongs. Desktop goes straight into practice; phones start at the
// read-only companion hub instead of the desktop-only typing surface.
export function load() {
	const mobile = browser && window.matchMedia(MOBILE_QUERY).matches;
	redirect(307, mobile ? '/play' : '/play/me');
}
