package metrics

import (
	"sync"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics holds all Prometheus metric collectors
type PrometheusMetrics struct {
	// Slot and epoch metrics
	Slot  *prometheus.GaugeVec
	Epoch *prometheus.GaugeVec

	// Network metrics
	CurrentPriceDollars        *prometheus.GaugeVec
	PendingDepositsCount       *prometheus.GaugeVec
	PendingDepositsValue       *prometheus.GaugeVec
	PendingConsolidationsCount *prometheus.GaugeVec
	PendingWithdrawalsCount    *prometheus.GaugeVec

	// Validator status metrics
	ValidatorStatusCount       *prometheus.GaugeVec
	ValidatorStatusScaledCount *prometheus.GaugeVec

	// Validator type metrics
	ValidatorTypeCount       *prometheus.GaugeVec
	ValidatorTypeScaledCount *prometheus.GaugeVec

	// Slashed validators
	SlashedValidators *prometheus.GaugeVec

	// Attestation metrics
	MissedAttestations       *prometheus.GaugeVec
	MissedAttestationsScaled *prometheus.GaugeVec
	SuboptimalSourcesRate    *prometheus.GaugeVec
	SuboptimalTargetsRate    *prometheus.GaugeVec
	SuboptimalHeadsRate      *prometheus.GaugeVec

	// Block production metrics
	BlockProposalsHeadTotal            *prometheus.CounterVec
	MissedBlockProposalsHeadTotal      *prometheus.CounterVec
	BlockProposalsFinalizedTotal       *prometheus.CounterVec
	MissedBlockProposalsFinalizedTotal *prometheus.CounterVec
	FutureBlockProposals               *prometheus.GaugeVec

	// Reward metrics
	IdealConsensusRewardsGwei  *prometheus.GaugeVec
	ActualConsensusRewardsGwei *prometheus.GaugeVec
	ConsensusRewardsRate       *prometheus.GaugeVec

	// Duty metrics at slot level
	MissedDutiesAtSlot       *prometheus.GaugeVec
	MissedDutiesAtSlotScaled *prometheus.GaugeVec
	PerformedDutiesAtSlot       *prometheus.GaugeVec
	PerformedDutiesAtSlotScaled *prometheus.GaugeVec

	// Duty metrics
	DutiesRate       *prometheus.GaugeVec
	DutiesRateScaled *prometheus.GaugeVec

	// Consecutive missed attestations
	MissedConsecutiveAttestations       *prometheus.GaugeVec
	MissedConsecutiveAttestationsScaled *prometheus.GaugeVec

	// Counter state tracking (last seen values for incrementing)
	counterState     map[string]counterValues
	counterStateMu   sync.RWMutex
}

// counterValues tracks the last seen values for counters
type counterValues struct {
	ProposedBlocks          uint64
	MissedBlocks            uint64
	ProposedBlocksFinalized uint64
	MissedBlocksFinalized   uint64
}

// NewPrometheusMetrics creates and registers all Prometheus metrics
func NewPrometheusMetrics(registry *prometheus.Registry) *PrometheusMetrics {
	m := &PrometheusMetrics{
		Slot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_slot",
			Help: "Current Ethereum slot number",
		}, []string{"network"}),
		Epoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_epoch",
			Help: "Current Ethereum epoch number",
		}, []string{"network"}),
		CurrentPriceDollars: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_current_price_dollars",
			Help: "Current ETH price in USD",
		}, []string{"network"}),
		PendingDepositsCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_pending_deposits_count",
			Help: "Number of pending deposits",
		}, []string{"network"}),
		PendingDepositsValue: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_pending_deposits_value",
			Help: "Total value of pending deposits in Gwei",
		}, []string{"network"}),
		PendingConsolidationsCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_pending_consolidations_count",
			Help: "Number of pending consolidations",
		}, []string{"network"}),
		PendingWithdrawalsCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_pending_withdrawals_count",
			Help: "Number of pending withdrawals",
		}, []string{"network"}),
		ValidatorStatusCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_status_count",
			Help: "Number of validators by status",
		}, []string{"scope", "status", "network"}),
		ValidatorStatusScaledCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_status_scaled_count",
			Help: "Number of validators by status, scaled by stake (32 ETH units)",
		}, []string{"scope", "status", "network"}),
		ValidatorTypeCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_type_count",
			Help: "Number of validators by withdrawal credentials type",
		}, []string{"scope", "type", "network"}),
		ValidatorTypeScaledCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_type_scaled_count",
			Help: "Number of validators by withdrawal credentials type, scaled by stake (32 ETH units)",
		}, []string{"scope", "type", "network"}),
		SlashedValidators: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_slashed_validators",
			Help: "Total number of slashed validators",
		}, []string{"scope", "network"}),
		MissedAttestations: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_missed_attestations",
			Help: "Number of missed attestations in the current epoch",
		}, []string{"scope", "network"}),
		MissedAttestationsScaled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_missed_attestations_scaled",
			Help: "Number of missed attestations in the current epoch, scaled by stake (32 ETH units)",
		}, []string{"scope", "network"}),
		SuboptimalSourcesRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_suboptimal_sources_rate",
			Help: "Rate of suboptimal source votes (0-1)",
		}, []string{"scope", "network"}),
		SuboptimalTargetsRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_suboptimal_targets_rate",
			Help: "Rate of suboptimal target votes (0-1)",
		}, []string{"scope", "network"}),
		SuboptimalHeadsRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_suboptimal_heads_rate",
			Help: "Rate of suboptimal head votes (0-1)",
		}, []string{"scope", "network"}),
		BlockProposalsHeadTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "eth_block_proposals_head_total",
			Help: "Total block proposals at head",
		}, []string{"scope", "network"}),
		MissedBlockProposalsHeadTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "eth_missed_block_proposals_head_total",
			Help: "Total missed block proposals at head",
		}, []string{"scope", "network"}),
		BlockProposalsFinalizedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "eth_block_proposals_finalized_total",
			Help: "Total number of finalized block proposals",
		}, []string{"scope", "network"}),
		MissedBlockProposalsFinalizedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "eth_missed_block_proposals_finalized_total",
			Help: "Total number of finalized missed block proposals",
		}, []string{"scope", "network"}),
		FutureBlockProposals: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_future_block_proposals",
			Help: "Number of upcoming block proposals in the next 2 epochs",
		}, []string{"scope", "network"}),
		IdealConsensusRewardsGwei: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_ideal_consensus_rewards_gwei",
			Help: "Ideal consensus rewards in Gwei",
		}, []string{"scope", "network"}),
		ActualConsensusRewardsGwei: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_actual_consensus_rewards_gwei",
			Help: "Actual consensus rewards in Gwei",
		}, []string{"scope", "network"}),
		ConsensusRewardsRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_consensus_rewards_rate",
			Help: "Consensus rewards rate (actual/ideal, 0-1)",
		}, []string{"scope", "network"}),
		MissedDutiesAtSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_missed_duties_at_slot",
			Help: "Missed validator duties in last slot",
		}, []string{"scope", "network"}),
		MissedDutiesAtSlotScaled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_missed_duties_at_slot_scaled",
			Help: "Stake-scaled missed validator duties in last slot",
		}, []string{"scope", "network"}),
		PerformedDutiesAtSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_performed_duties_at_slot",
			Help: "Performed validator duties in last slot",
		}, []string{"scope", "network"}),
		PerformedDutiesAtSlotScaled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_performed_duties_at_slot_scaled",
			Help: "Stake-scaled performed validator duties in last slot",
		}, []string{"scope", "network"}),
		DutiesRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_duties_rate",
			Help: "Attestation duties success rate (0-1)",
		}, []string{"scope", "network"}),
		DutiesRateScaled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_duties_rate_scaled",
			Help: "Attestation duties success rate, scaled by stake (0-1)",
		}, []string{"scope", "network"}),
		MissedConsecutiveAttestations: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_missed_consecutive_attestations",
			Help: "Maximum number of consecutive missed attestations",
		}, []string{"scope", "network"}),
		MissedConsecutiveAttestationsScaled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_missed_consecutive_attestations_scaled",
			Help: "Maximum number of consecutive missed attestations, scaled by stake (32 ETH units)",
		}, []string{"scope", "network"}),
		counterState: make(map[string]counterValues),
	}

	// Register all metrics
	registry.MustRegister(m.Slot)
	registry.MustRegister(m.Epoch)
	registry.MustRegister(m.CurrentPriceDollars)
	registry.MustRegister(m.PendingDepositsCount)
	registry.MustRegister(m.PendingDepositsValue)
	registry.MustRegister(m.PendingConsolidationsCount)
	registry.MustRegister(m.PendingWithdrawalsCount)
	registry.MustRegister(m.ValidatorStatusCount)
	registry.MustRegister(m.ValidatorStatusScaledCount)
	registry.MustRegister(m.ValidatorTypeCount)
	registry.MustRegister(m.ValidatorTypeScaledCount)
	registry.MustRegister(m.SlashedValidators)
	registry.MustRegister(m.MissedAttestations)
	registry.MustRegister(m.MissedAttestationsScaled)
	registry.MustRegister(m.SuboptimalSourcesRate)
	registry.MustRegister(m.SuboptimalTargetsRate)
	registry.MustRegister(m.SuboptimalHeadsRate)
	registry.MustRegister(m.BlockProposalsHeadTotal)
	registry.MustRegister(m.MissedBlockProposalsHeadTotal)
	registry.MustRegister(m.BlockProposalsFinalizedTotal)
	registry.MustRegister(m.MissedBlockProposalsFinalizedTotal)
	registry.MustRegister(m.FutureBlockProposals)
	registry.MustRegister(m.IdealConsensusRewardsGwei)
	registry.MustRegister(m.ActualConsensusRewardsGwei)
	registry.MustRegister(m.ConsensusRewardsRate)
	registry.MustRegister(m.MissedDutiesAtSlot)
	registry.MustRegister(m.MissedDutiesAtSlotScaled)
	registry.MustRegister(m.PerformedDutiesAtSlot)
	registry.MustRegister(m.PerformedDutiesAtSlotScaled)
	registry.MustRegister(m.DutiesRate)
	registry.MustRegister(m.DutiesRateScaled)
	registry.MustRegister(m.MissedConsecutiveAttestations)
	registry.MustRegister(m.MissedConsecutiveAttestationsScaled)

	return m
}

