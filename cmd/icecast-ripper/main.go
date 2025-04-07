package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kemko/icecast-ripper/internal/config"
	"github.com/kemko/icecast-ripper/internal/database"
	"github.com/kemko/icecast-ripper/internal/logger"
	"github.com/kemko/icecast-ripper/internal/recorder"
	"github.com/kemko/icecast-ripper/internal/rss"
	"github.com/kemko/icecast-ripper/internal/scheduler"
	"github.com/kemko/icecast-ripper/internal/server"
	"github.com/kemko/icecast-ripper/internal/streamchecker"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		_, err2 := fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		if err2 != nil {
			return
		}
		os.Exit(1)
	}

	// Setup logger
	logger.Setup(cfg.LogLevel)

	slog.Info("Starting icecast-ripper", "version", "0.1.0") // Updated version

	// Initialize database
	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	// Create a context that cancels on shutdown signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop() // Ensure context is cancelled even if not explicitly stopped later

	// --- Initialize components ---
	slog.Info("Initializing components...")
	streamChecker := streamchecker.New(cfg.StreamURL)
	recorderInstance, err := recorder.New(cfg.TempPath, cfg.RecordingsPath, db)
	if err != nil {
		slog.Error("Failed to initialize recorder", "error", err)
		os.Exit(1)
	}
	rssGenerator := rss.New(db, cfg, "Icecast Recordings", "Recordings from stream: "+cfg.StreamURL)
	schedulerInstance := scheduler.New(cfg.CheckInterval, streamChecker, recorderInstance)
	httpServer := server.New(cfg, rssGenerator)

	// --- Start components ---
	slog.Info("Starting services...")
	schedulerInstance.Start(ctx) // Pass the main context to the scheduler
	if err := httpServer.Start(); err != nil {
		slog.Error("Failed to start HTTP server", "error", err)
		stop() // Trigger shutdown if server fails to start
		os.Exit(1)
	}

	slog.Info("Application started successfully. Press Ctrl+C to shut down.")

	// Wait for context cancellation (due to signal)
	<-ctx.Done()

	slog.Info("Shutting down application...")

	// --- Graceful shutdown ---
	// Create a shutdown context with a timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop scheduler first (prevents new recordings)
	schedulerInstance.Stop() // This waits for the scheduler loop to exit

	// Stop the recorder (ensures any ongoing recording is finalized or cancelled)
	// Note: Stopping the scheduler doesn't automatically stop an *ongoing* recording.
	// We need to explicitly stop the recorder if we want it to terminate immediately.
	recorderInstance.StopRecording() // Signal any active recording to stop

	// Stop the HTTP server
	if err := httpServer.Stop(shutdownCtx); err != nil {
		slog.Warn("HTTP server shutdown error", "error", err)
	}

	// Database is closed by the deferred function call

	slog.Info("Application shut down gracefully")
}
