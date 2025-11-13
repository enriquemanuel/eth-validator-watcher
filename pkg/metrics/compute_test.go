package metrics

import (
	"testing"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/validator"
)

func TestComputeMetrics(t *testing.T) {
	// Create test validators
	validators := []*validator.WatchedValidator{
		{
			Validator: models.Validator{
				Index:   100,
				Balance: 32000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels:                []string{"scope:watched", "vc:val1"},
			Weight:                1.0,
			MissedAttestations:    3,
			SuboptimalSourceVotes: 1,
			SuboptimalTargetVotes: 2,
			SuboptimalHeadVotes:   1,
			ProposedBlocks:        5,
			MissedBlocks:          1,
			IdealConsensusRewards: 1000000,
			ConsensusRewards:      950000,
		},
		{
			Validator: models.Validator{
				Index:   200,
				Balance: 32000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels:                []string{"scope:watched", "vc:val2"},
			Weight:                1.0,
			MissedAttestations:    1,
			SuboptimalSourceVotes: 0,
			SuboptimalTargetVotes: 1,
			SuboptimalHeadVotes:   0,
			ProposedBlocks:        3,
			MissedBlocks:          0,
			IdealConsensusRewards: 1000000,
			ConsensusRewards:      980000,
		},
	}

	metricsByLabel := ComputeMetrics(validators, 1000)

	// Check scope:watched metrics
	watched, ok := metricsByLabel["scope:watched"]
	if !ok {
		t.Fatal("Expected to find scope:watched metrics")
	}

	if watched.ValidatorCount != 2 {
		t.Errorf("Expected 2 validators, got %d", watched.ValidatorCount)
	}

	if watched.StakeCount != 2.0 {
		t.Errorf("Expected stake count 2.0, got %f", watched.StakeCount)
	}

	if watched.MissedAttestations != 4 {
		t.Errorf("Expected 4 missed attestations, got %d", watched.MissedAttestations)
	}

	if watched.SuboptimalSourceVotes != 1 {
		t.Errorf("Expected 1 suboptimal source vote, got %d", watched.SuboptimalSourceVotes)
	}

	if watched.SuboptimalTargetVotes != 3 {
		t.Errorf("Expected 3 suboptimal target votes, got %d", watched.SuboptimalTargetVotes)
	}

	if watched.ProposedBlocks != 8 {
		t.Errorf("Expected 8 proposed blocks, got %d", watched.ProposedBlocks)
	}

	if watched.MissedBlocks != 1 {
		t.Errorf("Expected 1 missed block, got %d", watched.MissedBlocks)
	}

	// Check rewards rate calculation
	expectedRate := float64(950000+980000) / float64(1000000+1000000)
	if watched.ConsensusRewardsRate != expectedRate {
		t.Errorf("Expected consensus rewards rate %f, got %f", expectedRate, watched.ConsensusRewardsRate)
	}

	// Check vc:val1 metrics
	val1, ok := metricsByLabel["vc:val1"]
	if !ok {
		t.Fatal("Expected to find vc:val1 metrics")
	}

	if val1.ValidatorCount != 1 {
		t.Errorf("Expected 1 validator in vc:val1, got %d", val1.ValidatorCount)
	}

	if val1.MissedAttestations != 3 {
		t.Errorf("Expected 3 missed attestations in vc:val1, got %d", val1.MissedAttestations)
	}
}

func TestComputeMetricsStakeWeighting(t *testing.T) {
	validators := []*validator.WatchedValidator{
		{
			Validator: models.Validator{
				Index:   100,
				Balance: 32000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels:             []string{"scope:watched"},
			Weight:             1.0, // 32 ETH
			MissedAttestations: 2,
		},
		{
			Validator: models.Validator{
				Index:   200,
				Balance: 16000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels:             []string{"scope:watched"},
			Weight:             0.5, // 16 ETH
			MissedAttestations: 2,
		},
	}

	metricsByLabel := ComputeMetrics(validators, 1000)

	watched := metricsByLabel["scope:watched"]

	// Total missed: 2 + 2 = 4
	if watched.MissedAttestations != 4 {
		t.Errorf("Expected 4 missed attestations, got %d", watched.MissedAttestations)
	}

	// Stake-weighted: 2*1.0 + 2*0.5 = 3.0
	expectedStake := 3.0
	if watched.MissedAttestationsStake != expectedStake {
		t.Errorf("Expected stake-weighted missed attestations %f, got %f", expectedStake, watched.MissedAttestationsStake)
	}
}

func TestComputeMetricsStatusCounts(t *testing.T) {
	validators := []*validator.WatchedValidator{
		{
			Validator: models.Validator{
				Index:   100,
				Balance: 32000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels: []string{"scope:watched"},
			Weight: 1.0,
		},
		{
			Validator: models.Validator{
				Index:   200,
				Balance: 32000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels: []string{"scope:watched"},
			Weight: 1.0,
		},
		{
			Validator: models.Validator{
				Index:   300,
				Balance: 32000000000,
				Status:  models.StatusActiveExiting,
			},
			Labels: []string{"scope:watched"},
			Weight: 1.0,
		},
	}

	metricsByLabel := ComputeMetrics(validators, 1000)

	watched := metricsByLabel["scope:watched"]

	if watched.StatusCounts[models.StatusActiveOngoing] != 2 {
		t.Errorf("Expected 2 active ongoing validators, got %d", watched.StatusCounts[models.StatusActiveOngoing])
	}

	if watched.StatusCounts[models.StatusActiveExiting] != 1 {
		t.Errorf("Expected 1 active exiting validator, got %d", watched.StatusCounts[models.StatusActiveExiting])
	}

	if watched.StatusStakes[models.StatusActiveOngoing] != 2.0 {
		t.Errorf("Expected 2.0 active ongoing stake, got %f", watched.StatusStakes[models.StatusActiveOngoing])
	}
}

func TestComputeNetworkMetrics(t *testing.T) {
	validators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
		{
			Index:   200,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
		{
			Index:   300,
			Balance: 32000000000,
			Status:  models.StatusPendingQueued,
		},
	}
	for i := range validators {
		validators[i].Data.EffectiveBalance = 32000000000
	}

	metrics := ComputeNetworkMetrics(validators)

	if metrics.ValidatorCount != 3 {
		t.Errorf("Expected 3 validators, got %d", metrics.ValidatorCount)
	}

	if metrics.StakeCount != 3.0 {
		t.Errorf("Expected stake count 3.0, got %f", metrics.StakeCount)
	}

	if metrics.StatusCounts[models.StatusActiveOngoing] != 2 {
		t.Errorf("Expected 2 active ongoing, got %d", metrics.StatusCounts[models.StatusActiveOngoing])
	}

	if metrics.StatusCounts[models.StatusPendingQueued] != 1 {
		t.Errorf("Expected 1 pending queued, got %d", metrics.StatusCounts[models.StatusPendingQueued])
	}
}

func TestComputeMetricsConcurrency(t *testing.T) {
	// Create a large set of validators to test concurrent processing
	validators := make([]*validator.WatchedValidator, 10000)
	for i := 0; i < 10000; i++ {
		validators[i] = &validator.WatchedValidator{
			Validator: models.Validator{
				Index:   models.ValidatorIndex(i),
				Balance: 32000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels:                []string{"scope:watched"},
			Weight:                1.0,
			MissedAttestations:    uint64(i % 5),
			SuboptimalSourceVotes: uint64(i % 3),
			ProposedBlocks:        uint64(i % 10),
		}
	}

	metricsByLabel := ComputeMetrics(validators, 1000)

	watched := metricsByLabel["scope:watched"]

	if watched.ValidatorCount != 10000 {
		t.Errorf("Expected 10000 validators, got %d", watched.ValidatorCount)
	}

	// Verify stake count
	if watched.StakeCount != 10000.0 {
		t.Errorf("Expected stake count 10000.0, got %f", watched.StakeCount)
	}
}

func BenchmarkComputeMetrics(b *testing.B) {
	validators := make([]*validator.WatchedValidator, 1000)
	for i := 0; i < 1000; i++ {
		validators[i] = &validator.WatchedValidator{
			Validator: models.Validator{
				Index:   models.ValidatorIndex(i),
				Balance: 32000000000,
				Status:  models.StatusActiveOngoing,
			},
			Labels:                []string{"scope:watched", "vc:val1", "region:us"},
			Weight:                1.0,
			MissedAttestations:    uint64(i % 5),
			SuboptimalSourceVotes: uint64(i % 3),
			ProposedBlocks:        uint64(i % 10),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeMetrics(validators, 1000)
	}
}
