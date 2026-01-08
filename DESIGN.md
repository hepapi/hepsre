# Micro-SRE: Agentic SRE Debugging Assistant

## Overview

An intelligent debugging assistant that automatically gathers, analyzes, and correlates SRE incident data from AlertManager and Kubernetes to help engineers quickly identify root causes.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       Agent Orchestrator                     │
│  (LLM-powered reasoning engine for debugging workflows)     │
└──────────────────┬──────────────────────────────────────────┘
                   │
       ┌───────────┼───────────┐
       │           │           │
┌──────▼─────┐ ┌──▼──────┐ ┌─▼────────────┐
│ AlertManager│ │   K8S   │ │  Analysis    │
│  Collector  │ │Collector│ │   Engine     │
└─────────────┘ └─────────┘ └──────────────┘
       │           │              │
       └───────────┴──────────────┘
                   │
            ┌──────▼──────┐
            │  Data Store │
            │  (In-memory)│
            └─────────────┘
```

## Components

### 1. Agent Orchestrator
The brain of the system that:
- Receives alerts from AlertManager
- Determines what data to collect based on alert context
- Orchestrates parallel data gathering from multiple sources
- Analyzes collected data using LLM reasoning
- Generates debugging insights and recommendations

### 2. AlertManager Collector
- Fetches active and recent alerts
- Parses alert labels, annotations, and metadata
- Identifies affected services, namespaces, and pods

### 3. K8S Collector
Gathers cluster information:
- **Events**: Recent events filtered by namespace/pod
- **Pod Logs**: Configurable time range, supports multiple containers
- **Pod Configuration**: Specs, resource limits, env vars, volumes
- **Pod Status**: Current state, restarts, conditions
- **Recent Changes**: Deployments, ConfigMaps, Secrets (metadata only)

### 4. Analysis Engine
- Correlates alerts with K8S events and logs
- Identifies patterns (OOMKilled, CrashLoopBackOff, network issues)
- Extracts error messages and stack traces from logs
- Generates timeline of events leading to the alert

## Agentic Approach

The "agent" uses LLM-powered reasoning to:

1. **Context-Aware Data Collection**
   - Dynamically decides what data to fetch based on alert type
   - Example: Memory alert → fetch pod memory usage, OOMKilled events, resource limits

2. **Multi-Step Reasoning**
   - Analyzes initial data
   - Identifies gaps and fetches additional information
   - Iteratively narrows down root cause

3. **Pattern Recognition**
   - Recognizes common failure patterns
   - Suggests likely causes based on historical knowledge
   - Recommends specific commands or checks

4. **Natural Language Output**
   - Explains findings in human-readable format
   - Provides actionable next steps
   - Generates debugging runbooks on-the-fly

## Data Flow

```
1. Alert Triggered → AlertManager
2. Agent receives alert webhook/polls alerts
3. Agent analyzes alert metadata
4. Agent determines collection strategy:
   - Which namespace/pods to investigate
   - Time range for logs (default: last 1h, configurable)
   - Which events to filter
5. Parallel data collection from K8S API
6. Agent analyzes collected data:
   - Correlates timestamps
   - Identifies error patterns
   - Extracts relevant log snippets
7. Agent generates report with:
   - Root cause hypothesis
   - Supporting evidence
   - Recommended actions
   - Relevant log excerpts and events
```

## Technology Stack

### Backend (Go)
- **Web Framework**: Gin or Chi (lightweight HTTP routing)
- **K8S Client**: `client-go` (official Kubernetes client)
- **AlertManager Client**: HTTP client with custom types
- **LLM Integration**:
  - Anthropic Claude API (primary choice for reasoning)
  - Or OpenAI API
- **Configuration**: Viper for config management
- **Logging**: Zap (structured logging)

### Why Go?
- Strong K8S ecosystem (`client-go`)
- Excellent concurrency for parallel data collection
- Fast performance for real-time debugging
- Easy deployment as single binary

## Configuration

```yaml
# config.yaml
alertmanager:
  url: "http://alertmanager:9093"
  poll_interval: "30s"

kubernetes:
  kubeconfig: ""  # empty for in-cluster config
  context: ""     # optional

