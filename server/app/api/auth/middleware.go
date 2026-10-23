package auth

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func tokenFromRequest(c *gin.Context) string {
	if after, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok {
		return after
	}
	if cookie, err := c.Cookie("token"); err == nil && cookie != "" {
		return cookie
	}
	// Browsers cannot set an Authorization header on a WebSocket handshake.
	// The SPA passes the same JWT in the query string for that upgrade only,
	// so ordinary API requests never accept tokens in URLs.
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		if token := c.Query("token"); token != "" {
			return token
		}
	}
	return ""
}

// OptionalAuth identifies a user when a valid token is present, but never
// rejects the request. Public endpoints use it to personalise responses.
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if token := tokenFromRequest(c); token != "" {
			if sub, err := ValidateJWT(token); err == nil {
				if id, err := strconv.ParseUint(sub, 10, 32); err == nil {
					c.Set("userID", strconv.FormatUint(id, 10))
				}
			}
		}
		c.Next()
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAuth(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required", "code": "unauthorized"})
			return
		}
		c.Next()
	}
}

func IsAuth(c *gin.Context) bool {
	_, exists := c.Get("userID")
	return exists
}

func GetUserID(c *gin.Context) (uint32, error) {
	value, exists := c.Get("userID")
	if !exists {
		return 0, fmt.Errorf("no active user")
	}
	switch v := value.(type) {
	case string:
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid user id")
		}
		return uint32(id), nil
	case uint32:
		return v, nil
	case uint:
		return uint32(v), nil
	}
	return 0, fmt.Errorf("invalid user id")
}

func SetUserID(c *gin.Context, uid uint32) {
	c.Set("userID", strconv.FormatUint(uint64(uid), 10))
}
