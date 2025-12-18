package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/sirupsen/logrus"
)

const (
	maxRetries      = 3
	retryDelay      = 2 * time.Second
	contentTypeJSON = "application/json"
)

// Client represents a Beacon Chain API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewClient creates a new Beacon Chain API client
func NewClient(baseURL string, timeout time.Duration, logger *logrus.Logger) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
			c.logger.Debugf("Retrying request to %s (attempt %d/%d)", path, attempt+1, maxRetries)
		}

		var reqBody io.Reader
		if body != nil {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewBuffer(jsonData)
		}

		url := c.baseURL + path
		c.logger.Debugf("Making request: %s %s", method, url)
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		if body != nil {
			req.Header.Set("Content-Type", contentTypeJSON)
		}
		req.Header.Set("Accept", contentTypeJSON)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			// For error responses, read the body for the error message
			respBody, _ := io.ReadAll(resp.Body)
			// Provide helpful error messages
			if resp.StatusCode == 404 {
				lastErr = fmt.Errorf("endpoint not found (HTTP 404): %s - this beacon node may not support this API endpoint. Response: %s", url, string(respBody))
			} else {
				lastErr = fmt.Errorf("HTTP %d: %s - URL: %s", resp.StatusCode, string(respBody), url)
			}
			// Retry on 5xx errors
			if resp.StatusCode >= 500 {
				continue
			}
			return lastErr
		}

		if result != nil {
			// Use streaming JSON decoder to avoid loading entire response into memory
			// This is critical for large responses like 2M+ validators (~600MB)
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

// GetGenesis retrieves the genesis configuration
func (c *Client) GetGenesis(ctx context.Context) (*models.Genesis, error) {
	var response struct {
		Data models.Genesis `json:"data"`
	}

	if err := c.doRequest(ctx, http.MethodGet, "/eth/v1/beacon/genesis", nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get genesis: %w", err)
	}

	return &response.Data, nil
}

// GetSpec retrieves the beacon chain specification
func (c *Client) GetSpec(ctx context.Context) (*models.Spec, error) {
	var response struct {
		Data models.Spec `json:"data"`
	}

	if err := c.doRequest(ctx, http.MethodGet, "/eth/v1/config/spec", nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get spec: %w", err)
	}

	return &response.Data, nil
}

// GetHeader retrieves a block header by state ID
func (c *Client) GetHeader(ctx context.Context, stateID string) (*models.BeaconHeader, error) {
	var response struct {
		Data models.BeaconHeader `json:"data"`
	}

	path := fmt.Sprintf("/eth/v1/beacon/headers/%s", stateID)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get header: %w", err)
	}

	return &response.Data, nil
}

// GetValidators retrieves validators by indices (uses POST for large sets)
func (c *Client) GetValidators(ctx context.Context, stateID string, indices []models.ValidatorIndex) ([]models.Validator, error) {
	// Convert indices to strings for the request
	indicesStr := make([]string, len(indices))
	for i, idx := range indices {
		indicesStr[i] = fmt.Sprintf("%d", idx)
	}

	requestBody := map[string]interface{}{
		"ids": indicesStr,
	}

	var response models.ValidatorsResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID)

	if err := c.doRequest(ctx, http.MethodPost, path, requestBody, &response); err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	return response.Data, nil
}

// GetValidatorsByPubkeys retrieves validators by public keys (uses POST for large sets)
func (c *Client) GetValidatorsByPubkeys(ctx context.Context, stateID string, pubkeys []string) ([]models.Validator, error) {
	requestBody := map[string]interface{}{
		"ids": pubkeys,
	}

	var response models.ValidatorsResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID)

	c.logger.WithField("count", len(pubkeys)).Debug("Fetching validators by pubkeys")
	if err := c.doRequest(ctx, http.MethodPost, path, requestBody, &response); err != nil {
		return nil, fmt.Errorf("failed to get validators by pubkeys: %w", err)
	}

	c.logger.Debugf("Batch loaded %d validators", len(response.Data))
	return response.Data, nil
}

// GetAllValidators retrieves all validators (for loading the full 2M+ validator set)
// DEPRECATED: Use StreamAllValidators for memory-efficient processing
func (c *Client) GetAllValidators(ctx context.Context, stateID string) ([]models.Validator, error) {
	var response models.ValidatorsResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get all validators: %w", err)
	}

	c.logger.Infof("Loaded %d validators from beacon node", len(response.Data))
	return response.Data, nil
}

// ValidatorProcessor is called for each validator during streaming
// Return false to stop processing
type ValidatorProcessor func(v *models.Validator) bool

