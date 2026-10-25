// Wire types mirroring the V1 Go entities. Keep these in sync with server/app/entity.

export type LobbyDuration = 30 | 90 | 300;
export const LOBBY_DURATIONS: LobbyDuration[] = [30, 90, 300];

export type AppStateName = 'idle' | 'queue' | 'room' | 'game' | 'results';
export type RoomStatus = 'lobby' | 'playing' | 'closed';
export type RoomRole = 'owner' | 'mod' | 'member';
export type GameMode = 'ranked' | 'room' | 'contest' | 'practice';
export type GameStatus = 'countdown' | 'active' | 'finished' | 'abandoned';

export interface EloView {
	rating: number;
	peak_rating: number;
	games_played: number;
	wins: number;
}

export type ClockKey = '30' | '90' | '300';

export interface UserView {
	id: number;
	username: string;
	display_name?: string;
	bio?: string;
	avatar_url: string;
	rating: number;
	peak_rating: number;
	games_played: number;
	wins: number;
	is_bot: boolean;
	is_admin?: boolean;
	// Present only on the signed-in user's self view.
	save_run_data?: boolean;
	ratings: Record<ClockKey, EloView>;
	best_duration: number;
}

// userElo returns one clock's ladder stats. The fallback keeps older cached
// state and partial responses renderable while the server rolls forward.
export function userElo(user: UserView | null | undefined, duration: number): EloView {
	const key = String(duration) as ClockKey;
	const clock = user?.ratings?.[key];
	if (clock) return clock;
	return {
		rating: user?.rating ?? 400,
		peak_rating: user?.peak_rating ?? 400,
		games_played: user?.games_played ?? 0,
		wins: user?.wins ?? 0
	};
}

export interface MemberView {
	user_id: number;
	username: string;
	avatar_url: string;
	rating: number;
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
	contest_code?: string;
	contest_stage_id?: number;
	contest_stage_name?: string;
	contest_stage_index?: number;
	duration: number;
	language: string;
	seed: string;
	snippet: string;
	start_offset: number;
	temporary?: boolean;
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
	user_id: number;
	mode: GameMode;
	room_code?: string;
	room_private?: boolean;
	language: string;
	duration: number;
	wpm: number;
	accuracy: number;
	rank: number;
	players: number;
	rating_delta: number;
	flagged: boolean;
	played_at: string;
	run_name?: string;
	run_description?: string;
	pinned: boolean;
	pinned_at?: string | null;
}

export interface ProfileView {
	user: UserView;
	position: number;
	positions: Record<ClockKey, number>;
	history: MatchHistory[];
	pinned: MatchHistory[];
	best: MatchHistory | null;
	top_wpm: number;
	online: boolean;
	is_friend: boolean;
	is_followed: boolean;
	follows_you: boolean;
	replays_visible: boolean;
	anonymous: boolean;
}

export interface FriendView {
	user: UserView;
	online: boolean;
	is_friend: boolean;
	follows_you: boolean;
}

export type ProfileVisibility = 'public' | 'anonymous' | 'private';
export type RunSharing = 'all' | 'pinned' | 'none';

export interface ProfileSettings {
	username: string;
	display_name: string;
	bio: string;
	profile_visibility: ProfileVisibility;
	do_not_disturb: boolean;
	hide_online: boolean;
	share_runs: RunSharing;
	share_ranked: boolean;
	share_rooms: boolean;
	share_practice: boolean;
	share_contests: boolean;
	hide_private_rooms: boolean;
	share_replays: boolean;
	save_run_data: boolean;
	retention_limit: number;
	retention_keeps_pinned: boolean;
	retention_keeps_best: boolean;
}

export type ProfileSettingsInput = Partial<
	Pick<
		ProfileSettings,
		| 'display_name'
		| 'bio'
		| 'profile_visibility'
		| 'do_not_disturb'
		| 'hide_online'
		| 'share_runs'
		| 'share_ranked'
		| 'share_rooms'
		| 'share_practice'
		| 'share_contests'
		| 'hide_private_rooms'
		| 'share_replays'
		| 'save_run_data'
	>
>;

export interface LeaderboardRow {
	position: number;
	rating: number;
	peak_rating: number;
	games_played: number;
	wins: number;
	user: UserView;
}

export interface LeaderboardView {
	duration: number;
	leaderboard: LeaderboardRow[];
	nearby: LeaderboardRow[];
	my_position: number;
}

