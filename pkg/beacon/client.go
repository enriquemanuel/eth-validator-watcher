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
	maxRetries     = 3
	retryDelay     = 2 * time.Second
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
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode >= 400 {
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
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
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

// GetValidatorsByPubkeys retrieves validators by public keys (uses POST)
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

	c.logger.Infof("Loaded %d validators by pubkeys", len(response.Data))
	return response.Data, nil
}

// GetAllValidators retrieves all validators (for loading the full 2M+ validator set)
func (c *Client) GetAllValidators(ctx context.Context, stateID string) ([]models.Validator, error) {
	var response models.ValidatorsResponse
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get all validators: %w", err)
	}

	c.logger.Infof("Loaded %d validators from beacon node", len(response.Data))
	return response.Data, nil
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
func (c *Client) GetAttestations(ctx context.Context, slot models.Slot) ([]models.Attestation, error) {
	var response models.AttestationsResponse
	path := fmt.Sprintf("/eth/v1/beacon/blocks/%d/attestations", slot)

	if err := c.doRequest(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to get attestations: %w", err)
	}

	return response.Data, nil
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
