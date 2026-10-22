// Wire types mirroring the V1 Go entities. Keep these in sync with server/app/entity.

export type LobbyDuration = 30 | 90 | 300;
export const LOBBY_DURATIONS: LobbyDuration[] = [30, 90, 300];

export type AppStateName = 'idle' | 'queue' | 'room' | 'game' | 'results';
export type RoomStatus = 'lobby' | 'playing' | 'closed';
export type RoomRole = 'owner' | 'mod' | 'member';
export type GameMode = 'ranked' | 'room' | 'contest';
export type GameStatus = 'countdown' | 'active' | 'finished' | 'abandoned';

export interface RankInfo {
	tier: string;
	division: number;
	label: string;
	rating: number;
	next: number;
	progress: number;
}

export interface UserView {
	id: number;
	username: string;
	avatar_url: string;
	rating: number;
	peak_rating: number;
	games_played: number;
	wins: number;
	is_bot: boolean;
	rank: RankInfo;
}

export interface MemberView {
	user_id: number;
	username: string;
	avatar_url: string;
	rating: number;
	rank: RankInfo;
	role: RoomRole;
	joined_at: string;
	is_bot: boolean;
	online: boolean;
}

export interface RoomView {
	id: number;
	code: string;
	name: string;
	description: string;
	is_public: boolean;
	has_password: boolean;
	max_players: number;
	host_id: number;
	status: RoomStatus;
	duration: LobbyDuration | number;
	language: string;
	game_id: number;
	member_count: number;
	members: MemberView[];
	my_role?: RoomRole;
	invite_link?: string;
	created_at: string;
}

export interface InviteView {
	id: number;
	room_code: string;
	room_name: string;
	from: string;
	from_id: number;
	status: 'pending' | 'accepted' | 'declined' | 'expired';
	created_at: string;
	expires_at: string;
}

export interface ResultView {
	user_id: number;
	username: string;
	avatar_url: string;
	rank: number;
	wpm: number;
	raw: number;
	accuracy: number;
	correct: number;
	wrong: number;
	rating_before: number;
	rating_after: number;
	rating_delta: number;
	flagged: boolean;
	left: boolean;
	is_bot: boolean;
}

export interface PlayerProgress {
	user_id: number;
	username: string;
	avatar_url: string;
	current_points: number;
	wpm: number;
	accuracy: number;
	finished: boolean;
	flagged: boolean;
	is_bot: boolean;
}

export interface GameView {
	id: number;
	mode: GameMode;
	status: GameStatus;
	room_id: number;
	room_code: string;
	contest_id: number;
	duration: number;
	language: string;
	seed: string;
	snippet: string;
	start_offset: number;
	starts_at: string | null;
	ends_at: string | null;
	players: PlayerProgress[];
	leaderboard: ResultView[] | null;
	my_result?: ResultView | null;
	created_at: string;
}

export interface QueueStatus {
	duration: number;
	position: number;
	waiting: number;
	joined_at: string;
}

export interface StateView {
	state: AppStateName;
	user: UserView;
	queue?: QueueStatus | null;
	room?: RoomView | null;
	game?: GameView | null;
	results?: GameView | null;
	invites: InviteView[];
	server_time: string;
}

export interface MatchHistory {
	game_id: number;
	mode: GameMode;
	room_code?: string;
	language: string;
	duration: number;
	wpm: number;
	accuracy: number;
	rank: number;
	players: number;
	rating_delta: number;
	flagged: boolean;
	played_at: string;
}

export interface ProfileView {
	user: UserView;
	position: number;
	history: MatchHistory[];
	online: boolean;
	is_friend: boolean;
	is_followed: boolean;
}

export interface FriendView {
	user: UserView;
	online: boolean;
	is_friend: boolean;
	follows_you: boolean;
}

export interface LeaderboardRow {
	position: number;
	user: UserView;
}

export interface ContestView {
	id: number;
	room_code: string;
	title: string;
	description: string;
	starts_at: string;
	duration: number;
	language: string;
	status: 'upcoming' | 'lobby' | 'started' | 'ended';
	game_id: number;
	leaderboard?: ResultView[];
}

export interface SocketEvent<T = unknown> {
	type: string;
	data?: T;
}

export enum KeypressAction {
	KEYPRESS = 0,
	BACKSPACE = 1
}

export interface ClientKeypress {
	value: string;
	action: KeypressAction;
	time_ms?: number;
}

export const LANGUAGE_LABELS: Record<string, string> = {
	c: 'C',
	cpp: 'C++',
	go: 'Go',
	haskell: 'Haskell',
	html: 'HTML',
	java: 'Java',
	js: 'JavaScript',
	python: 'Python',
	rust: 'Rust',
	sql: 'SQL'
};
