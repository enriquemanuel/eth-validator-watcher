# Ethereum Validator Watcher Makefile

.PHONY: all build test clean install run fmt vet lint docker-build docker-run

# Binary name
BINARY_NAME=eth-validator-watcher
DOCKER_IMAGE=eth-validator-watcher

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build parameters
BUILD_DIR=build
VERSION?=1.0.0
COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

all: test build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/watcher
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/watcher
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/watcher
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/watcher
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-*"

build-all: build-linux build-darwin
	@echo "All builds complete"

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./pkg/metrics

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Install complete: /usr/local/bin/$(BINARY_NAME)"

run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) -config config.example.yaml

fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

vet:
	@echo "Running go vet..."
	$(GOVET) ./...

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .
	@echo "Docker build complete"

docker-run:
	@echo "Running Docker container..."
	docker run -p 8000:8000 -v $(PWD)/config.yaml:/config.yaml $(DOCKER_IMAGE):latest -config /config.yaml

help:
	@echo "Ethereum Validator Watcher - Makefile targets:"
	@echo "  make build         - Build the binary"
	@echo "  make build-linux   - Build for Linux"
	@echo "  make build-darwin  - Build for macOS"
	@echo "  make build-all     - Build for all platforms"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make bench         - Run benchmarks"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make install       - Install binary to /usr/local/bin"
	@echo "  make run           - Build and run with example config"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run linter"
	@echo "  make deps          - Download dependencies"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run in Docker container"
