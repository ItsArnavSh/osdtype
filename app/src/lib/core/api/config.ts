export const BASE_URL = "https://your-osdtype-server.com";
export const WS_BASE_URL = BASE_URL.replace(/^https?/, (s) =>
  s === "https" ? "wss" : "ws"
);

export const ENDPOINTS = {
  // Auth
  LOGIN_GITHUB: `${BASE_URL}/login/github`,
  AUTH_CALLBACK: `${BASE_URL}/auth/github/callback`,
  LOGIN_FAKE: `${BASE_URL}/login/github/fake`,

  // General
  PING: `${BASE_URL}/ping`,
  GET_USER: `${BASE_URL}/get-user`,

  // User (Protected)
  WHOAMI: `${BASE_URL}/user/whoami`,
  JOIN_LOBBY: `${BASE_URL}/user/join-lobby`,
  I_AM_ONLINE: `${BASE_URL}/user/imonline`,
  FOLLOW: `${BASE_URL}/user/follow`,
  UNFOLLOW: `${BASE_URL}/user/unfollow`,
  JOIN_CLOBBY: `${BASE_URL}/user/join-clobby`,
  INVITE_TO_LOBBY: `${BASE_URL}/user/invite-to-lobby`,

  // Room (Protected)
  ROOM_CREATE: `${BASE_URL}/room/create`,
  ROOM_ADD_MEMBER: `${BASE_URL}/room/add-member`,
  ROOM_PROMOTE: `${BASE_URL}/room/promote`,
  ROOM_DEMOTE: `${BASE_URL}/room/demote`,
  ROOM_BLOCK: `${BASE_URL}/room/block`,
  ROOM_UNBLOCK: `${BASE_URL}/room/unblock`,
  ROOM_REMOVE: `${BASE_URL}/room/remove`,
  ROOM_LIST: `${BASE_URL}/room/list`,

  // Contest (Protected)
  CONTEST_CREATE: `${BASE_URL}/room/contest/create`,
  CONTEST_LIST: `${BASE_URL}/room/contest/list`,
  CONTEST_GET: (jobId: string) => `${BASE_URL}/room/contest/${jobId}`,

  // WebSocket
  WS_SESSION: `${WS_BASE_URL}/user/imonline`,
} as const;
