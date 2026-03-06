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
  OWNER = 2,
}

export interface Room_User {
  room_id: number;
  user_id: number;
  role: RoomRole;
}

export interface Contest {
  [key: string]: unknown; // Extend as the Contest entity is defined server-side
}

// ─── Lobby ───────────────────────────────────────────────────────────────────

export type LobbyDuration = 30 | 90 | 300;

// ─── WebSocket Message Types (Client → Server) ────────────────────────────────

export enum KeypressAction {
  KEYPRESS = 0,
  BACKSPACE = 1,
}

export interface ClientKeypress {
  value: string;
  action: KeypressAction;
  time_ms: number;
}

// ─── WebSocket Message Types (Server → Client) ────────────────────────────────

export interface GameplayBroadcast {
  player_id: number;
  current_points: number;
  update: ClientKeypress;
}

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

/** A 4-byte big-endian uint32 as a string — used to seed the RNG */
export type GameSeed = string;

/** "nil" signals an unsubscription/disconnection for a specific module */
export type ControlMessage = "nil";

export type ServerMessage =
  | GameSeed
  | GameplayBroadcast
  | LobbyInvitation
  | Leaderboard
  | ControlMessage;
