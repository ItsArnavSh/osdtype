import { WS_BASE_URL } from './config';
import { ENDPOINTS } from './config';
import { authToken } from './session';
import type { SocketEvent } from './types';

type Handler = (event: SocketEvent) => void;

let socket: WebSocket | null = null;
let connecting: Promise<void> | null = null;
let intentionalClose = false;
let retryTimer: ReturnType<typeof setTimeout> | null = null;
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;
let retryDelay = 800;
const handlers = new Set<Handler>();

interface OutboundMessage {
	type: string;
	data?: unknown;
}

// Keystrokes typed while the socket is between reconnects must not be
// silently lost: losing them makes the local caret and the server's typed
// buffer disagree, which corrupts the live result and the replay. Hold
// keypresses until the next open, then flush them in order.
const outbound: OutboundMessage[] = [];
const MAX_QUEUED_KEYPRESSES = 2048;

function emit(event: SocketEvent) {
	for (const handler of handlers) handler(event);
}

function stopHeartbeat() {
	if (heartbeatTimer) clearInterval(heartbeatTimer);
	heartbeatTimer = null;
}

function startHeartbeat(ws: WebSocket) {
	stopHeartbeat();
	// Presence expires after 70s. Browsers do not send application messages
	// while idle, so keep the Redis presence key warm with a small heartbeat.
	heartbeatTimer = setInterval(() => {
		if (ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ type: 'ping' }));
		}
	}, 25_000);
}

function wsURL(): string {
	const base = `${WS_BASE_URL}${ENDPOINTS.WS}`;
	const token = authToken();
	return token ? `${base}?token=${encodeURIComponent(token)}` : base;
}

function flushOutbound(ws: WebSocket): void {
	while (outbound.length > 0 && ws.readyState === WebSocket.OPEN) {
		const message = outbound[0];
		try {
			ws.send(JSON.stringify(message));
			outbound.shift();
		} catch {
			// The socket closed between the readyState check and send. Leave
			// the message queued for the next connection.
			return;
		}
	}
}

export function clearQueuedKeypresses(): void {
	outbound.length = 0;
}

export function connectSocket(): Promise<void> {
	if (socket?.readyState === WebSocket.OPEN) return Promise.resolve();
	if (connecting) return connecting;

	intentionalClose = false;
	connecting = new Promise((resolve, reject) => {
		const ws = new WebSocket(wsURL());
		socket = ws;

		ws.onopen = () => {
			connecting = null;
			retryDelay = 800;
			if (retryTimer) clearTimeout(retryTimer);
			retryTimer = null;
			startHeartbeat(ws);
			// Send anything typed while reconnecting before the page asks for
			// authoritative state, so that resync sees the new input.
			flushOutbound(ws);
			emit({ type: 'socket_open' });
			resolve();
		};

		ws.onmessage = (message) => {
			try {
				const parsed = JSON.parse(String(message.data)) as SocketEvent;
				if (parsed && typeof parsed.type === 'string') emit(parsed);
			} catch (err) {
				console.warn('[socket] unreadable message', err);
			}
		};

		ws.onclose = () => {
			connecting = null;
			stopHeartbeat();
			if (socket === ws) socket = null;
			emit({ type: 'socket_close' });
			if (!intentionalClose) {
				const delay = retryDelay;
				retryDelay = Math.min(retryDelay * 1.8, 8000);
				retryTimer = setTimeout(() => {
					connectSocket().catch(() => {});
				}, delay);
			}
		};

		ws.onerror = () => {
			if (connecting) {
				connecting = null;
				reject(new Error('websocket connection failed'));
			}
		};
	});

	return connecting;
}

export function onSocketEvent(handler: Handler): () => void {
	handlers.add(handler);
	return () => handlers.delete(handler);
}

export function sendSocket(type: string, data?: unknown): boolean {
	const message = { type, data };
	if (socket?.readyState === WebSocket.OPEN) {
		try {
			socket.send(JSON.stringify(message));
			return true;
		} catch {
			// Fall through and queue the keypress for the next connection.
		}
	}
	if (type === 'keypress') {
		if (outbound.length >= MAX_QUEUED_KEYPRESSES) outbound.shift();
		outbound.push(message);
	}
	return false;
}

export function disconnectSocket(): void {
	intentionalClose = true;
	clearQueuedKeypresses();
	stopHeartbeat();
	if (retryTimer) clearTimeout(retryTimer);
	retryTimer = null;
	socket?.close();
	socket = null;
	connecting = null;
}

export function socketConnected(): boolean {
	return socket?.readyState === WebSocket.OPEN;
}
