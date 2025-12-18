package validator

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
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
	// Duties at slot tracking (for slot-specific metrics)
	DutiesSlot            models.Slot // Slot where validator had a duty
	DutiesPerformedAtSlot bool        // Whether duty was performed at that slot
	// Missed attestation tracking (for consecutive tracking)
	MissedAttestation         bool // Whether validator missed attestation in current epoch (boolean, like Python)
	PreviousMissedAttestation bool // Whether validator missed attestation in previous epoch
	// Per-epoch suboptimal tracking (booleans for current epoch, like Python)
	// Used for rate calculations: suboptimal / (suboptimal + optimal) * 100
	SuboptimalSourceThisEpoch bool // Whether validator had suboptimal source this epoch
	SuboptimalTargetThisEpoch bool // Whether validator had suboptimal target this epoch
	SuboptimalHeadThisEpoch   bool // Whether validator had suboptimal head this epoch
	HasRewardsData            bool // Whether validator has rewards data for current epoch

	// Sync committee tracking
	InSyncCommittee      bool              // Whether validator is in current sync committee
	SyncCommitteeDuties  uint64            // Total sync committee slots assigned
	SyncCommitteeSuccess uint64            // Sync committee slots where validator participated
	SyncCommitteeMissed  uint64            // Sync committee slots where validator missed
	SyncCommitteeRewards models.SignedGwei // Total sync committee rewards (can be negative for penalties)
}

// ValidatorType represents the withdrawal credential type
type ValidatorType string

const (
	ValidatorType0x00 ValidatorType = "0x00" // BLS withdrawal credentials
	ValidatorType0x01 ValidatorType = "0x01" // Execution layer withdrawal credentials
	ValidatorType0x02 ValidatorType = "0x02" // Compounding (MaxEB) withdrawal credentials
)

// NetworkAggregates holds pre-computed network-wide metrics
// This is computed once during validator loading and cached
// Memory: ~1KB instead of ~600MB for storing all validators
type NetworkAggregates struct {
	TotalValidators int
	TotalStake      float64 // Sum of effective_balance / 32 ETH (weight units)

	// Status breakdown
	StatusCounts map[models.ValidatorStatus]int
	StatusStakes map[models.ValidatorStatus]float64

	// Validator type breakdown (0x00, 0x01, 0x02)
	TypeCounts map[ValidatorType]int
	TypeStakes map[ValidatorType]float64 // Actual effective stake in ETH
}

// AllValidators represents the full validator set with efficient memory usage
// Instead of storing 2M+ validators, we only keep:
// - pubkeyMap: for fast lookups to find watched validators (~16MB for 2M validators)
// - aggregates: pre-computed network metrics (~1KB)
// This matches Python's approach: stream process during load, discard individual validators
type AllValidators struct {
	mu         sync.RWMutex
	pubkeyMap  map[string]models.ValidatorIndex // pubkey -> index for lookups
	aggregates *NetworkAggregates               // Pre-computed network metrics

	// Double-buffer for non-blocking streaming updates
	// During streaming, we build new data here without holding locks
	streamingPubkeyMap  map[string]models.ValidatorIndex
	streamingAggregates *NetworkAggregates
}

// NewAllValidators creates a new all validators registry
func NewAllValidators() *AllValidators {
	return &AllValidators{
		pubkeyMap: make(map[string]models.ValidatorIndex),
		aggregates: &NetworkAggregates{
			StatusCounts: make(map[models.ValidatorStatus]int),
			StatusStakes: make(map[models.ValidatorStatus]float64),
			TypeCounts:   make(map[ValidatorType]int),
			TypeStakes:   make(map[ValidatorType]float64),
		},
	}
}

// GetValidatorType extracts the validator type from withdrawal credentials
func GetValidatorType(withdrawalCredentials string) ValidatorType {
	if len(withdrawalCredentials) < 4 {
		return ValidatorType0x00 // Default to 0x00 if invalid
	}
	prefix := withdrawalCredentials[:4]
	switch prefix {
	case "0x00":
		return ValidatorType0x00
	case "0x01":
		return ValidatorType0x01
	case "0x02":
		return ValidatorType0x02
	default:
		return ValidatorType0x00 // Default
	}
}

