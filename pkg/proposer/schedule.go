package proposer

import (
	"context"
	"fmt"
	"sync"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/beacon"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/sirupsen/logrus"
)

// Schedule tracks block proposer duties
type Schedule struct {
	mu      sync.RWMutex
	duties  map[models.Slot]models.ValidatorIndex
	client  *beacon.Client
	logger  *logrus.Logger
	maxSlot models.Slot
}

// NewSchedule creates a new proposer schedule
func NewSchedule(client *beacon.Client, logger *logrus.Logger) *Schedule {
	return &Schedule{
		duties: make(map[models.Slot]models.ValidatorIndex),
		client: client,
		logger: logger,
	}
}

// Update fetches and updates the proposer schedule for an epoch
func (s *Schedule) Update(ctx context.Context, epoch models.Epoch) error {
	duties, err := s.client.GetProposerDuties(ctx, epoch)
	if err != nil {
		return fmt.Errorf("failed to fetch proposer duties for epoch %d: %w", epoch, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, duty := range duties {
		s.duties[duty.Slot] = duty.ValidatorIndex
		if duty.Slot > s.maxSlot {
			s.maxSlot = duty.Slot
		}
	}

	s.logger.Debugf("Updated proposer schedule for epoch %d: %d duties", epoch, len(duties))
	return nil
}

// GetProposer returns the validator index of the proposer for a slot
func (s *Schedule) GetProposer(slot models.Slot) (models.ValidatorIndex, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	proposer, ok := s.duties[slot]
	return proposer, ok
}

// HasProposer returns true if a proposer is scheduled for the slot
func (s *Schedule) HasProposer(slot models.Slot) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.duties[slot]
	return ok
}

// GetDuties returns all duties for a specific validator
func (s *Schedule) GetDuties(validatorIndex models.ValidatorIndex) []models.Slot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slots := make([]models.Slot, 0)
	for slot, proposer := range s.duties {
		if proposer == validatorIndex {
			slots = append(slots, slot)
		}
	}
	return slots
}

// Cleanup removes old duties before the specified slot
func (s *Schedule) Cleanup(beforeSlot models.Slot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for slot := range s.duties {
		if slot < beforeSlot {
			delete(s.duties, slot)
		}
	}
}

// Count returns the number of scheduled duties
func (s *Schedule) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.duties)
}
