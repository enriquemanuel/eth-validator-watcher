package validator

import (
	"fmt"
	"sync"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
)

// WatchedValidator represents a validator being watched with its labels
type WatchedValidator struct {
	models.Validator
	Labels                   []string
	Weight                   float64 // effective_balance / 32 ETH
	MissedAttestations       uint64
	SuboptimalSourceVotes    uint64
	SuboptimalTargetVotes    uint64
	SuboptimalHeadVotes      uint64
	IdealConsensusRewards    models.Gwei       // Ideal is always positive
	ConsensusRewards         models.SignedGwei // Actual can be negative (penalties)
	ProposedBlocks           uint64
	ProposedBlocksFinalized  uint64
	MissedBlocks             uint64
	MissedBlocksFinalized    uint64
	FutureBlockProposals     uint64
	AttestationDuties        uint64
	AttestationDutiesSuccess uint64
	ConsecutiveMissedAttest  uint64
}

// AllValidators represents the full validator set (2M+)
type AllValidators struct {
	mu         sync.RWMutex
	validators map[models.ValidatorIndex]*models.Validator
	pubkeyMap  map[string]models.ValidatorIndex
}

// NewAllValidators creates a new all validators registry
func NewAllValidators() *AllValidators {
	return &AllValidators{
		validators: make(map[models.ValidatorIndex]*models.Validator),
		pubkeyMap:  make(map[string]models.ValidatorIndex),
	}
}

// Update updates the full validator set
func (av *AllValidators) Update(validators []models.Validator) {
	av.mu.Lock()
	defer av.mu.Unlock()

	// Clear old data
	av.validators = make(map[models.ValidatorIndex]*models.Validator, len(validators))
	av.pubkeyMap = make(map[string]models.ValidatorIndex, len(validators))

	for i := range validators {
		v := &validators[i]
		av.validators[v.Index] = v
		av.pubkeyMap[v.Data.Pubkey] = v.Index
	}
}

// Get retrieves a validator by index
func (av *AllValidators) Get(index models.ValidatorIndex) (*models.Validator, bool) {
	av.mu.RLock()
	defer av.mu.RUnlock()

	v, ok := av.validators[index]
	return v, ok
}

// GetByPubkey retrieves a validator by public key
func (av *AllValidators) GetByPubkey(pubkey string) (*models.Validator, bool) {
	av.mu.RLock()
	defer av.mu.RUnlock()

	index, ok := av.pubkeyMap[pubkey]
	if !ok {
		return nil, false
	}

	return av.validators[index], true
}

// Count returns the total number of validators
func (av *AllValidators) Count() int {
	av.mu.RLock()
	defer av.mu.RUnlock()

	return len(av.validators)
}

// GetAll returns all validators (copy for safe iteration)
func (av *AllValidators) GetAll() []models.Validator {
	av.mu.RLock()
	defer av.mu.RUnlock()

	result := make([]models.Validator, 0, len(av.validators))
	for _, v := range av.validators {
		result = append(result, *v)
	}
	return result
}

// WatchedValidators represents the registry of watched validators
type WatchedValidators struct {
	mu         sync.RWMutex
	validators map[models.ValidatorIndex]*WatchedValidator
	pubkeyMap  map[string]models.ValidatorIndex
	labels     map[string][]models.ValidatorIndex // label -> validator indices
}

// NewWatchedValidators creates a new watched validators registry
func NewWatchedValidators() *WatchedValidators {
	return &WatchedValidators{
		validators: make(map[models.ValidatorIndex]*WatchedValidator),
		pubkeyMap:  make(map[string]models.ValidatorIndex),
		labels:     make(map[string][]models.ValidatorIndex),
	}
}

// Update updates the watched validators from API data
func (wv *WatchedValidators) Update(validators []models.Validator, config []models.WatchedKey) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	// Build pubkey to config map
	configMap := make(map[string]models.WatchedKey)
	for _, wk := range config {
		configMap[wk.PublicKey] = wk
	}

	// Clear old data
	wv.validators = make(map[models.ValidatorIndex]*WatchedValidator)
	wv.pubkeyMap = make(map[string]models.ValidatorIndex)
	wv.labels = make(map[string][]models.ValidatorIndex)

	for _, v := range validators {
		cfg, ok := configMap[v.Data.Pubkey]
		if !ok {
			continue
		}

		// Calculate weight (effective balance / 32 ETH)
		weight := float64(v.Data.EffectiveBalance) / 32_000_000_000.0

		// Build labels (always include scope labels)
		labels := []string{"scope:all-network", "scope:watched"}
		labels = append(labels, cfg.Labels...)

		watched := &WatchedValidator{
			Validator: v,
			Labels:    labels,
			Weight:    weight,
		}

		wv.validators[v.Index] = watched
		wv.pubkeyMap[v.Data.Pubkey] = v.Index

		// Update label index
		for _, label := range labels {
			wv.labels[label] = append(wv.labels[label], v.Index)
		}
	}

	// Add "scope:network" label for all validators
	for idx := range wv.validators {
		wv.labels["scope:network"] = append(wv.labels["scope:network"], idx)
	}

	return nil
}

// Get retrieves a watched validator by index
func (wv *WatchedValidators) Get(index models.ValidatorIndex) (*WatchedValidator, bool) {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	v, ok := wv.validators[index]
	return v, ok
}

// GetByPubkey retrieves a watched validator by public key
func (wv *WatchedValidators) GetByPubkey(pubkey string) (*WatchedValidator, bool) {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	index, ok := wv.pubkeyMap[pubkey]
	if !ok {
		return nil, false
	}

	return wv.validators[index], true
}

// GetAll returns all watched validators
func (wv *WatchedValidators) GetAll() []*WatchedValidator {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	result := make([]*WatchedValidator, 0, len(wv.validators))
	for _, v := range wv.validators {
		result = append(result, v)
	}
	return result
}

// GetByLabel returns all validators with a specific label
func (wv *WatchedValidators) GetByLabel(label string) []*WatchedValidator {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	indices, ok := wv.labels[label]
	if !ok {
		return nil
	}

	result := make([]*WatchedValidator, 0, len(indices))
	for _, idx := range indices {
		if v, ok := wv.validators[idx]; ok {
			result = append(result, v)
		}
	}
	return result
}

// GetLabels returns all unique labels
func (wv *WatchedValidators) GetLabels() []string {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	labels := make([]string, 0, len(wv.labels))
	for label := range wv.labels {
		labels = append(labels, label)
	}
	return labels
}

// Count returns the number of watched validators
func (wv *WatchedValidators) Count() int {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	return len(wv.validators)
}

// UpdateMetrics updates metrics for a validator
func (wv *WatchedValidators) UpdateMetrics(index models.ValidatorIndex, fn func(*WatchedValidator)) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	v, ok := wv.validators[index]
	if !ok {
		return fmt.Errorf("validator %d not found", index)
	}

	fn(v)
	return nil
}

// ResetMetrics resets all metrics for all validators
func (wv *WatchedValidators) ResetMetrics() {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	for _, v := range wv.validators {
		v.MissedAttestations = 0
		v.SuboptimalSourceVotes = 0
		v.SuboptimalTargetVotes = 0
		v.SuboptimalHeadVotes = 0
		v.IdealConsensusRewards = 0
		v.ConsensusRewards = 0
		v.ProposedBlocks = 0
		v.ProposedBlocksFinalized = 0
		v.MissedBlocks = 0
		v.MissedBlocksFinalized = 0
		v.FutureBlockProposals = 0
		v.AttestationDuties = 0
		v.AttestationDutiesSuccess = 0
		v.ConsecutiveMissedAttest = 0
	}
}
