# Micro-SRE: Agentic SRE Debugging Assistant

An intelligent debugging assistant that automatically gathers, analyzes, and correlates SRE incident data from AlertManager and Kubernetes to help engineers quickly identify root causes using LLM-powered reasoning.

## Features

- **Automated Data Collection**: Fetches alerts, pod logs, events, and configurations from Kubernetes
- **LLM-Powered Analysis**: Uses Claude/GPT to analyze incidents and identify root causes
- **Timeline Generation**: Creates chronological view of events leading to incidents
- **Actionable Recommendations**: Provides specific commands and steps to resolve issues
- **REST API**: Easy integration with existing monitoring tools
- **CLI Tool**: Quick debugging from the command line

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Agent Orchestrator (LLM)                  │
└──────────────────┬──────────────────────────────────────────┘
                   │
       ┌───────────┼───────────┐
       │           │           │
┌──────▼─────┐ ┌──▼──────┐ ┌─▼────────┐
│AlertManager│ │   K8S   │ │ Analysis │
│ Collector  │ │Collector│ │  Engine  │
└────────────┘ └─────────┘ └──────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Kubernetes cluster access (kubeconfig)
- Anthropic API key (or OpenAI)
- AlertManager (optional)

### Installation

1. Clone the repository:
```bash
cd /Users/emirozbir/go/src/micro-sre
```

2. Install dependencies:
```bash
make install-deps
```

3. Set up configuration:
```bash
cp config/config.yaml config/config.local.yaml
# Edit config/config.local.yaml with your settings
```

4. Export your API key:
```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

5. Build the application:
```bash
make build
```

### Running the Server

```bash
# Run directly
make run

# Or use the binary
./bin/micro-sre-server
```

The server will start on `http://localhost:8080`

### Using the CLI

```bash
# Analyze a specific pod
./bin/micro-sre-cli -namespace production -pod api-server-xyz -lookback 2h

# Or with make
make run-cli NAMESPACE=production POD=api-server-xyz LOOKBACK=2h
```

## API Usage

### Health Check

```bash
curl http://localhost:8080/health
```

### Analyze a Pod

```bash
curl -X POST http://localhost:8080/api/v1/analyze/pod \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default",
    "pod": "oom-killer-demo",
    "lookback": "1h"
  }'
```

### Analyze an Alert

```bash
curl -X POST http://localhost:8080/api/v1/analyze/alert \
  -H "Content-Type: application/json" \
  -d '{
    "alert_id": "abc123",
    "namespace": "production",
    "pod": "api-server-xyz",
    "lookback": "1h"
  }'
```

### Example Response

```json
{
  "alert": {
    "name": "PodCrashLooping",
    "severity": "critical",
    "namespace": "production",
    "pod": "api-server-xyz",
    "started_at": "2026-01-07T10:00:00Z"
  },
  "analysis": {
    "root_cause": "Database connection failure due to incorrect credentials",
    "confidence": "high",
    "reasoning": "Pod logs show repeated 'connection refused' errors...",
    "timeline": [
      {
        "timestamp": "2026-01-07T09:55:00Z",
        "event": "Deployment updated",
        "details": "New version deployed with updated DB config"
      },
      {
        "timestamp": "2026-01-07T10:00:00Z",
        "event": "Pod started crashing",
        "details": "Exit code 1, connection error"
      }
    ],
    "evidence": {
      "logs": [
        {
          "timestamp": "2026-01-07T10:00:15Z",
          "line": "FATAL: password authentication failed for user 'app'"
        }
      ],
      "events": [
        {
          "type": "Warning",
          "reason": "BackOff",
          "message": "Back-off restarting failed container"
        }
      ]
    },
    "recommendations": [
      {
        "priority": "high",
        "action": "Verify database credentials",
        "command": "kubectl get secret db-creds -n production -o yaml"
      },
      {
        "priority": "high",
        "action": "Test database connectivity",
        "command": "kubectl exec -it api-server-xyz -- nc -zv postgres-svc 5432"
      }
    ]
  },
  "collected_data": {
    "logs_lines": 1000,
    "events_count": 12,
    "time_range": "1h"
  }
}
```

## Configuration

Edit `config/config.yaml`:

```yaml
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
  include_previous: true

llm:
  provider: "anthropic"  # or "openai"
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-sonnet-4-5"
  max_tokens: 4096
  temperature: 0.2

server:
  port: 8080
  host: "0.0.0.0"
```

## Deployment

### Docker

```bash
# Build image
make docker-build

# Run container
make docker-run
```

### Kubernetes

```bash
# Apply manifests
kubectl apply -f deploy/k8s/
```
## Development

### Project Structure

```
micro-sre/
├── cmd/
│   ├── server/          # HTTP server
│   └── cli/             # CLI tool
├── internal/
│   ├── agent/           # Agent orchestrator
│   ├── collectors/      # Data collectors (K8S, AlertManager)
│   ├── llm/            # LLM client (Anthropic, OpenAI)
│   ├── models/         # Data models
│   ├── api/            # HTTP handlers
│   └── config/         # Configuration
├── config/             # Config files
├── examples/           # Example payloads
└── DESIGN.md          # Detailed design document
```

### Running Tests

```bash
make test
```

### Code Formatting

```bash
make fmt
```

## How It Works

1. **Alert Detection**: Receives alert from AlertManager (webhook or polling)
2. **Context Gathering**: Agent determines what data to collect based on alert metadata
3. **Parallel Collection**: Fetches pod logs, events, configurations from K8S API
4. **LLM Analysis**: Sends collected data to Claude/GPT for root cause analysis
5. **Result Structuring**: Parses LLM response into structured format
6. **Delivery**: Returns analysis via API or CLI

## Agentic Approach

The "agent" uses LLM reasoning to:
- Dynamically decide what data to fetch based on alert type
- Iteratively narrow down root causes through multi-step reasoning
- Recognize common failure patterns (OOMKilled, CrashLoopBackOff, etc.)
- Generate debugging runbooks on-the-fly
- Provide actionable recommendations with specific commands

## Roadmap

- [ ] Implement proper JSON parsing from LLM responses
- [ ] Add support for OpenAI provider
- [ ] Multi-cluster support
- [ ] Historical incident storage and pattern learning
- [ ] Slack/PagerDuty integration
- [ ] Auto-remediation capabilities
- [ ] Prometheus metrics correlation
- [ ] Distributed tracing integration

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License

## Support

For detailed design documentation, see [DESIGN.md](DESIGN.md)
