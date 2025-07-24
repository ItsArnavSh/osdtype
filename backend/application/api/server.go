package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func StartServer(ctx context.Context, log *zap.Logger) {
	r := gin.Default()

	var githubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_KEY"),
		ClientSecret: os.Getenv("GITHUB_AUTH"),
		RedirectURL:  "http://localhost:8080/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	if githubOauthConfig.ClientID == "" {
		log.Error("GithubKey Not Set")
		return
	}
	if githubOauthConfig.ClientSecret == "" {
		log.Error("GithubAuth not Set")
	}
	// Step 1: Redirect user to GitHub login
	r.GET("/login/github", func(c *gin.Context) {
		url := githubOauthConfig.AuthCodeURL("randomstate")
		c.Redirect(http.StatusTemporaryRedirect, url)
	})

	// Step 2: GitHub redirects back here
	r.GET("/auth/github/callback", func(c *gin.Context) {
		code := c.Query("code")
		token, err := githubOauthConfig.Exchange(context.Background(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
			return
		}

		// Step 3: Get user info
		client := githubOauthConfig.Client(context.Background(), token)
		resp, err := client.Get("https://api.github.com/user")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		defer resp.Body.Close()

		var user map[string]any
		json.NewDecoder(resp.Body).Decode(&user)
		c.JSON(http.StatusOK, user)
	})
	r.GET("ws", wsHandler)
	r.Run(":8080")
}
