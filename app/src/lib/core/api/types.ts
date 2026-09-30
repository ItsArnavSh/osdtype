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
export enum ContestStatus {
	UPCOMING = 0,
	LOBBY = 1,
	STARTED = 2,
	ENDED = 3
}
export interface WPMRes {
	name: string;
	id: number;
	raw: number;
	wpm: number;
	accuracy: number;
	correct: number;
	wrong: number;
}
export interface Contest {
	id: string;
	jobID: number;
	roomID: number;
	time: Date;
	data: string; // title, writeup etc
	lang: number;
	duration: LobbyDuration;
	lobbyID: number; // allotted by scheduler
	status: ContestStatus;
	leaderboard: LeaderboardEntry[];
}

// ─── Lobby ───────────────────────────────────────────────────────────────────

export const LobbyDuration = {
	SHORT: 30,
	MEDIUM: 90,
	LONG: 300
} as const;

export type LobbyDuration = (typeof LobbyDuration)[keyof typeof LobbyDuration];

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
	name: string;
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
