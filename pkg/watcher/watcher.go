package watcher

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/beacon"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/clock"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/duties"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/metrics"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/price"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/proposer"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/validator"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// ValidatorWatcher is the main orchestrator for validator monitoring
type ValidatorWatcher struct {
	config             *models.Config
	beaconClient       *beacon.Client
	clock              *clock.BeaconClock
	proposerSchedule   *proposer.Schedule
	allValidators      *validator.AllValidators
	watchedValidators  *validator.WatchedValidators
	prometheusMetrics  *metrics.PrometheusMetrics
	priceFetcher       *price.Fetcher
	registry           *prometheus.Registry
	logger             *logrus.Logger
	lastProcessedEpoch models.Epoch
	ready              bool // Tracks if watcher has successfully initialized
}

// NewValidatorWatcher creates a new validator watcher
func NewValidatorWatcher(cfg *models.Config, logger *logrus.Logger) (*ValidatorWatcher, error) {
	// Create beacon client
	beaconClient := beacon.NewClient(cfg.BeaconURL, cfg.BeaconTimeout.ToDuration(), logger)

	// Initialize registries
	allValidators := validator.NewAllValidators()
	watchedValidators := validator.NewWatchedValidators()

	// Create Prometheus registry and metrics
	registry := prometheus.NewRegistry()
	prometheusMetrics := metrics.NewPrometheusMetrics(registry)

	// Create price fetcher
	priceFetcher := price.NewFetcher(logger)

	watcher := &ValidatorWatcher{
		config:            cfg,
		beaconClient:      beaconClient,
		allValidators:     allValidators,
		watchedValidators: watchedValidators,
		prometheusMetrics: prometheusMetrics,
		priceFetcher:      priceFetcher,
		registry:          registry,
		logger:            logger,
	}

	return watcher, nil
}

// Run starts the validator watcher main loop
func (w *ValidatorWatcher) Run(ctx context.Context) error {
	// Initialize beacon clock
	if err := w.initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Start Prometheus HTTP server
	go w.startMetricsServer()

	// Main monitoring loop
	return w.mainLoop(ctx)
}

// initialize sets up the watcher by fetching initial data
func (w *ValidatorWatcher) initialize(ctx context.Context) error {
	w.logger.Info("Initializing validator watcher...")

	// Fetch genesis and spec (optional - some public RPC endpoints may not support these)
	genesis, err := w.beaconClient.GetGenesis(ctx)
	if err != nil {
		w.logger.WithError(err).Warn("Failed to get genesis - clock-based monitoring will be disabled")
		w.logger.Info("Continuing without clock initialization - can still fetch validator data")
		w.logger.Info("NOTE: Some public RPC endpoints do not support all Beacon API endpoints.")
		w.logger.Info("      You can still load validator snapshots, but real-time monitoring requires a full beacon node.")
		// Don't return error, just skip clock initialization
		genesis = nil
	}

	var spec *models.Spec
	if genesis != nil {
		spec, err = w.beaconClient.GetSpec(ctx)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to get spec - clock-based monitoring will be disabled")
			genesis = nil // Also disable clock if we can't get spec
		}
	}

	// Initialize clock only if we have genesis and spec
	if genesis != nil && spec != nil {
		w.clock = clock.NewBeaconClock(genesis, spec, w.logger)
		if w.config.ReplayStartAtTS != nil {
			w.clock.EnableReplayMode(w.config.ReplayStartAtTS, w.config.ReplayEndAtTS)
		}

		// Initialize proposer schedule
		w.proposerSchedule = proposer.NewSchedule(w.beaconClient, w.logger)

		w.logger.WithFields(logrus.Fields{
			"genesis_time":     genesis.GenesisTime,
			"seconds_per_slot": spec.SecondsPerSlot,
			"slots_per_epoch":  spec.SlotsPerEpoch,
			"current_slot":     w.clock.CurrentSlot(),
			"current_epoch":    w.clock.CurrentEpoch(),
		}).Info("Initialized beacon clock")
	} else {
		w.logger.Info("Clock not initialized - running in snapshot mode")
	}

	// Load validators immediately (this works without clock)
	if err := w.loadAllValidators(ctx); err != nil {
		return fmt.Errorf("failed to load validators: %w", err)
	}

	// Mark watcher as ready after successful initialization
	w.ready = true
	w.logger.Info("✅ Validator watcher ready - health checks will now pass")

	return nil
}

