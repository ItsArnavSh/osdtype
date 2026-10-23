import { api } from './client';
import { BASE_URL, ENDPOINTS } from './config';
import type {
	ContestAdvanceType,
	ContestLanguageMode,
	ContestListView,
	ContestStageFormat,
	ContestView,
	ContestVisibility,
	GameView,
	LobbyDuration
} from './types';

export interface ContestStageInput {
	name: string;
	format: ContestStageFormat;
	delay_minutes: number;
	window_minutes: number;
	duration: LobbyDuration;
	language_mode: ContestLanguageMode;
	language: string;
	max_tries: number;
	min_wpm: number;
	advance_type: ContestAdvanceType;
	advance_value: number;
	max_players: number;
}

export interface ContestInput {
	title: string;
	description?: string;
	banner_url?: string;
	accent?: string;
	custom_css?: string;
	visibility: ContestVisibility;
	password?: string;
	starts_at: string;
	duration: LobbyDuration;
	language: string;
	max_players: number;
	stages: ContestStageInput[];
}

export function listContests(): Promise<ContestListView> {
	return api<ContestListView>(ENDPOINTS.CONTESTS);
}

export function getContest(code: string): Promise<ContestView> {
	return api<ContestView>(ENDPOINTS.CONTEST(code));
}

export function createContest(input: ContestInput): Promise<ContestView> {
	return api<ContestView>(ENDPOINTS.CONTESTS, { method: 'POST', body: input });
}

export function updateContest(code: string, input: Partial<ContestInput>): Promise<ContestView> {
	return api<ContestView>(ENDPOINTS.CONTEST(code), { method: 'PATCH', body: input });
}

export function cancelContest(code: string): Promise<ContestView> {
	return api<ContestView>(ENDPOINTS.CONTEST(code), { method: 'DELETE' });
}

export function enrollContest(code: string, password = ''): Promise<ContestView> {
	return api<ContestView>(`${ENDPOINTS.CONTEST(code)}/enroll`, {
		method: 'POST',
		body: password ? { password } : {}
	});
}

export function leaveContest(code: string): Promise<ContestView> {
	return api<ContestView>(`${ENDPOINTS.CONTEST(code)}/enroll`, { method: 'DELETE' });
}

export function startContestAttempt(
	code: string,
	position: number,
	language = ''
): Promise<GameView> {
	return api<GameView>(`${ENDPOINTS.CONTEST(code)}/stages/${position}/attempts`, {
		method: 'POST',
		body: language ? { language } : {}
	});
}

export function contestCalendarURL(code: string): string {
	return `${BASE_URL}${ENDPOINTS.CONTEST_CALENDAR(code)}`;
}
