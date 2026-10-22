const rawBase = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';

export const BASE_URL = rawBase.replace(/\/$/, '');
export const WS_BASE_URL = BASE_URL.replace(/^https?/, (s) => (s === 'https' ? 'wss' : 'ws'));

export const ENDPOINTS = {
	PING: '/ping',
	ME: '/auth/me',
	DEV_LOGIN: '/auth/dev-login',
	LOGOUT: '/auth/logout',
	GITHUB_LOGIN: '/login/github',
	STATE: '/state',
	ACK_RESULTS: '/state/ack',
	SEARCH_USERS: '/user/search',
	FRIENDS: '/user/friends',
	FOLLOW: '/user/follow',
	UNFOLLOW: '/user/unfollow',
	PROFILE: (username: string) => `/user/${encodeURIComponent(username)}/profile`,
	LEADERBOARD: '/ranked/leaderboard',
	QUEUE: '/ranked/queue',
	ROOMS: '/room/list',
	MY_ROOMS: '/room/mine',
	CREATE_ROOM: '/room/create',
	ROOM: (code: string) => `/room/${encodeURIComponent(code)}`,
	ROOM_JOIN: (code: string) => `/room/${encodeURIComponent(code)}/join`,
	ROOM_LEAVE: (code: string) => `/room/${encodeURIComponent(code)}/leave`,
	ROOM_START: (code: string) => `/room/${encodeURIComponent(code)}/start`,
	ROOM_INVITE: (code: string) => `/room/${encodeURIComponent(code)}/invite`,
	ROOM_BOT: (code: string) => `/room/${encodeURIComponent(code)}/bot`,
	ROOM_KICK: (code: string) => `/room/${encodeURIComponent(code)}/kick`,
	ROOM_BAN: (code: string) => `/room/${encodeURIComponent(code)}/ban`,
	ROOM_PROMOTE: (code: string) => `/room/${encodeURIComponent(code)}/promote`,
	ROOM_DEMOTE: (code: string) => `/room/${encodeURIComponent(code)}/demote`,
	ROOM_CONTESTS: (code: string) => `/room/${encodeURIComponent(code)}/contests`,
	ROOM_CONTEST: (code: string) => `/room/${encodeURIComponent(code)}/contest`,
	INVITES: '/invites',
	ACCEPT_INVITE: (id: number) => `/invites/${id}/accept`,
	DECLINE_INVITE: (id: number) => `/invites/${id}/decline`,
	GAME: (id: number) => `/game/${id}`,
	GAME_LEAVE: (id: number) => `/game/${id}/leave`,
	CONTEST: (id: number) => `/contest/${id}`,
	WS: '/ws'
} as const;
