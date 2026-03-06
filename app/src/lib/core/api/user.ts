import { ENDPOINTS } from "./config";
import type { User, LobbyDuration } from "./types";

/** Shared fetch helper that always sends cookies */
async function authedFetch(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  const res = await fetch(url, { ...options, credentials: "include" });
  if (!res.ok) throw new Error(`Request failed: ${res.status} ${res.statusText}`);
  return res;
}

/**
 * Returns the currently authenticated user's data.
 */
export async function whoAmI(): Promise<User> {
  const res = await authedFetch(ENDPOINTS.WHOAMI);
  return res.json() as Promise<User>;
}

/**
 * Adds the user to the matchmaking queue for a ranked match.
 * @param duration - Match duration in seconds: 30, 90, or 300.
 */
export async function joinRankedLobby(duration: LobbyDuration): Promise<void> {
  const url = new URL(ENDPOINTS.JOIN_LOBBY);
  url.searchParams.set("duration", String(duration));
  await authedFetch(url.toString());
}

/**
 * Follow another user.
 * @param username - The username to follow.
 */
export async function followUser(username: string): Promise<void> {
  const url = new URL(ENDPOINTS.FOLLOW);
  url.searchParams.set("user", username);
  await authedFetch(url.toString(), { method: "POST" });
}

/**
 * Unfollow a user.
 * @param username - The username to unfollow.
 */
export async function unfollowUser(username: string): Promise<void> {
  const url = new URL(ENDPOINTS.UNFOLLOW);
  url.searchParams.set("user", username);
  await authedFetch(url.toString(), { method: "POST" });
}

/**
 * Join a manually-controlled lobby by its ID.
 * @param lobbyId - The ID of the lobby to join.
 */
export async function joinControlledLobby(lobbyId: number): Promise<void> {
  const url = new URL(ENDPOINTS.JOIN_CLOBBY);
  url.searchParams.set("lobbyid", String(lobbyId));
  await authedFetch(url.toString());
}

/**
 * Invite another player to a lobby.
 * @param inviteeId - The ID of the player to invite.
 * @param lobbyId   - The ID of the lobby.
 */
export async function inviteToLobby(
  inviteeId: number,
  lobbyId: number
): Promise<void> {
  const url = new URL(ENDPOINTS.INVITE_TO_LOBBY);
  url.searchParams.set("invitee", String(inviteeId));
  url.searchParams.set("lobbyid", String(lobbyId));
  await authedFetch(url.toString());
}
