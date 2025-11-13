package beacon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/models"
	"github.com/sirupsen/logrus"
)

func TestGetGenesis(t *testing.T) {
	expectedGenesis := models.Genesis{
		GenesisTime:           1606824023,
		GenesisValidatorsRoot: "0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eth/v1/beacon/genesis" {
			t.Errorf("Expected path /eth/v1/beacon/genesis, got %s", r.URL.Path)
		}

		response := struct {
			Data models.Genesis `json:"data"`
		}{Data: expectedGenesis}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	client := NewClient(server.URL, 10*time.Second, logger)

	genesis, err := client.GetGenesis(context.Background())
	if err != nil {
		t.Fatalf("GetGenesis failed: %v", err)
	}

	if genesis.GenesisTime != expectedGenesis.GenesisTime {
		t.Errorf("Expected genesis time %d, got %d", expectedGenesis.GenesisTime, genesis.GenesisTime)
	}

	if genesis.GenesisValidatorsRoot != expectedGenesis.GenesisValidatorsRoot {
		t.Errorf("Expected genesis validators root %s, got %s", expectedGenesis.GenesisValidatorsRoot, genesis.GenesisValidatorsRoot)
	}
}

func TestGetSpec(t *testing.T) {
	expectedSpec := models.Spec{
		SecondsPerSlot:               12,
		SlotsPerEpoch:                32,
		EpochsPerSyncCommitteePeriod: 256,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eth/v1/config/spec" {
			t.Errorf("Expected path /eth/v1/config/spec, got %s", r.URL.Path)
		}

		response := struct {
			Data models.Spec `json:"data"`
		}{Data: expectedSpec}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	client := NewClient(server.URL, 10*time.Second, logger)

	spec, err := client.GetSpec(context.Background())
	if err != nil {
		t.Fatalf("GetSpec failed: %v", err)
	}

	if spec.SecondsPerSlot != expectedSpec.SecondsPerSlot {
		t.Errorf("Expected seconds per slot %d, got %d", expectedSpec.SecondsPerSlot, spec.SecondsPerSlot)
	}

	if spec.SlotsPerEpoch != expectedSpec.SlotsPerEpoch {
		t.Errorf("Expected slots per epoch %d, got %d", expectedSpec.SlotsPerEpoch, spec.SlotsPerEpoch)
	}
}

func TestGetValidators(t *testing.T) {
	expectedValidators := []models.Validator{
		{
			Index:   100,
			Balance: 32000000000,
			Status:  models.StatusActiveOngoing,
		},
		{
			Index:   200,
			Balance: 32100000000,
			Status:  models.StatusActiveOngoing,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		response := models.ValidatorsResponse{
			Data: expectedValidators,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	client := NewClient(server.URL, 10*time.Second, logger)

	validators, err := client.GetValidators(context.Background(), "head", []models.ValidatorIndex{100, 200})
	if err != nil {
		t.Fatalf("GetValidators failed: %v", err)
	}

	if len(validators) != 2 {
		t.Errorf("Expected 2 validators, got %d", len(validators))
	}

	if validators[0].Index != 100 {
		t.Errorf("Expected validator index 100, got %d", validators[0].Index)
	}
}

func TestRetryLogic(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := struct {
			Data models.Genesis `json:"data"`
		}{
			Data: models.Genesis{
				GenesisTime:           1606824023,
				GenesisValidatorsRoot: "0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	client := NewClient(server.URL, 10*time.Second, logger)

	_, err := client.GetGenesis(context.Background())
	if err != nil {
		t.Fatalf("GetGenesis failed after retry: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	client := NewClient(server.URL, 10*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetGenesis(ctx)
	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}
}