// StreamAllValidators streams validators one at a time, calling the processor for each
// This is memory-efficient: only one validator is in memory at a time (~500 bytes)
// Instead of loading all 2M+ validators into a slice (~4GB)
func (c *Client) StreamAllValidators(ctx context.Context, stateID string, processor ValidatorProcessor) (int, error) {
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID)
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", contentTypeJSON)

	// Use a client with no timeout for streaming - context handles cancellation
	// Streaming 2M+ validators can take 5-10 minutes over remote APIs
	streamClient := &http.Client{Timeout: 0}
	resp, err := streamClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Use streaming JSON decoder
	decoder := json.NewDecoder(resp.Body)

	// Navigate to "data" array: {"data": [...]}
	// First token should be '{'
	if _, err := decoder.Token(); err != nil {
		return 0, fmt.Errorf("failed to read opening brace: %w", err)
	}

	// Read tokens until we find "data"
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return 0, fmt.Errorf("failed to read token: %w", err)
		}

		// Look for "data" key
		if key, ok := token.(string); ok && key == "data" {
			break
		}

		// Skip unknown values
		if _, err := decoder.Token(); err != nil {
			return 0, fmt.Errorf("failed to skip value: %w", err)
		}
	}

	// Next token should be '[' (start of data array)
	if _, err := decoder.Token(); err != nil {
		return 0, fmt.Errorf("failed to read array start: %w", err)
	}

	// Stream each validator
	count := 0
	for decoder.More() {
		var v models.Validator
		if err := decoder.Decode(&v); err != nil {
			return count, fmt.Errorf("failed to decode validator %d: %w", count, err)
		}

		// Process this validator
		if !processor(&v) {
			break // Processor requested stop
		}
		count++

		// Log progress every 500k validators
		if count%500000 == 0 {
			c.logger.Infof("Streamed %d validators...", count)
		}
	}

	c.logger.Infof("Streamed %d validators from beacon node", count)
	return count, nil
}

// GetProposerDuties retrieves proposer duties for an epoch
func (c *Client) GetProposerDuties(ctx context.Context, epoch models.Epoch) ([]models.ProposerDuty, error) {
	var response models.ProposerDutiesResponse
	path := fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get proposer duties: %w", err)
	}

	return response.Data, nil
}

// GetBlock retrieves a block by block ID
func (c *Client) GetBlock(ctx context.Context, blockID string) (*models.Block, error) {
	var response models.BlockResponse
	path := fmt.Sprintf("/eth/v2/beacon/blocks/%s", blockID)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return &response.Data, nil
}

// GetAttestations retrieves attestations for a slot
// Uses the full block endpoint since some providers (QuickNode) don't support /attestations
// Returns empty slice if block doesn't exist (missed block or not produced yet)
func (c *Client) GetAttestations(ctx context.Context, slot models.Slot) ([]models.Attestation, error) {
	// Get full block and extract attestations from body
	block, err := c.GetBlock(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		// Block might not exist (missed or not produced yet) - return empty, not error
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NOT_FOUND") {
			return []models.Attestation{}, nil
		}
		return nil, fmt.Errorf("failed to get attestations: %w", err)
	}

	return block.Message.Body.Attestations, nil
}

// GetCommittees retrieves committees for a slot
func (c *Client) GetCommittees(ctx context.Context, stateID string, epoch *models.Epoch, slot *models.Slot) ([]models.Committee, error) {
	var response models.CommitteesResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/committees", stateID)

	// Add query parameters if provided
	params := []string{}
	if epoch != nil {
		params = append(params, fmt.Sprintf("epoch=%d", *epoch))
	}
	if slot != nil {
		params = append(params, fmt.Sprintf("slot=%d", *slot))
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get committees: %w", err)
	}

	return response.Data, nil
}

// GetValidatorsLiveness retrieves validator liveness for an epoch
func (c *Client) GetValidatorsLiveness(ctx context.Context, epoch models.Epoch, indices []models.ValidatorIndex) ([]models.ValidatorLiveness, error) {
	// Convert indices to strings for the request
	indicesStr := make([]string, len(indices))
	for i, idx := range indices {
		indicesStr[i] = fmt.Sprintf("%d", idx)
	}

	var response models.ValidatorsLivenessResponse
	path := fmt.Sprintf("/eth/v1/validator/liveness/%d", epoch)

	if err := c.doRequest(ctx, http.MethodPost, path, indicesStr, &response); err != nil {
		return nil, fmt.Errorf("failed to get validators liveness: %w", err)
	}

	return response.Data, nil
}

