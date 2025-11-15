package metrics

import (
	"runtime"
	"strings"
	"sync"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/validator"
)

// getValidatorType extracts the validator type from withdrawal credentials
// Returns "0", "1", or "2" for 0x00 (BLS), 0x01 (execution), 0x02 (compounding)
func getValidatorType(withdrawalCredentials string) string {
	// Withdrawal credentials is a hex string like "0x00..." or "0x01..." or "0x02..."
	if len(withdrawalCredentials) < 4 {
		return "0" // Default to 0x00 (BLS) if invalid
	}

	// Remove "0x" prefix and get first byte
	cleaned := strings.TrimPrefix(withdrawalCredentials, "0x")
	if len(cleaned) < 2 {
		return "0"
	}

	// Extract first byte (first 2 hex chars)
	firstByte := cleaned[0:2]

	switch firstByte {
	case "00":
		return "0" // BLS withdrawal credentials
	case "01":
		return "1" // Execution layer withdrawal address (0x01)
	case "02":
		return "2" // Compounding credentials (EIP-7251)
	default:
		return "0" // Default to BLS
	}
}

// MetricsByLabel represents aggregated metrics per label
type MetricsByLabel struct {
	Label string

	// Counts
	ValidatorCount             int
	StakeCount                 float64
	MissedAttestations         uint64
	MissedAttestationsStake    float64
	SuboptimalSourceVotes      uint64
	SuboptimalSourceVotesStake float64
	SuboptimalTargetVotes      uint64
	SuboptimalTargetVotesStake float64
	SuboptimalHeadVotes        uint64
	SuboptimalHeadVotesStake   float64
	ProposedBlocks             uint64
	ProposedBlocksFinalized    uint64
	MissedBlocks               uint64
	MissedBlocksFinalized      uint64
	FutureBlockProposals       uint64

	// Rewards
	IdealConsensusRewards models.Gwei       // Ideal is always positive
	ConsensusRewards      models.SignedGwei // Actual can be negative (penalties)
	ConsensusRewardsRate  float64

	// Duties
	AttestationDuties        uint64
	AttestationDutiesSuccess uint64
	AttestationDutiesRate    float64
	AttestationDutiesStake   float64 // Stake-weighted duties

	// Status breakdown
	StatusCounts map[models.ValidatorStatus]int
	StatusStakes map[models.ValidatorStatus]float64

	// Validator type breakdown (by withdrawal credentials)
	ValidatorTypeCounts map[string]int     // "0", "1", "2" for 0x00, 0x01, 0x02
	ValidatorTypeStakes map[string]float64 // Stake by type

	// Slashing
	SlashedCount int
	SlashedStake float64

	// Consecutive missed attestations
	MaxConsecutiveMissed       uint64  // Max consecutive missed
	MaxConsecutiveMissedStake  float64 // Stake-weighted max consecutive missed

	// Details for logging (limited to 5)
	MissedAttestationDetails []ValidatorDetail
	SuboptimalSourceDetails  []ValidatorDetail
	SuboptimalTargetDetails  []ValidatorDetail
	SuboptimalHeadDetails    []ValidatorDetail
	MissedBlockDetails       []ValidatorDetail
}

// ValidatorDetail represents a validator detail for logging
type ValidatorDetail struct {
	Index  models.ValidatorIndex
	Pubkey string
	Value  uint64
}

