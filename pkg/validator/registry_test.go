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
	validators[1].Data.Pubkey = "0xdef456"

	av.Update(validators)

	if av.Count() != 2 {
		t.Errorf("Expected 2 validators, got %d", av.Count())
	}

	v, ok := av.Get(100)
	if !ok {
		t.Fatal("Expected to find validator 100")
	}
	if v.Index != 100 {
		t.Errorf("Expected index 100, got %d", v.Index)
	}

	v2, ok := av.GetByPubkey("0xabc123")
	if !ok {
		t.Fatal("Expected to find validator by pubkey")
	}
	if v2.Index != 100 {
		t.Errorf("Expected index 100, got %d", v2.Index)
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

	// Check labels include scope labels
	expectedLabels := []string{"scope:all-network", "scope:watched", "vc:val1", "region:us"}
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
	}

	av.Update(validators)

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				av.Get(models.ValidatorIndex(j))
				av.Count()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
