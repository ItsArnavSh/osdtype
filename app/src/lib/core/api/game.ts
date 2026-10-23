import { api } from './client';
import { ENDPOINTS } from './config';
import type {
	ClientKeypress,
	GameView,
	PracticeRunView,
	ReplayBundle,
	StateView
} from './types';

export function getGame(id: number): Promise<GameView> {
	return api<GameView>(ENDPOINTS.GAME(id));
}

export function leaveGame(id: number): Promise<{ ok: boolean; state: StateView }> {
	return api(ENDPOINTS.GAME_LEAVE(id), { method: 'POST' });
}

export function getGameReplays(id: number): Promise<ReplayBundle> {
	return api<ReplayBundle>(ENDPOINTS.GAME_REPLAYS(id));
}

export function getPublicReplay(gameID: number, userID: number): Promise<ReplayBundle> {
	return api<ReplayBundle>(ENDPOINTS.PUBLIC_REPLAY(gameID, userID));
}

export interface CreatePracticeRunInput {
	language: string;
	duration: number;
	duration_ms: number;
	seed: string;
	snippet: string;
	start_offset: number;
	final: string;
	events: ClientKeypress[];
}

export function createPracticeRun(input: CreatePracticeRunInput): Promise<PracticeRunView> {
	return api<PracticeRunView>(ENDPOINTS.PRACTICE_RUNS, { method: 'POST', body: input });
}

export function savePracticeRun(id: number): Promise<PracticeRunView> {
	return api<PracticeRunView>(ENDPOINTS.PRACTICE_SAVE(id), { method: 'POST' });
}
