package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func oauthConfig() *oauth2.Config {
	clientID := os.Getenv("GITHUB_KEY")
	if clientID == "" {
		clientID = viper.GetString("Core.oauth_client_id")
	}
	clientSecret := os.Getenv("GITHUB_AUTH")
	if clientSecret == "" {
		clientSecret = viper.GetString("Core.oauth_client_secret")
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  viper.GetString("Core.oauth_redirect_url"),
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}
}

func randomState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "state"
	}
	return hex.EncodeToString(b)
}

func (s *Server) githubLogin(c *gin.Context) {
	cfg := oauthConfig()
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		c.JSON(http.StatusServiceUnavailable, apiError{
			Error: "GitHub login is not configured on this server",
			Code:  "oauth_unavailable",
		})
		return
	}
	state := randomState()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   viper.GetBool("Core.cookie_secure"),
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusTemporaryRedirect, cfg.AuthCodeURL(state))
}

func (s *Server) githubCallback(c *gin.Context) {
	frontend := strings.TrimRight(viper.GetString("Core.frontend_url"), "/")
	stateCookie, _ := c.Cookie("oauth_state")
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   viper.GetBool("Core.cookie_secure"),
		SameSite: http.SameSiteLaxMode,
	})
	state := c.Query("state")
	if stateCookie == "" || state == "" ||
		subtle.ConstantTimeCompare([]byte(stateCookie), []byte(state)) != 1 {
		c.Redirect(http.StatusSeeOther, frontend+"/login?error=invalid_oauth_state")
		return
	}
	cfg := oauthConfig()
	token, err := cfg.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, frontend+"/login?error=oauth_exchange_failed")
		return
	}
	client := cfg.Client(context.Background(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.Redirect(http.StatusSeeOther, frontend+"/login?error=github_unavailable")
		return
	}
	defer resp.Body.Close()
	var gh struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil || gh.Login == "" {
		c.Redirect(http.StatusSeeOther, frontend+"/login?error=github_profile_failed")
		return
	}
	user, err := s.services.LoginUser(c, gh.Login, gh.AvatarURL)
	if err != nil {
		s.logger.Errorw("github login", "error", err)
		c.Redirect(http.StatusSeeOther, frontend+"/login?error=login_failed")
		return
	}
	jwtToken, err := auth.GenerateJWT(strconv.FormatUint(uint64(user.ID), 10))
	if err != nil {
		c.Redirect(http.StatusSeeOther, frontend+"/login?error=token_failed")
		return
	}
	auth.SetAuthCookie(c.Writer, jwtToken)
	c.Redirect(http.StatusSeeOther, frontend+"/play")
}

type devLoginRequest struct {
	Username string `json:"username"`
}

func (s *Server) devLogin(c *gin.Context) {
	if !viper.GetBool("Core.dev_login") {
		c.JSON(http.StatusForbidden, apiError{Error: "developer login is disabled", Code: "forbidden"})
		return
	}
	var req devLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Error: "username is required", Code: "bad_request"})
		return
	}
	s.finishDevLogin(c, req.Username)
}

func (s *Server) devLoginGET(c *gin.Context) {
	if !viper.GetBool("Core.dev_login") {
		c.JSON(http.StatusForbidden, apiError{Error: "developer login is disabled", Code: "forbidden"})
		return
	}
	username := c.Query("username")
	if username == "" {
		username = "testuser"
	}
	user, token, err := s.loginDev(c, username)
	if err != nil {
		s.fail(c, err)
		return
	}
	auth.SetAuthCookie(c.Writer, token)
	if c.Query("redirect") == "0" || strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{"user": s.services.UserView(user)})
		return
	}
	frontend := strings.TrimRight(viper.GetString("Core.frontend_url"), "/")
	c.Redirect(http.StatusSeeOther, frontend+"/play")
}

func (s *Server) finishDevLogin(c *gin.Context, username string) {
	user, token, err := s.loginDev(c, username)
	if err != nil {
		s.fail(c, err)
		return
	}
	auth.SetAuthCookie(c.Writer, token)
	c.JSON(http.StatusOK, gin.H{"user": s.services.UserView(user)})
}

func (s *Server) loginDev(c *gin.Context, username string) (entity.User, string, error) {
	username = strings.TrimSpace(username)
	// Dev accounts use the UI's local initial placeholder. GitHub's old
	// identicon endpoint now returns 404, and OAuth logins pass a real avatar.
	avatar := ""
	user, err := s.services.LoginUser(c, username, avatar)
	if err != nil {
		return entity.User{}, "", err
	}
	token, err := auth.GenerateJWT(strconv.FormatUint(uint64(user.ID), 10))
	if err != nil {
		return entity.User{}, "", err
	}
	return user, token, nil
}

func (s *Server) logout(c *gin.Context) {
	auth.ClearAuthCookie(c.Writer)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) whoami(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apiError{Error: "authentication required", Code: "unauthorized"})
		return
	}
	user, err := s.services.GetUser(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, s.services.UserView(user))
}