// loadAllValidators loads all validators from the beacon node
func (w *ValidatorWatcher) loadAllValidators(ctx context.Context) error {
	// Check if we should load all validators (default true)
	if !w.config.ShouldLoadAllValidators() {
		w.logger.Info("Skipping all validators load (load_all_validators=false)")
		w.logger.Info("Network-wide comparison metrics will not be available")
		return w.loadWatchedValidatorsOnly(ctx)
	}

	w.logger.Info("Loading all validators from beacon node (this may take 30-60 seconds for 2M+ validators)...")
	w.logger.Info("This enables network-wide performance comparison (like Kiln's original behavior)")

	allVals, err := w.beaconClient.GetAllValidators(ctx, "head")
	if err != nil {
		w.logger.WithError(err).Error("Failed to load all validators")
		w.logger.Warn("Network comparison will be unavailable - continuing with watched validators only")
		return w.loadWatchedValidatorsOnly(ctx)
	}

	w.allValidators.Update(allVals)
	w.logger.WithField("count", w.allValidators.Count()).Info("✅ Successfully loaded all validators")

	// Load watched validators
	if len(w.config.WatchedKeys) > 0 {
		w.logger.WithField("count", len(w.config.WatchedKeys)).Info("Loading watched validators...")

		var allWatchedVals []models.Validator

		if allVals != nil {
			// Use all validators to find indices (fast - no API call needed!)
			w.logger.Info("Using cached validator set to build watched validators (no API calls needed)")
			watchedIndices := make([]models.ValidatorIndex, 0)
			for _, wk := range w.config.WatchedKeys {
				if v, ok := w.allValidators.GetByPubkey(wk.PublicKey); ok {
					watchedIndices = append(watchedIndices, v.Index)
					// We already have the validator data, just extract it
					if fullVal, ok := w.allValidators.Get(v.Index); ok {
						allWatchedVals = append(allWatchedVals, *fullVal)
					}
				} else {
					w.logger.WithField("pubkey", wk.PublicKey[:10]+"...").Warn("Watched validator not found in all validators set")
				}
			}
			w.logger.WithField("found", len(allWatchedVals)).Info("Extracted watched validators from cached set")
		} else {
			// Can't use all validators, fetch by public keys in batches
			w.logger.Info("Fetching watched validators by public keys in batches (since all validators unavailable)...")
			batchSize := 100
			for i := 0; i < len(w.config.WatchedKeys); i += batchSize {
				end := i + batchSize
				if end > len(w.config.WatchedKeys) {
					end = len(w.config.WatchedKeys)
				}

				pubkeys := make([]string, end-i)
				for j, wk := range w.config.WatchedKeys[i:end] {
					pubkeys[j] = wk.PublicKey
				}

				w.logger.WithFields(logrus.Fields{
					"batch": i/batchSize + 1,
					"total": (len(w.config.WatchedKeys) + batchSize - 1) / batchSize,
					"size":  len(pubkeys),
				}).Debug("Fetching batch...")

				batchVals, err := w.beaconClient.GetValidatorsByPubkeys(ctx, "head", pubkeys)
				if err != nil {
					return fmt.Errorf("failed to get watched validators batch %d: %w", i/batchSize+1, err)
				}
				allWatchedVals = append(allWatchedVals, batchVals...)
			}
			w.logger.WithField("total", len(allWatchedVals)).Info("Fetched all watched validators in batches")
		}

		if len(allWatchedVals) > 0 {
			if err := w.watchedValidators.Update(allWatchedVals, w.config.WatchedKeys); err != nil {
				return fmt.Errorf("failed to update watched validators: %w", err)
			}
			w.logger.WithField("count", w.watchedValidators.Count()).Info("Successfully loaded watched validators")
		} else {
			w.logger.Warn("No watched validators found - check your configuration")
		}
	}

	return nil
}

// loadWatchedValidatorsOnly loads only the watched validators (when all validators load is disabled)
func (w *ValidatorWatcher) loadWatchedValidatorsOnly(ctx context.Context) error {
	if len(w.config.WatchedKeys) == 0 {
		w.logger.Warn("No watched validators configured")
		return nil
	}

	w.logger.WithField("count", len(w.config.WatchedKeys)).Info("Loading watched validators by public keys...")

	// Fetch by public keys in batches
	batchSize := 100
	var allWatchedVals []models.Validator

	for i := 0; i < len(w.config.WatchedKeys); i += batchSize {
		end := i + batchSize
		if end > len(w.config.WatchedKeys) {
			end = len(w.config.WatchedKeys)
		}

		pubkeys := make([]string, end-i)
		for j, wk := range w.config.WatchedKeys[i:end] {
			pubkeys[j] = wk.PublicKey
		}

		w.logger.WithFields(logrus.Fields{
			"batch": i/batchSize + 1,
			"total": (len(w.config.WatchedKeys) + batchSize - 1) / batchSize,
			"size":  len(pubkeys),
		}).Debug("Fetching batch...")

		batchVals, err := w.beaconClient.GetValidatorsByPubkeys(ctx, "head", pubkeys)
		if err != nil {
			return fmt.Errorf("failed to get watched validators batch %d: %w", i/batchSize+1, err)
		}
		allWatchedVals = append(allWatchedVals, batchVals...)
	}

	if len(allWatchedVals) > 0 {
		if err := w.watchedValidators.Update(allWatchedVals, w.config.WatchedKeys); err != nil {
			return fmt.Errorf("failed to update watched validators: %w", err)
		}
		w.logger.WithField("count", w.watchedValidators.Count()).Info("✅ Successfully loaded watched validators")
	} else {
		w.logger.Warn("No watched validators found - check your configuration")
	}

	return nil
}

