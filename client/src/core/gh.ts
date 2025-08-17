/**
 * Generates a GitHub avatar URL for a given username.
 * This URL redirects to the user's actual profile picture.
 *
 * @param username The GitHub username (e.g., "gaearon").
 * @param size The desired width of the avatar in pixels.
 *             GitHub will serve an image of this size or the next available larger size.
 *             If omitted, it defaults to the original upload size.
 * @returns The full URL to the user's GitHub avatar.
 */
function getGithubAvatarUrl(username: string, size?: number): string {
	if (!username) {
		return ''; // Return empty string or throw error if username is not provided
	}

	let url = `https://github.com/${username}.png`;

	if (size) {
		url += `?size=${size}`;
	}

	return url;
}

// --- Examples ---

// Get the default avatar for the user "orta"
const avatarUrlDefault = getGithubAvatarUrl('orta');
console.log(avatarUrlDefault);
// Expected output: https://github.com/orta.png

// Get a 200px version of the avatar for "vercel"
const avatarUrlSized = getGithubAvatarUrl('vercel', 200);
console.log(avatarUrlSized);
// Expected output: https://github.com/vercel.png?size=200

// Example with no username
const noUser = getGithubAvatarUrl('');
console.log(noUser);
// Expected output: ""
