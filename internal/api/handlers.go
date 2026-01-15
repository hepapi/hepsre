package api

import (
	"context"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/emirozbir/micro-sre/internal/agent"
	"github.com/emirozbir/micro-sre/internal/database"
	"github.com/emirozbir/micro-sre/internal/models"
)

type Handler struct {
	agent  *agent.Agent
	logger *zap.Logger
	db     *database.DB
	tmpl   *template.Template
}

func NewHandler(agent *agent.Agent, logger *zap.Logger, db *database.DB) *Handler {
	// Parse templates with helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("internal/templates/*.html"))

	return &Handler{
		agent:  agent,
		logger: logger,
		db:     db,
		tmpl:   tmpl,
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

	// Save to database
	if _, err := h.db.SaveAnalysis(result); err != nil {
		h.logger.Error("failed to save analysis to database", zap.Error(err))
		// Don't fail the request if DB save fails
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

	// Save to database
	if _, err := h.db.SaveAnalysis(result); err != nil {
		h.logger.Error("failed to save analysis to database", zap.Error(err))
		// Don't fail the request if DB save fails
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now(),
	})
}

// ReceiveAlertManagerWebhook handles incoming AlertManager webhook payloads
func (h *Handler) ReceiveAlertManagerWebhook(c *gin.Context) {
	var webhook models.AlertManagerWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		h.logger.Error("failed to bind webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload: " + err.Error()})
		return
	}

	h.logger.Info("received alertmanager webhook",
		zap.String("receiver", webhook.Receiver),
		zap.String("status", webhook.Status),
		zap.Int("alert_count", len(webhook.Alerts)))

	// Create context with timeout for batch processing (5 minutes)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Default lookback duration (1 hour)
	lookback := 1 * time.Hour

	// Prepare result structures
	var (
		results []models.AlertAnalysisResult
		errors  []models.AlertAnalysisError
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	// Process each alert in parallel
	for _, alert := range webhook.Alerts {
		wg.Add(1)
		go func(alert models.Alert) {
			defer wg.Done()

			// Extract namespace and pod from alert labels
			namespace := alert.GetNamespace()
			podName := alert.GetPodName()
			alertName := alert.GetAlertName()
			severity := alert.GetSeverity()

			// Skip alerts without namespace or pod
			if namespace == "" || podName == "" {
				h.logger.Warn("skipping alert without namespace or pod",
					zap.String("alert_name", alertName),
					zap.String("fingerprint", alert.Fingerprint))

				mu.Lock()
				errors = append(errors, models.AlertAnalysisError{
					Fingerprint: alert.Fingerprint,
					AlertName:   alertName,
					Error:       "missing namespace or pod in alert labels",
				})
				mu.Unlock()
				return
			}

			// Create analysis request
			analysisReq := agent.AnalysisRequest{
				AlertFingerprint: alert.Fingerprint,
				Namespace:        namespace,
				PodName:          podName,
				Lookback:         lookback,
			}

			// Perform analysis
			result, err := h.agent.AnalyzeAlert(ctx, analysisReq)
			if err != nil {
				h.logger.Error("alert analysis failed",
					zap.String("alert_name", alertName),
					zap.String("namespace", namespace),
					zap.String("pod", podName),
					zap.Error(err))

				mu.Lock()
				errors = append(errors, models.AlertAnalysisError{
					Fingerprint: alert.Fingerprint,
					AlertName:   alertName,
					Error:       err.Error(),
				})
				mu.Unlock()
				return
			}

			// Save to database
			if _, err := h.db.SaveAnalysis(result); err != nil {
				h.logger.Error("failed to save analysis to database",
					zap.String("alert_name", alertName),
					zap.Error(err))
				// Don't fail the analysis if DB save fails
			}

			// Add successful result
			mu.Lock()
			results = append(results, models.AlertAnalysisResult{
				Fingerprint:   alert.Fingerprint,
				AlertName:     alertName,
				Namespace:     namespace,
				Pod:           podName,
				Severity:      severity,
				Status:        alert.Status,
				Analysis:      &result.Analysis,
				CollectedData: &result.CollectedData,
			})
			mu.Unlock()

			h.logger.Info("alert analysis completed",
				zap.String("alert_name", alertName),
				zap.String("namespace", namespace),
				zap.String("pod", podName))
		}(alert)
	}

	// Wait for all analyses to complete
	wg.Wait()

	// Build response
	response := models.WebhookAnalysisResponse{
		Received: len(webhook.Alerts),
		Analyzed: len(results),
		Failed:   len(errors),
		Results:  results,
		Errors:   errors,
	}

	h.logger.Info("webhook processing completed",
		zap.Int("received", response.Received),
		zap.Int("analyzed", response.Analyzed),
		zap.Int("failed", response.Failed))

	// Return 200 even with partial failures
	c.JSON(http.StatusOK, response)
}

// ListAnalyses displays the HTML page with all analyses
func (h *Handler) ListAnalyses(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 20
	offset := (page - 1) * perPage

	// Get analyses from database
	analyses, err := h.db.ListAnalyses(perPage, offset)
	if err != nil {
		h.logger.Error("failed to list analyses", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load analyses")
		return
	}

	// Get total count
	total, err := h.db.CountAnalyses()
	if err != nil {
		h.logger.Error("failed to count analyses", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to count analyses")
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	// Render template
	data := gin.H{
		"Analyses":   analyses,
		"Total":      total,
		"Page":       page,
		"TotalPages": totalPages,
	}

	if err := h.tmpl.ExecuteTemplate(c.Writer, "list.html", data); err != nil {
		h.logger.Error("failed to render template", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to render page")
	}
}

// GetAnalysis displays the HTML page for a single analysis
func (h *Handler) GetAnalysis(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid analysis ID")
		return
	}

	analysis, err := h.db.GetAnalysis(id)
	if err != nil {
		h.logger.Error("failed to get analysis", zap.Int64("id", id), zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load analysis")
		return
	}

	if analysis == nil {
		c.String(http.StatusNotFound, "Analysis not found")
		return
	}

	// Render template
	if err := h.tmpl.ExecuteTemplate(c.Writer, "detail.html", analysis); err != nil {
		h.logger.Error("failed to render template", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to render page")
	}
}
