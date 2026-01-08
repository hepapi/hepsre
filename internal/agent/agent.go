package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/emirozbir/micro-sre/internal/collectors"
	"github.com/emirozbir/micro-sre/internal/config"
	"github.com/emirozbir/micro-sre/internal/llm"
	"github.com/emirozbir/micro-sre/internal/models"
	corev1 "k8s.io/api/core/v1"
)

type Agent struct {
	k8sCollector *collectors.KubernetesCollector
	amCollector  *collectors.AlertManagerCollector
	llmClient    llm.Client
	config       *config.Config
	logger       *zap.Logger
}

func NewAgent(cfg *config.Config, logger *zap.Logger) (*Agent, error) {
	k8sCollector, err := collectors.NewKubernetesCollector(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s collector: %w", err)
	}

	amCollector := collectors.NewAlertManagerCollector(cfg)

	llmClient, err := llm.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &Agent{
		k8sCollector: k8sCollector,
		amCollector:  amCollector,
		llmClient:    llmClient,
		config:       cfg,
		logger:       logger,
	}, nil
}

type AnalysisRequest struct {
	AlertFingerprint string
	Namespace        string
	PodName          string
	Lookback         time.Duration
}

func (a *Agent) AnalyzeAlert(ctx context.Context, req AnalysisRequest) (*models.AnalysisResult, error) {
	a.logger.Info("starting alert analysis",
		zap.String("namespace", req.Namespace),
		zap.String("pod", req.PodName),
		zap.Duration("lookback", req.Lookback),
	)

	// Collect data in parallel
	var (
		podInfo *collectors.PodInfo
		err     error
		wg      sync.WaitGroup
		mu      sync.Mutex
		errors  []error
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		pi, e := a.k8sCollector.GetPodInfo(ctx, req.Namespace, req.PodName, req.Lookback)
		mu.Lock()
		podInfo = pi
		if e != nil {
			errors = append(errors, e)
		}
		mu.Unlock()
	}()

	wg.Wait()

	if len(errors) > 0 {
		a.logger.Error("failed to collect data", zap.Errors("errors", errors))
		return nil, fmt.Errorf("failed to collect data: %v", errors)
	}

	// Build context for LLM
	prompt := a.buildAnalysisPrompt(req, podInfo)

	// Analyze with LLM
	a.logger.Info("sending data to LLM for analysis")
	analysisText, err := a.llmClient.Analyze(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse the response and structure it
	result := a.parseAnalysisResponse(req, podInfo, analysisText)

	a.logger.Info("analysis completed",
		zap.String("root_cause", result.Analysis.RootCause),
		zap.String("confidence", result.Analysis.Confidence),
	)

	return result, nil
}

func (a *Agent) buildAnalysisPrompt(req AnalysisRequest, podInfo *collectors.PodInfo) string {
	return fmt.Sprintf(`You are an expert SRE analyzing a Kubernetes incident. Analyze the following data and provide a detailed root cause analysis.

ALERT CONTEXT:
- Namespace: %s
- Pod: %s
- Time Range: Last %s

POD STATUS:
Phase: %s
Conditions: %v
Container Statuses: %v

POD CONFIGURATION:
Resources: %v
Image: %s

RECENT EVENTS:
%s

POD LOGS:
%s

TASK:
1. Identify the root cause of the issue
2. Provide a confidence level (high/medium/low)
3. Explain your reasoning
4. Create a timeline of key events
5. Extract relevant evidence (log lines, events)
6. Provide actionable recommendations with specific commands

Please respond in JSON format with the following structure:
{
  "root_cause": "brief description",
  "confidence": "high|medium|low",
  "reasoning": "detailed explanation",
  "timeline": [{"timestamp": "...", "event": "...", "details": "..."}],
  "evidence": {
    "logs": [{"timestamp": "...", "line": "..."}],
    "events": [{"type": "...", "reason": "...", "message": "..."}]
  },
  "recommendations": [
    {"priority": "high|medium|low", "action": "...", "details": "...", "command": "..."}
  ]
}`,
		req.Namespace,
		req.PodName,
		req.Lookback,
		podInfo.Pod.Status.Phase,
		podInfo.Pod.Status.Conditions,
		podInfo.Pod.Status.ContainerStatuses,
		podInfo.Pod.Spec.Containers[0].Resources,
		podInfo.Pod.Spec.Containers[0].Image,
		a.formatEvents(podInfo.Events),
		a.truncateLogs(podInfo.Logs, 5000),
	)
}

func (a *Agent) formatEvents(events []corev1.Event) string {
	if len(events) == 0 {
		return "No recent events found"
	}
	// Format events into readable text
	result := ""
	for i, event := range events {
		if i >= 10 { // Limit to 10 most recent events
			break
		}
		result += fmt.Sprintf("- [%s] %s: %s (reason: %s)\n",
			event.LastTimestamp.Format(time.RFC3339),
			event.Type,
			event.Message,
			event.Reason)
	}
	return result
}

func (a *Agent) truncateLogs(logs string, maxChars int) string {
	if len(logs) <= maxChars {
		return logs
	}
	return logs[len(logs)-maxChars:] + "\n... (truncated)"
}

func (a *Agent) parseAnalysisResponse(req AnalysisRequest, podInfo *collectors.PodInfo, analysisText string) *models.AnalysisResult {
	// For now, return a basic structure
	// TODO: Properly parse JSON response from LLM
	return &models.AnalysisResult{
		Alert: models.AlertSummary{
			Name:      "Alert",
			Namespace: req.Namespace,
			Pod:       req.PodName,
			StartedAt: time.Now().Add(-req.Lookback),
		},
		Analysis: models.Analysis{
			RootCause:  "Analysis in progress",
			Confidence: "medium",
			Reasoning:  analysisText,
			Timeline:   []models.TimelineEvent{},
			Evidence: models.Evidence{
				Logs:   []models.LogEntry{},
				Events: []models.EventEntry{},
			},
			Recommendations: []models.Recommendation{},
		},
		CollectedData: models.CollectedData{
			LogLines:    len(podInfo.Logs),
			EventsCount: len(podInfo.Events),
			TimeRange:   req.Lookback.String(),
		},
	}
}