log_collection:
  default_lookback: "1h"
  max_lookback: "24h"
  tail_lines: 1000
  include_previous: true  # include previous terminated container

event_collection:
  default_lookback: "1h"
  max_lookback: "24h"
  event_types: ["Warning", "Error"]

llm:
  provider: "anthropic"  # or "openai"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-sonnet-4-5"
  max_tokens: 4000
  temperature: 0.2

agent:
  max_parallel_fetches: 5
  analysis_timeout: "2m"
```

## API Design

### REST API Endpoints

```
POST /api/v1/analyze/alert
- Trigger analysis for a specific alert
- Body: { "alert_id": "...", "lookback": "2h" }
- Returns: Analysis report with recommendations

GET /api/v1/alerts
- List active alerts from AlertManager
- Query params: ?severity=critical&namespace=production

POST /api/v1/analyze/pod
- Analyze specific pod
- Body: { "namespace": "...", "pod": "...", "lookback": "1h" }

GET /api/v1/health
- Health check endpoint

POST /webhook/alertmanager
- Webhook endpoint for AlertManager notifications
```

### Example Analysis Response

```json
{
  "alert": {
    "name": "PodMemoryUsageHigh",
    "severity": "warning",
    "namespace": "production",
    "pod": "api-server-7d9f8c-xyz",
    "started_at": "2026-01-07T10:30:00Z"
  },
  "analysis": {
    "root_cause": "Memory leak in application code",
    "confidence": "high",
    "reasoning": "Pod memory usage has been steadily increasing over 6 hours. No memory limit set. Recent deployment introduced new caching layer without eviction policy.",
    "timeline": [
      {
        "timestamp": "2026-01-07T04:00:00Z",
        "event": "Deployment updated",
        "details": "Version v2.1.3 deployed with new Redis cache"
      },
      {
        "timestamp": "2026-01-07T10:15:00Z",
        "event": "Memory usage crossed 80%",
        "details": "RSS: 3.2GB, working set: 3.1GB"
      },
      {
        "timestamp": "2026-01-07T10:30:00Z",
        "event": "Alert fired",
        "details": "PodMemoryUsageHigh triggered"
      }
    ],
    "evidence": {
      "logs": [
        {
          "timestamp": "2026-01-07T10:25:00Z",
          "line": "Cache size: 1.2M entries, memory: 2.8GB"
        }
      ],
      "events": [
        {
          "type": "Warning",
          "reason": "MemoryPressure",
          "message": "Node experiencing memory pressure"
        }
      ],
      "pod_config": {
        "resources": {
          "requests": {"memory": "512Mi"},
          "limits": {}
        }
      }
    },
    "recommendations": [
      {
        "priority": "high",
        "action": "Set memory limit to prevent node exhaustion",
        "command": "kubectl set resources deployment/api-server --limits=memory=4Gi"
      },
      {
        "priority": "high",
        "action": "Implement cache eviction policy",
        "details": "Add TTL or LRU eviction to Redis cache configuration"
      },
      {
        "priority": "medium",
        "action": "Enable memory profiling",
        "command": "kubectl exec api-server-7d9f8c-xyz -- curl localhost:6060/debug/pprof/heap > heap.prof"
      }
    ]
  },
  "collected_data": {
    "logs_lines": 1000,
    "events_count": 12,
    "time_range": "2h"
  }
}
```

## Implementation Phases

### Phase 1: Core Data Collection (Week 1)
- Set up Go project structure
- Implement K8S client with basic queries
- Implement AlertManager client
- Create data models for alerts, events, pods, logs
- Build configuration system

### Phase 2: Agent Integration (Week 1-2)
- Integrate LLM API (Claude/OpenAI)
- Implement prompt templates for different alert types
- Build context construction (feeding data to LLM)
- Create analysis pipeline

### Phase 3: API & CLI (Week 2)
- Build REST API server
- Create CLI tool for testing
- Implement webhook handler for AlertManager
- Add logging and error handling

### Phase 4: Intelligence & Polish (Week 2-3)
- Enhance prompts with better reasoning strategies
- Add pattern recognition for common issues
- Implement caching to reduce API calls
- Add metrics and observability
- Create example alerts and test scenarios

## Project Structure

```
micro-sre/
├── cmd/
│   ├── server/          # HTTP server
│   │   └── main.go
│   └── cli/             # CLI tool for testing
│       └── main.go
├── internal/
│   ├── agent/           # Agent orchestrator
│   │   ├── agent.go
│   │   ├── analyzer.go
│   │   └── prompts.go
│   ├── collectors/      # Data collectors
│   │   ├── alertmanager.go
│   │   ├── kubernetes.go
│   │   ├── logs.go
│   │   └── events.go
│   ├── llm/            # LLM client
│   │   ├── client.go
│   │   ├── anthropic.go
│   │   └── openai.go
│   ├── models/         # Data models
│   │   ├── alert.go
│   │   ├── pod.go
│   │   └── analysis.go
│   ├── api/            # HTTP handlers
│   │   ├── handlers.go
│   │   └── routes.go
│   └── config/         # Configuration
│       └── config.go
├── pkg/                # Public packages
│   └── utils/
├── config/
│   └── config.yaml
├── examples/
│   └── alerts/         # Example alert payloads
├── go.mod
├── go.sum
├── Dockerfile
├── Makefile
└── README.md
```

## Example Agent Workflow

```
Alert: "PodCrashLooping" in namespace "payment-service"

