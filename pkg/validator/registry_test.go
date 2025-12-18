package validator

import (
	"testing"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
)

func TestAllValidatorsUpdate(t *testing.T) {
	av := NewAllValidators()

	validators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
		{
			Index:   200,
			Balance: 32100000000,
			Status:  models.StatusActiveOngoing,
		},
	}
	validators[0].Data.Pubkey = "0xabc123"
	validators[0].Data.EffectiveBalance = 32000000000
	validators[0].Data.WithdrawalCredentials = "0x01abc"
	validators[1].Data.Pubkey = "0xdef456"
	validators[1].Data.EffectiveBalance = 32000000000
	validators[1].Data.WithdrawalCredentials = "0x02def"

	av.Update(validators)

	if av.Count() != 2 {
		t.Errorf("Expected 2 validators, got %d", av.Count())
	}

	// Get() is no longer supported (we don't store individual validators)
	// Only index lookup via pubkey is supported
	_, ok := av.Get(100)
	if ok {
		t.Error("Expected Get() to return false (not supported)")
	}

	// GetByPubkey returns minimal validator with just index
	v2, ok := av.GetByPubkey("0xabc123")
	if !ok {
		t.Fatal("Expected to find validator by pubkey")
	}
	if v2.Index != 100 {
		t.Errorf("Expected index 100, got %d", v2.Index)
	}

	// GetIndexByPubkey is the preferred method
	idx, ok := av.GetIndexByPubkey("0xdef456")
	if !ok {
		t.Fatal("Expected to find validator index by pubkey")
	}
	if idx != 200 {
		t.Errorf("Expected index 200, got %d", idx)
	}
}

func TestAllValidatorsAggregates(t *testing.T) {
	av := NewAllValidators()

	validators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
		{
			Index:   200,
			Balance: 32000000000,
			Status:  models.StatusActiveExiting,
		},
		{
			Index:   300,
			Balance: 64000000000, // 64 ETH - MaxEB validator
			Status:  models.StatusActiveOngoing,
		},
	}
	validators[0].Data.Pubkey = "0xabc123"
	validators[0].Data.EffectiveBalance = 32000000000
	validators[0].Data.WithdrawalCredentials = "0x01abc"
	validators[1].Data.Pubkey = "0xdef456"
	validators[1].Data.EffectiveBalance = 32000000000
	validators[1].Data.WithdrawalCredentials = "0x01def"
	validators[2].Data.Pubkey = "0x789ghi"
	validators[2].Data.EffectiveBalance = 64000000000    // 64 ETH
	validators[2].Data.WithdrawalCredentials = "0x02xyz" // 0x02 type

	av.Update(validators)

	aggregates := av.GetNetworkAggregates()

	if aggregates.TotalValidators != 3 {
		t.Errorf("Expected 3 total validators, got %d", aggregates.TotalValidators)
	}

	// TotalStake should be 4 (32 ETH + 32 ETH + 64 ETH = 128 ETH = 4 * 32 ETH units)
	if aggregates.TotalStake != 4.0 {
		t.Errorf("Expected total stake 4.0, got %f", aggregates.TotalStake)
	}

	// Check status counts
	if aggregates.StatusCounts[models.StatusActiveOngoing] != 2 {
		t.Errorf("Expected 2 active_ongoing, got %d", aggregates.StatusCounts[models.StatusActiveOngoing])
	}
	if aggregates.StatusCounts[models.StatusActiveExiting] != 1 {
		t.Errorf("Expected 1 active_exiting, got %d", aggregates.StatusCounts[models.StatusActiveExiting])
	}

	// Check type counts
	if aggregates.TypeCounts[ValidatorType0x01] != 2 {
		t.Errorf("Expected 2 0x01 validators, got %d", aggregates.TypeCounts[ValidatorType0x01])
	}
	if aggregates.TypeCounts[ValidatorType0x02] != 1 {
		t.Errorf("Expected 1 0x02 validator, got %d", aggregates.TypeCounts[ValidatorType0x02])
	}

	// Check type stakes (in ETH)
	if aggregates.TypeStakes[ValidatorType0x02] != 64.0 {
		t.Errorf("Expected 64 ETH for 0x02 validators, got %f", aggregates.TypeStakes[ValidatorType0x02])
	}
}

func TestAllValidatorsGetNonExistent(t *testing.T) {
	av := NewAllValidators()

	_, ok := av.Get(999)
	if ok {
		t.Error("Expected not to find validator 999")
	}

	_, ok = av.GetByPubkey("0xnonexistent")
	if ok {
		t.Error("Expected not to find validator by pubkey")
	}
}

