package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	blockchain "tlng/blockchain/client"
	"tlng/config"
	"tlng/query/service/core"
	queryhttp "tlng/query/service/http"
	"tlng/storage/store"
)

const queryConfigPath = "./config/query.defaults.yml"

func main() {
	logger := log.New(os.Stdout, "[QUERY] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Query Service...")

	// 1. Load Query Config
	queryCfg, err := config.LoadQueryConfig(queryConfigPath)
	if err != nil {
		logger.Fatalf("FATAL: Failed to load query configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize Database Connection
	logger.Println("Initializing database connection...")
	dbStore, err := store.NewPostgresStore(
		ctx,
		queryCfg.Database.DSN,
		queryCfg.Database.MinConnections,
		queryCfg.Database.MaxConnections,
		logger,
	)
	if err != nil {
		logger.Fatalf("FATAL: Failed to initialize database store: %v", err)
	}
	defer dbStore.Close()

	// 3. Initialize Blockchain Client (conditionally)
	var bcClient blockchain.BlockchainClient
	if queryCfg.Blockchain.Enabled {
		logger.Println("Initializing blockchain client...")
		bcClient, err = blockchain.NewBlockchainClientFromFile(
			queryCfg.Blockchain.ChainMakerConfig,
			logger,
		)
		if err != nil {
			logger.Fatalf("FATAL: Failed to initialize blockchain client: %v", err)
		}
		defer bcClient.Close()
	} else {
		logger.Println("Blockchain client is disabled in configuration; skipping initialization.")
	}

	// 4. Create Query Service
	logger.Println("Initializing query service...")
	queryService := core.NewService(dbStore, bcClient, logger)

	// 5. Setup HTTP Server
	logger.Println("Setting up HTTP server...")
	mux := http.NewServeMux()

	// Register query API routes
	handler := queryhttp.NewHandler(queryService, logger)
	handler.RegisterRoutes(mux)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Parse timeout durations from config
	readTimeout, err := time.ParseDuration(queryCfg.Server.ReadTimeout)
	if err != nil {
		logger.Fatalf("FATAL: Invalid read_timeout: %v", err)
	}
	writeTimeout, err := time.ParseDuration(queryCfg.Server.WriteTimeout)
	if err != nil {
		logger.Fatalf("FATAL: Invalid write_timeout: %v", err)
	}
	idleTimeout, err := time.ParseDuration(queryCfg.Server.IdleTimeout)
	if err != nil {
		logger.Fatalf("FATAL: Invalid idle_timeout: %v", err)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", queryCfg.Server.HTTPPort),
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// 6. Start HTTP Server in goroutine
	go func() {
		logger.Printf("Query Service listening on port %d", queryCfg.Server.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("FATAL: HTTP server error: %v", err)
		}
	}()

	logger.Println("Query Service started successfully. Press Ctrl+C to stop.")

	// 7. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Received shutdown signal, initiating graceful shutdown...")

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("WARNING: HTTP server shutdown error: %v", err)
	}

	// Cancel main context
	cancel()

	logger.Println("Query Service shut down gracefully.")
}
