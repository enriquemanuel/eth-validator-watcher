# Quick Start Guide - Go Implementation

Get up and running with the Go-based Ethereum Validator Watcher in 5 minutes.

## Prerequisites

- Go 1.21+ (or use pre-built binary/Docker)
- Access to an Ethereum Beacon node
- 2-4 GB RAM
- Multi-core CPU recommended

## Option 1: Docker (Recommended)

```bash
# Pull image from GitHub Container Registry
docker pull ghcr.io/enriquemanuel/eth-validator-watcher:latest

# Create config file (see Configuration section below)
# Then run:
docker run -d \
  --name eth-validator-watcher \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/config/config.yaml \
  ghcr.io/enriquemanuel/eth-validator-watcher:latest
```

## Option 2: Pre-built Binary

```bash
# Download latest release
wget https://github.com/enriquemanuel/eth-validator-watcher/releases/latest/download/eth-validator-watcher-linux-amd64

# Make executable
chmod +x eth-validator-watcher-linux-amd64
mv eth-validator-watcher-linux-amd64 eth-validator-watcher

# Run
./eth-validator-watcher -version
```

## Option 3: Build from Source

```bash
# Build
make build

# Or build for your platform
make build-linux   # Linux
make build-darwin  # macOS

# Binary will be in build/
./build/eth-validator-watcher -version
```

## Configuration

Create `config.yaml`:

```yaml
network: mainnet
beacon_url: http://localhost:5052  # Your beacon node
beacon_timeout_sec: 90
metrics_port: 8000

watched_keys:
  - public_key: "0xYOUR_VALIDATOR_PUBKEY_HERE"
    labels:
      - "vc:my-validator"
```

**Get your validator public keys:**

```bash
# If you have validator keys
grep -r "pubkey" /path/to/validator/keys

# Or from beacon node
curl http://localhost:5052/eth/v1/beacon/states/head/validators | jq '.data[0].validator.pubkey'
```

## Run

```bash
# Test configuration
./eth-validator-watcher -config config.yaml -log-level debug

# Run in production
./eth-validator-watcher -config config.yaml
```

## Verify It's Working

### Health Checks

```bash
# Liveness check (always returns OK if server is running)
curl http://localhost:8080/health
# Response: OK

# Readiness check (returns OK only after initialization is complete)
curl http://localhost:8080/ready
# Response: READY (or NOT READY if still initializing)
```

### Check Logs

```bash
# You should see:
# INFO Starting Ethereum Validator Watcher
# INFO Initialized beacon clock
# INFO Loading all validators from beacon node...
# INFO ✅ Loaded all validators (batch mode)
# INFO ✅ Validator watcher ready - health checks will now pass
# INFO Processing epoch
```

### Check Metrics

```bash
# Open in browser
open http://localhost:8080/metrics

# Or via curl
curl http://localhost:8080/metrics | grep eth_validator_watcher

# You should see metrics like:
# eth_validator_watcher_validator_count{label="scope:watched"} 1
# eth_validator_watcher_missed_attestations{label="scope:watched"} 0
# eth_sync_committee_validators{label="scope:watched"} 0
```

### Check with Prometheus

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'eth-validator-watcher'
    static_configs:
      - targets: ['localhost:8080']
```

## Complete Monitoring Stack

Use docker-compose for full setup:

```bash
# Copy config
cp config.example.yaml config.yaml
# Edit config.yaml with your validators

# Start stack
docker-compose up -d

# Access services
# - Watcher metrics: http://localhost:8080/metrics
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
```

## Common Issues

### "Failed to get genesis"
- Check beacon node is running: `curl http://localhost:5052/eth/v1/node/version`
- Verify beacon_url in config.yaml

### "Watched validator not found"
- Verify pubkey format (must start with 0x, 98 chars total)
- Check validator exists: `curl http://localhost:5052/eth/v1/beacon/states/head/validators`

### High memory usage
- Normal for 2M validators (~500MB)
- Reduce watched validators if needed
- Set `load_all_validators: false` in config to skip network comparison

## Production Deployment

### Systemd Service

```bash
# Create service
sudo tee /etc/systemd/system/eth-validator-watcher.service > /dev/null <<EOF
[Unit]
Description=Ethereum Validator Watcher
After=network.target

[Service]
Type=simple
User=ethereum
ExecStart=/usr/local/bin/eth-validator-watcher -config /etc/eth-watcher/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable eth-validator-watcher
sudo systemctl start eth-validator-watcher

# Check status
sudo systemctl status eth-validator-watcher
```

### Monitoring

```bash
# View logs
journalctl -u eth-validator-watcher -f

# Check metrics
watch -n 5 'curl -s http://localhost:8080/metrics | grep missed_attestations'

# Monitor resources
top -p $(pgrep eth-validator-watcher)
```

## Next Steps

1. **Set up alerts**: Configure Prometheus AlertManager
2. **Create dashboards**: Import Grafana dashboards
3. **Add more validators**: Update config.yaml with more pubkeys
4. **Optimize labels**: Organize validators by client, region, etc.
5. **Read full docs**: See README.go.md

## Performance Expectations

After startup, you should see:

- **CPU**: 5-10% on 4-core system
- **Memory**: 400-600 MB
- **Metrics update**: Every 12 seconds (per slot)
- **Startup time**: 5-15 seconds

## Help

```bash
# Show help
./eth-validator-watcher -help

# Show version
./eth-validator-watcher -version

# Debug mode
./eth-validator-watcher -config config.yaml -log-level debug
```

## Documentation

- **README.md**: Full documentation
- **docs/METRICS_GUIDE.md**: Understanding metrics
- **docs/GO_IMPLEMENTATION_SUMMARY.md**: Technical details

## Support

- GitHub Issues: https://github.com/enriquemanuel/eth-validator-watcher/issues
- Logs: `journalctl -u eth-validator-watcher -f`
- Metrics: http://localhost:8080/metrics

Happy monitoring! 🚀
