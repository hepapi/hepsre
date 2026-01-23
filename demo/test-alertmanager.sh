#!/bin/bash

# AlertManager Webhook Test Script
# This script tests the AlertManager webhook endpoint with various scenarios

# Configuration
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
WEBHOOK_ENDPOINT="$SERVER_URL/api/v1/webhook/alertmanager"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print section headers
print_header() {
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
}

# Function to send alert and display response
send_alert() {
    local test_name=$1
    local payload=$2

    echo -e "${GREEN}Testing: $test_name${NC}"
    echo "Payload:"
    echo "$payload" | jq '.' 2>/dev/null || echo "$payload"
    echo ""
    echo "Response:"

    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        -X POST "$WEBHOOK_ENDPOINT" \
        -H "Content-Type: application/json" \
        -d "$payload")

    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    body=$(echo "$response" | sed '/HTTP_STATUS:/d')

    if [ "$http_status" -eq 200 ]; then
        echo -e "${GREEN}✓ Status: $http_status${NC}"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ Status: $http_status${NC}"
        echo "$body"
    fi

    echo ""
    echo "---"
    sleep 1
}

# Test 1: Single Alert - OOM Killed
print_header "Test 1: Single Alert - OOM Killed"

send_alert "OOM Killed Pod" '{
  "version": "4",
  "groupKey": "{}:{alertname=\"PodOOMKilled\"}",
  "truncatedAlerts": 0,
  "status": "firing",
  "receiver": "micro-sre-webhook",
  "groupLabels": {
    "alertname": "PodOOMKilled"
  },
  "commonLabels": {
    "alertname": "PodOOMKilled",
    "severity": "critical"
  },
  "commonAnnotations": {
    "summary": "Pod has been OOM killed"
  },
  "externalURL": "http://alertmanager:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "PodOOMKilled",
        "namespace": "default",
        "pod": "oom-killer-demo",
        "severity": "critical",
        "container": "app",
        "instance": "10.0.1.20:9090",
        "job": "kubernetes-pods"
      },
      "annotations": {
        "description": "Pod test-app-789xyz in namespace default was killed due to out of memory (OOM)",
        "summary": "Pod has been OOM killed",
        "runbook_url": "https://runbooks.example.com/PodOOMKilled"
      },
      "startsAt": "2026-01-15T10:30:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph",
      "fingerprint": "oom123test456"
    }
  ]
}'

# Test 2: Single Alert - CrashLoopBackOff
print_header "Test 2: Single Alert - CrashLoopBackOff"

send_alert "CrashLoopBackOff" '{
  "version": "4",
  "groupKey": "{}:{alertname=\"PodCrashLooping\"}",
  "status": "firing",
  "receiver": "micro-sre-webhook",
  "groupLabels": {
    "alertname": "PodCrashLooping"
  },
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "PodCrashLooping",
        "namespace": "default",
        "pod": "sidecar-injector-webhook-557c48bff4-2769j",
        "severity": "critical"
      },
      "annotations": {
        "description": "Pod sidecar-injector-webhook-557c48bff4-2769j is in CrashLoopBackOff state",
        "summary": "Pod is crash looping"
      },
      "startsAt": "2026-01-15T11:00:00Z",
      "fingerprint": "crash456loop789"
    }
  ]
}'

# Test 3: Multiple Alerts in Single Webhook
print_header "Test 3: Multiple Alerts (Batch)"

send_alert "Multiple Alerts" '{
  "version": "4",
  "groupKey": "{}:{severity=\"critical\"}",
  "status": "firing",
  "receiver": "micro-sre-webhook",
  "groupLabels": {
    "severity": "critical"
  },
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "PodNotReady",
        "namespace": "staging",
        "pod": "nginx-pod",
        "severity": "warning"
      },
      "annotations": {
        "description": "Pod nginx-pod is not ready",
        "summary": "Pod not ready"
      },
      "startsAt": "2026-01-15T11:15:00Z",
      "fingerprint": "notready123"
    },
    {
      "status": "firing",
      "labels": {
        "alertname": "HighMemoryUsage",
        "namespace": "production",
        "pod": "backend-service-456",
        "severity": "warning"
      },
      "annotations": {
        "description": "Pod backend-service-456 memory usage is above 90%",
        "summary": "High memory usage"
      },
      "startsAt": "2026-01-15T11:20:00Z",
      "fingerprint": "highmem456"
    },
    {
      "status": "firing",
      "labels": {
        "alertname": "PodPending",
        "namespace": "default",
        "pod": "worker-queue-789",
        "severity": "critical"
      },
      "annotations": {
        "description": "Pod worker-queue-789 has been pending for 10 minutes",
        "summary": "Pod stuck in pending state"
      },
      "startsAt": "2026-01-15T11:25:00Z",
      "fingerprint": "pending789"
    }
  ]
}'

