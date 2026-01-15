package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(handler *Handler) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/healthzzz", handler.Health)
	r.GET("/analyses", handler.ListAnalyses)
	r.GET("/analyses/:id", handler.GetAnalysis)

	// API v1
	v1 := r.Group("/api/v1")
	{
		v1.POST("/analyze/alert", handler.AnalyzeAlert)
		v1.POST("/analyze/pod", handler.AnalyzePod)
		v1.POST("/webhook/alertmanager", handler.ReceiveAlertManagerWebhook)
	}

	return r
}
