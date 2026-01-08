package models

import "time"

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	Status      string            `json:"status"`
	Fingerprint string            `json:"fingerprint"`
}

type AlertContext struct {
	Alert      Alert
	Namespace  string
	PodName    string
	Severity   string
	AlertName  string
	StartedAt  time.Time
}

func (a *Alert) GetNamespace() string {
	if ns, ok := a.Labels["namespace"]; ok {
		return ns
	}
	if ns, ok := a.Labels["kubernetes_namespace"]; ok {
		return ns
	}
	return ""
}

func (a *Alert) GetPodName() string {
	if pod, ok := a.Labels["pod"]; ok {
		return pod
	}
	if pod, ok := a.Labels["pod_name"]; ok {
		return pod
	}
	return ""
}

func (a *Alert) GetSeverity() string {
	if sev, ok := a.Labels["severity"]; ok {
		return sev
	}
	return "unknown"
}

func (a *Alert) GetAlertName() string {
	if name, ok := a.Labels["alertname"]; ok {
		return name
	}
	return "unknown"
}
