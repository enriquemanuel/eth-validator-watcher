package price

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// CoinGecko free API endpoint (no API key required)
	coingeckoURL = "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd"

	// Default refresh interval
	defaultRefreshInterval = 5 * time.Minute

	// Request timeout
	requestTimeout = 10 * time.Second
)

// Fetcher fetches ETH price from CoinGecko
type Fetcher struct {
	client          *http.Client
	logger          *logrus.Logger
	refreshInterval time.Duration

	mu        sync.RWMutex
	price     float64
	lastFetch time.Time
}

// coingeckoResponse represents the CoinGecko API response
type coingeckoResponse struct {
	Ethereum struct {
		USD float64 `json:"usd"`
	} `json:"ethereum"`
}

// NewFetcher creates a new price fetcher
func NewFetcher(logger *logrus.Logger) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		logger:          logger,
		refreshInterval: defaultRefreshInterval,
	}
}

// SetRefreshInterval sets the refresh interval for price fetching
func (f *Fetcher) SetRefreshInterval(interval time.Duration) {
	f.refreshInterval = interval
}

// Start starts the background price fetching loop
func (f *Fetcher) Start(ctx context.Context) {
	// Fetch immediately on start
	if err := f.fetch(ctx); err != nil {
		f.logger.WithError(err).Warn("Initial ETH price fetch failed")
	}

	// Start background refresh loop
	go func() {
		ticker := time.NewTicker(f.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				f.logger.Debug("ETH price fetcher stopping")
				return
			case <-ticker.C:
				if err := f.fetch(ctx); err != nil {
					f.logger.WithError(err).Warn("ETH price fetch failed")
				}
			}
		}
	}()

	f.logger.WithField("interval", f.refreshInterval).Info("ETH price fetcher started")
}

// fetch fetches the current ETH price from CoinGecko
func (f *Fetcher) fetch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coingeckoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result coingeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	f.mu.Lock()
	f.price = result.Ethereum.USD
	f.lastFetch = time.Now()
	f.mu.Unlock()

	f.logger.WithField("price_usd", f.price).Debug("ETH price updated")
	return nil
}

// GetPrice returns the current cached ETH price in USD
func (f *Fetcher) GetPrice() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.price
}

// GetPriceWithAge returns the current cached ETH price and its age
func (f *Fetcher) GetPriceWithAge() (float64, time.Duration) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	age := time.Since(f.lastFetch)
	return f.price, age
}

// IsStale returns true if the price data is older than the refresh interval
func (f *Fetcher) IsStale() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return time.Since(f.lastFetch) > f.refreshInterval*2
}