// mainLoop runs the main monitoring loop
func (w *ValidatorWatcher) mainLoop(ctx context.Context) error {
	// If no clock, we're in snapshot mode - just load data and exit
	if w.clock == nil {
		w.logger.Info("Running in snapshot mode - no continuous monitoring")
		w.logger.Info("Validator data loaded successfully")

		// Update metrics once
		allVals := w.allValidators.GetAll()
		watchedVals := w.watchedValidators.GetAll()

		w.logger.WithFields(logrus.Fields{
			"all_validators":     len(allVals),
			"watched_validators": len(watchedVals),
		}).Info("Snapshot complete")

		// Keep metrics server running
		select {
		case <-ctx.Done():
			w.logger.Info("Shutting down...")
			return ctx.Err()
		}
	}

	w.logger.Info("Starting main monitoring loop...")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Shutting down...")
			return ctx.Err()
		default:
		}

		// Check replay mode completion
		if w.clock.IsReplayMode() && w.clock.ReplayComplete() {
			w.logger.Info("Replay mode complete")
			return nil
		}

		// Get current slot
		currentSlot := w.clock.CurrentSlot()
		currentEpoch := w.clock.SlotToEpoch(currentSlot)

		// Log slot info every 10 slots or if it's the first slot of an epoch
		if currentSlot%10 == 0 || w.clock.IsFirstSlotOfEpoch(currentSlot) {
			w.logger.WithFields(logrus.Fields{
				"slot":               currentSlot,
				"epoch":              currentEpoch,
				"slot_in_epoch":      currentSlot % models.Slot(w.clock.SlotsPerEpoch()),
				"watched_validators": w.watchedValidators.Count(),
			}).Info("📊 Slot checkpoint")
		}

		// Process epoch if it's the first slot
		if w.clock.IsFirstSlotOfEpoch(currentSlot) {
			if err := w.processEpoch(ctx, currentEpoch); err != nil {
				w.logger.WithError(err).Error("Failed to process epoch")
			}
		}

		// Process slot-specific tasks
		if w.clock.IsSlotInEpoch(currentSlot, 16) {
			// Process liveness at slot 16
			if err := w.processLiveness(ctx, currentEpoch-1); err != nil {
				w.logger.WithError(err).Error("Failed to process liveness")
			}
		}

		if w.clock.IsSlotInEpoch(currentSlot, 17) {
			// Process rewards at slot 17 (for epoch - 2)
			if currentEpoch >= 2 {
				if err := w.processRewards(ctx, currentEpoch-2); err != nil {
					w.logger.WithError(err).Error("Failed to process rewards")
				}
			}
		}

		if w.clock.IsSlotInEpoch(currentSlot, 15) {
			// Reload config at slot 15
			if err := w.reloadConfig(); err != nil {
				w.logger.WithError(err).Error("Failed to reload config")
			}
		}

		// Process current slot
		if err := w.processSlot(ctx, currentSlot); err != nil {
			w.logger.WithError(err).Error("Failed to process slot")
		}

		// Update metrics
		w.updateMetrics(currentSlot, currentEpoch)

		// Wait for next slot
		if _, err := w.clock.WaitUntilNextSlot(ctx); err != nil {
			return err
		}

		// Cleanup old data
		w.cleanup(currentSlot)
	}
}