# Test 4: Resolved Alert
print_header "Test 4: Resolved Alert"

send_alert "Resolved Alert" '{
  "version": "4",
  "groupKey": "{}:{alertname=\"PodNotReady\"}",
  "status": "resolved",
  "receiver": "micro-sre-webhook",
  "alerts": [
    {
      "status": "resolved",
      "labels": {
        "alertname": "PodNotReady",
        "namespace": "production",
        "pod": "fixed-service-xyz",
        "severity": "warning"
      },
      "annotations": {
        "description": "Pod is now ready",
        "summary": "Pod recovered"
      },
      "startsAt": "2026-01-15T10:00:00Z",
      "endsAt": "2026-01-15T11:30:00Z",
      "fingerprint": "resolved123"
    }
  ]
}'

# Test 5: Alert without namespace/pod (should fail gracefully)
print_header "Test 5: Invalid Alert (Missing namespace/pod)"

send_alert "Invalid Alert" '{
  "version": "4",
  "groupKey": "{}:{alertname=\"ClusterAlert\"}",
  "status": "firing",
  "receiver": "micro-sre-webhook",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "ClusterAlert",
        "severity": "info"
      },
      "annotations": {
        "description": "This is a cluster-level alert without pod info",
        "summary": "Cluster alert"
      },
      "startsAt": "2026-01-15T11:35:00Z",
      "fingerprint": "cluster123"
    }
  ]
}'

# Test 6: Custom alert with all optional fields
print_header "Test 6: Detailed Alert with All Fields"

send_alert "Detailed Alert" '{
  "version": "4",
  "groupKey": "{}:{alertname=\"PodImagePullError\"}",
  "truncatedAlerts": 0,
  "status": "firing",
  "receiver": "micro-sre-webhook",
  "groupLabels": {
    "alertname": "PodImagePullError",
    "team": "platform"
  },
  "commonLabels": {
    "alertname": "PodImagePullError",
    "severity": "critical",
    "team": "platform",
    "environment": "production"
  },
  "commonAnnotations": {
    "summary": "Pod cannot pull container image",
    "dashboard": "https://grafana.example.com/d/pods"
  },
  "externalURL": "http://alertmanager:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "PodImagePullError",
        "namespace": "production",
        "pod": "web-frontend-v2-123",
        "severity": "critical",
        "container": "frontend",
        "image": "registry.example.com/frontend:v2.0.0",
        "team": "platform",
        "environment": "production"
      },
      "annotations": {
        "description": "Pod web-frontend-v2-123 cannot pull image registry.example.com/frontend:v2.0.0. Error: manifest not found",
        "summary": "Image pull failed",
        "runbook_url": "https://runbooks.example.com/ImagePullError",
        "dashboard": "https://grafana.example.com/d/pods?var-pod=web-frontend-v2-123"
      },
      "startsAt": "2026-01-15T11:40:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph?g0.expr=kube_pod_container_status_waiting_reason%7Breason%3D%22ImagePullBackOff%22%7D",
      "fingerprint": "imagepull789xyz"
    }
  ]
}'

print_header "All Tests Completed"
echo -e "${GREEN}Test script finished!${NC}"
echo ""
echo "Summary:"
echo "- Single alerts: OOM, CrashLoop, Resolved"
echo "- Batch alerts: Multiple alerts in one webhook"
echo "- Invalid alert: Missing namespace/pod"
echo "- Detailed alert: With all optional fields"
echo ""
echo "Check the server logs for detailed analysis results."
