# Metrics Guide

## Understanding Validator Metrics

### Stake Metrics

The `eth_validator_status_scaled_count` metric shows stake in **units of 32 ETH**:

```
eth_validator_status_scaled_count{scope="operator:lido1",status="active_ongoing",network="mainnet"} 1000
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
  labels: ["operator:lido1", "name:Lido1"]
```

**Results in metric series for each scope:**
```
eth_validator_status_count{scope="operator:lido1",status="active_ongoing",network="mainnet"} 1000
eth_validator_status_count{scope="name:Lido1",status="active_ongoing",network="mainnet"} 1000
eth_validator_status_count{scope="scope:watched",status="active_ongoing",network="mainnet"} 22752
eth_validator_status_count{scope="scope:all-network",status="active_ongoing",network="mainnet"} 2130801
```

This allows you to query by ANY scope:
- `operator:lido1` - All Lido validators
- `name:Lido1` - Specific Lido instance
- `key:0x88b3be1f4f` - Individual validator (auto-generated)
- `scope:watched` - All your watched validators
- `scope:network` - Network excluding your watched validators
- `scope:all-network` - Entire Ethereum network

## Key Metrics

### Network Metrics
- `eth_slot{network}` - Current slot number
- `eth_epoch{network}` - Current epoch number
- `eth_current_price_dollars{network}` - Current ETH price in USD
- `eth_pending_deposits_count{network}` - Pending deposit count
- `eth_pending_deposits_value{network}` - Pending deposit value in Gwei
- `eth_pending_consolidations_count{network}` - Pending consolidation count
- `eth_pending_withdrawals_count{network}` - Pending withdrawal count

### Validator Status
- `eth_validator_status_count{scope,status,network}` - Count by status
- `eth_validator_status_scaled_count{scope,status,network}` - Stake by status (32 ETH units)
- `eth_validator_type_count{scope,type,network}` - Count by withdrawal credential type
- `eth_validator_type_scaled_count{scope,type,network}` - Stake by type
- `eth_slashed_validators{scope,network}` - Slashed validator count

**Status Values:**
- `active_ongoing` - Active and attesting
- `active_exiting` - In exit queue
- `pending_initialized` - Deposit made, waiting
- `pending_queued` - In activation queue
- `exited_unslashed` - Exited normally
- `exited_slashed` - Slashed and exited
- `withdrawal_done` - Fully withdrawn

**Type Values:**
- `0x00` - BLS withdrawal credentials
- `0x01` - Execution withdrawal credentials
- `0x02` - Compounding withdrawal credentials

### Attestation Performance
- `eth_missed_attestations{scope,network}` - Count of missed attestations
- `eth_missed_attestations_scaled{scope,network}` - Stake-weighted missed attestations
- `eth_suboptimal_sources_rate{scope,network}` - Rate of suboptimal source votes (0-1)
- `eth_suboptimal_targets_rate{scope,network}` - Rate of suboptimal target votes (0-1)
- `eth_suboptimal_heads_rate{scope,network}` - Rate of suboptimal head votes (0-1)
- `eth_duties_rate{scope,network}` - Attestation success rate (0-100%)
- `eth_duties_rate_scaled{scope,network}` - Stake-weighted success rate

### Duty Tracking (per slot)
- `eth_performed_duties_at_slot{scope,network}` - Successful duties in current slot
- `eth_missed_duties_at_slot{scope,network}` - Missed duties in current slot
- `eth_performed_duties_at_slot_scaled{scope,network}` - Stake-weighted successful
- `eth_missed_duties_at_slot_scaled{scope,network}` - Stake-weighted missed
- `eth_missed_consecutive_attestations{scope,network}` - Max consecutive missed

### Block Proposals
- `eth_block_proposals_head_total{scope,network}` - Total proposed blocks (counter)
- `eth_missed_block_proposals_head_total{scope,network}` - Total missed blocks (counter)
- `eth_block_proposals_finalized_total{scope,network}` - Finalized proposals (counter)
- `eth_missed_block_proposals_finalized_total{scope,network}` - Finalized missed (counter)
- `eth_future_block_proposals{scope,network}` - Upcoming proposals in next 2 epochs

### Rewards
- `eth_ideal_consensus_rewards_gwei{scope,network}` - Maximum possible rewards
- `eth_actual_consensus_rewards_gwei{scope,network}` - Actual earned rewards
- `eth_consensus_rewards_rate{scope,network}` - Reward rate (0-100%)

## Example Queries

**Validator count by operator:**
```promql
eth_validator_status_count{scope=~"operator:.*",status="active_ongoing"}
```

**Total stake managed:**
```promql
eth_validator_status_scaled_count{scope="operator:lido1",status="active_ongoing"} * 32
```

**Performance rate:**
```promql
eth_consensus_rewards_rate{scope="scope:watched"}
```

**Compare your performance vs network:**
```promql
eth_consensus_rewards_rate{scope="scope:watched"} / eth_consensus_rewards_rate{scope="scope:all-network"}
```

**Missed attestations by operator:**
```promql
eth_missed_attestations{scope=~"operator:.*"}
```

**Block proposals in last 24h:**
```promql
increase(eth_block_proposals_head_total{scope=~"operator:.*"}[24h])
```

**Validators at risk (consecutive missed):**
```promql
eth_missed_consecutive_attestations{scope=~"operator:.*"} > 2
```

**ETH price:**
```promql
eth_current_price_dollars{network="mainnet"}
```

## Access Metrics

Metrics are available at:
```
http://localhost:8080/metrics
```

Or use Prometheus/Grafana for visualization and alerting.
