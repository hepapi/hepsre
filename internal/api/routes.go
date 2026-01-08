package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(handler *Handler) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", handler.Health)

	// API v1
	v1 := r.Group("/api/v1")
	{
		v1.POST("/analyze/alert", handler.AnalyzeAlert)
		v1.POST("/analyze/pod", handler.AnalyzePod)
	}

	return r
}
