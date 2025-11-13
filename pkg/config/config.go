package config

import (
	"fmt"
	"os"
	"time"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"gopkg.in/yaml.v3"
)

// DefaultConfig returns a default configuration
func DefaultConfig() *models.Config {
	return &models.Config{
		Network:       "mainnet",
		BeaconURL:     "http://localhost:5052",
		BeaconTimeout: models.Duration(90 * time.Second),
		MetricsPort:   8000,
		WatchedKeys:   []models.WatchedKey{},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*models.Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate
	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *models.Config) error {
	if cfg.Network == "" {
		return fmt.Errorf("network is required")
	}
	if cfg.BeaconURL == "" {
		return fmt.Errorf("beacon_url is required")
	}
	if cfg.MetricsPort <= 0 || cfg.MetricsPort > 65535 {
		return fmt.Errorf("metrics_port must be between 1 and 65535")
	}

	// Validate watched keys
	for i, key := range cfg.WatchedKeys {
		if key.PublicKey == "" {
			return fmt.Errorf("watched_keys[%d]: public_key is required", i)
		}
		if len(key.PublicKey) != 98 || key.PublicKey[:2] != "0x" {
			return fmt.Errorf("watched_keys[%d]: public_key must be a valid BLS public key (0x...)", i)
		}
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides
func applyEnvOverrides(cfg *models.Config) {
	if network := os.Getenv("ETH_WATCHER_NETWORK"); network != "" {
		cfg.Network = network
	}
	if beaconURL := os.Getenv("ETH_WATCHER_BEACON_URL"); beaconURL != "" {
		cfg.BeaconURL = beaconURL
	}
	if slackToken := os.Getenv("ETH_WATCHER_SLACK_TOKEN"); slackToken != "" {
		cfg.SlackToken = slackToken
	}
	if slackChannel := os.Getenv("ETH_WATCHER_SLACK_CHANNEL"); slackChannel != "" {
		cfg.SlackChannel = slackChannel
	}
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(cfg *models.Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
