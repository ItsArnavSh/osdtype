import { BASE_URL, WS_BASE_URL } from '$lib/config';

type MessageHandler = (data: any) => void;

let ws: WebSocket | null = null;
let isConnecting = false;
const handlers = new Set<MessageHandler>();

/* -------------------- CONNECT -------------------- */

export async function connect(useWs = true): Promise<void> {
	if (!useWs) {
		const res = await fetch(`${BASE_URL}/user/imonline`, {
			method: 'GET',
			credentials: 'include'
		});
		if (!res.ok) throw new Error('Session Not Marked Online');
		return;
	}

	if (ws && ws.readyState === WebSocket.OPEN) {
		console.log('Already connected');
		return;
	}

	if (isConnecting) return;

	isConnecting = true;

	return new Promise((resolve, reject) => {
		ws = new WebSocket(`${WS_BASE_URL}/user/imonline`);

		ws.onopen = () => {
			console.log('Online via WebSocket');
			isConnecting = false;
			resolve();
		};

		ws.onmessage = (event) => {
			try {
				const data = JSON.parse(event.data);
				console.log('Received:', data);
				handlers.forEach((fn) => fn(data));
			} catch (err) {
				console.error('Failed to parse message:', err);
			}
		};

		ws.onclose = () => {
			console.log('Disconnected from WebSocket');
			ws = null;
			isConnecting = false;
		};

		ws.onerror = (err) => {
			console.error('WS Error:', err);
			isConnecting = false;
			reject(err);
		};
	});
}

/* -------------------- SEND -------------------- */

export function send(data: any): void {
	if (!ws || ws.readyState !== WebSocket.OPEN) {
		console.error('WebSocket not connected');
		return;
	}
	ws.send(JSON.stringify(data));
	console.log('Sent:', data);
}

/* -------------------- SUBSCRIBE -------------------- */

export function onMessage(handler: MessageHandler): () => void {
	handlers.add(handler);
	return () => handlers.delete(handler);
}

/* -------------------- STATE -------------------- */

export function isConnected(): boolean {
	return ws !== null && ws.readyState === WebSocket.OPEN;
}

/* -------------------- DISCONNECT -------------------- */

export function disconnect(): void {
	if (ws) {
		ws.close();
		ws = null;
	}
}
