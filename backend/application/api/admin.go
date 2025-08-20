package api

import (
	"fmt"
	"net/http"
	"osdtype/application/entity"
	langauge "osdtype/application/services/language"

	"github.com/gin-gonic/gin"
)

// All the admin only functions in the code
func (s *Server) admin_ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "You are in"})
}

func (s *Server) insert_snippet(c *gin.Context) {
	var snippet entity.LangData

	// Bind JSON into struct
	if err := c.ShouldBindJSON(&snippet); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(snippet)
	// Now you can use snippet.Language and snippet.Snippet
	err := langauge.InsertSnippet(
		c.Request.Context(),
		*s.essen.Db,
		snippet.Language,
		snippet.Snippet,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "snippet inserted"})
}
