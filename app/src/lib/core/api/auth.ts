import { ENDPOINTS } from "./config";

/**
 * Redirects the browser to GitHub for OAuth login.
 * Call this in a browser context only.
 */
export function redirectToGithubLogin(): void {
  window.location.href = ENDPOINTS.LOGIN_GITHUB;
}

/**
 * Fake login for development — bypasses OAuth and sets the token cookie.
 * @param username - The username to log in as (default: "testuser")
 */
export async function fakeLogin(username = "testuser"): Promise<void> {
  const url = new URL(ENDPOINTS.LOGIN_FAKE);
  url.searchParams.set("username", username);

  const res = await fetch(url.toString(), {
    method: "GET",
    credentials: "include",
  });

  if (!res.ok) {
    throw new Error(`Fake login failed: ${res.status} ${res.statusText}`);
  }
}