// processEpoch processes epoch-specific tasks
func (w *ValidatorWatcher) processEpoch(ctx context.Context, epoch models.Epoch) error {
	w.logger.WithField("epoch", epoch).Info("Processing epoch")

	// Load ALL validators (full 2M+ set) in background - non-blocking
	// This is used for network-wide comparison metrics
	if w.config.ShouldLoadAllValidators() {
		go func() {
			allVals, err := w.beaconClient.GetAllValidators(ctx, "head")
			if err != nil {
				w.logger.WithError(err).Warn("Failed to load all validators (background)")
				return
			}
			w.allValidators.Update(allVals)
			w.logger.WithField("count", w.allValidators.Count()).Debug("✅ Updated all validators cache (background)")
		}()
	}

	// Load watched validators
	watchedIndices := make([]models.ValidatorIndex, 0)
	for _, wk := range w.config.WatchedKeys {
		if v, ok := w.allValidators.GetByPubkey(wk.PublicKey); ok {
			watchedIndices = append(watchedIndices, v.Index)
		} else {
			w.logger.WithField("pubkey", wk.PublicKey).Warn("Watched validator not found")
		}
	}

	if len(watchedIndices) > 0 {
		watchedVals, err := w.beaconClient.GetValidators(ctx, "head", watchedIndices)
		if err != nil {
			return fmt.Errorf("failed to get watched validators: %w", err)
		}
		if err := w.watchedValidators.Update(watchedVals, w.config.WatchedKeys); err != nil {
			return fmt.Errorf("failed to update watched validators: %w", err)
		}
		w.logger.WithField("count", w.watchedValidators.Count()).Info("Updated watched validators")
	}

	// Update proposer schedule for current and next epoch
	if err := w.proposerSchedule.Update(ctx, epoch); err != nil {
		w.logger.WithError(err).Warn("Failed to update proposer schedule for current epoch")
	}
	if err := w.proposerSchedule.Update(ctx, epoch+1); err != nil {
		w.logger.WithError(err).Warn("Failed to update proposer schedule for next epoch")
	}

	// Fetch pending deposits, consolidations, withdrawals
	if _, err := w.beaconClient.GetPendingDeposits(ctx, "head"); err != nil {
		w.logger.WithError(err).Debug("Failed to get pending deposits")
	}
	if _, err := w.beaconClient.GetPendingConsolidations(ctx, "head"); err != nil {
		w.logger.WithError(err).Debug("Failed to get pending consolidations")
	}
	if _, err := w.beaconClient.GetPendingWithdrawals(ctx, "head"); err != nil {
		w.logger.WithError(err).Debug("Failed to get pending withdrawals")
	}

	w.lastProcessedEpoch = epoch
	return nil
}

// processSlot processes slot-specific tasks
func (w *ValidatorWatcher) processSlot(ctx context.Context, slot models.Slot) error {
	// Process block
	if err := w.processBlock(ctx, slot); err != nil {
		w.logger.WithError(err).Debug("Failed to process block (may not exist)")
	}

	// Process attestations
	if err := w.processAttestations(ctx, slot); err != nil {
		w.logger.WithError(err).Debug("Failed to process attestations")
	}

	return nil
}

// processBlock processes a block and updates block production metrics
func (w *ValidatorWatcher) processBlock(ctx context.Context, slot models.Slot) error {
	block, err := w.beaconClient.GetBlock(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		// Block may not exist (missed)
		if proposerIndex, ok := w.proposerSchedule.GetProposer(slot); ok {
			if v, ok := w.watchedValidators.Get(proposerIndex); ok {
				w.watchedValidators.UpdateMetrics(proposerIndex, func(wv *validator.WatchedValidator) {
					wv.MissedBlocks++
				})

				// Get primary label (non-scope label)
				primaryLabel := "unknown"
				for _, label := range v.Labels {
					if !strings.HasPrefix(label, "scope:") && !strings.HasPrefix(label, "key:") {
						primaryLabel = label
						break
					}
				}

				w.logger.WithFields(logrus.Fields{
					"slot":            slot,
					"validator_index": proposerIndex,
					"pubkey":          v.Data.Pubkey[:14] + "...",
					"label":           primaryLabel,
					"total_missed":    v.MissedBlocks + 1,
				}).Warn("❌ MISSED BLOCK")
			}
		}
		return err
	}

	// Block was proposed
	proposerIndex := models.ValidatorIndex(block.Message.ProposerIndex)
	if v, ok := w.watchedValidators.Get(proposerIndex); ok {
		w.watchedValidators.UpdateMetrics(proposerIndex, func(wv *validator.WatchedValidator) {
			wv.ProposedBlocks++
		})

		// Get primary label
		primaryLabel := "unknown"
		for _, label := range v.Labels {
			if !strings.HasPrefix(label, "scope:") && !strings.HasPrefix(label, "key:") {
				primaryLabel = label
				break
			}
		}

		// Get fee recipient if available
		feeRecipient := "unknown"
		if block.Message.Body.ExecutionPayload != nil {
			feeRecipient = block.Message.Body.ExecutionPayload.FeeRecipient[:10] + "..."
		}

		w.logger.WithFields(logrus.Fields{
			"slot":            slot,
			"validator_index": proposerIndex,
			"pubkey":          v.Data.Pubkey[:14] + "...",
			"label":           primaryLabel,
			"fee_recipient":   feeRecipient,
			"total_proposed":  v.ProposedBlocks + 1,
		}).Info("✅ BLOCK PROPOSED")
	}

	return nil
}

