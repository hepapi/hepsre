package models

import "time"

type AnalysisResult struct {
	Alert          AlertSummary    `json:"alert"`
	Analysis       Analysis        `json:"analysis"`
	CollectedData  CollectedData   `json:"collected_data"`
}

type AlertSummary struct {
	Name      string    `json:"name"`
	Severity  string    `json:"severity"`
	Namespace string    `json:"namespace"`
	Pod       string    `json:"pod"`
	StartedAt time.Time `json:"started_at"`
}

type Analysis struct {
	RootCause       string           `json:"root_cause"`
	Confidence      string           `json:"confidence"`
	Reasoning       string           `json:"reasoning"`
	Timeline        []TimelineEvent  `json:"timeline"`
	Evidence        Evidence         `json:"evidence"`
	Recommendations []Recommendation `json:"recommendations"`
}

type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Details   string    `json:"details"`
}

type Evidence struct {
	Logs      []LogEntry   `json:"logs"`
	Events    []EventEntry `json:"events"`
	PodConfig interface{}  `json:"pod_config,omitempty"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Line      string    `json:"line"`
	Container string    `json:"container,omitempty"`
}

type EventEntry struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type Recommendation struct {
	Priority string `json:"priority"`
	Action   string `json:"action"`
	Details  string `json:"details,omitempty"`
	Command  string `json:"command,omitempty"`
}

type CollectedData struct {
	LogLines    int    `json:"logs_lines"`
	EventsCount int    `json:"events_count"`
	TimeRange   string `json:"time_range"`
}
