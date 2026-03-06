import { connect, send, onMessage, isConnected } from './websocket';
import type {
	ClientKeypress,
	GameplayBroadcast,
	GameplayBroadcastFrame,
	LobbyInvitation,
	Leaderboard,
	LeaderboardEntry,
	GameSeed,
	ControlMessage
} from './types';

// ─── Handlers ────────────────────────────────────────────────────────────────

export interface SessionHandlers {
	onSeed?: (seed: GameSeed) => void;
	onBroadcast?: (frame: GameplayBroadcastFrame) => void;
	onInvitation?: (invite: LobbyInvitation) => void;
	onLeaderboard?: (board: Leaderboard) => void;
	onControl?: (msg: ControlMessage) => void;
	onOpen?: () => void;
	onClose?: (event: CloseEvent) => void;
	onError?: (event: Event) => void;
}

// ─── Type Guards ─────────────────────────────────────────────────────────────

function isSingleBroadcast(obj: unknown): obj is GameplayBroadcast {
	return (
		typeof obj === 'object' &&
		obj !== null &&
		'player_id' in obj &&
		'current_points' in obj &&
		'update' in obj
	);
}

/** Server sends []OutGoing — an array of broadcast entries */
function isBroadcastFrame(obj: unknown): obj is GameplayBroadcastFrame {
	return Array.isArray(obj) && obj.length > 0 && isSingleBroadcast(obj[0]);
}

function isLobbyInvitation(obj: unknown): obj is LobbyInvitation {
	return typeof obj === 'object' && obj !== null && 'from' in obj && 'lobby_id' in obj;
}

function isLeaderboard(obj: unknown): obj is Leaderboard {
	return (
		Array.isArray(obj) && obj.length > 0 && typeof (obj[0] as LeaderboardEntry).wpm === 'number'
	);
}

// ─── Seed envelope ───────────────────────────────────────────────────────────

interface SeedEnvelope {
	type: 'seed';
	value: string;
}

function isSeedEnvelope(obj: unknown): obj is SeedEnvelope {
	return (
		typeof obj === 'object' &&
		obj !== null &&
		(obj as SeedEnvelope).type === 'seed' &&
		typeof (obj as SeedEnvelope).value === 'string'
	);
}

// ─── Session ─────────────────────────────────────────────────────────────────

export class OSDTypeSession {
	private unsub: (() => void) | null = null;
	private handlers: SessionHandlers;

	constructor(handlers: SessionHandlers = {}) {
		this.handlers = handlers;
	}

	async connect(): Promise<void> {
		await connect();
		this.handlers.onOpen?.();
		this.unsub = onMessage((data: unknown) => this.handleMessage(data));
	}

	disconnect(): void {
		this.unsub?.();
		this.unsub = null;
		this.handlers.onClose?.(new CloseEvent('close'));
	}

	sendKeypress(keypress: ClientKeypress): void {
		if (!isConnected()) throw new Error('WebSocket is not connected.');
		send(keypress);
	}

	get isConnected(): boolean {
		return isConnected();
	}

	// ─── Private ─────────────────────────────────────────────────────────────

	private handleMessage(data: unknown): void {
		if (data === 'nil') {
			this.handlers.onControl?.(data as ControlMessage);
			return;
		}

		// Seed envelope — wrapped by websocket.ts
		if (isSeedEnvelope(data)) {
			this.handlers.onSeed?.(data.value as GameSeed);
			return;
		}

		// Seed arriving as a raw base64 string (websocket.ts didn't wrap it)
		if (typeof data === 'string') {
			try {
				const decoded = atob(data);
				if (/^\d+$/.test(decoded)) {
					this.handlers.onSeed?.(decoded as GameSeed);
					return;
				}
			} catch {
				// not base64 — fall through
			}
			console.warn('[OSDTypeSession] Unknown string message:', data);
			return;
		}

		// Leaderboard check must come before broadcast —
		// both are arrays but leaderboard entries have `wpm`, broadcast entries have `player_id`
		if (isLeaderboard(data)) {
			this.handlers.onLeaderboard?.(data);
		} else if (isBroadcastFrame(data)) {
			this.handlers.onBroadcast?.(data);
		} else if (isLobbyInvitation(data)) {
			this.handlers.onInvitation?.(data);
		} else {
			console.warn('[OSDTypeSession] Unknown message shape:', data);
		}
	}
}