// processAttestations processes attestations for a slot
func (w *ValidatorWatcher) processAttestations(ctx context.Context, slot models.Slot) error {
	// Per Ethereum consensus: attestations in the current slot are FOR the previous slot
	// We need to:
	// 1. Get attestations from current slot's block
	// 2. Get committees from PREVIOUS slot
	// 3. Filter attestations to only those for previous slot

	if slot == 0 {
		return nil // No previous slot
	}

	previousSlot := slot - 1

	// Get attestations from current slot's block
	attestations, err := w.beaconClient.GetAttestations(ctx, slot)
	if err != nil {
		return err
	}

	// Get committees for the PREVIOUS slot (where validators had duties)
	committees, err := w.beaconClient.GetCommittees(ctx, "head", nil, &previousSlot)
	if err != nil {
		return err
	}

	// Filter attestations to only those for the previous slot
	filteredAttestations := make([]models.Attestation, 0)
	for _, att := range attestations {
		if att.Data.Slot == previousSlot {
			filteredAttestations = append(filteredAttestations, att)
		}
	}

	// Build set of validators with duties in the PREVIOUS slot
	validatorsWithDuties := make(map[models.ValidatorIndex]bool)
	for _, committee := range committees {
		for _, validatorStr := range committee.Validators {
			var validatorIdx models.ValidatorIndex
			fmt.Sscanf(validatorStr, "%d", &validatorIdx)
			validatorsWithDuties[validatorIdx] = true
		}
	}

	// Process attestations (for previous slot)
	attested, err := duties.ProcessAttestations(filteredAttestations, committees)
	if err != nil {
		return err
	}

	// Update attestation duty metrics - ONLY for validators with duties this slot
	missedCount := 0
	dutiesCount := 0
	var missedDetails []string
	missedByLabel := make(map[string]int) // Track misses by primary label

	for validatorIdx := range validatorsWithDuties {
		// Only process if this is one of our watched validators
		v, ok := w.watchedValidators.Get(validatorIdx)
		if !ok {
			continue
		}

		dutiesCount++

		if attested[validatorIdx] {
			// Successfully attested
			w.watchedValidators.UpdateMetrics(validatorIdx, func(wv *validator.WatchedValidator) {
				wv.AttestationDutiesSuccess++
				wv.AttestationDuties++
				wv.ConsecutiveMissedAttest = 0
			})
		} else {
			// Missed attestation
			missedCount++

			// Get primary label
			primaryLabel := "unknown"
			for _, label := range v.Labels {
				if !strings.HasPrefix(label, "scope:") && !strings.HasPrefix(label, "key:") {
					primaryLabel = label
					break
				}
			}
			missedByLabel[primaryLabel]++

			w.watchedValidators.UpdateMetrics(validatorIdx, func(wv *validator.WatchedValidator) {
				wv.ConsecutiveMissedAttest++
				wv.AttestationDuties++
			})

			// Log first 5 missed attestations with details
			if len(missedDetails) < 5 {
				missedDetails = append(missedDetails, fmt.Sprintf("v%d (%s, consecutive: %d)",
					validatorIdx, primaryLabel, v.ConsecutiveMissedAttest+1))
			}
		}
	}

	// Log attestation summary if there were any misses
	if missedCount > 0 {
		logFields := logrus.Fields{
			"current_slot":   slot,
			"attesting_slot": previousSlot,
			"missed_count":   missedCount,
			"duties_count":   dutiesCount,
			"miss_rate":      fmt.Sprintf("%.2f%%", float64(missedCount)*100/float64(dutiesCount)),
		}

		if len(missedDetails) > 0 {
			logFields["examples"] = strings.Join(missedDetails, "; ")
		}

		if missedCount > 5 {
			logFields["more"] = fmt.Sprintf("+%d more", missedCount-5)
		}

		// Show breakdown by label
		if len(missedByLabel) > 0 {
			labelBreakdown := make([]string, 0)
			for label, count := range missedByLabel {
				labelBreakdown = append(labelBreakdown, fmt.Sprintf("%s:%d", label, count))
			}
			logFields["by_label"] = strings.Join(labelBreakdown, ", ")
		}

		w.logger.WithFields(logFields).Warn("⚠️  MISSED ATTESTATIONS")
	} else if dutiesCount > 0 {
		// All attestations successful - log occasionally
		if dutiesCount > 100 || slot%32 == 0 { // Log if many duties or once per epoch
			w.logger.WithFields(logrus.Fields{
				"current_slot":   slot,
				"attesting_slot": previousSlot,
				"duties_count":   dutiesCount,
			}).Debug("✅ All attestations successful")
		}
	}

	return nil
}

