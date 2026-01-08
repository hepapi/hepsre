package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	// Try to extract JSON from the response
	analysis := a.extractAndParseJSON(analysisText)

	// Build the complete result
	result := &models.AnalysisResult{
		Alert: models.AlertSummary{
			Name:      "PodIncident",
			Namespace: req.Namespace,
			Pod:       req.PodName,
			StartedAt: time.Now().Add(-req.Lookback),
		},
		Analysis: analysis,
		CollectedData: models.CollectedData{
			LogLines:    len(podInfo.Logs),
			EventsCount: len(podInfo.Events),
			TimeRange:   req.Lookback.String(),
		},
	}

	// If parsing failed, include the raw text in reasoning
	if analysis.RootCause == "" && analysis.Reasoning == "" {
		result.Analysis.Reasoning = analysisText
		result.Analysis.RootCause = "Unable to parse LLM response"
		result.Analysis.Confidence = "unknown"
	}

	return result
}

func (a *Agent) extractAndParseJSON(text string) models.Analysis {
	// Try to find JSON in the text
	jsonStr := a.extractJSON(text)
	if jsonStr == "" {
		a.logger.Warn("no JSON found in LLM response, using raw text")
		return models.Analysis{
			Reasoning: text,
		}
	}

	// Parse the JSON
	var response struct {
		RootCause   string `json:"root_cause"`
		Confidence  string `json:"confidence"`
		Reasoning   string `json:"reasoning"`
		Timeline    []struct {
			Timestamp string `json:"timestamp"`
			Event     string `json:"event"`
			Details   string `json:"details"`
		} `json:"timeline"`
		Evidence struct {
			Logs []struct {
				Timestamp string `json:"timestamp"`
				Line      string `json:"line"`
				Container string `json:"container,omitempty"`
			} `json:"logs"`
			Events []struct {
				Type      string `json:"type"`
				Reason    string `json:"reason"`
				Message   string `json:"message"`
				Timestamp string `json:"timestamp,omitempty"`
			} `json:"events"`
		} `json:"evidence"`
		Recommendations []struct {
			Priority string `json:"priority"`
			Action   string `json:"action"`
			Details  string `json:"details,omitempty"`
			Command  string `json:"command,omitempty"`
		} `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		a.logger.Warn("failed to parse JSON from LLM response",
			zap.Error(err),
			zap.String("json", jsonStr[:min(200, len(jsonStr))]),
		)
		return models.Analysis{
			Reasoning: text,
		}
	}

	// Convert to models.Analysis
	analysis := models.Analysis{
		RootCause:       response.RootCause,
		Confidence:      response.Confidence,
		Reasoning:       response.Reasoning,
		Timeline:        make([]models.TimelineEvent, 0),
		Evidence:        models.Evidence{Logs: []models.LogEntry{}, Events: []models.EventEntry{}},
		Recommendations: make([]models.Recommendation, 0),
	}

	// Parse timeline
	for _, t := range response.Timeline {
		timestamp := a.parseTimestamp(t.Timestamp)
		analysis.Timeline = append(analysis.Timeline, models.TimelineEvent{
			Timestamp: timestamp,
			Event:     t.Event,
			Details:   t.Details,
		})
	}

	// Parse evidence logs
	for _, l := range response.Evidence.Logs {
		timestamp := a.parseTimestamp(l.Timestamp)
		analysis.Evidence.Logs = append(analysis.Evidence.Logs, models.LogEntry{
			Timestamp: timestamp,
			Line:      l.Line,
			Container: l.Container,
		})
	}

	// Parse evidence events
	for _, e := range response.Evidence.Events {
		timestamp := a.parseTimestamp(e.Timestamp)
		analysis.Evidence.Events = append(analysis.Evidence.Events, models.EventEntry{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Timestamp: timestamp,
		})
	}

	// Parse recommendations
	for _, r := range response.Recommendations {
		analysis.Recommendations = append(analysis.Recommendations, models.Recommendation{
			Priority: r.Priority,
			Action:   r.Action,
			Details:  r.Details,
			Command:  r.Command,
		})
	}

	return analysis
}

func (a *Agent) extractJSON(text string) string {
	// Try to find JSON object in the text
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		return ""
	}

	// Find the matching closing brace
	braceCount := 0
	inString := false
	escaped := false

	for i := startIdx; i < len(text); i++ {
		char := text[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--
			if braceCount == 0 {
				return text[startIdx : i+1]
			}
		}
	}

	return ""
}

func (a *Agent) parseTimestamp(ts string) time.Time {
	// Try multiple timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, ts); err == nil {
			return t
		}
	}

	// If we can't parse it, return current time
	return time.Now()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
