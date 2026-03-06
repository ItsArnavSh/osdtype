import { ENDPOINTS } from "./config";
import type { Room, Room_User } from "./types";

/** Shared fetch helper that always sends cookies */
async function authedFetch(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  const res = await fetch(url, { ...options, credentials: "include" });
  if (!res.ok) throw new Error(`Request failed: ${res.status} ${res.statusText}`);
  return res;
}

/** Shared POST helper for Room_User actions */
async function roomUserAction(endpoint: string, body: Room_User): Promise<void> {
  await authedFetch(endpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

/**
 * Create a new room.
 * @param room - Room entity to create.
 */
export async function createRoom(room: Room): Promise<void> {
  await authedFetch(ENDPOINTS.ROOM_CREATE, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(room),
  });
}

/**
 * Add a member to a room.
 */
export async function addMember(roomUser: Room_User): Promise<void> {
  await roomUserAction(ENDPOINTS.ROOM_ADD_MEMBER, roomUser);
}

/**
 * Promote a member to moderator.
 */
export async function promoteToModerator(roomUser: Room_User): Promise<void> {
  await roomUserAction(ENDPOINTS.ROOM_PROMOTE, roomUser);
}

/**
 * Demote a moderator back to member.
 */
export async function demoteToMember(roomUser: Room_User): Promise<void> {
  await roomUserAction(ENDPOINTS.ROOM_DEMOTE, roomUser);
}

/**
 * Block a user from the room.
 */
export async function blockUser(roomUser: Room_User): Promise<void> {
  await roomUserAction(ENDPOINTS.ROOM_BLOCK, roomUser);
}

/**
 * Unblock a previously blocked user.
 */
export async function unblockUser(roomUser: Room_User): Promise<void> {
  await roomUserAction(ENDPOINTS.ROOM_UNBLOCK, roomUser);
}

/**
 * Remove a user from the room.
 */
export async function removeUser(roomUser: Room_User): Promise<void> {
  await roomUserAction(ENDPOINTS.ROOM_REMOVE, roomUser);
}

/**
 * List rooms with pagination.
 * @param index - Pagination index.
 */
export async function listRooms(index: number): Promise<Room[]> {
  const url = new URL(ENDPOINTS.ROOM_LIST);
  url.searchParams.set("index", String(index));
  const res = await authedFetch(url.toString());
  return res.json() as Promise<Room[]>;
}
