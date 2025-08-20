package api

import (
	"encoding/json"
	langauge "osdtype/application/services/language"

	"github.com/gin-gonic/gin"
)

// All the public routes inside the app
func (s *Server) ping(g *gin.Context) {

	g.JSON(200, gin.H{"reply": "pong"})
}

func (s *Server) get_snippet(g *gin.Context) {
	lang := g.Query("lang")
	if lang == "" {
		g.JSON(404, gin.H{"error": "Language Param not found"})
	}
	snippet, err := langauge.GetSnippet(g.Request.Context(), s.essen, lang)
	if err != nil {
		g.JSON(500, gin.H{"error": err.Error()})
	}
	snip, err := json.Marshal(snippet)
	if err != nil {
		g.JSON(500, gin.H{"error": err.Error()})
	}
	g.JSON(200, snip)
}
