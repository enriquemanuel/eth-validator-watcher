package duties

import (
	"testing"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
)

func TestDecodeBitVector(t *testing.T) {
	tests := []struct {
		name     string
		hexStr   string
		size     int
		expected map[int]bool
	}{
		{
			name:   "all zeros",
			hexStr: "0x00",
			size:   8,
			expected: map[int]bool{},
		},
		{
			name:   "all ones",
			hexStr: "0xff",
			size:   8,
			expected: map[int]bool{
				0: true, 1: true, 2: true, 3: true,
				4: true, 5: true, 6: true, 7: true,
			},
		},
		{
			name:   "first bit set",
			hexStr: "0x01",
			size:   8,
			expected: map[int]bool{0: true},
		},
		{
			name:   "last bit set",
			hexStr: "0x80",
			size:   8,
			expected: map[int]bool{7: true},
		},
		{
			name:   "alternating bits",
			hexStr: "0x55", // 01010101
			size:   8,
			expected: map[int]bool{
				0: true, 2: true, 4: true, 6: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeBitVector(tt.hexStr, tt.size)
			if err != nil {
				t.Fatalf("DecodeBitVector failed: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d bits set, got %d", len(tt.expected), len(result))
			}

			for bit, expectedSet := range tt.expected {
				if result[bit] != expectedSet {
					t.Errorf("Expected bit %d to be %v, got %v", bit, expectedSet, result[bit])
				}
			}
		})
	}
}

func TestProcessAttestations(t *testing.T) {
	// Create test committees
	committees := []models.Committee{
		{
			Index:      0,
			Slot:       100,
			Validators: []string{"10", "20", "30", "40"},
		},
		{
			Index:      1,
			Slot:       100,
			Validators: []string{"50", "60", "70", "80"},
		},
	}

	// Create attestations
	attestations := []models.Attestation{
		{
			AggregationBits: "0x05", // Binary: 10100000 -> validators 0 and 2 (indices 10 and 30)
			Data: models.AttestationData{
				Index: 0,
				Slot:  100,
			},
		},
		{
			AggregationBits: "0x03", // Binary: 11000000 -> validators 0 and 1 (indices 50 and 60)
			Data: models.AttestationData{
				Index: 1,
				Slot:  100,
			},
		},
	}

	attested, err := ProcessAttestations(attestations, committees)
	if err != nil {
		t.Fatalf("ProcessAttestations failed: %v", err)
	}

	// Check that the correct validators attested
	expectedAttested := map[models.ValidatorIndex]bool{
		10: true,
		30: true,
		50: true,
		60: true,
	}

	if len(attested) != len(expectedAttested) {
		t.Errorf("Expected %d validators attested, got %d", len(expectedAttested), len(attested))
	}

	for idx := range expectedAttested {
		if !attested[idx] {
			t.Errorf("Expected validator %d to have attested", idx)
		}
	}

	// Check that other validators did not attest
	if attested[20] {
		t.Error("Validator 20 should not have attested")
	}
	if attested[40] {
		t.Error("Validator 40 should not have attested")
	}
}

func TestProcessLiveness(t *testing.T) {
	liveness := []models.ValidatorLiveness{
		{Index: 100, IsLive: true},
		{Index: 200, IsLive: false},
		{Index: 300, IsLive: true},
	}

	result := ProcessLiveness(liveness)

	if len(result) != 3 {
		t.Errorf("Expected 3 liveness entries, got %d", len(result))
	}

	if !result[100] {
		t.Error("Expected validator 100 to be live")
	}

	if result[200] {
		t.Error("Expected validator 200 to not be live")
	}

	if !result[300] {
		t.Error("Expected validator 300 to be live")
	}
}

func TestProcessRewards(t *testing.T) {
	rewards := &models.RewardsResponse{
		Data: struct {
			IdealRewards []models.IdealReward `json:"ideal_rewards"`
			TotalRewards []models.TotalReward `json:"total_rewards"`
		}{
			IdealRewards: []models.IdealReward{
				{
					EffectiveBalance: 32000000000,
					Head:             1000,
					Target:           2000,
					Source:           3000,
				},
			},
			TotalRewards: []models.TotalReward{
				{
					ValidatorIndex: 100,
					Head:           900,
					Target:         2000,
					Source:         2500,
				},
				{
					ValidatorIndex: 200,
					Head:           1000,
					Target:         2000,
					Source:         3000,
				},
			},
		},
	}

	result, err := ProcessRewards(rewards, []models.ValidatorIndex{100, 200})
	if err != nil {
		t.Fatalf("ProcessRewards failed: %v", err)
	}

	// Check validator 100 (suboptimal head and source)
	reward100 := result[100]
	if !reward100.SuboptimalHead {
		t.Error("Expected validator 100 to have suboptimal head")
	}
	if !reward100.SuboptimalSource {
		t.Error("Expected validator 100 to have suboptimal source")
	}
	if reward100.SuboptimalTarget {
		t.Error("Expected validator 100 to not have suboptimal target")
	}

	// Check validator 200 (all optimal)
	reward200 := result[200]
	if reward200.SuboptimalHead {
		t.Error("Expected validator 200 to not have suboptimal head")
	}
	if reward200.SuboptimalSource {
		t.Error("Expected validator 200 to not have suboptimal source")
	}
	if reward200.SuboptimalTarget {
		t.Error("Expected validator 200 to not have suboptimal target")
	}
}
