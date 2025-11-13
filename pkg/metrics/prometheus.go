package metrics

import (
	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics holds all Prometheus metric collectors
type PrometheusMetrics struct {
	// Slot and epoch metrics
	CurrentSlot  prometheus.Gauge
	CurrentEpoch prometheus.Gauge

	// Validator counts
	ValidatorCount *prometheus.GaugeVec
	StakeCount     *prometheus.GaugeVec

	// Attestation metrics
	MissedAttestations      *prometheus.GaugeVec
	MissedAttestationsStake *prometheus.GaugeVec
	SuboptimalSourceVotes   *prometheus.GaugeVec
	SuboptimalSourceStake   *prometheus.GaugeVec
	SuboptimalTargetVotes   *prometheus.GaugeVec
	SuboptimalTargetStake   *prometheus.GaugeVec
	SuboptimalHeadVotes     *prometheus.GaugeVec
	SuboptimalHeadStake     *prometheus.GaugeVec

	// Block production metrics
	ProposedBlocks          *prometheus.GaugeVec
	ProposedBlocksFinalized *prometheus.GaugeVec
	MissedBlocks            *prometheus.GaugeVec
	MissedBlocksFinalized   *prometheus.GaugeVec
	FutureBlockProposals    *prometheus.GaugeVec

	// Reward metrics
	IdealConsensusRewards *prometheus.GaugeVec
	ConsensusRewards      *prometheus.GaugeVec
	ConsensusRewardsRate  *prometheus.GaugeVec

	// Duty metrics
	AttestationDuties     *prometheus.GaugeVec
	AttestationDutiesSuccess *prometheus.GaugeVec
	AttestationDutiesRate *prometheus.GaugeVec

	// Status metrics
	StatusCount *prometheus.GaugeVec
	StatusStake *prometheus.GaugeVec

	// Consecutive missed attestations
	ConsecutiveMissedAttestations *prometheus.GaugeVec
}

// NewPrometheusMetrics creates and registers all Prometheus metrics
func NewPrometheusMetrics(registry *prometheus.Registry) *PrometheusMetrics {
	m := &PrometheusMetrics{
		CurrentSlot: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_current_slot",
			Help: "Current slot number",
		}),
		CurrentEpoch: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_current_epoch",
			Help: "Current epoch number",
		}),
		ValidatorCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_validator_count",
			Help: "Number of validators",
		}, []string{"label"}),
		StakeCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_stake_count",
			Help: "Total stake (in units of 32 ETH)",
		}, []string{"label"}),
		MissedAttestations: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_missed_attestations",
			Help: "Number of missed attestations",
		}, []string{"label"}),
		MissedAttestationsStake: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_missed_attestations_stake",
			Help: "Stake-weighted missed attestations",
		}, []string{"label"}),
		SuboptimalSourceVotes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_suboptimal_source_votes",
			Help: "Number of suboptimal source votes",
		}, []string{"label"}),
		SuboptimalSourceStake: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_suboptimal_source_stake",
			Help: "Stake-weighted suboptimal source votes",
		}, []string{"label"}),
		SuboptimalTargetVotes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_suboptimal_target_votes",
			Help: "Number of suboptimal target votes",
		}, []string{"label"}),
		SuboptimalTargetStake: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_suboptimal_target_stake",
			Help: "Stake-weighted suboptimal target votes",
		}, []string{"label"}),
		SuboptimalHeadVotes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_suboptimal_head_votes",
			Help: "Number of suboptimal head votes",
		}, []string{"label"}),
		SuboptimalHeadStake: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_suboptimal_head_stake",
			Help: "Stake-weighted suboptimal head votes",
		}, []string{"label"}),
		ProposedBlocks: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_proposed_blocks",
			Help: "Number of proposed blocks",
		}, []string{"label"}),
		ProposedBlocksFinalized: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_proposed_blocks_finalized",
			Help: "Number of proposed blocks that were finalized",
		}, []string{"label"}),
		MissedBlocks: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_missed_blocks",
			Help: "Number of missed block proposals",
		}, []string{"label"}),
		MissedBlocksFinalized: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_missed_blocks_finalized",
			Help: "Number of missed block proposals (finalized)",
		}, []string{"label"}),
		FutureBlockProposals: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_future_block_proposals",
			Help: "Number of upcoming block proposals",
		}, []string{"label"}),
		IdealConsensusRewards: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_ideal_consensus_rewards_gwei",
			Help: "Ideal consensus rewards in Gwei",
		}, []string{"label"}),
		ConsensusRewards: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_consensus_rewards_gwei",
			Help: "Actual consensus rewards in Gwei",
		}, []string{"label"}),
		ConsensusRewardsRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_consensus_rewards_rate",
			Help: "Consensus rewards rate (actual/ideal)",
		}, []string{"label"}),
		AttestationDuties: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_attestation_duties",
			Help: "Total number of attestation duties",
		}, []string{"label"}),
		AttestationDutiesSuccess: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_attestation_duties_success",
			Help: "Number of successful attestation duties",
		}, []string{"label"}),
		AttestationDutiesRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_attestation_duties_rate",
			Help: "Attestation duties success rate",
		}, []string{"label"}),
		StatusCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_status_count",
			Help: "Number of validators by status",
		}, []string{"label", "status"}),
		StatusStake: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_status_stake",
			Help: "Stake by validator status",
		}, []string{"label", "status"}),
		ConsecutiveMissedAttestations: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "eth_validator_watcher_consecutive_missed_attestations",
			Help: "Maximum consecutive missed attestations",
		}, []string{"label", "validator_index", "pubkey"}),
	}

	// Register all metrics
	registry.MustRegister(m.CurrentSlot)
	registry.MustRegister(m.CurrentEpoch)
	registry.MustRegister(m.ValidatorCount)
	registry.MustRegister(m.StakeCount)
	registry.MustRegister(m.MissedAttestations)
	registry.MustRegister(m.MissedAttestationsStake)
	registry.MustRegister(m.SuboptimalSourceVotes)
	registry.MustRegister(m.SuboptimalSourceStake)
	registry.MustRegister(m.SuboptimalTargetVotes)
	registry.MustRegister(m.SuboptimalTargetStake)
	registry.MustRegister(m.SuboptimalHeadVotes)
	registry.MustRegister(m.SuboptimalHeadStake)
	registry.MustRegister(m.ProposedBlocks)
	registry.MustRegister(m.ProposedBlocksFinalized)
	registry.MustRegister(m.MissedBlocks)
	registry.MustRegister(m.MissedBlocksFinalized)
	registry.MustRegister(m.FutureBlockProposals)
	registry.MustRegister(m.IdealConsensusRewards)
	registry.MustRegister(m.ConsensusRewards)
	registry.MustRegister(m.ConsensusRewardsRate)
	registry.MustRegister(m.AttestationDuties)
	registry.MustRegister(m.AttestationDutiesSuccess)
	registry.MustRegister(m.AttestationDutiesRate)
	registry.MustRegister(m.StatusCount)
	registry.MustRegister(m.StatusStake)
	registry.MustRegister(m.ConsecutiveMissedAttestations)

	return m
}

