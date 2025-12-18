# Metrics Comparison: Python Legacy vs Go Implementation

## Summary
This document compares all metrics and computations between the Python/C++ legacy code and the Go implementation.

## ✅ Fully Implemented Metrics

### Core Metrics
- ✅ Validator status counts (with stake scaling)
- ✅ Missed attestations (count + stake-scaled)
- ✅ Suboptimal votes (source/target/head) - counts and stake
- ✅ Consensus rewards (ideal, actual, rate)
- ✅ Block proposals (proposed, missed, finalized, future)
- ✅ Attestation duties (total, success, rate)
- ✅ Consecutive missed attestations (tracked differently - see below)
- ✅ Slashed validators count
- ✅ ETH price (metric exists, needs price fetcher integration)
- ✅ Queue metrics (pending deposits, consolidations, withdrawals)
- ✅ Validator type metrics (for network aggregates only)

## ✅ Fixed / Now Implemented

### 1. Duties at Slot Metrics
**Python**: Tracks `duties_slot` and `duties_performed_at_slot` per validator. When computing metrics, checks `if (slot == v.duties_slot)` to count performed/missed duties at that specific slot.

**Go**: ✅ **NOW IMPLEMENTED**
- Added `DutiesSlot` and `DutiesPerformedAtSlot` fields to `WatchedValidator`
- Tracked when processing attestations in `processAttestations`
- Populated `PerformedDutiesAtSlot` and `MissedDutiesAtSlot` in `ComputeMetrics` when `slot == v.DutiesSlot`

**Status**: ✅ IMPLEMENTED

### 2. Consecutive Missed Attestations Logic
**Python**: Tracks `previous_missed_attestation` (from previous epoch) and `missed_attestation` (current epoch). Counts consecutive when **both are true** (exactly 2 consecutive misses).

**Go**: ✅ **NOW IMPLEMENTED**
- Added `MissedAttestation` (boolean) and `PreviousMissedAttestation` (boolean) fields to `WatchedValidator`
- Updated `processLiveness` to track `PreviousMissedAttestation = MissedAttestation` before updating current
- Updated `ComputeMetrics` to check `v.PreviousMissedAttestation && v.MissedAttestation` (matches Python logic)

**Status**: ✅ IMPLEMENTED (matches Python logic)

### 3. Validator Type Metrics Per Label
**Python**: Computes `validator_type_count` and `validator_type_scaled_count` **for each label** (watched validators).

**Go**: ✅ **NOW IMPLEMENTED**
- Added `TypeCounts` and `TypeStakes` maps to `MetricsByLabel`
- Compute validator type for each watched validator using `GetValidatorType()`
- Populate type counts/stakes per label in `ComputeMetrics`
- Export to Prometheus metrics per label

**Status**: ✅ IMPLEMENTED

### 4. Suboptimal Rates
**Python**: Computes rates as `suboptimal_count / total_active_validators`.

**Go**: ✅ Implemented - computes rates correctly.

**Status**: ✅ IMPLEMENTED

## Missing Fields in WatchedValidator

The Go `WatchedValidator` struct is missing:
1. `DutiesSlot` - Which slot the validator had a duty
2. `DutiesPerformedAtSlot` - Whether they performed the duty at that slot
3. `PreviousMissedAttestation` - Whether they missed in the previous epoch (for consecutive tracking)

## Recommendations

1. **Add duties at slot tracking** - Track `duties_slot` and `duties_performed_at_slot` in `WatchedValidator`
2. **Add validator type per label** - Compute validator type counts for watched validators, not just network
3. **Clarify consecutive missed logic** - Decide if counting all consecutive misses (Go) or just 2+ consecutive (Python) is desired

