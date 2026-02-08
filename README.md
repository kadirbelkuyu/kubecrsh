# kubecrsh

[![Go Version](https://img.shields.io/github/go-mod/go-version/kadirbelkuyu/kubecrsh?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/kadirbelkuyu/kubecrsh?style=flat&logo=github)](https://github.com/kadirbelkuyu/kubecrsh/releases/latest)
[![License](https://img.shields.io/github/license/kadirbelkuyu/kubecrsh?style=flat)](LICENSE)
[![GitHub Repo](https://img.shields.io/badge/GitHub-kadirbelkuyu%2Fkubecrsh-black?style=flat&logo=github)](https://github.com/kadirbelkuyu/kubecrsh)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=kadirbelkuyu_kubecrsh&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=kadirbelkuyu_kubecrsh)

kubecrsh is a Kubernetes debugging tool that captures logs, events, and pod state right before a container restarts. It ensures you have the necessary data to investigate crashes without losing ephemeral information.

## Features

- Real-time pod crash monitoring via Kubernetes informers
- Automatic capture of container logs (current and previous)
- Kubernetes events collection from the past hour
- Environment variables and exit code preservation
- Slack and webhook notifications for instant alerts
- Interactive terminal UI for forensic analysis
- Prometheus metrics for observability
- JSON-based report storage with optional compression

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                        │
│                                                               │
│   ┌─────────────────┐                                         │
│   │    kubecrsh     │  Watches pods for crashes               │
│   │     daemon      │  Collects forensic data                 │
│   │                 │  Sends notifications                    │
│   └────────┬────────┘                                         │
│            │                                                  │
│            ▼                                                  │
│   ┌─────────────────┐     ┌─────────────────┐                 │
│   │  Slack/Webhook  │     │   PVC/Reports   │                 │
│   │   Notifications │     │     Storage     │                 │
│   └─────────────────┘     └─────────────────┘                 │
│                                                               │
└──────────────────────────────────────────────────────────────┘
            ▲
            │ kubeconfig
            │
┌───────────┴───────────┐
│    Local Machine      │
│                       │
│   kubecrsh watch      │  Real-time TUI monitoring
│   kubecrsh list       │  Browse saved reports
└───────────────────────┘
```

## Cluster Deployment

Deploy kubecrsh as a daemon in your Kubernetes cluster to continuously monitor pod crashes.

### Using Helm (Recommended)

```bash
helm install kubecrsh oci://ghcr.io/kadirbelkuyu/kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace
```

With Slack notifications:

```bash
helm install kubecrsh oci://ghcr.io/kadirbelkuyu/kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace \
  --set notifiers.slack.enabled=true \
  --set secrets.slackWebhook="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

For production with HA:

```bash
helm install kubecrsh oci://ghcr.io/kadirbelkuyu/kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace \
  -f charts/kubecrsh/values-production.yaml \
  --set secrets.slackWebhook="$SLACK_WEBHOOK"
```

Cluster-wide monitoring (all namespaces):

```bash
helm install kubecrsh oci://ghcr.io/kadirbelkuyu/kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace \
  --set rbac.clusterWide=true
```

See [charts/kubecrsh/README.md](charts/kubecrsh/README.md) for all configuration options.

### Using Raw Manifests

You can deploy kubecrsh to your cluster using the provided manifests.

```bash
kubectl apply -f manifests/namespace.yaml
kubectl apply -f manifests/rbac.yaml
kubectl apply -f manifests/configmap.yaml

cp manifests/secret.yaml.example manifests/secret.yaml
kubectl apply -f manifests/secret.yaml

kubectl apply -f manifests/service.yaml
kubectl apply -f manifests/deployment.yaml
```

## CLI Usage

The CLI provides a terminal interface to monitor crashes in real-time or browse saved reports.

### Installation

Download the latest release from [GitHub Releases](https://github.com/kadirbelkuyu/kubecrsh/releases/latest):

```bash
curl -LO https://github.com/kadirbelkuyu/kubecrsh/releases/latest/download/kubecrsh_linux_amd64.tar.gz
tar -xzf kubecrsh_linux_amd64.tar.gz
sudo mv kubecrsh /usr/local/bin/
```

macOS (Apple Silicon):

```bash
curl -LO https://github.com/kadirbelkuyu/kubecrsh/releases/latest/download/kubecrsh_darwin_arm64.tar.gz
tar -xzf kubecrsh_darwin_arm64.tar.gz
sudo mv kubecrsh /usr/local/bin/
```

Or using Go:

```bash
go install github.com/kadirbelkuyu/kubecrsh/cmd/kubecrsh@latest
```

### Commands

Watch for crashes in real-time with the interactive TUI:

```bash
kubecrsh watch
kubecrsh watch -n my-namespace
```

Browse previously saved crash reports:

```bash
kubecrsh list
```

### TUI Controls

| Key | Action |
| --- | --- |
| `↑` / `↓` | Move through the list |
| `Enter` | View detailed crash information |
| `Tab` | Switch between different tabs |
| `Esc` | Go back to the previous screen |
| `q` | Quit the application |

## Collected Data

During a crash event, the tool captures the following information and stores it as JSON:

- Container logs (both current and previous)
- Kubernetes events from the past hour
- Environment variables
- Exit codes, restart counts, and timestamps

## Configuration

You can configure the tool using environment variables or a ConfigMap.

| Variable | Description | Default |
| --- | --- | --- |
| `SLACK_WEBHOOK` | Slack incoming webhook URL | - |
| `WEBHOOK_URL` | Generic webhook endpoint | - |
| `HTTP_ADDR` | Metrics server address | `:8080` |
| `NAMESPACE` | Namespace to watch (empty = all) | - |

## Endpoints

The service exposes the following HTTP endpoints:

| Path | Description |
| --- | --- |
| `/health` | Liveness probe |
| `/ready` | Readiness probe |
| `/metrics` | Prometheus metrics |
| `/reports` | List saved crash reports (optional, disabled by default) |
| `/reports/{id}` | Get a single crash report (optional, disabled by default) |

After deployment you can port-forward and validate:

```bash
kubectl -n kubecrsh port-forward svc/kubecrsh 8080:8080
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/ready
curl -fsS http://127.0.0.1:8080/metrics | head
```

## Reports API (Optional)

The Reports API is disabled by default. When enabled, it provides read-only access to stored reports.

Environment variables:

```bash
KUBECRSH_API_REPORTS_ENABLED=true
KUBECRSH_API_TOKEN=your-token
KUBECRSH_API_ALLOW_FULL=false
```

Examples:

```bash
curl -fsS -H "Authorization: Bearer $KUBECRSH_API_TOKEN" "http://127.0.0.1:8080/reports?limit=50&offset=0"
curl -fsS -H "Authorization: Bearer $KUBECRSH_API_TOKEN" "http://127.0.0.1:8080/reports/<report-id>"
```

Full report output is gated. To allow it, set `KUBECRSH_API_ALLOW_FULL=true` and request `full=1`:

```bash
curl -fsS -H "Authorization: Bearer $KUBECRSH_API_TOKEN" "http://127.0.0.1:8080/reports/<report-id>?full=1"
```

## Metrics

```text
kubecrsh_crashes_total{namespace,reason}
kubecrsh_notifications_sent_total{notifier,status}
kubecrsh_report_size_bytes
```

## Project Structure

```bash
kubecrsh/
├── cmd/kubecrsh/        # CLI entrypoints (Cobra)
├── internal/
│   ├── domain/          # Core entities (CrashReport, PodInfo)
│   ├── watcher/         # Kubernetes informer
│   ├── collector/       # Log and event collection
│   ├── notifier/        # Slack, webhook integrations
│   ├── reporter/        # JSON storage
│   ├── daemon/          # HTTP server + metrics
│   └── tui/             # Terminal UI (Bubble Tea)
├── charts/              # Helm chart
├── manifests/           # Kubernetes deployment files
├── monitoring/          # Grafana dashboard, Prometheus config
└── Dockerfile
```

## Development

For local development, build and run the binary:

```bash
go build -o bin/kubecrsh ./cmd/kubecrsh
./bin/kubecrsh daemon --http-addr :8080
```

## License

MIT
