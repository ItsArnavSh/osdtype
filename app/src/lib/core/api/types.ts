// ─── Core Entities ───────────────────────────────────────────────────────────

export interface User {
	id: number;
	username: string;
	current_rank: number;
}

export interface Room {
	name: string;
	description: string;
	private: boolean;
}

export enum RoomRole {
	MEMBER = 0,
	MODERATOR = 1,
	OWNER = 2
}

export interface Room_User {
	room_id: number;
	user_id: number;
	role: RoomRole;
}

export interface Contest {
	[key: string]: unknown;
}

// ─── Lobby ───────────────────────────────────────────────────────────────────

export type LobbyDuration = 30 | 90 | 300;

// ─── WebSocket Message Types (Client → Server) ────────────────────────────────

export enum KeypressAction {
	KEYPRESS = 0,
	BACKSPACE = 1
}

export interface ClientKeypress {
	value: string;
	action: KeypressAction;
	time_ms: number;
}

// ─── WebSocket Message Types (Server → Client) ────────────────────────────────

export interface GameplayBroadcast {
	player_id: number; // uint32
	current_points: number; // uint16 — character offset of this player
	update: ClientKeypress;
}

/** Full state snapshot — server sends []OutGoing per tick */
export type GameplayBroadcastFrame = GameplayBroadcast[];

export interface LobbyInvitation {
	from: string;
	lobby_id: number;
}

export interface LeaderboardEntry {
	id: number;
	raw: number;
	wpm: number;
	accuracy: number;
	wrong: number;
}

export type Leaderboard = LeaderboardEntry[];

/** Base64-encoded uint32 seed string */
export type GameSeed = string;

/** "nil" signals an unsubscription/disconnection for a specific module */
export type ControlMessage = 'nil';

export type ServerMessage =
	| GameSeed
	| GameplayBroadcastFrame
	| LobbyInvitation
	| Leaderboard
	| ControlMessage;