// ComputeMetrics computes metrics for all validators grouped by labels
// Uses concurrent processing for performance with large validator sets
func ComputeMetrics(validators []*validator.WatchedValidator, slot models.Slot) map[string]*MetricsByLabel {
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Split validators into chunks for parallel processing
	chunkSize := (len(validators) + numWorkers - 1) / numWorkers

	type workerResult struct {
		metrics map[string]*MetricsByLabel
	}

	resultsChan := make(chan workerResult, numWorkers)
	var wg sync.WaitGroup

	// Process chunks in parallel
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		if start >= len(validators) {
			break
		}

		end := start + chunkSize
		if end > len(validators) {
			end = len(validators)
		}

		wg.Add(1)
		go func(chunk []*validator.WatchedValidator) {
			defer wg.Done()

			// Process chunk
			localMetrics := make(map[string]*MetricsByLabel)

			for _, v := range chunk {
				for _, label := range v.Labels {
					metrics, ok := localMetrics[label]
					if !ok {
						metrics = &MetricsByLabel{
							Label:              label,
							StatusCounts:       make(map[models.ValidatorStatus]int),
							StatusStakes:       make(map[models.ValidatorStatus]float64),
							ValidatorTypeCounts: make(map[string]int),
							ValidatorTypeStakes: make(map[string]float64),
						}
						localMetrics[label] = metrics
					}

					// Check if validator is active (should be attesting)
					isActive := v.Status == models.StatusActiveOngoing ||
						v.Status == models.StatusActiveExiting ||
						v.Status == models.StatusActiveSlashed

					// Always count all validators for status breakdown
					metrics.ValidatorCount++
					metrics.StakeCount += v.Weight
					metrics.StatusCounts[v.Status]++
					metrics.StatusStakes[v.Status] += v.Weight

					// Track validator type from withdrawal credentials
					validatorType := getValidatorType(v.Data.WithdrawalCredentials)
					metrics.ValidatorTypeCounts[validatorType]++
					metrics.ValidatorTypeStakes[validatorType] += v.Weight

					// Track slashed validators
					if v.Data.Slashed {
						metrics.SlashedCount++
						metrics.SlashedStake += v.Weight
					}

					// Track max consecutive missed attestations
					if v.ConsecutiveMissedAttest > metrics.MaxConsecutiveMissed {
						metrics.MaxConsecutiveMissed = v.ConsecutiveMissedAttest
					}
					consecStakeWeighted := float64(v.ConsecutiveMissedAttest) * v.Weight
					if consecStakeWeighted > metrics.MaxConsecutiveMissedStake {
						metrics.MaxConsecutiveMissedStake = consecStakeWeighted
					}

					// Only aggregate performance metrics for ACTIVE validators
					if isActive {
						metrics.MissedAttestations += v.MissedAttestations
						metrics.MissedAttestationsStake += float64(v.MissedAttestations) * v.Weight
						metrics.SuboptimalSourceVotes += v.SuboptimalSourceVotes
						metrics.SuboptimalSourceVotesStake += float64(v.SuboptimalSourceVotes) * v.Weight
						metrics.SuboptimalTargetVotes += v.SuboptimalTargetVotes
						metrics.SuboptimalTargetVotesStake += float64(v.SuboptimalTargetVotes) * v.Weight
						metrics.SuboptimalHeadVotes += v.SuboptimalHeadVotes
						metrics.SuboptimalHeadVotesStake += float64(v.SuboptimalHeadVotes) * v.Weight
						metrics.MissedBlocksFinalized += v.MissedBlocksFinalized
						metrics.FutureBlockProposals += v.FutureBlockProposals
						metrics.IdealConsensusRewards += v.IdealConsensusRewards
						metrics.ConsensusRewards += v.ConsensusRewards
						metrics.AttestationDuties += v.AttestationDuties
						metrics.AttestationDutiesSuccess += v.AttestationDutiesSuccess
						metrics.AttestationDutiesStake += float64(v.AttestationDuties) * v.Weight
					}

					// Block proposals should be counted regardless of validator status
					// A validator can propose a block even when exiting or in other states
					metrics.ProposedBlocks += v.ProposedBlocks
					metrics.ProposedBlocksFinalized += v.ProposedBlocksFinalized
					metrics.MissedBlocks += v.MissedBlocks

					// Collect details (limited to 5 per label)
					if v.MissedAttestations > 0 && len(metrics.MissedAttestationDetails) < 5 {
						metrics.MissedAttestationDetails = append(metrics.MissedAttestationDetails, ValidatorDetail{
							Index:  v.Index,
							Pubkey: v.Data.Pubkey,
							Value:  v.MissedAttestations,
						})
					}
					if v.SuboptimalSourceVotes > 0 && len(metrics.SuboptimalSourceDetails) < 5 {
						metrics.SuboptimalSourceDetails = append(metrics.SuboptimalSourceDetails, ValidatorDetail{
							Index:  v.Index,
							Pubkey: v.Data.Pubkey,
							Value:  v.SuboptimalSourceVotes,
						})
					}
					if v.SuboptimalTargetVotes > 0 && len(metrics.SuboptimalTargetDetails) < 5 {
						metrics.SuboptimalTargetDetails = append(metrics.SuboptimalTargetDetails, ValidatorDetail{
							Index:  v.Index,
							Pubkey: v.Data.Pubkey,
							Value:  v.SuboptimalTargetVotes,
						})
					}
					if v.SuboptimalHeadVotes > 0 && len(metrics.SuboptimalHeadDetails) < 5 {
						metrics.SuboptimalHeadDetails = append(metrics.SuboptimalHeadDetails, ValidatorDetail{
							Index:  v.Index,
							Pubkey: v.Data.Pubkey,
							Value:  v.SuboptimalHeadVotes,
						})
					}
					if v.MissedBlocks > 0 && len(metrics.MissedBlockDetails) < 5 {
						metrics.MissedBlockDetails = append(metrics.MissedBlockDetails, ValidatorDetail{
							Index:  v.Index,
							Pubkey: v.Data.Pubkey,
							Value:  v.MissedBlocks,
						})
					}
				}
			}

			resultsChan <- workerResult{metrics: localMetrics}
		}(validators[start:end])
	}

	// Wait for all workers and close channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Merge results from all workers
	finalMetrics := make(map[string]*MetricsByLabel)

	for result := range resultsChan {
		for label, metrics := range result.metrics {
			if _, ok := finalMetrics[label]; !ok {
				finalMetrics[label] = &MetricsByLabel{
					Label:              label,
					StatusCounts:       make(map[models.ValidatorStatus]int),
					StatusStakes:       make(map[models.ValidatorStatus]float64),
					ValidatorTypeCounts: make(map[string]int),
					ValidatorTypeStakes: make(map[string]float64),
				}
			}

			fm := finalMetrics[label]

			// Merge metrics
			fm.ValidatorCount += metrics.ValidatorCount
			fm.StakeCount += metrics.StakeCount
			fm.MissedAttestations += metrics.MissedAttestations
			fm.MissedAttestationsStake += metrics.MissedAttestationsStake
			fm.SuboptimalSourceVotes += metrics.SuboptimalSourceVotes
			fm.SuboptimalSourceVotesStake += metrics.SuboptimalSourceVotesStake
			fm.SuboptimalTargetVotes += metrics.SuboptimalTargetVotes
			fm.SuboptimalTargetVotesStake += metrics.SuboptimalTargetVotesStake
			fm.SuboptimalHeadVotes += metrics.SuboptimalHeadVotes
			fm.SuboptimalHeadVotesStake += metrics.SuboptimalHeadVotesStake
			fm.ProposedBlocks += metrics.ProposedBlocks
			fm.ProposedBlocksFinalized += metrics.ProposedBlocksFinalized
			fm.MissedBlocks += metrics.MissedBlocks
			fm.MissedBlocksFinalized += metrics.MissedBlocksFinalized
			fm.FutureBlockProposals += metrics.FutureBlockProposals
			fm.IdealConsensusRewards += metrics.IdealConsensusRewards
			fm.ConsensusRewards += metrics.ConsensusRewards
			fm.AttestationDuties += metrics.AttestationDuties
			fm.AttestationDutiesSuccess += metrics.AttestationDutiesSuccess
			fm.AttestationDutiesStake += metrics.AttestationDutiesStake

			// Merge slashing metrics
			fm.SlashedCount += metrics.SlashedCount
			fm.SlashedStake += metrics.SlashedStake

			// Merge consecutive missed attestations (take max)
			if metrics.MaxConsecutiveMissed > fm.MaxConsecutiveMissed {
				fm.MaxConsecutiveMissed = metrics.MaxConsecutiveMissed
			}
			if metrics.MaxConsecutiveMissedStake > fm.MaxConsecutiveMissedStake {
				fm.MaxConsecutiveMissedStake = metrics.MaxConsecutiveMissedStake
			}

			// Merge status counts
			for status, count := range metrics.StatusCounts {
				fm.StatusCounts[status] += count
			}
			for status, stake := range metrics.StatusStakes {
				fm.StatusStakes[status] += stake
			}

			// Merge validator type counts
			for validatorType, count := range metrics.ValidatorTypeCounts {
				fm.ValidatorTypeCounts[validatorType] += count
			}
			for validatorType, stake := range metrics.ValidatorTypeStakes {
				fm.ValidatorTypeStakes[validatorType] += stake
			}

			// Merge details (keep first 5)
			for _, detail := range metrics.MissedAttestationDetails {
				if len(fm.MissedAttestationDetails) < 5 {
					fm.MissedAttestationDetails = append(fm.MissedAttestationDetails, detail)
				}
			}
			for _, detail := range metrics.SuboptimalSourceDetails {
				if len(fm.SuboptimalSourceDetails) < 5 {
					fm.SuboptimalSourceDetails = append(fm.SuboptimalSourceDetails, detail)
				}
			}
			for _, detail := range metrics.SuboptimalTargetDetails {
				if len(fm.SuboptimalTargetDetails) < 5 {
					fm.SuboptimalTargetDetails = append(fm.SuboptimalTargetDetails, detail)
				}
			}
			for _, detail := range metrics.SuboptimalHeadDetails {
				if len(fm.SuboptimalHeadDetails) < 5 {
					fm.SuboptimalHeadDetails = append(fm.SuboptimalHeadDetails, detail)
				}
			}
			for _, detail := range metrics.MissedBlockDetails {
				if len(fm.MissedBlockDetails) < 5 {
					fm.MissedBlockDetails = append(fm.MissedBlockDetails, detail)
				}
			}
		}
	}

	// Calculate rates
	for _, metrics := range finalMetrics {
		if metrics.IdealConsensusRewards > 0 {
			metrics.ConsensusRewardsRate = float64(metrics.ConsensusRewards) / float64(metrics.IdealConsensusRewards)
		}
		if metrics.AttestationDuties > 0 {
			metrics.AttestationDutiesRate = float64(metrics.AttestationDutiesSuccess) / float64(metrics.AttestationDuties)
		}
	}

	return finalMetrics
}

// ComputeNetworkMetrics computes aggregate network-wide metrics from all validators
func ComputeNetworkMetrics(allValidators []models.Validator) *MetricsByLabel {
	metrics := &MetricsByLabel{
		Label:              "scope:all-network",
		StatusCounts:       make(map[models.ValidatorStatus]int),
		StatusStakes:       make(map[models.ValidatorStatus]float64),
		ValidatorTypeCounts: make(map[string]int),
		ValidatorTypeStakes: make(map[string]float64),
	}

	for _, v := range allValidators {
		weight := float64(v.Data.EffectiveBalance) / 32_000_000_000.0

		metrics.ValidatorCount++
		metrics.StakeCount += weight
		metrics.StatusCounts[v.Status]++
		metrics.StatusStakes[v.Status] += weight

		// Track validator type
		validatorType := getValidatorType(v.Data.WithdrawalCredentials)
		metrics.ValidatorTypeCounts[validatorType]++
		metrics.ValidatorTypeStakes[validatorType] += weight

		// Track slashed validators
		if v.Data.Slashed {
			metrics.SlashedCount++
			metrics.SlashedStake += weight
		}
	}

	return metrics
}
