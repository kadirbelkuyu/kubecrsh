# Kubecrsh Helm Chart

Production-ready Helm chart for deploying Kubecrsh - Kubernetes Pod Crash Forensic Analyzer.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PV provisioner (if persistence enabled)
- Prometheus Operator (if ServiceMonitor enabled)

## Installation

### Method 1: Helm Repository (Recommended)

```bash
helm repo add kubecrsh https://kadirbelkuyu.github.io/kubecrsh
helm repo update

helm install kubecrsh kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace
```

### Method 2: OCI Registry (GHCR)

```bash
helm install kubecrsh oci://ghcr.io/kadirbelkuyu/charts/kubecrsh \
  --namespace kubecrsh \
  --create-namespace
```

### Method 3: From Source

```bash
git clone https://github.com/kadirbelkuyu/kubecrsh.git
cd kubecrsh

helm install kubecrsh ./charts/kubecrsh \
  --namespace kubecrsh \
  --create-namespace
```

### With Slack Notifications

```bash
helm install kubecrsh kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace \
  --set notifiers.slack.enabled=true \
  --set secrets.slackWebhook="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### Production Deployment

```bash
helm install kubecrsh kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace \
  -f values-production.yaml \
  --set secrets.slackWebhook="$SLACK_WEBHOOK"
```

### Cluster-Wide Monitoring

```bash
helm install kubecrsh kubecrsh/kubecrsh \
  --namespace kubecrsh \
  --create-namespace \
  --set rbac.clusterWide=true \
  --set config.namespace=""
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/kadirbelkuyu/kubecrsh` |
| `image.tag` | Image tag (defaults to Chart appVersion) | `""` |
| `rbac.clusterWide` | Enable cluster-wide monitoring | `false` |
| `persistence.enabled` | Enable persistent storage for reports | `false` |
| `persistence.size` | PVC size | `5Gi` |
| `notifiers.slack.enabled` | Enable Slack notifications | `false` |
| `notifiers.webhook.enabled` | Enable generic webhook | `false` |
| `metrics.serviceMonitor.enabled` | Create ServiceMonitor | `false` |
| `config.watch.reasons` | Crash reasons to watch | `[OOMKilled, Error, CrashLoopBackOff]` |
| `config.reports.redaction.enabled` | Enable sensitive data redaction | `false` |

### RBAC Modes

**Namespace-Scoped (Default):**

```yaml
rbac:
  clusterWide: false
config:
  namespace: ""  # Uses release namespace
```

**Cluster-Wide:**

```yaml
rbac:
  clusterWide: true
config:
  namespace: ""  # Watches all namespaces
```

### Secrets Management

**Using Helm Values:**

```yaml
secrets:
  create: true
  slackWebhook: "https://hooks.slack.com/..."
  webhookUrl: "https://your-webhook.com/..."
  webhookToken: "your-token"
```

**Using Existing Secret:**

```yaml
secrets:
  create: false
  existingSecret: "my-kubecrsh-secrets"
```

### Monitoring Integration

**ServiceMonitor for Prometheus Operator:**

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      release: prometheus
```

### Security Hardening

The chart implements security best practices by default:

- Non-root user (UID 1000)
- Read-only root filesystem
- Dropped capabilities
- Seccomp profile (RuntimeDefault)
- No privilege escalation

### High Availability

For production, use `values-production.yaml`:

- Multiple replicas with PodDisruptionBudget
- Pod anti-affinity for zone distribution
- HorizontalPodAutoscaler
- Increased resource limits

## Upgrading

```bash
helm upgrade kubecrsh ./charts/kubecrsh \
  --namespace kubecrsh \
  -f my-values.yaml
```

## Uninstalling

```bash
helm uninstall kubecrsh --namespace kubecrsh
```

## Testing

```bash
helm test kubecrsh --namespace kubecrsh
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n kubecrsh -l app.kubernetes.io/name=kubecrsh
```

### View Logs

```bash
kubectl logs -n kubecrsh -l app.kubernetes.io/name=kubecrsh -f
```

### Verify RBAC

```bash
kubectl auth can-i --as=system:serviceaccount:kubecrsh:kubecrsh \
  list pods --namespace kubecrsh
```