// UpdateMetrics updates Prometheus metrics from computed metrics
func (m *PrometheusMetrics) UpdateMetrics(metricsByLabel map[string]*MetricsByLabel, slot models.Slot, epoch models.Epoch, network string) {
	// Update slot and epoch (now with network label)
	m.Slot.WithLabelValues(network).Set(float64(slot))
	m.Epoch.WithLabelValues(network).Set(float64(epoch))

	// Note: Network-level metrics (price, pending deposits, etc.) should be set by the caller
	// as they require beacon client access which we don't have in this method.
	// These can be set separately via dedicated methods if needed:
	// - SetNetworkMetrics(network string, price float64, deposits, consolidations, withdrawals counts)

	// Reset scope-based metrics
	m.ValidatorStatusCount.Reset()
	m.ValidatorStatusScaledCount.Reset()
	m.ValidatorTypeCount.Reset()
	m.ValidatorTypeScaledCount.Reset()
	m.SlashedValidators.Reset()
	m.MissedAttestations.Reset()
	m.MissedAttestationsScaled.Reset()
	m.SuboptimalSourcesRate.Reset()
	m.SuboptimalTargetsRate.Reset()
	m.SuboptimalHeadsRate.Reset()
	m.FutureBlockProposals.Reset()
	m.ConsensusRewardsRate.Reset()
	m.DutiesRate.Reset()
	m.DutiesRateScaled.Reset()
	m.MissedConsecutiveAttestations.Reset()
	m.MissedConsecutiveAttestationsScaled.Reset()

	// Update metrics for each scope
	for label, metrics := range metricsByLabel {
		scope := label // Labels are already in the format "scope:watched", "scope:network", etc.

		// Validator status metrics
		for status, count := range metrics.StatusCounts {
			m.ValidatorStatusCount.WithLabelValues(scope, string(status), network).Set(float64(count))
		}
		for status, stake := range metrics.StatusStakes {
			// Scaled count = stake / 32 (since each validator has 32 ETH effective balance)
			scaledCount := stake / 32.0
			m.ValidatorStatusScaledCount.WithLabelValues(scope, string(status), network).Set(scaledCount)
		}

		// Validator type metrics (0x00 BLS, 0x01 execution, 0x02 compounding)
		for validatorType, count := range metrics.ValidatorTypeCounts {
			m.ValidatorTypeCount.WithLabelValues(scope, validatorType, network).Set(float64(count))
		}
		for validatorType, stake := range metrics.ValidatorTypeStakes {
			scaledCount := stake / 32.0
			m.ValidatorTypeScaledCount.WithLabelValues(scope, validatorType, network).Set(scaledCount)
		}

		// Slashed validators
		m.SlashedValidators.WithLabelValues(scope, network).Set(float64(metrics.SlashedCount))

		// Attestation metrics
		m.MissedAttestations.WithLabelValues(scope, network).Set(float64(metrics.MissedAttestations))
		m.MissedAttestationsScaled.WithLabelValues(scope, network).Set(metrics.MissedAttestationsStake / 32.0)

		// Calculate suboptimal rates
		if metrics.AttestationDuties > 0 {
			sourceRate := float64(metrics.SuboptimalSourceVotes) / float64(metrics.AttestationDuties)
			targetRate := float64(metrics.SuboptimalTargetVotes) / float64(metrics.AttestationDuties)
			headRate := float64(metrics.SuboptimalHeadVotes) / float64(metrics.AttestationDuties)

			m.SuboptimalSourcesRate.WithLabelValues(scope, network).Set(sourceRate)
			m.SuboptimalTargetsRate.WithLabelValues(scope, network).Set(targetRate)
			m.SuboptimalHeadsRate.WithLabelValues(scope, network).Set(headRate)
		}

		// Block proposal metrics
		m.FutureBlockProposals.WithLabelValues(scope, network).Set(float64(metrics.FutureBlockProposals))

		// Block proposal counters - increment based on delta from last seen value
		scopeKey := network + ":" + scope
		m.counterStateMu.Lock()
		lastValues, exists := m.counterState[scopeKey]

		// Calculate deltas for all counters
		proposedHeadDelta := uint64(0)
		missedHeadDelta := uint64(0)
		proposedFinalizedDelta := uint64(0)
		missedFinalizedDelta := uint64(0)

		if exists {
			// Only increment if values increased (handle potential resets)
			if metrics.ProposedBlocks >= lastValues.ProposedBlocks {
				proposedHeadDelta = metrics.ProposedBlocks - lastValues.ProposedBlocks
			}
			if metrics.MissedBlocks >= lastValues.MissedBlocks {
				missedHeadDelta = metrics.MissedBlocks - lastValues.MissedBlocks
			}
			if metrics.ProposedBlocksFinalized >= lastValues.ProposedBlocksFinalized {
				proposedFinalizedDelta = metrics.ProposedBlocksFinalized - lastValues.ProposedBlocksFinalized
			}
			if metrics.MissedBlocksFinalized >= lastValues.MissedBlocksFinalized {
				missedFinalizedDelta = metrics.MissedBlocksFinalized - lastValues.MissedBlocksFinalized
			}
		} else {
			// First time seeing this scope - use current values
			proposedHeadDelta = metrics.ProposedBlocks
			missedHeadDelta = metrics.MissedBlocks
			proposedFinalizedDelta = metrics.ProposedBlocksFinalized
			missedFinalizedDelta = metrics.MissedBlocksFinalized
		}

		// Update state
		m.counterState[scopeKey] = counterValues{
			ProposedBlocks:          metrics.ProposedBlocks,
			MissedBlocks:            metrics.MissedBlocks,
			ProposedBlocksFinalized: metrics.ProposedBlocksFinalized,
			MissedBlocksFinalized:   metrics.MissedBlocksFinalized,
		}
		m.counterStateMu.Unlock()

		// Increment counters by delta (note: label order is scope, network)
		// Always call Add() to initialize counters, even with 0, so they appear in metrics output
		m.BlockProposalsHeadTotal.WithLabelValues(scope, network).Add(float64(proposedHeadDelta))
		m.MissedBlockProposalsHeadTotal.WithLabelValues(scope, network).Add(float64(missedHeadDelta))
		m.BlockProposalsFinalizedTotal.WithLabelValues(scope, network).Add(float64(proposedFinalizedDelta))
		m.MissedBlockProposalsFinalizedTotal.WithLabelValues(scope, network).Add(float64(missedFinalizedDelta))

		// Reward metrics
		m.IdealConsensusRewardsGwei.WithLabelValues(scope, network).Set(float64(metrics.IdealConsensusRewards))
		m.ActualConsensusRewardsGwei.WithLabelValues(scope, network).Set(float64(metrics.ConsensusRewards))
		m.ConsensusRewardsRate.WithLabelValues(scope, network).Set(metrics.ConsensusRewardsRate)

		// Duty metrics at slot level (these track current epoch performance)
		m.PerformedDutiesAtSlot.WithLabelValues(scope, network).Set(float64(metrics.AttestationDutiesSuccess))
		m.MissedDutiesAtSlot.WithLabelValues(scope, network).Set(float64(metrics.AttestationDuties - metrics.AttestationDutiesSuccess))

		// Scaled versions
		successStake := float64(metrics.AttestationDutiesSuccess) * (metrics.StakeCount / float64(metrics.ValidatorCount))
		missedStake := float64(metrics.AttestationDuties-metrics.AttestationDutiesSuccess) * (metrics.StakeCount / float64(metrics.ValidatorCount))
		m.PerformedDutiesAtSlotScaled.WithLabelValues(scope, network).Set(successStake / 32.0)
		m.MissedDutiesAtSlotScaled.WithLabelValues(scope, network).Set(missedStake / 32.0)

		// Duty rate metrics
		m.DutiesRate.WithLabelValues(scope, network).Set(metrics.AttestationDutiesRate)
		if metrics.AttestationDutiesStake > 0 {
			// For scaled rate, we need to weight success by stake
			scaledSuccessRate := float64(metrics.AttestationDutiesSuccess) / float64(metrics.AttestationDuties)
			m.DutiesRateScaled.WithLabelValues(scope, network).Set(scaledSuccessRate)
		}

		// Consecutive missed attestations
		m.MissedConsecutiveAttestations.WithLabelValues(scope, network).Set(float64(metrics.MaxConsecutiveMissed))
		m.MissedConsecutiveAttestationsScaled.WithLabelValues(scope, network).Set(metrics.MaxConsecutiveMissedStake / 32.0)
	}
}

// SetNetworkMetrics sets network-level metrics that require external data
func (m *PrometheusMetrics) SetNetworkMetrics(network string, ethPriceDollars float64, pendingDepositsCount, pendingDepositsValue, pendingConsolidationsCount, pendingWithdrawalsCount float64) {
	if ethPriceDollars > 0 {
		m.CurrentPriceDollars.WithLabelValues(network).Set(ethPriceDollars)
	}
	m.PendingDepositsCount.WithLabelValues(network).Set(pendingDepositsCount)
	m.PendingDepositsValue.WithLabelValues(network).Set(pendingDepositsValue)
	m.PendingConsolidationsCount.WithLabelValues(network).Set(pendingConsolidationsCount)
	m.PendingWithdrawalsCount.WithLabelValues(network).Set(pendingWithdrawalsCount)
}