// processLiveness processes validator liveness data
func (w *ValidatorWatcher) processLiveness(ctx context.Context, epoch models.Epoch) error {
	indices := make([]models.ValidatorIndex, 0)
	for _, v := range w.watchedValidators.GetAll() {
		indices = append(indices, v.Index)
	}

	if len(indices) == 0 {
		return nil
	}

	liveness, err := w.beaconClient.GetValidatorsLiveness(ctx, epoch, indices)
	if err != nil {
		return err
	}

	livenessMap := duties.ProcessLiveness(liveness)

	notLiveCount := 0
	var notLiveDetails []string

	for idx, isLive := range livenessMap {
		if !isLive {
			notLiveCount++
			w.watchedValidators.UpdateMetrics(idx, func(wv *validator.WatchedValidator) {
				wv.MissedAttestations++
			})

			// Collect details for first 5 non-live validators
			if notLiveCount <= 5 {
				if v, ok := w.watchedValidators.Get(idx); ok {
					primaryLabel := "unknown"
					for _, label := range v.Labels {
						if !strings.HasPrefix(label, "scope:") && !strings.HasPrefix(label, "key:") {
							primaryLabel = label
							break
						}
					}
					notLiveDetails = append(notLiveDetails, fmt.Sprintf("%d (%s)", idx, primaryLabel))
				}
			}
		}
	}

	// Log liveness summary
	liveCount := len(livenessMap) - notLiveCount
	logFields := logrus.Fields{
		"epoch":      epoch,
		"live":       liveCount,
		"not_live":   notLiveCount,
		"total":      len(livenessMap),
		"percentage": fmt.Sprintf("%.1f%%", float64(liveCount)*100/float64(len(livenessMap))),
	}

	if notLiveCount > 0 && len(notLiveDetails) > 0 {
		logFields["not_live_validators"] = strings.Join(notLiveDetails, "; ")
		if notLiveCount > 5 {
			logFields["more"] = fmt.Sprintf("+%d more", notLiveCount-5)
		}
		w.logger.WithFields(logFields).Warn("🔴 Liveness check: some validators not live")
	} else {
		w.logger.WithFields(logFields).Info("🟢 Liveness check: all validators live")
	}

	return nil
}

// processRewards processes reward data
func (w *ValidatorWatcher) processRewards(ctx context.Context, epoch models.Epoch) error {
	// Build map of validator index -> effective balance
	validatorBalances := make(map[models.ValidatorIndex]models.Gwei)
	for _, v := range w.watchedValidators.GetAll() {
		validatorBalances[v.Index] = v.Data.EffectiveBalance
	}

	if len(validatorBalances) == 0 {
		return nil
	}

	// Convert to indices slice for API call
	indices := make([]models.ValidatorIndex, 0, len(validatorBalances))
	for idx := range validatorBalances {
		indices = append(indices, idx)
	}

	rewards, err := w.beaconClient.GetRewards(ctx, epoch, indices)
	if err != nil {
		return err
	}

	rewardData, err := duties.ProcessRewards(rewards, validatorBalances)
	if err != nil {
		return err
	}

	// Track statistics
	suboptimalSourceCount := 0
	suboptimalTargetCount := 0
	suboptimalHeadCount := 0
	negativeRewardsCount := 0
	var totalIdeal models.Gwei
	var totalActual models.SignedGwei

	for idx, data := range rewardData {
		w.watchedValidators.UpdateMetrics(idx, func(wv *validator.WatchedValidator) {
			if data.SuboptimalSource {
				wv.SuboptimalSourceVotes++
			}
			if data.SuboptimalTarget {
				wv.SuboptimalTargetVotes++
			}
			if data.SuboptimalHead {
				wv.SuboptimalHeadVotes++
			}
			wv.IdealConsensusRewards = data.IdealTotal
			wv.ConsensusRewards = data.ActualTotal
		})

		// Aggregate stats
		if data.SuboptimalSource {
			suboptimalSourceCount++
		}
		if data.SuboptimalTarget {
			suboptimalTargetCount++
		}
		if data.SuboptimalHead {
			suboptimalHeadCount++
		}
		if data.ActualTotal < 0 {
			negativeRewardsCount++
		}
		totalIdeal += data.IdealTotal
		totalActual += data.ActualTotal
	}

	// Calculate performance rate
	performanceRate := 0.0
	if totalIdeal > 0 {
		performanceRate = float64(totalActual) / float64(totalIdeal) * 100
	}

	// Log rewards summary
	logFields := logrus.Fields{
		"epoch":            epoch,
		"validators":       len(rewardData),
		"ideal_gwei":       totalIdeal,
		"actual_gwei":      totalActual,
		"performance_rate": fmt.Sprintf("%.2f%%", performanceRate),
		"penalties":        negativeRewardsCount,
	}

	if suboptimalSourceCount > 0 || suboptimalTargetCount > 0 || suboptimalHeadCount > 0 {
		logFields["suboptimal_source"] = suboptimalSourceCount
		logFields["suboptimal_target"] = suboptimalTargetCount
		logFields["suboptimal_head"] = suboptimalHeadCount
		w.logger.WithFields(logFields).Warn("⚠️  Rewards processed: suboptimal attestations detected")
	} else if negativeRewardsCount > 0 {
		w.logger.WithFields(logFields).Warn("⚠️  Rewards processed: penalties detected")
	} else {
		w.logger.WithFields(logFields).Info("💰 Rewards processed: optimal performance")
	}

	return nil
}

