# AlertManager Webhook Integration

This document explains how to integrate Micro-SRE with Prometheus AlertManager to automatically analyze alerts as they fire.

## Overview

The AlertManager webhook endpoint (`/api/v1/webhook/alertmanager`) receives alert notifications from AlertManager and automatically performs LLM-powered root cause analysis for each alert.

## Features

- **Standard AlertManager Format**: Accepts the official AlertManager webhook payload format
- **Batch Processing**: Processes multiple alerts in parallel for efficiency
- **Full Pod Analysis**: For each alert, extracts namespace/pod information and performs comprehensive analysis including:
  - Pod status and configuration
  - Recent logs (default 1-hour lookback)
  - Kubernetes events
  - LLM-powered root cause analysis
- **Partial Success Handling**: Returns results even if some alerts fail to analyze
- **Structured Responses**: Provides detailed analysis with confidence levels, timelines, evidence, and recommendations

## AlertManager Configuration

Add the following to your `alertmanager.yml` configuration file:

```yaml
route:
  receiver: 'default'
  routes:
    - match:
        severity: critical|warning
      receiver: 'micro-sre-webhook'

receivers:
  - name: 'default'
    # Your default receiver configuration

  - name: 'micro-sre-webhook'
    webhook_configs:
      - url: 'http://micro-sre-server:8080/api/v1/webhook/alertmanager'
        send_resolved: false
        max_alerts: 10
```

### Configuration Options

- **url**: The Micro-SRE webhook endpoint URL
- **send_resolved**: Set to `false` to only receive firing alerts (recommended)
- **max_alerts**: Maximum number of alerts to batch per webhook call (default: 0 = unlimited)

### Docker Compose Example

```yaml
version: '3.8'
services:
  micro-sre:
    image: micro-sre:latest
    ports:
      - "8080:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    volumes:
      - ~/.kube/config:/root/.kube/config:ro

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
```

## Request Format

The webhook accepts the standard AlertManager webhook format:

```json
{
  "version": "4",
  "groupKey": "{}:{alertname=\"AlertName\"}",
  "status": "firing",
  "receiver": "micro-sre-webhook",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "PodNotReady",
        "namespace": "default",
        "pod": "nginx-pod",
        "severity": "critical"
      },
      "annotations": {
        "description": "Pod is in not ready",
        "summary": "Pod continuously failing"
      },
      "startsAt": "2026-01-09T10:00:00Z",
      "fingerprint": "asdasdasad"
    }
  ]
}
```

### Required Alert Labels

For the webhook to analyze an alert, it **must** contain:
- `namespace`: The Kubernetes namespace
- `pod` or `pod_name`: The pod name

Alerts without these labels will be skipped and reported in the `errors` array.

## Response Format

```json
{
  "received": 3,
  "analyzed": 2,
  "failed": 1,
  "results": [
    {
      "fingerprint": "abc123",
      "alert_name": "PodCrashLooping",
      "namespace": "production",
      "pod": "api-server-xyz",
      "severity": "critical",
      "status": "firing",
      "analysis": {
        "root_cause": "Container failed due to...",
        "confidence": "high",
        "reasoning": "Based on the logs and events...",
        "timeline": [...],
        "evidence": {...},
        "recommendations": [...]
      },
      "collected_data": {
        "logs_lines": 150,
        "events_count": 8,
        "time_range": "1h"
      }
    }
  ],
  "errors": [
    {
      "fingerprint": "def456",
      "alert_name": "NetworkAlert",
      "error": "missing namespace or pod in alert labels"
    }
  ]
}
```

### Response Fields

- **received**: Total number of alerts in the webhook
- **analyzed**: Number of successfully analyzed alerts
- **failed**: Number of alerts that failed analysis
- **results**: Array of successful analysis results
- **errors**: Array of alerts that could not be analyzed (with reasons)

## Testing

### Test with Single Alert

```bash
curl -X POST http://localhost:8080/api/v1/webhook/alertmanager \
  -H "Content-Type: application/json" \
  -d @examples/alerts/alertmanager-webhook.json
```

### Test with Multiple Alerts

```bash
curl -X POST http://localhost:8080/api/v1/webhook/alertmanager \
  -H "Content-Type: application/json" \
  -d @examples/alerts/alertmanager-webhook.json
```

### Test with Invalid Payload

```bash
curl -X POST http://localhost:8080/api/v1/webhook/alertmanager \
  -H "Content-Type: application/json" \
  -d '{"invalid": "payload"}'
```

