# Metrics Guide

## Understanding Validator Metrics

### Stake Metrics

The `eth_validator_watcher_status_stake` metric shows stake in **units of 32 ETH**:

```
eth_validator_watcher_status_stake{label="operator:lido1",status="active_ongoing"} 1000
```

This means:
- **1000** validators with that label
- **32,000 ETH** total stake (1000 × 32 ETH)

### Common Values Explained

| Metric Value | Meaning |
|--------------|---------|
| `0` | Validator pending activation or exited (0 effective balance) |
| `1` | 1 validator = 32 ETH |
| `0.03` | Validator with 1 ETH effective balance (compounding rewards) |
| `1000` | 1000 validators = 32,000 ETH total |

### Label Structure

Each validator can have multiple labels, and metrics are created for EACH label separately:

**Your Config:**
```yaml
- public_key: '0x88b3be1f4f...'
  labels: ["operator:lido1", "name:Lido1", "key:0x88b3be1f4f"]
```

**Results in 5 Metric Series:**
```
eth_validator_watcher_status_stake{label="operator:lido1",status="active_ongoing"} 1000
eth_validator_watcher_status_stake{label="name:Lido1",status="active_ongoing"} 1000
eth_validator_watcher_status_stake{label="key:0x88b3be1f4f",status="active_ongoing"} 1
eth_validator_watcher_status_stake{label="scope:watched",status="active_ongoing"} 22752
eth_validator_watcher_status_stake{label="scope:all-network",status="active_ongoing"} 2130801
```

This allows you to query by ANY label:
- `operator:lido1` - All Lido validators
- `name:Lido1` - Specific Lido instance
- `key:0x88b3be1f4f` - Individual validator
- `scope:watched` - All your watched validators
- `scope:all-network` - Entire Ethereum network

### Querying Metrics

**See all validators for an operator:**
```promql
sum(eth_validator_watcher_status_stake{label="operator:lido1"})
```

**Compare your validators vs network:**
```promql
eth_validator_watcher_consensus_rewards_rate{label="scope:watched"}
  vs
eth_validator_watcher_consensus_rewards_rate{label="scope:all-network"}
```

**Individual validator performance:**
```promql
eth_validator_watcher_missed_attestations{label="key:0x88b3be1f4f"}
```

## Key Metrics

### Attestation Performance
- `eth_validator_watcher_missed_attestations` - Count of missed attestations
- `eth_validator_watcher_attestation_duties_rate` - Success rate (0.0 to 1.0)
- `eth_validator_watcher_suboptimal_source_votes` - Suboptimal source votes
- `eth_validator_watcher_suboptimal_target_votes` - Suboptimal target votes
- `eth_validator_watcher_suboptimal_head_votes` - Suboptimal head votes

### Block Proposals
- `eth_validator_watcher_proposed_blocks` - Successfully proposed blocks
- `eth_validator_watcher_missed_blocks` - Missed block proposals
- `eth_validator_watcher_future_block_proposals` - Upcoming proposals

### Rewards
- `eth_validator_watcher_consensus_rewards_gwei` - Actual consensus rewards
- `eth_validator_watcher_ideal_consensus_rewards_gwei` - Ideal consensus rewards
- `eth_validator_watcher_consensus_rewards_rate` - Actual/Ideal ratio (0.0 to 1.0)

### Validator Status
- `eth_validator_watcher_status_count` - Count by status
- `eth_validator_watcher_status_stake` - Stake by status
- `eth_validator_watcher_validator_count` - Total validators per label

**Status Values:**
- `active_ongoing` - Active and attesting
- `active_exiting` - In exit queue
- `pending_initialized` - Deposit made, waiting
- `pending_queued` - In activation queue
- `exited_unslashed` - Exited normally
- `exited_slashed` - Slashed and exited
- `withdrawal_done` - Fully withdrawn

## Current Status

**Loaded:**
- ✅ **2,130,801** total validators (full Ethereum network)
- ✅ **22,752** watched validators (from your config)

**Note:** 22,752 found out of 45,803 configured keys. The difference may be due to:
- Validators not yet activated
- Invalid/duplicate public keys in config
- Exited validators

## Example Queries

**Validator uptime:**
```promql
eth_validator_watcher_attestation_duties_rate{label="operator:lido1"} * 100
```

**Total stake managed:**
```promql
eth_validator_watcher_stake_count{label="operator:lido1"} * 32
```

**Performance vs network:**
```promql
(eth_validator_watcher_consensus_rewards_rate{label="scope:watched"} /
 eth_validator_watcher_consensus_rewards_rate{label="scope:all-network"}) * 100
```

**Validators at risk (consecutive missed attestations):**
```promql
eth_validator_watcher_consecutive_missed_attestations > 2
```

## Access Metrics

Metrics are available at:
```
http://localhost:8000/metrics
```

Or use Prometheus/Grafana for visualization and alerting.
