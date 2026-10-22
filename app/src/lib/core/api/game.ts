import { api } from './client';
import { ENDPOINTS } from './config';
import type { GameView, StateView } from './types';

export function getGame(id: number): Promise<GameView> {
	return api<GameView>(ENDPOINTS.GAME(id));
}

export function leaveGame(id: number): Promise<{ ok: boolean; state: StateView }> {
	return api(ENDPOINTS.GAME_LEAVE(id), { method: 'POST' });
}