Expected response: `400 Bad Request` with error message

## Troubleshooting

### Alerts Not Being Analyzed

**Issue**: Webhook receives alerts but they appear in the `errors` array

**Solution**: Check that alerts contain both `namespace` and `pod` labels:

```yaml
# In your Prometheus rules
groups:
  - name: pod_alerts
    rules:
      - alert: PodCrashLooping
        expr: rate(kube_pod_container_status_restarts_total[5m]) > 0
        labels:
          severity: critical
          namespace: "{{ $labels.namespace }}"  # Required
          pod: "{{ $labels.pod }}"              # Required
        annotations:
          description: "Pod {{ $labels.pod }} is crash looping"
```

### Webhook Timeout

**Issue**: AlertManager webhook times out

**Solution**: The webhook has a 5-minute timeout for batch processing. If you have many alerts:
- Reduce `max_alerts` in AlertManager configuration
- Check Kubernetes API performance
- Verify LLM API is responsive

### Authentication Errors

**Issue**: `401 Unauthorized` or LLM analysis fails

**Solution**: Ensure proper API keys are configured:

```bash
# For Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# For OpenAI
export OPENAI_API_KEY="sk-..."
```

Check `config/config.yaml`:

```yaml
llm:
  provider: "anthropic"  # or "openai"
  api_key: "${ANTHROPIC_API_KEY}"
```

### Missing Kubernetes Access

**Issue**: Cannot fetch pod information

**Solution**: Ensure Micro-SRE has proper Kubernetes access:

**In-cluster (recommended):**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: micro-sre
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: micro-sre-reader
rules:
  - apiGroups: [""]
    resources: ["pods", "pods/log", "events"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: micro-sre-reader-binding
subjects:
  - kind: ServiceAccount
    name: micro-sre
roleRef:
  kind: ClusterRole
  name: micro-sre-reader
  apiGroup: rbac.authorization.k8s.io
```

**Out-of-cluster:**
Mount kubeconfig with read access:
```bash
docker run -v ~/.kube/config:/root/.kube/config:ro micro-sre:latest
```

## Performance Considerations

- **Parallel Processing**: Alerts are processed in parallel using goroutines
- **Lookback Window**: Default 1-hour log lookback (configurable in `config.yaml`)
- **Batch Size**: Consider setting `max_alerts` in AlertManager if you have high alert volumes
- **Context Timeout**: 5-minute timeout for entire batch processing
- **Response Time**: Typically 5-30 seconds per alert (depends on LLM and Kubernetes API latency)

## Example Use Cases

### Critical Alert Auto-Analysis

Configure AlertManager to send only critical alerts for immediate LLM analysis:

```yaml
route:
  routes:
    - match:
        severity: critical
      receiver: 'micro-sre-webhook'
```

### Namespace-Specific Routing

Route alerts from specific namespaces:

```yaml
route:
  routes:
    - match:
        namespace: production
      receiver: 'micro-sre-webhook'
```

### Alert Aggregation

Use AlertManager's grouping to batch related alerts:

```yaml
route:
  group_by: ['alertname', 'namespace']
  group_wait: 30s
  group_interval: 5m
  receiver: 'micro-sre-webhook'
```

## Integration with Other Tools

### Slack Notifications

Chain webhook with Slack receiver for enriched notifications:

```yaml
receivers:
  - name: 'micro-sre-chain'
    webhook_configs:
      - url: 'http://micro-sre:8080/api/v1/webhook/alertmanager'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#incidents'
        title: 'Alert Analysis Available'
        text: 'Check Micro-SRE for detailed analysis'
```

### PagerDuty Integration

Use Micro-SRE analysis to enrich PagerDuty incidents by processing the webhook response.

## API Rate Limits

Be aware of rate limits:
- **Anthropic API**: Check your plan's rate limits
- **OpenAI API**: Check your plan's rate limits
- **Kubernetes API**: Typically no limits for read operations

## Security Best Practices

1. **Network Security**: Deploy Micro-SRE in the same network as AlertManager
2. **API Keys**: Use secrets management (Kubernetes Secrets, Vault, etc.)
3. **RBAC**: Grant minimal Kubernetes permissions (read-only on pods, logs, events)
4. **Authentication**: Consider adding authentication to the webhook endpoint for production use
5. **TLS**: Use HTTPS in production deployments

## Support

For issues or questions:
- GitHub Issues: [https://github.com/emirozbir/micro-sre/issues](https://github.com/emirozbir/micro-sre/issues)
- Documentation: Check the main README.md for general configuration
