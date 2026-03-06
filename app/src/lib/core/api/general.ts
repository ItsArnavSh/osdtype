import { ENDPOINTS } from "./config";
import type { User } from "./types";

/**
 * Health check — returns "pong" if the server is up.
 */
export async function ping(): Promise<string> {
  const res = await fetch(ENDPOINTS.PING);
  if (!res.ok) throw new Error(`Ping failed: ${res.status}`);
  const data: { reply: string } = await res.json();
  return data.reply;
}

/**
 * Fetch public info for any user by username.
 * @param username - The username to look up.
 */
export async function getUserInfo(username: string): Promise<User> {
  const url = new URL(ENDPOINTS.GET_USER);
  url.searchParams.set("user", username);

  const res = await fetch(url.toString());
  if (!res.ok) throw new Error(`Get user failed: ${res.status}`);
  return res.json() as Promise<User>;
}
