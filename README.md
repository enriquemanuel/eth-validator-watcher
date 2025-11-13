# Ethereum Validator Watcher (Go)

A high-performance Ethereum validator monitoring tool written in Go. Monitors validator performance, attestations, block proposals, and consensus rewards across the entire Ethereum network (2M+ validators).

## Quick Start

### Using Docker (Recommended)

```bash
# Pull image from GitHub Container Registry
docker pull ghcr.io/enriquemanuel/eth-validator-watcher:latest

# Create config file
cat > config.yaml <<EOF
network: mainnet
beacon_url: http://your-beacon-node:5052
beacon_timeout_sec: 30
metrics_port: 8080
watched_keys:
  - public_key: "0x1234..."
    labels: [operator:my-operator]
EOF

# Run
docker run -d \
  --name eth-validator-watcher \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/config/config.yaml \
  ghcr.io/enriquemanuel/eth-validator-watcher:latest
```

### Using Helm (Kubernetes)

```bash
# Add Helm repository
helm repo add eth-validator-watcher https://enriquemanuel.github.io/eth-validator-watcher
helm repo update

# Install
helm install eth-validator-watcher eth-validator-watcher/eth-validator-watcher \
  --namespace monitoring \
  --create-namespace \
  --set config="$(cat config.yaml)"
```

### From Source

```bash
# Build
make build

# Configure
cp config.example.yaml config.yaml
vim config.yaml

# Run
./build/eth-validator-watcher -config config.yaml
```

### Health Checks

```bash
curl http://localhost:8080/health   # Liveness check
curl http://localhost:8080/ready    # Readiness check
curl http://localhost:8080/metrics  # Prometheus metrics
```

## Features

- **Real-time Monitoring**: Slot-by-slot processing of all validators
- **Performance Metrics**: Attestation success rate, consensus rewards, block proposals
- **Label-based Organization**: Group validators by operator, region, client, etc.
- **Prometheus Export**: Industry-standard metrics format
- **Concurrent Processing**: Parallel metrics computation across CPU cores
- **Network Comparison**: Compare your validators against all 2M+ validators
- **Health Checks**: Kubernetes-ready liveness and readiness probes

## Configuration

Create `config.yaml`:

```yaml
network: mainnet
beacon_url: http://your-beacon-node:5052
beacon_timeout_sec: 30
metrics_port: 8080

# Optional: Disable full validator set loading (faster startup, no network comparison)
# load_all_validators: false

watched_keys:
  - public_key: "0x1234..."
    labels:
      - operator:my-operator
      - region:us-east
      - client:lighthouse

  - public_key: "0x5678..."
    labels:
      - operator:my-operator
      - region:eu-west
      - client:prysm
```

## Understanding the Metrics

### Performance Rate vs Miss Rate

**Performance Rate** (`consensus_rewards_rate`):
- Formula: `actual_rewards / ideal_rewards`
- Includes penalties for suboptimal votes (wrong head/source/target)
- Includes penalties for late attestations
- **99.95% is excellent** - means you got 99.95% of maximum possible rewards

**Miss Rate** (`missed_attestations / attestation_duties`):
- Only counts completely missed attestations
- **0% is perfect** - you never failed to attest

Example: `performance_rate=99.95%, miss_rate=0.00%`
- You never missed an attestation ✅
- But lost 0.05% rewards due to suboptimal votes or timing

### Key Metrics

**Validator Counts:**
- `eth_validator_watcher_validator_count{label}` - Total validators
- `eth_validator_watcher_status_count{label,status}` - By status (active/exited/pending)

**Performance:**
- `eth_validator_watcher_consensus_rewards_rate{label}` - Performance rate (0-1.0)
- `eth_validator_watcher_missed_attestations{label}` - Missed attestations count
- `eth_validator_watcher_attestation_duties{label}` - Total duties assigned
- `eth_validator_watcher_attestation_duties_success{label}` - Successful attestations

**Suboptimal Votes (reduce rewards but not "misses"):**
- `eth_validator_watcher_suboptimal_head_votes{label}` - Wrong head block
- `eth_validator_watcher_suboptimal_source_votes{label}` - Wrong source checkpoint
- `eth_validator_watcher_suboptimal_target_votes{label}` - Wrong target checkpoint

**Block Proposals:**
- `eth_validator_watcher_proposed_blocks{label}` - Blocks proposed
- `eth_validator_watcher_proposed_blocks_finalized{label}` - Finalized proposals
- `eth_validator_watcher_missed_blocks{label}` - Missed proposals

**Rewards:**
- `eth_validator_watcher_ideal_consensus_rewards_gwei{label}` - Maximum possible
- `eth_validator_watcher_consensus_rewards_gwei{label}` - Actual earned

### Labels

Every metric has a `label` dimension for grouping:

**Default labels:**
- `scope:all-network` - All 2M+ Ethereum validators
- `scope:watched` - Your watched validators only

**Custom labels** (from your config):
- `operator:name` - Group by operator/infrastructure
- `region:location` - Geographic grouping
- `client:software` - Consensus client type
- Any custom labels you define

## Prometheus Queries