// reloadConfig reloads the configuration
func (w *ValidatorWatcher) reloadConfig() error {
	// Re-read config file if path is available
	// For now, just log
	w.logger.Debug("Config reload requested (not implemented yet)")
	return nil
}

// updateMetrics updates Prometheus metrics
func (w *ValidatorWatcher) updateMetrics(slot models.Slot, epoch models.Epoch) {
	// Compute metrics from watched validators
	watchedVals := w.watchedValidators.GetAll()
	metricsByLabel := metrics.ComputeMetrics(watchedVals, slot)

	// Add network-wide metrics
	allVals := w.allValidators.GetAll()
	networkMetrics := metrics.ComputeNetworkMetrics(allVals)
	metricsByLabel["scope:all-network"] = networkMetrics

	// Update Prometheus
	w.prometheusMetrics.UpdateMetrics(metricsByLabel, slot, epoch, w.config.Network)

	// Fetch and update network-level metrics
	w.updateNetworkMetrics()

	// Log summary
	if watchedMetrics, ok := metricsByLabel["scope:watched"]; ok {
		w.logger.WithFields(logrus.Fields{
			"validators":          watchedMetrics.ValidatorCount,
			"missed_attestations": watchedMetrics.MissedAttestations,
			"proposed_blocks":     watchedMetrics.ProposedBlocks,
			"missed_blocks":       watchedMetrics.MissedBlocks,
			"consensus_rate":      watchedMetrics.ConsensusRewardsRate,
		}).Info("Metrics updated")
	}

	// Log operator-level performance breakdown (only if rewards have been processed)
	if watchedMetrics, ok := metricsByLabel["scope:watched"]; ok && watchedMetrics.IdealConsensusRewards > 0 {
		for label, metrics := range metricsByLabel {
			// Skip scope labels, keys, and name: labels (only show operator: labels to avoid duplicates)
			if strings.HasPrefix(label, "scope:") || strings.HasPrefix(label, "key:") || strings.HasPrefix(label, "name:") {
				continue
			}

			// Calculate active validator count and metrics (excluding exited/pending validators)
			activeCount := 0
			activeCount += metrics.StatusCounts[models.StatusActiveOngoing]
			activeCount += metrics.StatusCounts[models.StatusActiveExiting]
			activeCount += metrics.StatusCounts[models.StatusActiveSlashed]

			// Calculate performance rate as percentage
			performanceRate := metrics.ConsensusRewardsRate * 100
			missRate := 0.0
			if metrics.AttestationDuties > 0 {
				missRate = float64(metrics.MissedAttestations) * 100 / float64(metrics.AttestationDuties)
			}

			logFields := logrus.Fields{
				"label":               label,
				"validators":          metrics.ValidatorCount,
				"active_validators":   activeCount,
				"performance_rate":    fmt.Sprintf("%.2f%%", performanceRate),
				"missed_attestations": metrics.MissedAttestations,
				"attestation_duties":  metrics.AttestationDuties,
				"miss_rate":           fmt.Sprintf("%.2f%%", missRate),
			}

			if metrics.ProposedBlocks > 0 || metrics.MissedBlocks > 0 {
				logFields["proposed_blocks"] = metrics.ProposedBlocks
				logFields["missed_blocks"] = metrics.MissedBlocks
			}

			// Skip performance evaluation if no active validators (expected to have 0% performance)
			if activeCount == 0 {
				w.logger.WithFields(logFields).Debug("📊 Operator performance: no active validators")
				continue
			}

			// Color-code based on performance and add validator details for poor performers
			if performanceRate >= 100.0 {
				w.logger.WithFields(logFields).Info("📊 Operator performance: excellent")
			} else if performanceRate >= 95.0 {
				w.logger.WithFields(logFields).Info("📊 Operator performance: good")
			} else if performanceRate >= 90.0 {
				w.logger.WithFields(logFields).Warn("📊 Operator performance: needs attention")
			} else {
				// For critical performance, show top offending validators
				offendingValidators := w.getTopOffendingValidators(label, 5)
				if len(offendingValidators) > 0 {
					logFields["top_offenders"] = offendingValidators
				}
				w.logger.WithFields(logFields).Error("📊 Operator performance: critical")
			}
		}
	}
}

