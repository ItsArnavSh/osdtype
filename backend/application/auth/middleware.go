package auth

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		claims := &jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims,
			func(token *jwt.Token) (any, error) {
				return jwtKey, nil
			})
		if err == nil || token.Valid {
			c.Set("userID", claims.Subject)
		}
		//We are keeping auth optional so that non logged users can also play
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