```promql
# Performance rate by operator
eth_validator_watcher_consensus_rewards_rate{label=~"operator:.*"} * 100

# Miss rate by operator
(eth_validator_watcher_missed_attestations{label=~"operator:.*"} /
 eth_validator_watcher_attestation_duties{label=~"operator:.*"}) * 100

# Active validators by operator
eth_validator_watcher_status_count{label=~"operator:.*", status="active_ongoing"}

# Block proposals in last 24h
increase(eth_validator_watcher_proposed_blocks{label=~"operator:.*"}[24h])

# Compare your performance vs network
eth_validator_watcher_consensus_rewards_rate{label="scope:watched"} /
eth_validator_watcher_consensus_rewards_rate{label="scope:all-network"}
```

## Kubernetes Deployment

### Using Helm (Recommended)

```bash
# Install from local chart
helm install eth-validator-watcher ./charts/eth-validator-watcher \
  --namespace monitoring \
  --create-namespace \
  --set config="$(cat config.yaml)"

# Or customize with values file
helm install eth-validator-watcher ./charts/eth-validator-watcher \
  --namespace monitoring \
  --values my-values.yaml
```

The Helm chart includes:
- ✅ Health checks (`/health` and `/ready` endpoints)
- ✅ Startup probe (150s for loading validators)
- ✅ PodMonitor for Prometheus Operator
- ✅ ConfigMap for configuration
- ✅ ServiceAccount

See `charts/eth-validator-watcher/values.yaml` for all configuration options.

## Log Output Examples

**Excellent Performance:**
```
INFO[...] 📊 Operator performance: excellent
  label="operator:my-operator"
  validators=100
  active_validators=100
  performance_rate="100.00%"
  miss_rate="0.00%"
```

**Good Performance with Minor Issues:**
```
INFO[...] 📊 Operator performance: good
  label="operator:my-operator"
  validators=100
  active_validators=98
  performance_rate="99.85%"
  miss_rate="0.12%"
  missed_attestations=2
```

**Critical Performance:**
```
ERRO[...] 📊 Operator performance: critical
  label="operator:my-operator"
  performance_rate="85.00%"
  top_offenders="123(0x1234...):missed=10,perf=80.5%; 456(0x5678...):missed=8,perf=82.3%"
```

**No Active Validators (Not an Error):**
```
DEBU[...] 📊 Operator performance: no active validators
  label="operator:exited-validators"
  validators=100
  active_validators=0
```

## Architecture

```
┌─────────────────┐
│  Beacon Client  │ ← Fetches data from Ethereum Beacon Chain
└────────┬────────┘
         │
┌────────▼────────┐
│ Validator       │ ← Manages 2M+ validators + watched subset
│ Registry        │
└────────┬────────┘
         │
┌────────▼────────┐
│ Duties          │ ← Processes attestations, rewards, blocks
│ Processor       │
└────────┬────────┘
         │
┌────────▼────────┐
│ Metrics Engine  │ ← Concurrent aggregation by labels
└────────┬────────┘
         │
┌────────▼────────┐
│ Prometheus      │ ← Exports at :8080/metrics
│ Exporter        │
└─────────────────┘
```

### Key Design Decisions

1. **Load All Validators (Default)**: Enables network-wide comparison, takes 30-60s on startup
2. **Active-Only Metrics**: Only active validators contribute to performance metrics (exited validators ignored)
3. **Block Proposals Always Counted**: Unlike attestations, block proposals count regardless of validator status
4. **Concurrent Metrics**: Uses worker pools across CPU cores for fast aggregation

## Performance

- **Startup**: ~60s (loading 2.1M validators)
- **Memory**: ~500MB (full validator set + watched validators)
- **Metrics Update**: <100ms (10k validators, 8 cores)
- **Binary Size**: ~10MB (single static binary)

## Troubleshooting

**Q: Why is performance_rate 99.95% but miss_rate 0%?**
A: Performance includes suboptimal votes and timing. You didn't miss attestations, but some had suboptimal head/source/target votes.

**Q: Why does my exited validator show in logs?**
A: At DEBUG level only. Exited validators show `active_validators=0` and don't affect performance calculations.

**Q: Metrics endpoint slow to load?**
A: Use `/health` or `/ready` for health checks. The `/metrics` endpoint is comprehensive and may take longer with many validators.

**Q: Block proposals not showing in metrics?**
A: Check `eth_validator_watcher_proposed_blocks{label="operator:..."}`. Block proposals are rare events (depends on validator count).

**Q: Can I disable loading all validators?**
A: Yes! Set `load_all_validators: false` in config. Faster startup but loses network comparison.

## Development

```bash
# Build
make build

# Test
make test

# Run locally
./build/eth-validator-watcher -config config.yaml -log-level debug

# Format code
go fmt ./...

# Project structure
pkg/
├── beacon/      # Beacon API client
├── clock/       # Slot/epoch timing
├── config/      # Config loading
├── duties/      # Attestation/reward processing
├── metrics/     # Prometheus metrics
├── models/      # Data types
├── proposer/    # Block proposer schedule
├── validator/   # Validator registry
└── watcher/     # Main orchestrator
```

## Migration from Python Version

This Go implementation is a drop-in replacement:
- ✅ Same Prometheus metric names
- ✅ Same configuration format
- ✅ Same functionality
- ✅ 3-5x faster performance
- ✅ 40% lower memory usage
- ✅ Single binary (no Python/C++ deps)

## Credits

**Original Implementation:** [Kiln](https://github.com/kilnfi) - Python/C++ version
**Go Refactor:** [Enrique Valenzuela](https://github.com/enriquemanuel)

Both implementations are MIT licensed.

## License

MIT License - See LICENSE file
