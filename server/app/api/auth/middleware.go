package auth

import (
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
func GetUserID(c *gin.Context) (uint32, error) {
	userid := c.GetString("userID")
	if userid == "" {
		return 0, fmt.Errorf("No Active User Logged in")
	}
	uid, _ := strconv.ParseUint(userid, 10, 32)

	return uint32(uid), nil
}
func SetUserID(c *gin.Context, uid uint32) {
	c.Set("userID", uid)
}
