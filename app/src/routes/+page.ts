import { redirect } from '@sveltejs/kit';

// There is no landing once the backend (and login) is out of the picture —
// the proof of concept starts straight in singleplayer.
export function load() {
	redirect(307, '/play/me');
}
