import { ENDPOINTS } from "./config";
import type { Contest } from "./types";

/** Shared fetch helper that always sends cookies */
async function authedFetch(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  const res = await fetch(url, { ...options, credentials: "include" });
  if (!res.ok) throw new Error(`Request failed: ${res.status} ${res.statusText}`);
  return res;
}

/**
 * Create a new contest.
 * @param contest - Contest entity to create.
 */
export async function createContest(contest: Contest): Promise<void> {
  await authedFetch(ENDPOINTS.CONTEST_CREATE, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(contest),
  });
}

/**
 * List contests for a room with pagination.
 * @param roomId - The room's ID.
 * @param index  - Pagination index.
 */
export async function listContests(
  roomId: number,
  index: number
): Promise<Contest[]> {
  const url = new URL(ENDPOINTS.CONTEST_LIST);
  url.searchParams.set("room_id", String(roomId));
  url.searchParams.set("index", String(index));
  const res = await authedFetch(url.toString());
  return res.json() as Promise<Contest[]>;
}

/**
 * Fetch data for a specific contest by its job ID.
 * @param jobId - The contest's job ID.
 */
export async function getContestData(jobId: string): Promise<Contest> {
  const res = await authedFetch(ENDPOINTS.CONTEST_GET(jobId));
  return res.json() as Promise<Contest>;
}