Agent Reasoning:
1. "This is a crash loop, I should check:
   - Recent pod events for crash reasons
   - Container exit codes
   - Last 100 lines of logs before crash
   - Recent configuration changes"

2. Collects data in parallel:
   - Fetch pod status → ExitCode: 1, Restarts: 15
   - Fetch events → "Back-off restarting failed container"
   - Fetch logs → "FATAL: Unable to connect to database"
   - Fetch pod config → Check env vars for DB connection

3. Analyzes:
   - "Exit code 1 indicates application error"
   - "Log shows database connection failure"
   - "Checking if DB credentials secret exists..."
   - Fetches secret metadata → Secret exists
   - "Checking if DB service is healthy..."
   - Fetches DB pod status → DB pod is running

4. Hypothesis:
   - "Database credentials may be incorrect or
      database is not accepting connections"

5. Recommendations:
   - Verify DB connectivity: kubectl exec -it payment-service-xyz -- nc -zv postgres-svc 5432
   - Check DB credentials: kubectl get secret db-creds -o yaml
   - View DB logs: kubectl logs postgres-0
```

## Deployment

### Local Development
```bash
# Run with local kubeconfig
export ANTHROPIC_API_KEY="sk-..."
go run cmd/server/main.go

# Or with Docker
docker build -t micro-sre:latest .
docker run -p 8080:8080 \
  -v ~/.kube/config:/config \
  -e KUBECONFIG=/config \
  -e ANTHROPIC_API_KEY="sk-..." \
  micro-sre:latest
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: micro-sre
  namespace: monitoring
spec:
  replicas: 1
  template:
    spec:
      serviceAccountName: micro-sre
      containers:
      - name: micro-sre
        image: micro-sre:latest
        env:
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: llm-credentials
              key: api-key
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: micro-sre
  namespace: monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: micro-sre-reader
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "events", "configmaps", "secrets"]
  verbs: ["get", "list"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list"]
```

## Security Considerations

1. **RBAC**: Minimal read-only permissions for K8S API
2. **Secrets**: Never log or expose secret values in analysis
3. **API Key**: Store LLM API key in K8S secrets
4. **Network**: Restrict API access with network policies
5. **Audit**: Log all analysis requests and data access

## Future Enhancements

1. **Multi-cluster support**: Analyze across multiple K8S clusters
2. **Historical analysis**: Store past incidents for pattern learning
3. **Slack/PagerDuty integration**: Send analysis directly to incident channels
4. **Auto-remediation**: Execute approved fixes automatically
5. **Custom playbooks**: User-defined debugging workflows
6. **Metric correlation**: Integrate with Prometheus for metric analysis
7. **Distributed tracing**: Integrate with Jaeger/Zipkin for request tracing
