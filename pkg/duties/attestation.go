package duties

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
)

// DecodeBitVector decodes an SSZ BitVector from hex string to a map of set positions
func DecodeBitVector(bitVectorHex string, size int) (map[int]bool, error) {
	// Remove 0x prefix if present
	bitVectorHex = strings.TrimPrefix(bitVectorHex, "0x")

	// Decode hex to bytes
	bytes, err := hex.DecodeString(bitVectorHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	result := make(map[int]bool)

	// Process each byte
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			bitPos := i*8 + j
			if bitPos >= size {
				break
			}

			// Check if bit is set (LSB first within each byte)
			if b&(1<<j) != 0 {
				result[bitPos] = true
			}
		}
	}

	return result, nil
}

// ProcessAttestations processes attestations for a slot and returns validator indices that attested
// Post-Electra format: attestations can span multiple committees using committee_bits
func ProcessAttestations(attestations []models.Attestation, committees []models.Committee) (map[models.ValidatorIndex]bool, error) {
	attested := make(map[models.ValidatorIndex]bool)

	// Build committee index map (committees are indexed 0..63 per slot)
	committeeMap := make(map[uint64]models.Committee)
	for _, committee := range committees {
		committeeMap[committee.Index] = committee
	}

	for _, attestation := range attestations {
		// Post-Electra: committee_bits is a 64-bit bitfield indicating which committees are attesting
		// If committee_bits is empty/missing, fall back to single committee (pre-Electra)
		if attestation.CommitteeBits == "" || attestation.CommitteeBits == "0x" {
			// Pre-Electra format: single committee per attestation
			committee, ok := committeeMap[attestation.Data.Index]
			if !ok {
				continue
			}

			// Decode aggregation bits
			bits, err := DecodeBitVector(attestation.AggregationBits, len(committee.Validators))
			if err != nil {
				return nil, fmt.Errorf("failed to decode aggregation bits: %w", err)
			}

			// Mark validators as attested
			for pos, isSet := range bits {
				if isSet && pos < len(committee.Validators) {
					// Parse validator index from string
					var validatorIndex models.ValidatorIndex
					fmt.Sscanf(committee.Validators[pos], "%d", &validatorIndex)
					attested[validatorIndex] = true
				}
			}
		} else {
			// Post-Electra format: decode committee_bits to find active committees
			// committee_bits is a 64-bit bitfield (one bit per committee index 0-63)
			committeeBits, err := DecodeBitVector(attestation.CommitteeBits, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to decode committee bits: %w", err)
			}

			// Decode aggregation bits (aggregated across all active committees)
			// We need to calculate total size first
			totalValidators := 0
			activeCommittees := make([]models.Committee, 0)
			for committeeIndex := 0; committeeIndex < 64; committeeIndex++ {
				if committeeBits[committeeIndex] {
					committee, ok := committeeMap[uint64(committeeIndex)]
					if ok {
						activeCommittees = append(activeCommittees, committee)
						totalValidators += len(committee.Validators)
					}
				}
			}

			if len(activeCommittees) == 0 {
				continue
			}

			// Decode aggregation bits
			aggregationBits, err := DecodeBitVector(attestation.AggregationBits, totalValidators)
			if err != nil {
				return nil, fmt.Errorf("failed to decode aggregation bits: %w", err)
			}

			// Process each active committee with committee_offset
			// This follows the Python logic at lines 112-120 of duties.py
			committeeOffset := 0
			for _, committee := range activeCommittees {
				// For each validator in this committee
				for i := 0; i < len(committee.Validators); i++ {
					bitPosition := committeeOffset + i

					// Check if this validator attested
					if aggregationBits[bitPosition] {
						// Parse validator index from string
						var validatorIndex models.ValidatorIndex
						fmt.Sscanf(committee.Validators[i], "%d", &validatorIndex)
						attested[validatorIndex] = true
					}
				}

				// Move offset for next committee
				committeeOffset += len(committee.Validators)
			}
		}
	}

	return attested, nil
}

// ProcessRewards processes reward data and updates validator metrics
func ProcessRewards(rewards *models.RewardsResponse, validators map[models.ValidatorIndex]models.Gwei) (map[models.ValidatorIndex]RewardData, error) {
	result := make(map[models.ValidatorIndex]RewardData)

	// Build ideal rewards map by effective balance
	idealByBalance := make(map[models.Gwei]models.IdealReward)
	for _, ideal := range rewards.Data.IdealRewards {
		idealByBalance[ideal.EffectiveBalance] = ideal
	}

	// Build total rewards map
	totalRewardsMap := make(map[models.ValidatorIndex]models.TotalReward)
	for _, total := range rewards.Data.TotalRewards {
		totalRewardsMap[total.ValidatorIndex] = total
	}

	// Calculate suboptimal attestations
	for idx, effectiveBalance := range validators {
		total, ok := totalRewardsMap[idx]
		if !ok {
			continue
		}

		// Find matching ideal reward using validator's actual effective balance
		ideal, ok := idealByBalance[effectiveBalance]
		if !ok {
			// If exact match not found, use 32 ETH as fallback
			ideal, ok = idealByBalance[32_000_000_000]
			if !ok {
				// Last resort: pick any available ideal reward
				for _, idealReward := range idealByBalance {
					ideal = idealReward
					break
				}
			}
		}

		data := RewardData{
			IdealHead:   ideal.Head,
			IdealTarget: ideal.Target,
			IdealSource: ideal.Source,
			ActualHead:  total.Head,
			ActualTarget: total.Target,
			ActualSource: total.Source,
		}

		// Calculate suboptimal votes (compare signed actual vs unsigned ideal)
		if total.Source < models.SignedGwei(ideal.Source) {
			data.SuboptimalSource = true
		}
		if total.Target < models.SignedGwei(ideal.Target) {
			data.SuboptimalTarget = true
		}
		if total.Head < models.SignedGwei(ideal.Head) {
			data.SuboptimalHead = true
		}

		data.IdealTotal = ideal.Source + ideal.Target + ideal.Head
		data.ActualTotal = total.Source + total.Target + total.Head

		result[idx] = data
	}

	return result, nil
}

// RewardData represents reward information for a validator
type RewardData struct {
	IdealHead        models.Gwei
	IdealTarget      models.Gwei
	IdealSource      models.Gwei
	IdealTotal       models.Gwei
	ActualHead       models.SignedGwei
	ActualTarget     models.SignedGwei
	ActualSource     models.SignedGwei
	ActualTotal      models.SignedGwei
	SuboptimalSource bool
	SuboptimalTarget bool
	SuboptimalHead   bool
}

// ProcessLiveness processes validator liveness data
func ProcessLiveness(liveness []models.ValidatorLiveness) map[models.ValidatorIndex]bool {
	result := make(map[models.ValidatorIndex]bool)

	for _, l := range liveness {
		result[l.Index] = l.IsLive
	}

	return result
}

// BitvectorToBigInt converts a hex bitvector to a big integer
func BitvectorToBigInt(hexStr string) (*big.Int, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	val, ok := new(big.Int).SetString(hexStr, 16)
	if !ok {
		return nil, fmt.Errorf("failed to parse bitvector: %s", hexStr)
	}
	return val, nil
}