// getTopOffendingValidators returns the top N validators with most issues for a given label
func (w *ValidatorWatcher) getTopOffendingValidators(label string, limit int) string {
	type validatorIssue struct {
		index              models.ValidatorIndex
		pubkey             string
		status             models.ValidatorStatus
		missedAttestations uint64
		performance        float64
	}

	var issues []validatorIssue

	// Get all validators with this label
	for _, v := range w.watchedValidators.GetAll() {
		hasLabel := false
		for _, l := range v.Labels {
			if l == label {
				hasLabel = true
				break
			}
		}
		if !hasLabel {
			continue
		}

		// Skip validators that are not expected to be attesting
		// Only include active validators (active_ongoing, active_exiting, active_slashed)
		if v.Status != models.StatusActiveOngoing &&
			v.Status != models.StatusActiveExiting &&
			v.Status != models.StatusActiveSlashed {
			continue
		}

		// Calculate validator's performance rate
		performance := 0.0
		if v.IdealConsensusRewards > 0 {
			performance = float64(v.ConsensusRewards) / float64(v.IdealConsensusRewards) * 100
		}

		// Include if has issues
		if v.MissedAttestations > 0 || performance < 90.0 {
			issues = append(issues, validatorIssue{
				index:              v.Index,
				pubkey:             v.Data.Pubkey[:14] + "...", // Truncate for readability
				status:             v.Status,
				missedAttestations: v.MissedAttestations,
				performance:        performance,
			})
		}
	}

	// Sort by missed attestations (descending)
	for i := 0; i < len(issues)-1; i++ {
		for j := i + 1; j < len(issues); j++ {
			if issues[j].missedAttestations > issues[i].missedAttestations {
				issues[i], issues[j] = issues[j], issues[i]
			}
		}
	}

	// Format top N
	if len(issues) > limit {
		issues = issues[:limit]
	}

	if len(issues) == 0 {
		return ""
	}

	result := ""
	for i, issue := range issues {
		if i > 0 {
			result += "; "
		}
		result += fmt.Sprintf("%d(%s):missed=%d,perf=%.1f%%",
			issue.index, issue.pubkey, issue.missedAttestations, issue.performance)
	}

	return result
}

// cleanup removes old data
func (w *ValidatorWatcher) cleanup(currentSlot models.Slot) {
	// Keep last 2 epochs worth of proposer duties
	cleanupSlot := currentSlot
	if currentSlot > models.Slot(w.clock.SlotsPerEpoch()*2) {
		cleanupSlot = currentSlot - models.Slot(w.clock.SlotsPerEpoch()*2)
	}
	w.proposerSchedule.Cleanup(cleanupSlot)
}

// startMetricsServer starts the Prometheus metrics HTTP server
func (w *ValidatorWatcher) startMetricsServer() {
	addr := fmt.Sprintf(":%d", w.config.MetricsPort)
	w.logger.WithField("address", addr).Info("Starting metrics server")

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(w.registry, promhttp.HandlerOpts{}))

	// Health check - always returns 200 OK if server is running
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness check - returns 200 OK only after successful initialization
	mux.HandleFunc("/ready", func(rw http.ResponseWriter, r *http.Request) {
		if w.ready {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("READY"))
		} else {
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Write([]byte("NOT READY"))
		}
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		w.logger.WithError(err).Error("Metrics server failed")
	}
}

// updateNetworkMetrics fetches and updates network-level metrics (price, pending operations)
func (w *ValidatorWatcher) updateNetworkMetrics() {
	ctx := context.Background()
	network := w.config.Network

	// Fetch ETH price from Coinbase
	ethPrice := w.priceFetcher.GetCurrentETHPrice()

	// Fetch pending deposits
	var pendingDepositsCount, pendingDepositsValue float64
	if deposits, err := w.beaconClient.GetPendingDeposits(ctx, "head"); err == nil {
		pendingDepositsCount = float64(len(deposits))
		for _, deposit := range deposits {
			pendingDepositsValue += float64(deposit.Amount)
		}
	} else {
		w.logger.WithError(err).Debug("Failed to fetch pending deposits")
	}

	// Fetch pending consolidations
	var pendingConsolidationsCount float64
	if consolidations, err := w.beaconClient.GetPendingConsolidations(ctx, "head"); err == nil {
		pendingConsolidationsCount = float64(len(consolidations))
	} else {
		w.logger.WithError(err).Debug("Failed to fetch pending consolidations")
	}

	// Fetch pending withdrawals
	var pendingWithdrawalsCount float64
	if withdrawals, err := w.beaconClient.GetPendingWithdrawals(ctx, "head"); err == nil {
		pendingWithdrawalsCount = float64(len(withdrawals))
	} else {
		w.logger.WithError(err).Debug("Failed to fetch pending withdrawals")
	}

	// Set network metrics
	w.prometheusMetrics.SetNetworkMetrics(
		network,
		ethPrice,
		pendingDepositsCount,
		pendingDepositsValue,
		pendingConsolidationsCount,
		pendingWithdrawalsCount,
	)

	w.logger.WithFields(logrus.Fields{
		"eth_price":             ethPrice,
		"pending_deposits":      pendingDepositsCount,
		"pending_consolidations": pendingConsolidationsCount,
		"pending_withdrawals":   pendingWithdrawalsCount,
	}).Debug("Updated network metrics")
}