// ProcessAndAggregate processes validators in a streaming fashion:
// - Builds pubkey->index lookup map
// - Computes network aggregates (including validator type breakdown)
// - Does NOT store individual validators (they are discarded after processing)
// This is the key memory optimization - we never hold 2M validators in memory
func (av *AllValidators) ProcessAndAggregate(validators []models.Validator) {
	av.mu.Lock()
	defer av.mu.Unlock()

	// Reset aggregates
	av.aggregates = &NetworkAggregates{
		StatusCounts: make(map[models.ValidatorStatus]int),
		StatusStakes: make(map[models.ValidatorStatus]float64),
		TypeCounts:   make(map[ValidatorType]int),
		TypeStakes:   make(map[ValidatorType]float64),
	}

	// Clear and recreate pubkey map with appropriate capacity
	// Maps don't have cap(), so we always recreate with proper size hint
	av.pubkeyMap = make(map[string]models.ValidatorIndex, len(validators))

	// Stream process: compute aggregates while building lookup map
	for i := range validators {
		v := &validators[i]

		// Build lookup map (this is what we keep)
		// Normalize to lowercase for case-insensitive matching
		av.pubkeyMap[strings.ToLower(v.Data.Pubkey)] = v.Index

		// Accumulate network aggregates
		weight := float64(v.Data.EffectiveBalance) / 32_000_000_000.0
		effectiveETH := float64(v.Data.EffectiveBalance) / 1_000_000_000.0 // Actual ETH

		av.aggregates.TotalValidators++
		av.aggregates.TotalStake += weight

		// Status breakdown
		av.aggregates.StatusCounts[v.Status]++
		av.aggregates.StatusStakes[v.Status] += weight

		// Validator type breakdown (0x00, 0x01, 0x02)
		valType := GetValidatorType(v.Data.WithdrawalCredentials)
		av.aggregates.TypeCounts[valType]++
		av.aggregates.TypeStakes[valType] += effectiveETH
	}
	// validators slice is NOT stored - will be garbage collected after this function returns
	// Force GC to release the ~600MB validator slice immediately
	// This prevents memory spikes when background refresh runs
	runtime.GC()
	// Force return of memory to OS (Go normally keeps it for future allocations)
	debug.FreeOSMemory()
}

// Update is an alias for ProcessAndAggregate for backward compatibility
func (av *AllValidators) Update(validators []models.Validator) {
	av.ProcessAndAggregate(validators)
}

// StartStreaming prepares for streaming validator processing
// Call this before processing validators one at a time with ProcessOne
// Uses double-buffering: builds new data without locking, then swaps atomically
func (av *AllValidators) StartStreaming(expectedCount int) {
	// NO LOCK HERE - we build new data in temporary buffers
	// This allows reads to continue using existing data during streaming

	// Create new aggregates buffer
	av.streamingAggregates = &NetworkAggregates{
		StatusCounts: make(map[models.ValidatorStatus]int),
		StatusStakes: make(map[models.ValidatorStatus]float64),
		TypeCounts:   make(map[ValidatorType]int),
		TypeStakes:   make(map[ValidatorType]float64),
	}

	// Pre-allocate new pubkey map buffer
	if expectedCount > 0 {
		av.streamingPubkeyMap = make(map[string]models.ValidatorIndex, expectedCount)
	} else {
		av.streamingPubkeyMap = make(map[string]models.ValidatorIndex, 2200000) // Default for mainnet
	}
}

// ProcessOne processes a single validator during streaming
// This is called for each validator as it's decoded from JSON
// Memory efficient: validator is processed and discarded immediately
// NO LOCK - writes to streaming buffers, not the main data
func (av *AllValidators) ProcessOne(v *models.Validator) {
	// No lock - writing to streaming buffers (double-buffer pattern)
	// Reads continue using the main pubkeyMap/aggregates

	// Build lookup map (this is what we keep)
	// Normalize to lowercase for case-insensitive matching
	av.streamingPubkeyMap[strings.ToLower(v.Data.Pubkey)] = v.Index

	// Accumulate network aggregates
	weight := float64(v.Data.EffectiveBalance) / 32_000_000_000.0
	effectiveETH := float64(v.Data.EffectiveBalance) / 1_000_000_000.0 // Actual ETH

	av.streamingAggregates.TotalValidators++
	av.streamingAggregates.TotalStake += weight

	// Status breakdown
	av.streamingAggregates.StatusCounts[v.Status]++
	av.streamingAggregates.StatusStakes[v.Status] += weight

	// Validator type breakdown (0x00, 0x01, 0x02)
	valType := GetValidatorType(v.Data.WithdrawalCredentials)
	av.streamingAggregates.TypeCounts[valType]++
	av.streamingAggregates.TypeStakes[valType] += effectiveETH
}

