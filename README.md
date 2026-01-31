# kubecrsh

[![Go Version](https://img.shields.io/badge/Go-1.25.6-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/badge/Release-v0.0.1-blue?style=flat&logo=github)](https://github.com/kadirbelkuyu/kubecrsh/releases/tag/v0.0.1)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub Repo](https://img.shields.io/badge/GitHub-kadirbelkuyu%2Fkubecrsh-black?style=flat&logo=github)](https://github.com/kadirbelkuyu/kubecrsh)

kubecrsh is a Kubernetes debugging tool that captures logs, events, and pod state right before a container restarts. It ensures you have the necessary data to investigate crashes without losing ephemeral information.

## Features

The application monitors pod lifecycle events and automatically gathers data when a crash occurs. It supports sending alerts via Slack or webhooks to notify you of production failures immediately. There is also an integrated terminal interface that allows you to browse and analyze crash reports directly from your CLI.

## Quickstart

You can deploy kubecrsh to your cluster using the provided manifests.

```bash
kubectl apply -f manifests/namespace.yaml
kubectl apply -f manifests/rbac.yaml
kubectl apply -f manifests/configmap.yaml

cp manifests/secret.yaml.example manifests/secret.yaml
# edit manifests/secret.yaml and set your webhook values
kubectl apply -f manifests/secret.yaml

kubectl apply -f manifests/service.yaml
kubectl apply -f manifests/deployment.yaml
```

For local development, build and run the binary:

```bash
go build -o bin/kubecrsh ./cmd/kubecrsh
./bin/kubecrsh daemon --http-addr :8080
```

To inspect the collected crash reports:

```bash
./bin/kubecrsh watch
```

## Collected Data

During a crash event, the tool captures the following information and stores it as JSON:

* Container logs (both current and previous)
* Kubernetes events from the past hour
* Environment variables
* Exit codes, restart counts, and timestamps

## TUI Controls

You can navigate the interface using the following keys:

| Key | Action |
| --- | --- |
| `↑` / `↓` | Move through the list |
| `Enter` | View detailed crash information |
| `Tab` | Switch between different tabs |
| `Esc` | Go back to the previous screen |
| `q` | Quit the application |

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
├── manifests/           # Kubernetes deployment files
├── monitoring/          # Grafana dashboard, Prometheus config
└── Dockerfile
```

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

## License

MIT
