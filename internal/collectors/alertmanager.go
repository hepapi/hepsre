package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/emirozbir/micro-sre/internal/config"
	"github.com/emirozbir/micro-sre/internal/models"
)

type AlertManagerCollector struct {
	baseURL string
	client  *http.Client
}

func NewAlertManagerCollector(cfg *config.Config) *AlertManagerCollector {
	return &AlertManagerCollector{
		baseURL: cfg.AlertManager.URL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type AlertManagerResponse struct {
	Status string          `json:"status"`
	Data   []models.Alert  `json:"data"`
}

func (a *AlertManagerCollector) GetAlerts(ctx context.Context) ([]models.Alert, error) {
	url := fmt.Sprintf("%s/api/v2/alerts", a.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alerts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alertmanager returned status %d", resp.StatusCode)
	}

	var alerts []models.Alert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return nil, fmt.Errorf("failed to decode alerts: %w", err)
	}

	return alerts, nil
}

func (a *AlertManagerCollector) GetActiveAlerts(ctx context.Context) ([]models.Alert, error) {
	alerts, err := a.GetAlerts(ctx)
	if err != nil {
		return nil, err
	}

	var activeAlerts []models.Alert
	for _, alert := range alerts {
		if alert.Status == "firing" {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts, nil
}

func (a *AlertManagerCollector) GetAlertsByNamespace(ctx context.Context, namespace string) ([]models.Alert, error) {
	alerts, err := a.GetActiveAlerts(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []models.Alert
	for _, alert := range alerts {
		if alert.GetNamespace() == namespace {
			filtered = append(filtered, alert)
		}
	}

	return filtered, nil
}
