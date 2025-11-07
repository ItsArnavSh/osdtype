package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GitHubUser represents the minimal response structure
type GitHubUser struct {
	AvatarURL string `json:"avatar_url"`
}

// GetGitHubAvatar fetches a GitHub user's avatar URL from their username.
func GetGitHubAvatar(ctx context.Context, username string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/users/%s", username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Go-GitHub-Client") // GitHub requires a User-Agent header

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return user.AvatarURL, nil
}
