package auth

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func AdminMiddleware() gin.HandlerFunc {
	adminKey := os.Getenv("ADMINKEY")

	return func(c *gin.Context) {
		password := c.GetHeader("X-Admin-Key")
		if password == "" {
			password = c.Query("admin_key")
		}

		if password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing admin key"})
			c.Abort()
			return
		}
		if password != adminKey {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid admin key"})
			c.Abort()
			return
		}

		// Passed check → grant access
		c.Next()
	}
}
