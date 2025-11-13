package clock

import (
	"context"
	"time"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultSlotLagSeconds is the default lag to wait for attestations after a slot
	DefaultSlotLagSeconds = 8
)

// BeaconClock manages slot timing and synchronization
type BeaconClock struct {
	genesisTime    uint64
	secondsPerSlot uint64
	slotsPerEpoch  uint64
	slotLagSeconds uint64
	logger         *logrus.Logger
	replayMode     bool
	replayStartTS  *uint64
	replayEndTS    *uint64
}

// NewBeaconClock creates a new beacon clock
func NewBeaconClock(genesis *models.Genesis, spec *models.Spec, logger *logrus.Logger) *BeaconClock {
	return &BeaconClock{
		genesisTime:    genesis.GenesisTime,
		secondsPerSlot: spec.SecondsPerSlot,
		slotsPerEpoch:  spec.SlotsPerEpoch,
		slotLagSeconds: DefaultSlotLagSeconds,
		logger:         logger,
		replayMode:     false,
	}
}

// EnableReplayMode enables replay mode with start and end timestamps
func (c *BeaconClock) EnableReplayMode(startTS, endTS *uint64) {
	c.replayMode = true
	c.replayStartTS = startTS
	c.replayEndTS = endTS
}

// CurrentSlot returns the current slot number
func (c *BeaconClock) CurrentSlot() models.Slot {
	now := uint64(time.Now().Unix())
	if c.replayMode && c.replayStartTS != nil {
		now = *c.replayStartTS
	}

	if now < c.genesisTime {
		return 0
	}

	return models.Slot((now - c.genesisTime) / c.secondsPerSlot)
}

// SlotToEpoch converts a slot to an epoch
func (c *BeaconClock) SlotToEpoch(slot models.Slot) models.Epoch {
	return models.Epoch(uint64(slot) / c.slotsPerEpoch)
}

// EpochToSlot converts an epoch to its first slot
func (c *BeaconClock) EpochToSlot(epoch models.Epoch) models.Slot {
	return models.Slot(uint64(epoch) * c.slotsPerEpoch)
}

// CurrentEpoch returns the current epoch number
func (c *BeaconClock) CurrentEpoch() models.Epoch {
	return c.SlotToEpoch(c.CurrentSlot())
}

// SlotStartTime returns the start time of a slot
func (c *BeaconClock) SlotStartTime(slot models.Slot) time.Time {
	timestamp := c.genesisTime + (uint64(slot) * c.secondsPerSlot)
	return time.Unix(int64(timestamp), 0)
}

// SlotEndTime returns the end time of a slot (including lag for attestations)
func (c *BeaconClock) SlotEndTime(slot models.Slot) time.Time {
	timestamp := c.genesisTime + (uint64(slot) * c.secondsPerSlot) + c.secondsPerSlot + c.slotLagSeconds
	return time.Unix(int64(timestamp), 0)
}

// TimeToSlot converts a timestamp to a slot number
func (c *BeaconClock) TimeToSlot(timestamp uint64) models.Slot {
	if timestamp < c.genesisTime {
		return 0
	}
	return models.Slot((timestamp - c.genesisTime) / c.secondsPerSlot)
}

// WaitUntilSlot waits until the specified slot has finished (including lag)
func (c *BeaconClock) WaitUntilSlot(ctx context.Context, slot models.Slot) error {
	if c.replayMode {
		// In replay mode, don't actually wait
		return nil
	}

	targetTime := c.SlotEndTime(slot)
	now := time.Now()

	if now.Before(targetTime) {
		waitDuration := targetTime.Sub(now)
		c.logger.Debugf("Waiting %v for slot %d to complete", waitDuration, slot)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
			return nil
		}
	}

	return nil
}

// WaitUntilNextSlot waits until the next slot starts
func (c *BeaconClock) WaitUntilNextSlot(ctx context.Context) (models.Slot, error) {
	currentSlot := c.CurrentSlot()
	nextSlot := currentSlot + 1

	if err := c.WaitUntilSlot(ctx, currentSlot); err != nil {
		return 0, err
	}

	return nextSlot, nil
}

// IsFirstSlotOfEpoch returns true if the slot is the first slot of an epoch
func (c *BeaconClock) IsFirstSlotOfEpoch(slot models.Slot) bool {
	return uint64(slot)%c.slotsPerEpoch == 0
}

// IsSlotInEpoch returns true if the slot is at the specified position in the epoch
func (c *BeaconClock) IsSlotInEpoch(slot models.Slot, position uint64) bool {
	return uint64(slot)%c.slotsPerEpoch == position
}

// SlotsPerEpoch returns the number of slots per epoch
func (c *BeaconClock) SlotsPerEpoch() uint64 {
	return c.slotsPerEpoch
}

// SecondsPerSlot returns the number of seconds per slot
func (c *BeaconClock) SecondsPerSlot() uint64 {
	return c.secondsPerSlot
}

// GenesisTime returns the genesis timestamp
func (c *BeaconClock) GenesisTime() uint64 {
	return c.genesisTime
}

// IsReplayMode returns true if in replay mode
func (c *BeaconClock) IsReplayMode() bool {
	return c.replayMode
}

// ReplayComplete returns true if replay has reached the end timestamp
func (c *BeaconClock) ReplayComplete() bool {
	if !c.replayMode || c.replayEndTS == nil {
		return false
	}

	currentTime := uint64(time.Now().Unix())
	if c.replayStartTS != nil {
		currentTime = *c.replayStartTS
	}

	return currentTime >= *c.replayEndTS
}
