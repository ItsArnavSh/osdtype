export const MOBILE_QUERY = '(max-width: 767px)';

// Phones are a read-only companion for now. Keep the hub as a landing page,
// but send the actual typing/game surfaces to the desktop-only gate.
const MOBILE_PATHS = [
	'/',
	'/play',
	'/login',
	'/friends',
	'/leaderboard',
	'/contests',
	'/docs',
	'/welcome',
	'/bugs',
	'/settings'
];
// Contest pages are viewable on phones. Entering a live typing stage still
// lands on the desktop-only game surface.
const MOBILE_PREFIXES = ['/profile/', '/replay/', '/contests/'];

export function mobileAllowed(path: string): boolean {
	// /admin stays untouched so its existing 404 behavior is preserved for
	// non-admins. It is not linked from the mobile navigation.
	if (path === '/admin') return true;
	return MOBILE_PATHS.includes(path) || MOBILE_PREFIXES.some((prefix) => path.startsWith(prefix));
}
