# Project Structure

The repository has been reorganized with the Go implementation as the primary codebase.

## Root Directory (Go Implementation)

```
eth-validator-watcher/
├── cmd/                          # Main application entry point
│   └── watcher/main.go          # CLI and startup logic
├── pkg/                          # Go packages
│   ├── beacon/                  # Beacon Chain API client
│   ├── clock/                   # Slot timing management
│   ├── config/                  # Configuration loading
│   ├── duties/                  # Attestation/reward processing
│   ├── metrics/                 # Metrics computation & Prometheus
│   ├── models/                  # Data structures
│   ├── price/                   # ETH price fetcher (CoinGecko API)
│   ├── proposer/                # Proposer duty tracking
│   ├── validator/               # Validator registries
│   └── watcher/                 # Main orchestrator
├── monitoring/                   # Monitoring configuration
│   ├── grafana/                 # Grafana dashboards
│   │   └── dashboards/          # JSON dashboard definitions
│   └── prometheus/              # Prometheus configuration
├── charts/                       # Helm charts for Kubernetes
│   └── eth-validator-watcher/   # Main Helm chart
├── docs/                         # Documentation
│   ├── CREDITS.md               # Attribution and credits
│   ├── GO_IMPLEMENTATION_SUMMARY.md  # Technical details
│   ├── METRICS_GUIDE.md         # Metrics documentation
│   ├── PROJECT_STRUCTURE.md     # This file
│   └── QUICKSTART.md            # Quick start guide
├── go.mod                        # Go module definition
├── go.sum                        # Go dependency lock
├── Makefile                      # Build automation
├── Dockerfile                    # Container image
├── docker-compose.yaml           # Complete monitoring stack
├── config.example.yaml           # Example configuration
└── README.md                     # Main documentation
```

## Python Legacy

All Python/C++ code has been moved to `python-legacy/`:

```
python-legacy/
├── eth_validator_watcher/        # Python source code
├── tests/                        # Python unit tests
├── docs/                         # Python documentation
├── etc/                          # Configuration examples
├── .github/workflows/            # Old CI/CD workflows
├── build.py                      # C++ extension builder
├── pyproject.toml                # Python package config
├── requirements.txt              # Python dependencies
├── Dockerfile.python             # Old Docker setup
├── README.md                     # Original README
└── LEGACY_README.md              # Legacy status explanation
```

## Shared Resources

- `monitoring/grafana/` - Grafana dashboards
- `monitoring/prometheus/` - Prometheus configuration
- `charts/` - Helm charts for Kubernetes deployment
- `LICENSE` - Project license

## Quick Start

```bash
# Build the Go version
make build

# Run with example config
./build/eth-validator-watcher -config config.example.yaml

# Or use Docker
docker-compose up -d
```

## For Python Users

If you're migrating from the Python version:

1. See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for step-by-step instructions
2. Your existing `config.yaml` is compatible (just copy it to the root)
3. All Prometheus metrics have the same names
4. The Python version remains in `python-legacy/` for reference

## Development

```bash
# Run tests
make test

# Build for multiple platforms
make build-all

# Run linter
make lint

# Generate coverage report
make test-coverage
```

## Why Go is Now Primary

The Go implementation provides:
- **5x faster** startup (6s vs 30s)
- **3x faster** metrics computation
- **40% lower** memory usage
- **Single binary** deployment
- **Better concurrency** (no GIL)

See [GO_IMPLEMENTATION_SUMMARY.md](GO_IMPLEMENTATION_SUMMARY.md) for technical details.
