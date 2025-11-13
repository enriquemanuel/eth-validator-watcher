// Ethereum Validator Watcher - Go Implementation
//
// Copyright (c) 2023 Kiln - Original Python/C++ implementation
// Copyright (c) 2025 Enrique Manuel Valenzuela - Go refactor
//
// Licensed under the MIT License. See LICENSE file for details.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/enriquemanuel/eth-validator-watcher/pkg/config"
	"github.com/enriquemanuel/eth-validator-watcher/pkg/watcher"
	"github.com/sirupsen/logrus"
)

var (
	configPath  = flag.String("config", "config.yaml", "Path to configuration file")
	logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	showVersion = flag.Bool("version", false, "Show version information")
)

const (
	version = "1.0.0"
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("eth-validator-watcher version %s (Go)\n", version)
		os.Exit(0)
	}

	// Setup logger
	logger := setupLogger(*logLevel)

	logger.WithFields(logrus.Fields{
		"version": version,
		"config":  *configPath,
	}).Info("Starting Ethereum Validator Watcher")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	logger.WithFields(logrus.Fields{
		"network":         cfg.Network,
		"beacon_url":      cfg.BeaconURL,
		"metrics_port":    cfg.MetricsPort,
		"watched_count":   len(cfg.WatchedKeys),
	}).Info("Configuration loaded")

	// Create watcher
	w, err := watcher.NewValidatorWatcher(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create validator watcher")
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.WithField("signal", sig).Info("Received shutdown signal")
		cancel()
	}()

	// Run watcher
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		logger.WithError(err).Fatal("Validator watcher failed")
	}

	logger.Info("Shutdown complete")
}

func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: false,
	})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logger.Warn("Invalid log level, using info")
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	return logger
}
