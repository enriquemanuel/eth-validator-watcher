package price

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewFetcher(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	fetcher := NewFetcher(logger)

	if fetcher == nil {
		t.Fatal("Expected non-nil fetcher")
	}

	if fetcher.client == nil {
		t.Error("Expected non-nil HTTP client")
	}

	// Initial price should be 0
	if fetcher.GetPrice() != 0 {
		t.Errorf("Expected initial price 0, got %f", fetcher.GetPrice())
	}
}

func TestFetcherGetPriceWithAge(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	fetcher := NewFetcher(logger)

	price, age := fetcher.GetPriceWithAge()

	if price != 0 {
		t.Errorf("Expected initial price 0, got %f", price)
	}

	// Age should be large since lastFetch is zero time
	if age < time.Hour {
		t.Errorf("Expected large age for unfetched price, got %v", age)
	}
}

func TestFetcherIsStale(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	fetcher := NewFetcher(logger)

	// Should be stale initially since we haven't fetched
	if !fetcher.IsStale() {
		t.Error("Expected price to be stale initially")
	}
}

func TestFetcherSetRefreshInterval(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	fetcher := NewFetcher(logger)

	newInterval := 10 * time.Minute
	fetcher.SetRefreshInterval(newInterval)

	if fetcher.refreshInterval != newInterval {
		t.Errorf("Expected interval %v, got %v", newInterval, fetcher.refreshInterval)
	}
}

// TestFetcherIntegration tests actual API call (skipped in CI)
func TestFetcherIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	fetcher := NewFetcher(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := fetcher.fetch(ctx)
	if err != nil {
		t.Logf("Warning: fetch failed (may be rate limited): %v", err)
		return
	}

	price := fetcher.GetPrice()
	if price <= 0 {
		t.Errorf("Expected positive price, got %f", price)
	}

	// ETH price should be reasonably within range
	if price < 100 || price > 100000 {
		t.Errorf("Price %f seems unreasonable", price)
	}

	t.Logf("Fetched ETH price: $%.2f", price)
}
