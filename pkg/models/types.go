package models

import "time"

// Duration wraps time.Duration to support YAML unmarshaling from seconds
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler to parse seconds as integer
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var seconds int
	if err := unmarshal(&seconds); err != nil {
		return err
	}
	*d = Duration(time.Duration(seconds) * time.Second)
	return nil
}

// MarshalYAML implements yaml.Marshaler
func (d Duration) MarshalYAML() (interface{}, error) {
	return int(time.Duration(d).Seconds()), nil
}

// ToDuration converts to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// String implements Stringer
func (d Duration) String() string {
	return time.Duration(d).String()
}

// Slot represents an Ethereum slot number
type Slot uint64

// Epoch represents an Ethereum epoch number
type Epoch uint64

// ValidatorIndex represents a validator index
type ValidatorIndex uint64

// Gwei represents an amount in Gwei (always positive)
type Gwei uint64

// SignedGwei represents a signed amount in Gwei (can be negative for penalties)
type SignedGwei int64

// ValidatorStatus represents the status of a validator
type ValidatorStatus string

const (
	StatusPendingInitialized ValidatorStatus = "pending_initialized"
	StatusPendingQueued      ValidatorStatus = "pending_queued"
	StatusActiveOngoing      ValidatorStatus = "active_ongoing"
	StatusActiveExiting      ValidatorStatus = "active_exiting"
	StatusActiveSlashed      ValidatorStatus = "active_slashed"
	StatusExitedUnslashed    ValidatorStatus = "exited_unslashed"
	StatusExitedSlashed      ValidatorStatus = "exited_slashed"
	StatusWithdrawalPossible ValidatorStatus = "withdrawal_possible"
	StatusWithdrawalDone     ValidatorStatus = "withdrawal_done"
)

