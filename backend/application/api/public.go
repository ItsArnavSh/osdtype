package api

import (
	langauge "osdtype/application/services/language"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// All the public routes inside the app
func (s *Server) ping(g *gin.Context) {

	g.JSON(200, gin.H{"reply": "pong"})
}

func (s *Server) get_snippet(g *gin.Context) {
	s.essen.Logger.Info("Snippet handler hit")

	lang := g.Query("lang")
	if lang == "" {
		s.essen.Logger.Warn("Language param missing")
		g.JSON(404, gin.H{"error": "Language Param not found"})
		return
	}

	s.essen.Logger.Info("Language param found", zap.String("lang", lang))

	snippet, err := langauge.GetSnippet(g.Request.Context(), s.essen, lang)
	if err != nil {
		s.essen.Logger.Error("Failed to fetch snippet", zap.Error(err))
		g.JSON(500, gin.H{"error": err.Error()})
		return
	}

	g.JSON(200, snippet)
}
