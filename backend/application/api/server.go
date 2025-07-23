package api

import (
	"context"

	"github.com/gin-gonic/gin"
)

func StartServer(ctx context.Context) {
	router := gin.Default()
	router.GET("ws", wsHandler)
	router.Run(":8080")
}
