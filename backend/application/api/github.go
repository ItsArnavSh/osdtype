package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"osdtype/application/auth"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func GitHubAuth(log *zap.Logger, r *gin.Engine) {
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

	r.GET("/auth/github/callback", func(c *gin.Context) {
		code := c.Query("code")
		token, err := githubOauthConfig.Exchange(context.Background(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
			return
		}

		// Get user info from GitHub
		client := githubOauthConfig.Client(context.Background(), token)
		resp, err := client.Get("https://api.github.com/user")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		defer resp.Body.Close()

		fmt.Println(resp.Body)
		var user map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
			return
		}
		login, ok := user["login"].(string)
		avatar_url, ok := user[""].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login field not found"})
			return
		}
		fmt.Print(login)
		jwt, err := auth.GenerateJWT(string(login))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Store JWT in cookie
		c.SetCookie("token", jwt, 3600, "/", "localhost", false, true)

		c.SetCookie("username", login, 3600, "/", "localhost", false, true)
		c.SetCookie("avatar", avatar_url, 3600, "/", "localhost", false, true)

		c.Redirect(http.StatusSeeOther, "http://localhost:5173/")
	})

}
