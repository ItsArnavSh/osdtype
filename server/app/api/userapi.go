package api

import (
	"fmt"
	"net/http"
	"osdtyp/app/api/auth"

	"github.com/gin-gonic/gin"
)

func (s *Server) whoami(c *gin.Context) {
	// 1. Get the token from the cookie
	jwtToken, err := c.Cookie("token")
	fmt.Println("Token is: ", jwtToken)
	if err != nil {
		// If no cookie is present, the user is not logged in
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// 2. Validate the JWT (your auth.ValidateJWT function would do this)
	// For this example, we'll assume auth.ValidateJWT returns the username
	username, err := auth.ValidateJWT(jwtToken)
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// 3. Return the user's information
	c.JSON(http.StatusOK, gin.H{"login": username})
}
