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

const version = "0.2.0"

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger.Setup(cfg.LogLevel)
	slog.Info("Starting icecast-ripper", "version", version)

	// Validate essential configuration
	if cfg.StreamURL == "" {
		slog.Error("Configuration error: STREAM_URL must be set")
		os.Exit(1)
	}

	// Extract stream name for GUID generation
	streamName := extractStreamName(cfg.StreamURL)
	slog.Info("Using stream name for identification", "name", streamName)

	// Initialize file store
	storePath := ""
	if cfg.DatabasePath != "" {
		storePath = changeExtension(cfg.DatabasePath, ".json")
	}

	fileStore, err := database.InitDB(storePath)
	if err != nil {
		slog.Error("Failed to initialize file store", "error", err)
		os.Exit(1)
	}
	// Properly handle Close() error
	defer func() {
		if err := fileStore.Close(); err != nil {
			slog.Error("Error closing file store", "error", err)
		}
	}()

	// Create main context that cancels on shutdown signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize components
	streamChecker := streamchecker.New(cfg.StreamURL)
	recorderInstance, err := recorder.New(cfg.TempPath, cfg.RecordingsPath, fileStore, streamName)
	if err != nil {
		slog.Error("Failed to initialize recorder", "error", err)
		os.Exit(1)
	}

	rssGenerator := rss.New(fileStore, cfg, "Icecast Recordings", "Recordings from stream: "+cfg.StreamURL)
	schedulerInstance := scheduler.New(cfg.CheckInterval, streamChecker, recorderInstance)
	httpServer := server.New(cfg, rssGenerator)

	// Start services
	slog.Info("Starting services...")

	// Start the scheduler which will check for streams and record them
	schedulerInstance.Start(ctx)

	// Start the HTTP server for RSS feed
	if err := httpServer.Start(); err != nil {
		slog.Error("Failed to start HTTP server", "error", err)
		stop()
		os.Exit(1)
	}

	slog.Info("Application started successfully. Press Ctrl+C to shut down.")

	// Wait for termination signal
	<-ctx.Done()
	slog.Info("Shutting down application...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// First stop the scheduler to prevent new recordings
	schedulerInstance.Stop()

	// Then stop any ongoing recording
	recorderInstance.StopRecording()

	// Finally, stop the HTTP server
	if err := httpServer.Stop(shutdownCtx); err != nil {
		slog.Warn("HTTP server shutdown error", "error", err)
	}

	slog.Info("Application shut down gracefully")
}

// extractStreamName extracts a meaningful identifier from the URL
func extractStreamName(streamURL string) string {
	parsedURL, err := url.Parse(streamURL)
	if err != nil {
		return streamURL
	}

	streamName := parsedURL.Hostname()

	if parsedURL.Path != "" && parsedURL.Path != "/" {
		path := parsedURL.Path
		if path[0] == '/' {
			path = path[1:]
		}

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
