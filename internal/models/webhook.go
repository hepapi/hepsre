package models

// AlertManagerWebhook represents the standard AlertManager webhook payload
type AlertManagerWebhook struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"` // "firing" or "resolved"
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

// WebhookAnalysisResponse represents the response for batch alert analysis
type WebhookAnalysisResponse struct {
	Received int                   `json:"received"`
	Analyzed int                   `json:"analyzed"`
	Failed   int                   `json:"failed"`
	Results  []AlertAnalysisResult `json:"results"`
	Errors   []AlertAnalysisError  `json:"errors,omitempty"`
}

// AlertAnalysisResult represents the analysis result for a single alert
type AlertAnalysisResult struct {
	Fingerprint   string         `json:"fingerprint"`
	AlertName     string         `json:"alert_name"`
	Namespace     string         `json:"namespace"`
	Pod           string         `json:"pod,omitempty"`
	Severity      string         `json:"severity"`
	Status        string         `json:"status"`
	Analysis      *Analysis      `json:"analysis"`
	CollectedData *CollectedData `json:"collected_data"`
}

// AlertAnalysisError represents an error that occurred during alert analysis
type AlertAnalysisError struct {
	Fingerprint string `json:"fingerprint"`
	AlertName   string `json:"alert_name"`
	Error       string `json:"error"`
}