// FinishStreaming completes streaming processing
// Call this after all validators have been processed with ProcessOne
// Atomically swaps the new data into place (brief lock)
func (av *AllValidators) FinishStreaming() {
	// Brief lock to swap the buffers atomically
	av.mu.Lock()
	av.pubkeyMap = av.streamingPubkeyMap
	av.aggregates = av.streamingAggregates
	av.streamingPubkeyMap = nil // Allow GC
	av.streamingAggregates = nil
	av.mu.Unlock()

	// Force GC to release any temporary allocations
	runtime.GC()
	// Force return of memory to OS (Go normally keeps it for future allocations)
	debug.FreeOSMemory()
}

// GetIndexByPubkey returns the validator index for a given pubkey
// This is O(1) lookup from the in-memory map
// Case-insensitive: pubkeys are normalized to lowercase
func (av *AllValidators) GetIndexByPubkey(pubkey string) (models.ValidatorIndex, bool) {
	av.mu.RLock()
	defer av.mu.RUnlock()

	// Normalize to lowercase for case-insensitive matching
	index, ok := av.pubkeyMap[strings.ToLower(pubkey)]
	return index, ok
}

// GetByPubkey returns a minimal validator (just index) for backward compatibility
// Note: This does NOT return full validator data since we don't store it
// Use GetIndexByPubkey for the common case of just needing the index
// Case-insensitive: pubkeys are normalized to lowercase
func (av *AllValidators) GetByPubkey(pubkey string) (*models.Validator, bool) {
	av.mu.RLock()
	defer av.mu.RUnlock()

	// Normalize to lowercase for case-insensitive matching
	index, ok := av.pubkeyMap[strings.ToLower(pubkey)]
	if !ok {
		return nil, false
	}

	// Return minimal validator with just the index
	// Full data is not available since we don't store individual validators
	return &models.Validator{Index: index}, true
}

// Get is not supported in the optimized version
// We don't store individual validators to save memory
func (av *AllValidators) Get(index models.ValidatorIndex) (*models.Validator, bool) {
	// Not available - we don't store individual validators
	return nil, false
}

// Count returns the total number of validators
func (av *AllValidators) Count() int {
	av.mu.RLock()
	defer av.mu.RUnlock()

	return len(av.pubkeyMap)
}

// GetNetworkAggregates returns the pre-computed network metrics
// This is O(1) and doesn't allocate any memory
func (av *AllValidators) GetNetworkAggregates() *NetworkAggregates {
	av.mu.RLock()
	defer av.mu.RUnlock()

	return av.aggregates
}

// GetAll is deprecated - returns empty slice
// Network metrics should be obtained via GetNetworkAggregates()
func (av *AllValidators) GetAll() []models.Validator {
	// Return empty - we don't store individual validators
	// Use GetNetworkAggregates() for network metrics instead
	return nil
}

// Close is a no-op for the in-memory implementation
func (av *AllValidators) Close() error {
	return nil
}

