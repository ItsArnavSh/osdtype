import { api, newIdempotencyKey } from './client';
import { ENDPOINTS } from './config';
import type { GameView, InviteView, LobbyDuration, RoomView } from './types';

export interface CreateRoomInput {
	name: string;
	description?: string;
	is_public: boolean;
	password?: string;
	max_players: number;
	duration: LobbyDuration;
	language: string;
}

export interface UpdateRoomInput {
	name?: string;
	description?: string;
	is_public?: boolean;
	password?: string;
	max_players?: number;
	duration?: LobbyDuration;
	language?: string;
}

export async function listRooms(offset = 0, limit = 30): Promise<RoomView[]> {
	const res = await api<{ rooms: RoomView[] }>(`${ENDPOINTS.ROOMS}?offset=${offset}&limit=${limit}`);
	return res.rooms ?? [];
}

export async function myRooms(): Promise<RoomView[]> {
	const res = await api<{ rooms: RoomView[] }>(ENDPOINTS.MY_ROOMS);
	return res.rooms ?? [];
}

export async function createRoom(input: CreateRoomInput): Promise<RoomView> {
	return api<RoomView>(ENDPOINTS.CREATE_ROOM, {
		method: 'POST',
		body: input,
		idempotencyKey: newIdempotencyKey()
	});
}

export function getRoom(code: string): Promise<RoomView> {
	return api<RoomView>(ENDPOINTS.ROOM(code));
}

export function joinRoom(code: string, password = ''): Promise<RoomView> {
	return api<RoomView>(ENDPOINTS.ROOM_JOIN(code), { method: 'POST', body: { password } });
}

export function leaveRoom(code: string): Promise<{ ok: boolean }> {
	return api(ENDPOINTS.ROOM_LEAVE(code), { method: 'POST' });
}

export function closeRoom(code: string): Promise<{ ok: boolean }> {
	return api(ENDPOINTS.ROOM(code), { method: 'DELETE' });
}

export function updateRoom(code: string, input: UpdateRoomInput): Promise<RoomView> {
	return api<RoomView>(ENDPOINTS.ROOM(code), { method: 'PATCH', body: input });
}

export function startRoomGame(code: string): Promise<GameView> {
	return api<GameView>(ENDPOINTS.ROOM_START(code), { method: 'POST', body: {} });
}

export function inviteToRoom(code: string, username: string): Promise<InviteView> {
	return api<InviteView>(ENDPOINTS.ROOM_INVITE(code), { method: 'POST', body: { username } });
}

export function addRoomBot(code: string, persona: 'ACE' | 'NOVA' | 'RUNE'): Promise<RoomView> {
	return api<RoomView>(ENDPOINTS.ROOM_BOT(code), { method: 'POST', body: { persona } });
}

function roomAction(path: string, userID: number): Promise<{ ok: boolean }> {
	return api(path, { method: 'POST', body: { user_id: userID } });
}

export const kickFromRoom = (code: string, userID: number) =>
	roomAction(ENDPOINTS.ROOM_KICK(code), userID);
export const banFromRoom = (code: string, userID: number) =>
	roomAction(ENDPOINTS.ROOM_BAN(code), userID);
export const promoteInRoom = (code: string, userID: number) =>
	roomAction(ENDPOINTS.ROOM_PROMOTE(code), userID);
export const demoteInRoom = (code: string, userID: number) =>
	roomAction(ENDPOINTS.ROOM_DEMOTE(code), userID);

export async function listInvites(): Promise<InviteView[]> {
	const res = await api<{ invites: InviteView[] }>(ENDPOINTS.INVITES);
	return res.invites ?? [];
}

export function acceptInvite(id: number): Promise<unknown> {
	return api(ENDPOINTS.ACCEPT_INVITE(id), { method: 'POST' });
}

export function declineInvite(id: number): Promise<unknown> {
	return api(ENDPOINTS.DECLINE_INVITE(id), { method: 'POST' });
}
