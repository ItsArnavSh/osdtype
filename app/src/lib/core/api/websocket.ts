import { ENDPOINTS } from "./config";
import type {
  ClientKeypress,
  GameplayBroadcast,
  LobbyInvitation,
  Leaderboard,
  LeaderboardEntry,
  GameSeed,
  ControlMessage,
} from "./types";

// ─── Handlers ────────────────────────────────────────────────────────────────

export interface SessionHandlers {
  onSeed?: (seed: GameSeed) => void;
  onBroadcast?: (msg: GameplayBroadcast) => void;
  onInvitation?: (invite: LobbyInvitation) => void;
  onLeaderboard?: (board: Leaderboard) => void;
  onControl?: (msg: ControlMessage) => void;
  onOpen?: () => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (event: Event) => void;
}

// ─── Type Guards ─────────────────────────────────────────────────────────────

function isGameplayBroadcast(obj: unknown): obj is GameplayBroadcast {
  return (
    typeof obj === "object" &&
    obj !== null &&
    "player_id" in obj &&
    "current_points" in obj &&
    "update" in obj
  );
}

function isLobbyInvitation(obj: unknown): obj is LobbyInvitation {
  return (
    typeof obj === "object" &&
    obj !== null &&
    "from" in obj &&
    "lobby_id" in obj
  );
}

function isLeaderboard(obj: unknown): obj is Leaderboard {
  return (
    Array.isArray(obj) &&
    obj.length > 0 &&
    typeof (obj[0] as LeaderboardEntry).wpm === "number"
  );
}

// ─── Session ─────────────────────────────────────────────────────────────────

export class OSDTypeSession {
  private ws: WebSocket | null = null;
  private handlers: SessionHandlers;

  constructor(handlers: SessionHandlers = {}) {
    this.handlers = handlers;
  }

  /**
   * Establish the WebSocket session with the server.
   * Requires the token cookie to already be set.
   */
  connect(): void {
    if (this.ws) this.disconnect();

    this.ws = new WebSocket(ENDPOINTS.WS_SESSION);

    this.ws.onopen = () => {
      this.handlers.onOpen?.();
    };

    this.ws.onclose = (event) => {
      this.handlers.onClose?.(event);
    };

    this.ws.onerror = (event) => {
      this.handlers.onError?.(event);
    };

    this.ws.onmessage = (event: MessageEvent) => {
      this.handleMessage(event.data as string);
    };
  }

  /** Close the WebSocket connection. */
  disconnect(): void {
    this.ws?.close();
    this.ws = null;
  }

  /**
   * Send a keypress event to the server during gameplay.
   * @param keypress - The keypress payload.
   */
  sendKeypress(keypress: ClientKeypress): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error("WebSocket is not connected.");
    }
    this.ws.send(JSON.stringify(keypress));
  }

  /** Returns true if the socket is currently open. */
  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  // ─── Private ───────────────────────────────────────────────────────────────

  private handleMessage(raw: string): void {
    // Control message
    if (raw === "nil") {
      this.handlers.onControl?.(raw as ControlMessage);
      return;
    }

    // Game seed — a 4-byte big-endian uint32 as a plain numeric string
    if (/^\d{8}$/.test(raw)) {
      this.handlers.onSeed?.(raw as GameSeed);
      return;
    }

    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      console.warn("[OSDTypeSession] Could not parse message:", raw);
      return;
    }

    if (isLeaderboard(parsed)) {
      this.handlers.onLeaderboard?.(parsed);
    } else if (isGameplayBroadcast(parsed)) {
      this.handlers.onBroadcast?.(parsed);
    } else if (isLobbyInvitation(parsed)) {
      this.handlers.onInvitation?.(parsed);
    } else {
      console.warn("[OSDTypeSession] Unknown message shape:", parsed);
    }
  }
}
