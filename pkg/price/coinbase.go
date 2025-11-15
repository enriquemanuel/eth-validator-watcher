package price

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	coinbaseURL = "https://api.exchange.coinbase.com/products/ETH-USD/trades"
	cacheTTL    = 10 * time.Minute
)

// CoinbaseTrade represents a trade from Coinbase API
type CoinbaseTrade struct {
	TradeID int     `json:"trade_id"`
	Price   string  `json:"price"`
	Size    string  `json:"size"`
	Time    string  `json:"time"`
	Side    string  `json:"side"`
}

// Fetcher fetches and caches ETH price from Coinbase
type Fetcher struct {
	client      *http.Client
	logger      *logrus.Logger
	mu          sync.RWMutex
	cachedPrice float64
	cacheTime   time.Time
}

// NewFetcher creates a new price fetcher
func NewFetcher(logger *logrus.Logger) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// GetCurrentETHPrice fetches the current ETH price in USD from Coinbase
// Returns 0.0 if fetching fails (this feature is optional)
// Caches the result for 10 minutes
func (f *Fetcher) GetCurrentETHPrice() float64 {
	// Check cache first
	f.mu.RLock()
	if time.Since(f.cacheTime) < cacheTTL && f.cachedPrice > 0 {
		price := f.cachedPrice
		f.mu.RUnlock()
		return price
	}
	f.mu.RUnlock()

	// Fetch new price
	price := f.fetchPrice()

	// Update cache
	f.mu.Lock()
	f.cachedPrice = price
	f.cacheTime = time.Now()
	f.mu.Unlock()

	return price
}

// fetchPrice makes the actual HTTP request to Coinbase
func (f *Fetcher) fetchPrice() float64 {
	req, err := http.NewRequest("GET", coinbaseURL, nil)
	if err != nil {
		f.logger.WithError(err).Debug("Failed to create Coinbase request")
		return 0.0
	}

	q := req.URL.Query()
	q.Add("limit", "1")
	req.URL.RawQuery = q.Encode()

	resp, err := f.client.Do(req)
	if err != nil {
		f.logger.WithError(err).Debug("Failed to fetch ETH price from Coinbase")
		return 0.0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		f.logger.WithField("status", resp.StatusCode).Debug("Coinbase API returned non-200 status")
		return 0.0
	}

	var trades []CoinbaseTrade
	if err := json.NewDecoder(resp.Body).Decode(&trades); err != nil {
		f.logger.WithError(err).Debug("Failed to decode Coinbase response")
		return 0.0
	}

	if len(trades) == 0 {
		f.logger.Debug("Coinbase returned empty trades list")
		return 0.0
	}

	// Parse price from string
	var price float64
	if _, err := parseFloat(trades[0].Price, &price); err != nil {
		f.logger.WithError(err).Debug("Failed to parse price from Coinbase")
		return 0.0
	}

	f.logger.WithField("price", price).Debug("Fetched ETH price from Coinbase")
	return price
}

// parseFloat parses a float from a string
func parseFloat(s string, dest *float64) (int, error) {
	var val float64
	n, err := fmt.Sscanf(s, "%f", &val)
	if err != nil {
		return n, err
	}
	*dest = val
	return n, nil
}
