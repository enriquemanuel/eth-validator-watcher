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
	"time"

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
	// Log process start immediately
	fmt.Fprintf(os.Stderr, "PROCESS STARTED: PID=%d\n", os.Getpid())

	// Recover from any panics and log them before exiting
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC in main: %v\n", r)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "PROCESS EXITING: PID=%d\n", os.Getpid())
	}()

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
		"network":       cfg.Network,
		"beacon_url":    cfg.BeaconURL,
		"metrics_port":  cfg.MetricsPort,
		"watched_count": len(cfg.WatchedKeys),
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

	// Run watcher with panic recovery - keep retrying if it exits unexpectedly
	// Wrap in panic recovery to ensure process never crashes
	defer func() {
		if r := recover(); r != nil {
			logger.WithField("panic", r).Error("PANIC in main - process should not exit!")
			// Don't exit - let Docker restart policy handle it if needed
			// But log the panic so we can debug
		}
	}()

	logger.Info("Entering restart loop - watcher will auto-restart on exit")

	for {
		logger.Info("Starting watcher iteration...")
		runWatcherWithRecovery(w, ctx, logger)

		logger.Info("runWatcherWithRecovery returned - checking exit conditions...")

		// Check if we should exit (context cancelled)
		select {
		case <-ctx.Done():
			logger.Info("Shutdown complete - context was cancelled")
			return
		default:
			// If mainLoop returned without context cancellation, something unexpected happened
			// Log it and restart the watcher
			logger.Warn("Watcher main loop exited unexpectedly - restarting in 5 seconds...")
			time.Sleep(5 * time.Second)

			// Recreate watcher in case it's in a bad state
			logger.Info("Recreating watcher instance...")
			newWatcher, err := watcher.NewValidatorWatcher(cfg, logger)
			if err != nil {
				logger.WithError(err).Error("Failed to recreate validator watcher - will retry in 10 seconds")
				time.Sleep(10 * time.Second)
				continue // Retry instead of exiting
			}
			w = newWatcher
			logger.Info("Watcher recreated successfully - restarting loop")
		}
	}
}

// runWatcherWithRecovery runs the watcher with panic recovery
func runWatcherWithRecovery(w *watcher.ValidatorWatcher, ctx context.Context, logger *logrus.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.WithField("panic", r).Error("PANIC recovered in watcher - this should not happen!")
			// Don't exit - let the outer loop handle restart
		}
	}()

	logger.Info("Starting watcher Run()...")

	// Run watcher - keep running even on errors, only exit on context cancellation
	err := w.Run(ctx)

	if err != nil {
		if err == context.Canceled {
			logger.Info("Watcher stopped due to context cancellation")
			return
		}
		logger.WithError(err).Error("Validator watcher encountered an error - will restart")
		// Return so outer loop can restart
		return
	}

	// If Run() returned nil without context cancellation, log it
	logger.WithFields(logrus.Fields{
		"context_done": ctx.Err(),
		"error":        err,
	}).Warn("Watcher Run() returned nil unexpectedly - this should only happen on context cancellation")
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
