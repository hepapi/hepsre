package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/emirozbir/micro-sre/internal/agent"
)

type Handler struct {
	agent  *agent.Agent
	logger *zap.Logger
}

func NewHandler(agent *agent.Agent, logger *zap.Logger) *Handler {
	return &Handler{
		agent:  agent,
		logger: logger,
	}
}

type AnalyzeAlertRequest struct {
	AlertID   string `json:"alert_id"`
	Namespace string `json:"namespace" binding:"required"`
	Pod       string `json:"pod" binding:"required"`
	Lookback  string `json:"lookback"`
}

func (h *Handler) AnalyzeAlert(c *gin.Context) {
	var req AnalyzeAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lookback := 1 * time.Hour
	if req.Lookback != "" {
		var err error
		lookback, err = time.ParseDuration(req.Lookback)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lookback duration"})
			return
		}
	}

	analysisReq := agent.AnalysisRequest{
		AlertFingerprint: req.AlertID,
		Namespace:        req.Namespace,
		PodName:          req.Pod,
		Lookback:         lookback,
	}

	result, err := h.agent.AnalyzeAlert(c.Request.Context(), analysisReq)
	if err != nil {
		h.logger.Error("analysis failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type AnalyzePodRequest struct {
	Namespace string `json:"namespace" binding:"required"`
	Pod       string `json:"pod" binding:"required"`
	Lookback  string `json:"lookback"`
}

func (h *Handler) AnalyzePod(c *gin.Context) {
	var req AnalyzePodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lookback := 1 * time.Hour
	if req.Lookback != "" {
		var err error
		lookback, err = time.ParseDuration(req.Lookback)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lookback duration"})
			return
		}
	}

	analysisReq := agent.AnalysisRequest{
		Namespace: req.Namespace,
		PodName:   req.Pod,
		Lookback:  lookback,
	}

	result, err := h.agent.AnalyzeAlert(c.Request.Context(), analysisReq)
	if err != nil {
		h.logger.Error("analysis failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now(),
	})
}
