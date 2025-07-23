package auth

import (
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var GithubOathConfig = &oauth2.Config{
	ClientID:     os.Getenv("GITHUB_KEY"),
	ClientSecret: os.Getenv("GITHUB_AUTH"),
	RedirectURL:  "http://localhost:8080/auth/github/callback",
	Scopes:       []string{"user:email"},
	Endpoint:     github.Endpoint,
}