// GetRewards retrieves attestation rewards for an epoch
func (c *Client) GetRewards(ctx context.Context, epoch models.Epoch, indices []models.ValidatorIndex) (*models.RewardsResponse, error) {
	// Convert indices to strings for the request
	indicesStr := make([]string, len(indices))
	for i, idx := range indices {
		indicesStr[i] = fmt.Sprintf("%d", idx)
	}

	var response models.RewardsResponse
	path := fmt.Sprintf("/eth/v1/beacon/rewards/attestations/%d", epoch)

	if err := c.doRequest(ctx, http.MethodPost, path, indicesStr, &response); err != nil {
		return nil, fmt.Errorf("failed to get rewards: %w", err)
	}

	return &response, nil
}

// GetPendingDeposits retrieves pending deposits
func (c *Client) GetPendingDeposits(ctx context.Context, stateID string) ([]models.PendingDeposit, error) {
	var response models.PendingDepositsResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/pending_deposits", stateID)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		// Not all beacon nodes support this endpoint
		c.logger.Debugf("Failed to get pending deposits (may not be supported): %v", err)
		return []models.PendingDeposit{}, nil
	}

	return response.Data, nil
}

// GetPendingConsolidations retrieves pending consolidations
func (c *Client) GetPendingConsolidations(ctx context.Context, stateID string) ([]models.PendingConsolidation, error) {
	var response models.PendingConsolidationsResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/pending_consolidations", stateID)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		// Not all beacon nodes support this endpoint
		c.logger.Debugf("Failed to get pending consolidations (may not be supported): %v", err)
		return []models.PendingConsolidation{}, nil
	}

	return response.Data, nil
}

// GetPendingWithdrawals retrieves pending withdrawals
func (c *Client) GetPendingWithdrawals(ctx context.Context, stateID string) ([]models.PendingWithdrawal, error) {
	var response models.PendingWithdrawalsResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/withdrawal_queue", stateID)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		// Not all beacon nodes support this endpoint
		c.logger.Debugf("Failed to get pending withdrawals (may not be supported): %v", err)
		return []models.PendingWithdrawal{}, nil
	}

	return response.Data, nil
}

// GetSyncCommitteeDuties retrieves sync committee duties for validators in an epoch
// POST /eth/v1/validator/duties/sync/{epoch}
func (c *Client) GetSyncCommitteeDuties(ctx context.Context, epoch models.Epoch, validatorIndices []models.ValidatorIndex) ([]models.SyncCommitteeDuty, error) {
	var response models.SyncCommitteeDutiesResponse
	path := fmt.Sprintf("/eth/v1/validator/duties/sync/%d", epoch)

	// Convert indices to strings for the request body
	indices := make([]string, len(validatorIndices))
	for i, idx := range validatorIndices {
		indices[i] = fmt.Sprintf("%d", idx)
	}

	if err := c.doRequest(ctx, http.MethodPost, path, indices, &response); err != nil {
		return nil, fmt.Errorf("failed to get sync committee duties: %w", err)
	}

	return response.Data, nil
}

// GetSyncCommitteeRewards retrieves sync committee rewards for a block
// POST /eth/v1/beacon/rewards/sync_committee/{block_id}
func (c *Client) GetSyncCommitteeRewards(ctx context.Context, blockID string, validatorIndices []models.ValidatorIndex) ([]models.SyncCommitteeReward, error) {
	var response models.SyncCommitteeRewardsResponse
	path := fmt.Sprintf("/eth/v1/beacon/rewards/sync_committee/%s", blockID)

	// Convert indices to strings for the request body
	indices := make([]string, len(validatorIndices))
	for i, idx := range validatorIndices {
		indices[i] = fmt.Sprintf("%d", idx)
	}

	if err := c.doRequest(ctx, http.MethodPost, path, indices, &response); err != nil {
		// Not all beacon nodes support this endpoint or block may not exist
		c.logger.Debugf("Failed to get sync committee rewards for block %s: %v", blockID, err)
		return nil, err
	}

	return response.Data, nil
}

// GetSyncCommittee retrieves the sync committee for a state
// GET /eth/v1/beacon/states/{state_id}/sync_committees
func (c *Client) GetSyncCommittee(ctx context.Context, stateID string, epoch *models.Epoch) (*models.SyncCommittee, error) {
	var response models.SyncCommitteeResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/sync_committees", stateID)
	if epoch != nil {
		path = fmt.Sprintf("%s?epoch=%d", path, *epoch)
	}

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get sync committee: %w", err)
	}

	return &response.Data, nil
}
