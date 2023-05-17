package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/wx-shi/utxo-indexer/internal/config"
	"github.com/wx-shi/utxo-indexer/internal/db"
	"github.com/wx-shi/utxo-indexer/internal/indexer"
	"github.com/wx-shi/utxo-indexer/internal/server"
	"github.com/wx-shi/utxo-indexer/pkg"
	"go.uber.org/zap"
)

func main() {

	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, _ := pkg.NewLogger(cfg.LogLevel)
	defer logger.Sync()

	// Initialize BadgerDB
	badgerDB, err := db.NewBadgerDB(cfg.BadgerDB, logger)
	if err != nil {
		logger.Fatal("Error initializing BadgerDB", zap.Error(err))
	}
	defer badgerDB.Close()

	// Initialize Bitcoin JSON-RPC client
	btcClient, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         cfg.RPC.URL,
		User:         cfg.RPC.User,
		Pass:         cfg.RPC.Password,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}, nil)
	if err != nil {
		logger.Fatal("Error initializing Bitcoin RPC client", zap.Error(err))
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start UTXO indexer
	indexer := indexer.NewIndexer(ctx, cfg.Indexer, logger, btcClient, badgerDB)
	indexer.Sync()

	// Start HTTP server
	httpServer := server.NewServer(cfg.Server, logger, badgerDB)
	httpServer.Run()

	// Wait for signal
	<-sigCh
	logger.Info("Shutting down...")

	// Shutdown context
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Error shutting down HTTP server", zap.Error(err))
	}
}
