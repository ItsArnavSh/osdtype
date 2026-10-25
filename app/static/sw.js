// OSDTYP's PWA service worker.
//
// The app is client-rendered, so one cached shell is enough to start any route
// offline. API and websocket traffic is never cached: admin data and live games
// must always come from the network.
const CACHE = 'osdtype-pwa-v3';
const SHELL = '/play';
const PRECACHE = [
	SHELL,
	'/manifest.webmanifest',
	'/icons/icon-192.png',
	'/icons/icon-512.png',
	'/icons/icon-maskable-192.png',
	'/icons/icon-maskable-512.png'
];
const STATIC_DESTINATIONS = new Set(['script', 'style', 'font', 'image', 'manifest']);

self.addEventListener('install', (event) => {
	event.waitUntil(
		caches
			.open(CACHE)
			.then((cache) => cache.addAll(PRECACHE))
			.then(() => self.skipWaiting())
	);
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) =>
				Promise.all(
					keys.filter((key) => key !== CACHE && key.startsWith('osdtype-pwa-')).map((key) => caches.delete(key))
				)
			)
			.then(() => self.clients.claim())
	);
});

async function cacheFirst(request) {
	const cached = await caches.match(request);
	if (cached) return cached;

	const response = await fetch(request);
	if (response.ok) {
		const cache = await caches.open(CACHE);
		await cache.put(request, response.clone());
	}
	return response;
}

async function networkFirstShell(request) {
	try {
		const response = await fetch(request);
		if (response.ok && response.headers.get('content-type')?.includes('text/html')) {
			const cache = await caches.open(CACHE);
			await cache.put(SHELL, response.clone());
		}
		return response;
	} catch {
		const cached = await caches.match(SHELL);
		if (cached) return cached;

		return new Response('OSDTYP is offline. Reconnect and try again.', {
			status: 503,
			headers: { 'Content-Type': 'text/plain; charset=utf-8' }
		});
	}
}

self.addEventListener('fetch', (event) => {
	const request = event.request;
	if (request.method !== 'GET') return;

	const url = new URL(request.url);
	if (url.origin !== self.location.origin) return;

	if (request.mode === 'navigate') {
		event.respondWith(networkFirstShell(request));
		return;
	}

	if (
		STATIC_DESTINATIONS.has(request.destination) ||
		url.pathname.startsWith('/_app/') ||
		url.pathname.startsWith('/icons/') ||
		url.pathname === '/manifest.webmanifest'
	) {
		event.respondWith(cacheFirst(request));
	}
});