// WatchedValidators represents the registry of watched validators
// Optimized for 45k+ validators with efficient memory usage
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
// Optimized for 45k+ validators
// IMPORTANT: Preserves existing metrics (rewards, attestations, blocks) when updating validator data
func (wv *WatchedValidators) Update(validators []models.Validator, config []models.WatchedKey) error {
	wv.mu.Lock()
	defer wv.mu.Unlock()

	// Build pubkey to config map
	configMap := make(map[string]models.WatchedKey, len(config))
	for _, wk := range config {
		configMap[wk.PublicKey] = wk
	}

	// Keep reference to old validators to preserve metrics
	oldValidators := wv.validators

	// Pre-allocate maps with expected capacity
	expectedSize := len(validators)
	wv.validators = make(map[models.ValidatorIndex]*WatchedValidator, expectedSize)
	wv.pubkeyMap = make(map[string]models.ValidatorIndex, expectedSize)
	wv.labels = make(map[string][]models.ValidatorIndex)

	// Pre-count labels for efficient slice allocation
	labelCounts := make(map[string]int)
	for _, v := range validators {
		cfg, ok := configMap[v.Data.Pubkey]
		if !ok {
			continue
		}
		// Count scope labels
		labelCounts["scope:all-network"]++
		labelCounts["scope:watched"]++
		labelCounts["scope:network"]++
		// Count custom labels
		for _, label := range cfg.Labels {
			labelCounts[label]++
		}
	}

	// Pre-allocate label slices
	for label, count := range labelCounts {
		wv.labels[label] = make([]models.ValidatorIndex, 0, count)
	}

	for _, v := range validators {
		cfg, ok := configMap[v.Data.Pubkey]
		if !ok {
			continue
		}

		// Calculate weight (effective balance / 32 ETH)
		weight := float64(v.Data.EffectiveBalance) / 32_000_000_000.0

		// Build labels (always include scope labels)
		// Python auto-generates key:<pubkey_prefix> for each validator
		keyPrefix := v.Data.Pubkey
		if len(keyPrefix) > 12 {
			keyPrefix = keyPrefix[:12] // e.g., "0x8001e3eff1"
		}
		keyLabel := "key:" + keyPrefix
		labels := make([]string, 0, 4+len(cfg.Labels))
		labels = append(labels, "scope:all-network", "scope:watched", keyLabel)
		labels = append(labels, cfg.Labels...)

		watched := &WatchedValidator{
			Validator: v,
			Labels:    labels,
			Weight:    weight,
		}

		// CRITICAL: Preserve existing metrics from the old validator
		// This ensures rewards, attestations, and block data persist across epoch updates
		if oldV, exists := oldValidators[v.Index]; exists {
			watched.MissedAttestations = oldV.MissedAttestations
			watched.SuboptimalSourceVotes = oldV.SuboptimalSourceVotes
			watched.SuboptimalTargetVotes = oldV.SuboptimalTargetVotes
			watched.SuboptimalHeadVotes = oldV.SuboptimalHeadVotes
			watched.IdealConsensusRewards = oldV.IdealConsensusRewards
			watched.ConsensusRewards = oldV.ConsensusRewards
			watched.ProposedBlocks = oldV.ProposedBlocks
			watched.ProposedBlocksFinalized = oldV.ProposedBlocksFinalized
			watched.MissedBlocks = oldV.MissedBlocks
			watched.MissedBlocksFinalized = oldV.MissedBlocksFinalized
			watched.FutureBlockProposals = oldV.FutureBlockProposals
			watched.AttestationDuties = oldV.AttestationDuties
			watched.AttestationDutiesSuccess = oldV.AttestationDutiesSuccess
			watched.ConsecutiveMissedAttest = oldV.ConsecutiveMissedAttest
			watched.DutiesSlot = oldV.DutiesSlot
			watched.DutiesPerformedAtSlot = oldV.DutiesPerformedAtSlot
			watched.MissedAttestation = oldV.MissedAttestation
			watched.PreviousMissedAttestation = oldV.PreviousMissedAttestation
			// Per-epoch suboptimal tracking
			watched.SuboptimalSourceThisEpoch = oldV.SuboptimalSourceThisEpoch
			watched.SuboptimalTargetThisEpoch = oldV.SuboptimalTargetThisEpoch
			watched.SuboptimalHeadThisEpoch = oldV.SuboptimalHeadThisEpoch
			watched.HasRewardsData = oldV.HasRewardsData
		}

		wv.validators[v.Index] = watched
		wv.pubkeyMap[v.Data.Pubkey] = v.Index

		// Update label index
		for _, label := range labels {
			wv.labels[label] = append(wv.labels[label], v.Index)
		}

		// Also add to scope:network (same as watched for our validators)
		wv.labels["scope:network"] = append(wv.labels["scope:network"], v.Index)
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
// For 45k validators, this allocates ~45k pointers (~360KB)
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

// GetIndices returns all validator indices (useful for batch API calls)
func (wv *WatchedValidators) GetIndices() []models.ValidatorIndex {
	wv.mu.RLock()
	defer wv.mu.RUnlock()

	indices := make([]models.ValidatorIndex, 0, len(wv.validators))
	for idx := range wv.validators {
		indices = append(indices, idx)
	}
	return indices
}
