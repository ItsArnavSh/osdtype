const TOKEN_KEY = 'osdtype.auth.token';
let memoryToken: string | null = null;

// The API normally keeps the session in an HttpOnly cookie. When the frontend
// and API are on different sites (Vercel + the AWS API hostname), browsers with
// third-party cookie blocking will not send that cookie from the frontend.
// The OAuth callback also hands the JWT back in the URL fragment so the SPA can
// use a Bearer token as a first-party-compatible fallback. Fragments are never
// sent to Vercel or to the API, and we remove it from the address bar as soon
// as it is read.
export function authToken(): string | null {
	if (memoryToken) return memoryToken;
	if (typeof window === 'undefined') return null;
	try {
		return window.localStorage.getItem(TOKEN_KEY);
	} catch {
		return null;
	}
}

export function setAuthToken(token: string): void {
	memoryToken = token;
	if (typeof window === 'undefined') return;
	try {
		window.localStorage.setItem(TOKEN_KEY, token);
	} catch {
		// Storage can be unavailable in private browsing. The in-memory copy
		// keeps the tab signed in, and the cookie path still works when the
		// browser allows it.
	}
}

export function clearAuthToken(): void {
	memoryToken = null;
	if (typeof window === 'undefined') return;
	try {
		window.localStorage.removeItem(TOKEN_KEY);
	} catch {
		// Ignore storage failures.
	}
}

export function consumeAuthTokenFromURL(): void {
	if (typeof window === 'undefined') return;
	const hash = window.location.hash.startsWith('#')
		? window.location.hash.slice(1)
		: window.location.hash;
	if (!hash) return;
	const token = new URLSearchParams(hash).get('token');
	if (!token) return;
	setAuthToken(token);
	window.history.replaceState(
		window.history.state,
		'',
		`${window.location.pathname}${window.location.search}`
	);
}
