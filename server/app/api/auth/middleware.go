package auth

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks for a valid JWT.
// It is "optional" - if a token is present and valid, it sets the userID in the context.
// If not, it proceeds without setting a user, allowing for guest access.
func AuthMiddleware() gin.HandlerFunc {
	fmt.Println("Here")

	return func(c *gin.Context) {
		var tokenString string

		// 1. Try to get the token from the cookie first
		cookie, err := c.Cookie("token")
		if err == nil {
			tokenString = cookie
		}

		// 2. If no cookie, fall back to the Authorization header
		if tokenString == "" {
			bearerToken := c.GetHeader("Authorization")
			// The header should be in the format "Bearer <token>"
			if after, ok := strings.CutPrefix(bearerToken, "Bearer "); ok {
				tokenString = after
			}
		}

		// 3. If no token was found in either place, proceed as a guest
		if tokenString == "" {
			c.Next()
			return
		}

		// 4. Validate the token using your dedicated function
		userID, err := ValidateJWT(tokenString)
		if err == nil {
			// Token is valid. Set the userID in the context for other handlers to use.
			c.Set("userID", userID)
		}
		// If err is not nil, the token is invalid (expired, bad signature, etc.).
		// We do nothing and treat them as a guest user.

		// 5. Always proceed to the next handler
		c.Next()
	}
}

func IsAuth(c *gin.Context) bool {
	_, exists := c.Get("userID")
	return exists
}
func GetUser(c *gin.Context) (string, error) {
	userid := c.GetString("UserID")
	if userid == "" {
		return "", fmt.Errorf("No Active User Logged in")
	}
	return userid, nil
}
