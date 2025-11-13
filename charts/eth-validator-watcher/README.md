# Ethereum Validator Watcher Helm Chart

## Installation

```bash
# Install with default values
helm install eth-validator-watcher . \
  --namespace monitoring \
  --create-namespace

# Install with custom config
helm install eth-validator-watcher . \
  --namespace monitoring \
  --set config="$(cat your-config.yaml)"

# Install with custom values file
helm install eth-validator-watcher . \
  --namespace monitoring \
  --values custom-values.yaml
```

## Configuration

### Essential Values

Edit `values.yaml` or create your own values file:

```yaml
# Image configuration
image:
  repository: your-registry/eth-validator-watcher
  tag: latest
  pullPolicy: IfNotPresent

# Resource limits
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2000m"

# Validator configuration (REQUIRED - edit this!)
config: |
  network: mainnet
  beacon_url: http://beacon-node:5052
  beacon_timeout_sec: 30
  metrics_port: 8080
  watched_keys:
    - public_key: '0x1234...'
      labels:
        - operator:my-operator
```

### Health Checks

The chart includes pre-configured health checks:

- **Liveness Probe** (`/health`): Checks if process is running
- **Readiness Probe** (`/ready`): Checks if initialization completed
- **Startup Probe** (`/ready`): Allows 150 seconds for loading validators

## Prometheus Integration

PodMonitor is enabled by default for Prometheus Operator:

```yaml
podMonitor:
  enabled: true
  interval: 12s
```

## Upgrade

```bash
helm upgrade eth-validator-watcher . --namespace monitoring
```

## Uninstall

```bash
helm uninstall eth-validator-watcher --namespace monitoring
```
