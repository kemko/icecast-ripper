package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
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

	// Validate essential config
	if cfg.StreamURL == "" {
		slog.Error("Configuration error: STREAM_URL must be set.")
		os.Exit(1)
	}

	// Extract stream name from URL for use in GUID generation
	streamName := extractStreamName(cfg.StreamURL)
	slog.Info("Using stream name for GUID generation", "name", streamName)

	// Initialize file store (replacing SQLite database)
	// If DatabasePath is provided, use it for JSON persistence
	storePath := ""
	if cfg.DatabasePath != "" {
		// Change extension from .db to .json for the file store
		storePath = changeExtension(cfg.DatabasePath, ".json")
	}

	fileStore, err := database.InitDB(storePath)
	if err != nil {
		slog.Error("Failed to initialize file store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := fileStore.Close(); err != nil {
			slog.Error("Failed to close file store", "error", err)
		}
	}()

	// Create a context that cancels on shutdown signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop() // Ensure context is cancelled even if not explicitly stopped later

	// --- Initialize components ---
	slog.Info("Initializing components...")
	streamChecker := streamchecker.New(cfg.StreamURL)
	recorderInstance, err := recorder.New(cfg.TempPath, cfg.RecordingsPath, fileStore, streamName)
	if err != nil {
		slog.Error("Failed to initialize recorder", "error", err)
		os.Exit(1)
	}
	rssGenerator := rss.New(fileStore, cfg, "Icecast Recordings", "Recordings from stream: "+cfg.StreamURL)
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

	// File store is closed by the deferred function call

	slog.Info("Application shut down gracefully")
}

// extractStreamName extracts a meaningful stream name from the URL.
// This is used as part of the GUID generation.
func extractStreamName(streamURL string) string {
	// Try to parse the URL
	parsedURL, err := url.Parse(streamURL)
	if err != nil {
		// If we can't parse it, just use the hostname part or the whole URL
		return streamURL
	}

	// Use the host as the base name
	streamName := parsedURL.Hostname()

	// If there's a path that appears to be a stream identifier, use that too
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		// Remove leading slash and use just the first part of the path if it has multiple segments
		path := parsedURL.Path
		if path[0] == '/' {
			path = path[1:]
		}

		// Take just the first segment of the path
		pathSegments := filepath.SplitList(path)
		if len(pathSegments) > 0 {
			streamName += "_" + pathSegments[0]
		}
	}

	return streamName
}

// changeExtension changes a file extension
func changeExtension(path string, newExt string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + newExt
	}
	return path[:len(path)-len(ext)] + newExt
}