func TestWatchedValidatorsUpdate(t *testing.T) {
	wv := NewWatchedValidators()

	validators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
		{
			Index:   200,
			Balance: 32100000000,
			Status:  models.StatusActiveOngoing,
		},
	}
	validators[0].Data.Pubkey = "0xabc123"
	validators[0].Data.EffectiveBalance = 32000000000
	validators[1].Data.Pubkey = "0xdef456"
	validators[1].Data.EffectiveBalance = 32000000000

	config := []models.WatchedKey{
		{
			PublicKey: "0xabc123",
			Labels:    []string{"vc:val1", "region:us"},
		},
		{
			PublicKey: "0xdef456",
			Labels:    []string{"vc:val2", "region:eu"},
		},
	}

	err := wv.Update(validators, config)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if wv.Count() != 2 {
		t.Errorf("Expected 2 watched validators, got %d", wv.Count())
	}

	v, ok := wv.Get(100)
	if !ok {
		t.Fatal("Expected to find validator 100")
	}

	// Check labels include scope labels and auto-generated key label
	expectedLabels := []string{"scope:all-network", "scope:watched", "key:0xabc123", "vc:val1", "region:us"}
	if len(v.Labels) != len(expectedLabels) {
		t.Errorf("Expected %d labels, got %d", len(expectedLabels), len(v.Labels))
	}

	// Check weight calculation (32 ETH = 1.0)
	if v.Weight != 1.0 {
		t.Errorf("Expected weight 1.0, got %f", v.Weight)
	}
}

func TestWatchedValidatorsGetByLabel(t *testing.T) {
	wv := NewWatchedValidators()

	validators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
		{
			Index:   200,
			Balance: 32100000000,
			Status:  models.StatusActiveOngoing,
		},
	}
	validators[0].Data.Pubkey = "0xabc123"
	validators[0].Data.EffectiveBalance = 32000000000
	validators[1].Data.Pubkey = "0xdef456"
	validators[1].Data.EffectiveBalance = 32000000000

	config := []models.WatchedKey{
		{
			PublicKey: "0xabc123",
			Labels:    []string{"region:us"},
		},
		{
			PublicKey: "0xdef456",
			Labels:    []string{"region:eu"},
		},
	}

	wv.Update(validators, config)

	usValidators := wv.GetByLabel("region:us")
	if len(usValidators) != 1 {
		t.Errorf("Expected 1 US validator, got %d", len(usValidators))
	}

	watchedValidators := wv.GetByLabel("scope:watched")
	if len(watchedValidators) != 2 {
		t.Errorf("Expected 2 watched validators, got %d", len(watchedValidators))
	}
}

func TestWatchedValidatorsUpdateMetrics(t *testing.T) {
	wv := NewWatchedValidators()

	validators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
	}
	validators[0].Data.Pubkey = "0xabc123"
	validators[0].Data.EffectiveBalance = 32000000000

	config := []models.WatchedKey{
		{
			PublicKey: "0xabc123",
			Labels:    []string{},
		},
	}

	wv.Update(validators, config)

	err := wv.UpdateMetrics(100, func(v *WatchedValidator) {
		v.MissedAttestations = 5
		v.ProposedBlocks = 3
	})
	if err != nil {
		t.Fatalf("UpdateMetrics failed: %v", err)
	}

	v, _ := wv.Get(100)
	if v.MissedAttestations != 5 {
		t.Errorf("Expected 5 missed attestations, got %d", v.MissedAttestations)
	}
	if v.ProposedBlocks != 3 {
		t.Errorf("Expected 3 proposed blocks, got %d", v.ProposedBlocks)
	}
}

func TestWatchedValidatorsResetMetrics(t *testing.T) {
	wv := NewWatchedValidators()

	validators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
	}
	validators[0].Data.Pubkey = "0xabc123"
	validators[0].Data.EffectiveBalance = 32000000000

	config := []models.WatchedKey{
		{
			PublicKey: "0xabc123",
			Labels:    []string{},
		},
	}

	wv.Update(validators, config)

	wv.UpdateMetrics(100, func(v *WatchedValidator) {
		v.MissedAttestations = 5
		v.ProposedBlocks = 3
	})

	wv.ResetMetrics()

	v, _ := wv.Get(100)
	if v.MissedAttestations != 0 {
		t.Errorf("Expected 0 missed attestations after reset, got %d", v.MissedAttestations)
	}
	if v.ProposedBlocks != 0 {
		t.Errorf("Expected 0 proposed blocks after reset, got %d", v.ProposedBlocks)
	}
}

func TestAllValidatorsConcurrency(t *testing.T) {
	av := NewAllValidators()

	validators := make([]models.Validator, 1000)
	for i := 0; i < 1000; i++ {
		validators[i] = models.Validator{
			Index:   models.ValidatorIndex(i),
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		}
		validators[i].Data.Pubkey = string(rune(i))
		validators[i].Data.EffectiveBalance = 32000000000
		validators[i].Data.WithdrawalCredentials = "0x01abc"
	}

	av.Update(validators)

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				av.GetIndexByPubkey(string(rune(j)))
				av.Count()
				av.GetNetworkAggregates()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGetIndices(t *testing.T) {
	wv := NewWatchedValidators()

	validators := []models.Validator{
		{Index: 100},
		{Index: 200},
		{Index: 300},
	}
	validators[0].Data.Pubkey = "0xabc123"
	validators[0].Data.EffectiveBalance = 32000000000
	validators[1].Data.Pubkey = "0xdef456"
	validators[1].Data.EffectiveBalance = 32000000000
	validators[2].Data.Pubkey = "0x789ghi"
	validators[2].Data.EffectiveBalance = 32000000000

	config := []models.WatchedKey{
		{PublicKey: "0xabc123"},
		{PublicKey: "0xdef456"},
		{PublicKey: "0x789ghi"},
	}

	wv.Update(validators, config)

	indices := wv.GetIndices()
	if len(indices) != 3 {
		t.Errorf("Expected 3 indices, got %d", len(indices))
	}
}
