package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func (s *Server) GitHubAuth() {
	var githubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_KEY"),
		ClientSecret: os.Getenv("GITHUB_AUTH"),
		RedirectURL:  "http://localhost:8080/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	if githubOauthConfig.ClientID == "" {
		s.logger.Errorf("GithubKey Not Set")
		return
	}
	if githubOauthConfig.ClientSecret == "" {
		s.logger.Errorf("GithubAuth not Set")
	}
	// Step 1: Redirect user to GitHub login
	s.gin_engine.GET("/login/github", func(c *gin.Context) {
		url := githubOauthConfig.AuthCodeURL("randomstate")
		c.Redirect(http.StatusTemporaryRedirect, url)
	})

	s.gin_engine.GET("/auth/github/callback", func(c *gin.Context) {
		code := c.Query("code")
		token, err := githubOauthConfig.Exchange(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
			return
		}

		// Get user info from GitHub
		client := githubOauthConfig.Client(c.Request.Context(), token)
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet,
			"https://api.github.com/user", nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build user info request"})
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				s.logger.Warnw("could not close the GitHub response body", "error", closeErr)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}

		var user struct {
			ID    int64  `json:"id"`
			Login string `json:"login"`
		}
		if decodeErr := json.NewDecoder(resp.Body).Decode(&user); decodeErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
			return
		}
		if user.Login == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login field not found"})
			return
		}

		// The JWT subject has to be the numeric user id: everything downstream
		// resolves a user by id, and AuthMiddleware's GetUserID parses the
		// subject as a uint32. This used to mint the subject from the GitHub
		// login name, so every real GitHub login produced an unparseable token
		// and no handler could identify the caller.
		subject, err := s.services.LoginUser(c, entity.User{Username: user.Login})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user to db"})
			return
		}

		jwt, err := auth.GenerateJWT(strconv.FormatUint(uint64(subject), 10))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Store JWT in cookie
		c.SetCookie("token", jwt, 3600, "/", "localhost", true, true)

		c.Redirect(http.StatusSeeOther, "http://localhost:5173/")
	})
}

func (s *Server) FakeGitHubAuth() {
	s.gin_engine.GET("/login/github/fake", func(c *gin.Context) {
		testLogin := c.Query("username")
		if testLogin == "" {
			testLogin = "testuser"
		}

		userid, err := s.services.LoginUser(c, entity.User{Username: testLogin})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed To Save user to db"})
			return
		}

		// Generate JWT for the test user
		jwt, err := auth.GenerateJWT(strconv.FormatUint(uint64(userid), 10))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Set JWT token cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "token",
			Value:    jwt,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			// Secure is false here because the dev frontend is served over
			// plain http on localhost. A browser drops a Secure cookie sent
			// over http, so leaving it true made the flag meaningless locally.
			// Deployments behind TLS should set OSDTYPE_SECURE_COOKIES=1.
			Secure:      os.Getenv("OSDTYPE_SECURE_COOKIES") == "1",
			SameSite:    http.SameSiteNoneMode,
			Partitioned: true, // requires Go 1.22+
		})
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Logged in as %s with userid  %d", testLogin, userid)})
	})
}