// UpdateMetrics updates Prometheus metrics from computed metrics
func (m *PrometheusMetrics) UpdateMetrics(metricsByLabel map[string]*MetricsByLabel, slot models.Slot, epoch models.Epoch) {
	// Update slot and epoch
	m.CurrentSlot.Set(float64(slot))
	m.CurrentEpoch.Set(float64(epoch))

	// Reset label-based metrics
	m.ValidatorCount.Reset()
	m.StakeCount.Reset()
	m.MissedAttestations.Reset()
	m.MissedAttestationsStake.Reset()
	m.SuboptimalSourceVotes.Reset()
	m.SuboptimalSourceStake.Reset()
	m.SuboptimalTargetVotes.Reset()
	m.SuboptimalTargetStake.Reset()
	m.SuboptimalHeadVotes.Reset()
	m.SuboptimalHeadStake.Reset()
	m.ProposedBlocks.Reset()
	m.ProposedBlocksFinalized.Reset()
	m.MissedBlocks.Reset()
	m.MissedBlocksFinalized.Reset()
	m.FutureBlockProposals.Reset()
	m.IdealConsensusRewards.Reset()
	m.ConsensusRewards.Reset()
	m.ConsensusRewardsRate.Reset()
	m.AttestationDuties.Reset()
	m.AttestationDutiesSuccess.Reset()
	m.AttestationDutiesRate.Reset()
	m.StatusCount.Reset()
	m.StatusStake.Reset()

	// Update metrics for each label
	for label, metrics := range metricsByLabel {
		m.ValidatorCount.WithLabelValues(label).Set(float64(metrics.ValidatorCount))
		m.StakeCount.WithLabelValues(label).Set(metrics.StakeCount)
		m.MissedAttestations.WithLabelValues(label).Set(float64(metrics.MissedAttestations))
		m.MissedAttestationsStake.WithLabelValues(label).Set(metrics.MissedAttestationsStake)
		m.SuboptimalSourceVotes.WithLabelValues(label).Set(float64(metrics.SuboptimalSourceVotes))
		m.SuboptimalSourceStake.WithLabelValues(label).Set(metrics.SuboptimalSourceVotesStake)
		m.SuboptimalTargetVotes.WithLabelValues(label).Set(float64(metrics.SuboptimalTargetVotes))
		m.SuboptimalTargetStake.WithLabelValues(label).Set(metrics.SuboptimalTargetVotesStake)
		m.SuboptimalHeadVotes.WithLabelValues(label).Set(float64(metrics.SuboptimalHeadVotes))
		m.SuboptimalHeadStake.WithLabelValues(label).Set(metrics.SuboptimalHeadVotesStake)
		m.ProposedBlocks.WithLabelValues(label).Set(float64(metrics.ProposedBlocks))
		m.ProposedBlocksFinalized.WithLabelValues(label).Set(float64(metrics.ProposedBlocksFinalized))
		m.MissedBlocks.WithLabelValues(label).Set(float64(metrics.MissedBlocks))
		m.MissedBlocksFinalized.WithLabelValues(label).Set(float64(metrics.MissedBlocksFinalized))
		m.FutureBlockProposals.WithLabelValues(label).Set(float64(metrics.FutureBlockProposals))
		m.IdealConsensusRewards.WithLabelValues(label).Set(float64(metrics.IdealConsensusRewards))
		m.ConsensusRewards.WithLabelValues(label).Set(float64(metrics.ConsensusRewards))
		m.ConsensusRewardsRate.WithLabelValues(label).Set(metrics.ConsensusRewardsRate)
		m.AttestationDuties.WithLabelValues(label).Set(float64(metrics.AttestationDuties))
		m.AttestationDutiesSuccess.WithLabelValues(label).Set(float64(metrics.AttestationDutiesSuccess))
		m.AttestationDutiesRate.WithLabelValues(label).Set(metrics.AttestationDutiesRate)

		// Update status metrics
		for status, count := range metrics.StatusCounts {
			m.StatusCount.WithLabelValues(label, string(status)).Set(float64(count))
		}
		for status, stake := range metrics.StatusStakes {
			m.StatusStake.WithLabelValues(label, string(status)).Set(stake)
		}
	}
}