// Genesis represents the genesis configuration
type Genesis struct {
	GenesisTime           uint64 `json:"genesis_time,string"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
}

// Spec represents the beacon chain specification
type Spec struct {
	SecondsPerSlot               uint64 `json:"SECONDS_PER_SLOT,string"`
	SlotsPerEpoch                uint64 `json:"SLOTS_PER_EPOCH,string"`
	EpochsPerSyncCommitteePeriod uint64 `json:"EPOCHS_PER_SYNC_COMMITTEE_PERIOD,string"`
}

// BeaconHeader represents a beacon block header
type BeaconHeader struct {
	Root   string `json:"root"`
	Header struct {
		Message struct {
			Slot          Slot   `json:"slot,string"`
			ProposerIndex uint64 `json:"proposer_index,string"`
			ParentRoot    string `json:"parent_root"`
			StateRoot     string `json:"state_root"`
			BodyRoot      string `json:"body_root"`
		} `json:"message"`
	} `json:"header"`
}

// Validator represents a beacon chain validator
type Validator struct {
	Index     ValidatorIndex  `json:"index,string"`
	Balance   Gwei            `json:"balance,string"`
	Status    ValidatorStatus `json:"status"`
	Data      struct {
		Pubkey                     string `json:"pubkey"`
		WithdrawalCredentials      string `json:"withdrawal_credentials"`
		EffectiveBalance           Gwei   `json:"effective_balance,string"`
		Slashed                    bool   `json:"slashed"`
		ActivationEligibilityEpoch Epoch  `json:"activation_eligibility_epoch,string"`
		ActivationEpoch            Epoch  `json:"activation_epoch,string"`
		ExitEpoch                  Epoch  `json:"exit_epoch,string"`
		WithdrawableEpoch          Epoch  `json:"withdrawable_epoch,string"`
	} `json:"validator"`
}

// ValidatorsResponse represents the API response for validators
type ValidatorsResponse struct {
	Data []Validator `json:"data"`
}

// ProposerDuty represents a block proposer duty
type ProposerDuty struct {
	Pubkey         string         `json:"pubkey"`
	ValidatorIndex ValidatorIndex `json:"validator_index,string"`
	Slot           Slot           `json:"slot,string"`
}

// ProposerDutiesResponse represents the API response for proposer duties
type ProposerDutiesResponse struct {
	Data []ProposerDuty `json:"data"`
}

// Block represents a beacon block
type Block struct {
	Message struct {
		Slot          Slot   `json:"slot,string"`
		ProposerIndex uint64 `json:"proposer_index,string"`
		Body          struct {
			ExecutionPayload *struct {
				FeeRecipient string `json:"fee_recipient"`
			} `json:"execution_payload,omitempty"`
		} `json:"body"`
	} `json:"message"`
}

// BlockResponse represents the API response for a block
type BlockResponse struct {
	Data Block `json:"data"`
}

// AttestationData represents attestation data
type AttestationData struct {
	Slot            Slot `json:"slot,string"`
	Index           uint64 `json:"index,string"`
	BeaconBlockRoot string `json:"beacon_block_root"`
	Source          struct {
		Epoch Epoch  `json:"epoch,string"`
		Root  string `json:"root"`
	} `json:"source"`
	Target struct {
		Epoch Epoch  `json:"epoch,string"`
		Root  string `json:"root"`
	} `json:"target"`
}

// Attestation represents an attestation (post-Electra format with committee_bits)
type Attestation struct {
	AggregationBits string          `json:"aggregation_bits"`
	CommitteeBits   string          `json:"committee_bits"` // Electra: bitfield of active committees
	Data            AttestationData `json:"data"`
	Signature       string          `json:"signature"`
}

// AttestationsResponse represents the API response for attestations
type AttestationsResponse struct {
	Data []Attestation `json:"data"`
}

// Committee represents a beacon committee
type Committee struct {
	Index      uint64   `json:"index,string"`
	Slot       Slot     `json:"slot,string"`
	Validators []string `json:"validators"`
}

// CommitteesResponse represents the API response for committees
type CommitteesResponse struct {
	Data []Committee `json:"data"`
}

// ValidatorLiveness represents validator liveness data
type ValidatorLiveness struct {
	Index  ValidatorIndex `json:"index,string"`
	IsLive bool           `json:"is_live"`
}

// ValidatorsLivenessResponse represents the API response for validators liveness
type ValidatorsLivenessResponse struct {
	Data []ValidatorLiveness `json:"data"`
}

// IdealReward represents ideal rewards for a validator
type IdealReward struct {
	EffectiveBalance Gwei `json:"effective_balance,string"`
	Head             Gwei `json:"head,string"`
	Target           Gwei `json:"target,string"`
	Source           Gwei `json:"source,string"`
}

// TotalReward represents total rewards (can be negative for penalties)
type TotalReward struct {
	ValidatorIndex ValidatorIndex `json:"validator_index,string"`
	Head           SignedGwei     `json:"head,string"`
	Target         SignedGwei     `json:"target,string"`
	Source         SignedGwei     `json:"source,string"`
}

// RewardsResponse represents the API response for rewards
type RewardsResponse struct {
	Data struct {
		IdealRewards []IdealReward `json:"ideal_rewards"`
		TotalRewards []TotalReward `json:"total_rewards"`
	} `json:"data"`
}

// PendingDeposit represents a pending deposit
type PendingDeposit struct {
	Pubkey string `json:"pubkey"`
	Amount Gwei   `json:"amount,string"`
}

// PendingDepositsResponse represents the API response for pending deposits
type PendingDepositsResponse struct {
	Data []PendingDeposit `json:"data"`
}

// PendingConsolidation represents a pending consolidation
type PendingConsolidation struct {
	SourceIndex ValidatorIndex `json:"source_index,string"`
	TargetIndex ValidatorIndex `json:"target_index,string"`
}

// PendingConsolidationsResponse represents the API response for pending consolidations
type PendingConsolidationsResponse struct {
	Data []PendingConsolidation `json:"data"`
}

// PendingWithdrawal represents a pending withdrawal
type PendingWithdrawal struct {
	Index          uint64         `json:"index,string"`
	ValidatorIndex ValidatorIndex `json:"validator_index,string"`
	Amount         Gwei           `json:"amount,string"`
}

// PendingWithdrawalsResponse represents the API response for pending withdrawals
type PendingWithdrawalsResponse struct {
	Data []PendingWithdrawal `json:"data"`
}

// StateIDRequest represents a state ID request parameter
type StateIDRequest struct {
	StateID string
}

// Config represents the watcher configuration
type Config struct {
	Network             string       `yaml:"network"`
	BeaconURL           string       `yaml:"beacon_url"`
	BeaconTimeout       Duration     `yaml:"beacon_timeout_sec"`
	MetricsPort         int          `yaml:"metrics_port"`
	WatchedKeys         []WatchedKey `yaml:"watched_keys"`
	SlackToken          string       `yaml:"slack_token,omitempty"`
	SlackChannel        string       `yaml:"slack_channel,omitempty"`
	ReplayStartAtTS     *uint64      `yaml:"replay_start_at_ts,omitempty"`
	ReplayEndAtTS       *uint64      `yaml:"replay_end_at_ts,omitempty"`
	LoadAllValidators   *bool        `yaml:"load_all_validators,omitempty"` // Default true - load full 2M+ validator set for network comparison
}

// ShouldLoadAllValidators returns whether to load the full validator set (default true)
func (c *Config) ShouldLoadAllValidators() bool {
	if c.LoadAllValidators == nil {
		return true // Default behavior: load all validators like Kiln
	}
	return *c.LoadAllValidators
}

// WatchedKey represents a watched validator configuration
type WatchedKey struct {
	PublicKey string   `yaml:"public_key"`
	Labels    []string `yaml:"labels,omitempty"`
}
