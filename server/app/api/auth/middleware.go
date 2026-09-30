package auth

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks for a valid JWT.
// It is "optional" - if a token is present and valid, it sets the userID in the context.
// If not, it proceeds without setting a user, allowing for guest access.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		cookie, err := c.Cookie("token")
		if err == nil {
			tokenString = cookie
		}

		if tokenString == "" {
			bearerToken := c.GetHeader("Authorization")
			// The header should be in the format "Bearer <token>"
			if after, ok := strings.CutPrefix(bearerToken, "Bearer "); ok {
				tokenString = after
			}
		}

		if tokenString == "" {
			c.Next()
			return
		}

		// 4. Validate the token using your dedicated function
		userID, err := ValidateJWT(tokenString)
		if err == nil {
			c.Set("userID", userID)
		}
		c.Next()
	}
}

func IsAuth(c *gin.Context) bool {
	_, exists := c.Get("userID")
	return exists
}

// GetUserID reads the user id that AuthMiddleware or SetUserID stored.
//
// The middleware puts the token subject in as a string, and SetUserID used to
// put a uint32 in. Reading with c.GetString only works for the string shape, so
// the two could never round-trip; the value is normalised to a string on the
// way in so there is exactly one representation.
func GetUserID(c *gin.Context) (uint32, error) {
	raw, exists := c.Get("userID")
	if !exists {
		return 0, errors.New("no active user logged in")
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return 0, errors.New("no active user logged in")
		}
		uid, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("malformed user id %q: %w", v, err)
		}
		return uint32(uid), nil
	case uint32:
		return v, nil
	case int:
		return uint32(v), nil
	case uint64:
		return uint32(v), nil
	default:
		return 0, errors.New("no active user logged in")
	}
}

func SetUserID(c *gin.Context, uid uint32) {
	c.Set("userID", strconv.FormatUint(uint64(uid), 10))
}
