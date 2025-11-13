package clock

import (
	"context"
	"testing"
	"time"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/sirupsen/logrus"
)

func TestBeaconClockSlotCalculation(t *testing.T) {
	genesis := &models.Genesis{
		GenesisTime: 1606824023,
	}
	spec := &models.Spec{
		SecondsPerSlot: 12,
		SlotsPerEpoch:  32,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	clock := NewBeaconClock(genesis, spec, logger)

	// Test slot 0
	slot0Time := clock.SlotStartTime(0)
	expectedSlot0 := time.Unix(int64(genesis.GenesisTime), 0)
	if !slot0Time.Equal(expectedSlot0) {
		t.Errorf("Expected slot 0 at %v, got %v", expectedSlot0, slot0Time)
	}

	// Test slot 100
	slot100Time := clock.SlotStartTime(100)
	expectedSlot100 := time.Unix(int64(genesis.GenesisTime+100*12), 0)
	if !slot100Time.Equal(expectedSlot100) {
		t.Errorf("Expected slot 100 at %v, got %v", expectedSlot100, slot100Time)
	}
}

func TestBeaconClockEpochConversion(t *testing.T) {
	genesis := &models.Genesis{
		GenesisTime: 1606824023,
	}
	spec := &models.Spec{
		SecondsPerSlot: 12,
		SlotsPerEpoch:  32,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	clock := NewBeaconClock(genesis, spec, logger)

	// Test slot to epoch
	epoch := clock.SlotToEpoch(64)
	if epoch != 2 {
		t.Errorf("Expected epoch 2 for slot 64, got %d", epoch)
	}

	epoch = clock.SlotToEpoch(0)
	if epoch != 0 {
		t.Errorf("Expected epoch 0 for slot 0, got %d", epoch)
	}

	epoch = clock.SlotToEpoch(31)
	if epoch != 0 {
		t.Errorf("Expected epoch 0 for slot 31, got %d", epoch)
	}

	epoch = clock.SlotToEpoch(32)
	if epoch != 1 {
		t.Errorf("Expected epoch 1 for slot 32, got %d", epoch)
	}

	// Test epoch to slot
	slot := clock.EpochToSlot(0)
	if slot != 0 {
		t.Errorf("Expected slot 0 for epoch 0, got %d", slot)
	}

	slot = clock.EpochToSlot(1)
	if slot != 32 {
		t.Errorf("Expected slot 32 for epoch 1, got %d", slot)
	}

	slot = clock.EpochToSlot(10)
	if slot != 320 {
		t.Errorf("Expected slot 320 for epoch 10, got %d", slot)
	}
}

func TestBeaconClockIsFirstSlotOfEpoch(t *testing.T) {
	genesis := &models.Genesis{
		GenesisTime: 1606824023,
	}
	spec := &models.Spec{
		SecondsPerSlot: 12,
		SlotsPerEpoch:  32,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	clock := NewBeaconClock(genesis, spec, logger)

	if !clock.IsFirstSlotOfEpoch(0) {
		t.Error("Expected slot 0 to be first slot of epoch")
	}

	if !clock.IsFirstSlotOfEpoch(32) {
		t.Error("Expected slot 32 to be first slot of epoch")
	}

	if clock.IsFirstSlotOfEpoch(1) {
		t.Error("Expected slot 1 to not be first slot of epoch")
	}

	if clock.IsFirstSlotOfEpoch(31) {
		t.Error("Expected slot 31 to not be first slot of epoch")
	}
}

func TestBeaconClockIsSlotInEpoch(t *testing.T) {
	genesis := &models.Genesis{
		GenesisTime: 1606824023,
	}
	spec := &models.Spec{
		SecondsPerSlot: 12,
		SlotsPerEpoch:  32,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	clock := NewBeaconClock(genesis, spec, logger)

	// Test slot 15 (position 15 in epoch 0)
	if !clock.IsSlotInEpoch(15, 15) {
		t.Error("Expected slot 15 to be at position 15")
	}

	// Test slot 47 (position 15 in epoch 1)
	if !clock.IsSlotInEpoch(47, 15) {
		t.Error("Expected slot 47 to be at position 15")
	}

	// Test slot 16 (position 16 in epoch 0)
	if clock.IsSlotInEpoch(16, 15) {
		t.Error("Expected slot 16 to not be at position 15")
	}
}

func TestBeaconClockReplayMode(t *testing.T) {
	genesis := &models.Genesis{
		GenesisTime: 1606824023,
	}
	spec := &models.Spec{
		SecondsPerSlot: 12,
		SlotsPerEpoch:  32,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	clock := NewBeaconClock(genesis, spec, logger)

	if clock.IsReplayMode() {
		t.Error("Expected replay mode to be disabled by default")
	}

	startTS := uint64(1606824023 + 1000)
	endTS := uint64(1606824023 + 2000)
	clock.EnableReplayMode(&startTS, &endTS)

	if !clock.IsReplayMode() {
		t.Error("Expected replay mode to be enabled")
	}

	// In replay mode, WaitUntilSlot should not actually wait
	ctx := context.Background()
	start := time.Now()
	clock.WaitUntilSlot(ctx, 1000)
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("Expected wait to be instant in replay mode, took %v", duration)
	}
}

func TestBeaconClockTimeToSlot(t *testing.T) {
	genesis := &models.Genesis{
		GenesisTime: 1606824023,
	}
	spec := &models.Spec{
		SecondsPerSlot: 12,
		SlotsPerEpoch:  32,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	clock := NewBeaconClock(genesis, spec, logger)

	// Test genesis time
	slot := clock.TimeToSlot(genesis.GenesisTime)
	if slot != 0 {
		t.Errorf("Expected slot 0 at genesis, got %d", slot)
	}

	// Test 12 seconds after genesis
	slot = clock.TimeToSlot(genesis.GenesisTime + 12)
	if slot != 1 {
		t.Errorf("Expected slot 1 at genesis+12s, got %d", slot)
	}

	// Test before genesis
	slot = clock.TimeToSlot(genesis.GenesisTime - 1000)
	if slot != 0 {
		t.Errorf("Expected slot 0 before genesis, got %d", slot)
	}
}