export interface LeaderboardSearchView {
	duration: number;
	results: LeaderboardRow[];
}

export interface RunMetaInput {
	name?: string;
	description?: string;
	pinned?: boolean;
}

export type ContestStatus = 'upcoming' | 'lobby' | 'started' | 'ended' | 'cancelled';
export type ContestVisibility = 'public' | 'private';
export type ContestStageFormat =
	| 'qualifier'
	| 'hotlaps'
	| 'practice'
	| 'multiplayer'
	| 'ranked';
export type ContestStageStatus = 'pending' | 'active' | 'completed' | 'cancelled';
export type ContestLanguageMode = 'fixed' | 'choice';
export type ContestAdvanceType = 'count' | 'percent' | 'all';

export interface ContestParticipant {
	user: UserView;
	joined_at: string;
	current_stage: number;
	eliminated: boolean;
	eliminated_at_stage: number;
}

export interface ContestStageStanding {
	rank: number;
	user: UserView;
	wpm: number;
	raw: number;
	accuracy: number;
	attempts: number;
	game_id: number;
	advanced: boolean;
	eliminated: boolean;
	flagged: boolean;
	left: boolean;
}

export interface ContestStageRun {
	rank: number;
	user: UserView;
	wpm: number;
	raw: number;
	accuracy: number;
	game_id: number;
	flagged: boolean;
	left: boolean;
}

export interface ContestStageView {
	id: number;
	position: number;
	name: string;
	format: ContestStageFormat;
	status: ContestStageStatus;
	starts_at: string;
	ends_at: string | null;
	delay_minutes: number;
	window_minutes: number;
	duration: number;
	language_mode: ContestLanguageMode;
	language: string;
	max_tries: number;
	min_wpm: number;
	advance_type: ContestAdvanceType;
	advance_value: number;
	max_players: number;
	entry_count: number;
	qualified_count: number;
	advance_count: number;
	run_count: number;
	viewer_attempts: number;
	viewer_best_wpm: number;
	viewer_game_id: number;
	viewer_can_attempt: boolean;
	viewer_attempts_left: number;
	standings?: ContestStageStanding[];
	top_runs?: ContestStageRun[];
}

export interface ContestView {
	id: number;
	code: string;
	title: string;
	description: string;
	banner_url: string;
	accent: string;
	custom_css: string;
	visibility: ContestVisibility;
	starts_at: string;
	duration: number;
	language: string;
	status: ContestStatus;
	max_players: number;
	entry_count: number;
	host: UserView;
	is_host: boolean;
	is_enrolled: boolean;
	requires_password: boolean;
	share_link: string;
	game_id: number;
	stage_count: number;
	current_stage: number;
	stages: ContestStageView[];
	participants?: ContestParticipant[];
	leaderboard?: ResultView[];
	created_at: string;
}

export interface ContestListView {
	contests: ContestView[];
	trending: ContestView[];
	mine: ContestView[];
	server_time: string;
}

export type BugReportStatus =
	| 'new'
	| 'triaged'
	| 'in_progress'
	| 'resolved'
	| 'closed'
	| 'wontfix';

export interface BugReport {
	public_id: string;
	title: string;
	category: string;
	severity: string;
	description: string;
	steps: string;
	expected: string;
	actual: string;
	page_url: string;
	user_agent: string;
	app_version: string;
	request_id: string;
	contact_email: string;
	status: BugReportStatus;
	created_at: string;
	updated_at: string;
	resolved_at?: string | null;
}

export interface BugReportInput {
	title: string;
	category: string;
	severity: string;
	description: string;
	steps?: string;
	expected?: string;
	actual?: string;
	page_url?: string;
	user_agent?: string;
	app_version?: string;
	request_id?: string;
	contact_email?: string;
	submission_id: string;
	website?: string;
}

export interface BugReportResponse {
	report: BugReport;
	duplicate: boolean;
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

export interface ReplayRun {
	user_id: number;
	username: string;
	avatar_url: string;
	wpm: number;
	raw: number;
	accuracy: number;
	correct: number;
	wrong: number;
	is_bot: boolean;
	final: string;
	events: ClientKeypress[] | null;
}

export interface ReplayBundle {
	game: GameView;
	runs: ReplayRun[];
}

export interface PracticeRunView {
	id: number;
	temporary: boolean;
	url: string;
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
