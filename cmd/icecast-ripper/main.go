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
	"github.com/kemko/icecast-ripper/internal/logger"
	"github.com/kemko/icecast-ripper/internal/recorder"
	"github.com/kemko/icecast-ripper/internal/rss"
	"github.com/kemko/icecast-ripper/internal/scheduler"
	"github.com/kemko/icecast-ripper/internal/server"
	"github.com/kemko/icecast-ripper/internal/streamchecker"
)

const version = "0.3.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load and validate configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Setup logger with text format for better human readability
	logger.Setup(cfg.LogLevel, logger.Text)
	slog.Info("Starting icecast-ripper", "version", version)

	// Extract stream name for identification
	streamName := extractStreamName(cfg.StreamURL)
	slog.Info("Using stream identifier", "name", streamName)

	// Create shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create common User-Agent for all HTTP requests
	userAgent := fmt.Sprintf("icecast-ripper/%s", version)

	// Initialize components
	streamChecker := streamchecker.New(
		cfg.StreamURL,
		streamchecker.WithUserAgent(userAgent),
	)

	recorderInstance, err := recorder.New(
		cfg.TempPath,
		cfg.RecordingsPath,
		streamName,
		recorder.WithUserAgent(userAgent),
	)
	if err != nil {
		return fmt.Errorf("recorder initialization failed: %w", err)
	}

	feedTitle := "Icecast Recordings"
	feedDesc := "Recordings from stream: " + cfg.StreamURL
	rssGenerator := rss.New(cfg, feedTitle, feedDesc, streamName)

	schedulerInstance := scheduler.New(cfg.CheckInterval, streamChecker, recorderInstance)
	httpServer := server.New(cfg, rssGenerator)

	// Start services
	schedulerInstance.Start(ctx)

	if err := httpServer.Start(); err != nil {
		stop() // Cancel context before returning
		return fmt.Errorf("HTTP server failed to start: %w", err)
	}

	slog.Info("Application started successfully. Press Ctrl+C to shut down.")

	// Wait for termination signal
	<-ctx.Done()
	slog.Info("Shutting down application...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop components in reverse order of dependency
	schedulerInstance.Stop()
	recorderInstance.StopRecording()

	if err := httpServer.Stop(shutdownCtx); err != nil {
		slog.Warn("HTTP server shutdown error", "error", err)
	}

	slog.Info("Application shut down gracefully")
	return nil
}

// extractStreamName derives a meaningful identifier from the stream URL
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
