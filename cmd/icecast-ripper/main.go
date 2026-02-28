package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kemko/icecast-ripper/internal/config"
	"github.com/kemko/icecast-ripper/internal/recorder"
	"github.com/kemko/icecast-ripper/internal/rss"
	"github.com/kemko/icecast-ripper/internal/scheduler"
	"github.com/kemko/icecast-ripper/internal/server"
	"github.com/kemko/icecast-ripper/internal/streamchecker"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	setupLogger(cfg.LogLevel)
	slog.Info("Starting icecast-ripper", "version", version)

	streamName := extractStreamName(cfg.StreamURL)
	slog.Info("Using stream identifier", "name", streamName)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	userAgent := fmt.Sprintf("icecast-ripper/%s", version)

	checkerLog := slog.Default().With("component", "checker")
	recorderLog := slog.Default().With("component", "recorder")
	schedulerLog := slog.Default().With("component", "scheduler")
	rssLog := slog.Default().With("component", "rss")
	serverLog := slog.Default().With("component", "server")

	streamChecker := streamchecker.New(cfg.StreamURL, userAgent, checkerLog)

	recorderInstance, err := recorder.New(cfg.TempPath, cfg.RecordingsPath, streamName, userAgent, recorderLog)
	if err != nil {
		return fmt.Errorf("recorder initialization failed: %w", err)
	}

	feedTitle := "Icecast Recordings"
	feedDesc := "Recordings from stream: " + cfg.StreamURL
	rssGenerator := rss.New(cfg.PublicURL, cfg.RecordingsPath, feedTitle, feedDesc, streamName, rssLog)

	schedulerInstance := scheduler.New(cfg.CheckInterval, streamChecker, recorderInstance, cfg.RetentionDays, schedulerLog)
	httpServer := server.New(cfg.BindAddress, cfg.RecordingsPath, rssGenerator, serverLog)

	schedulerInstance.Start(ctx)

	if err := httpServer.Start(); err != nil {
		stop()
		return fmt.Errorf("HTTP server failed to start: %w", err)
	}

	slog.Info("Application started successfully. Press Ctrl+C to shut down.")

	<-ctx.Done()
	slog.Info("Shutting down application...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	schedulerInstance.Stop()
	recorderInstance.StopRecording()

	if err := httpServer.Stop(shutdownCtx); err != nil {
		slog.Warn("HTTP server shutdown error", "error", err)
	}

	slog.Info("Application shut down gracefully")
	return nil
}

func setupLogger(level string) {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}

func extractStreamName(streamURL string) string {
	u, err := url.Parse(streamURL)
	if err != nil {
		return "unknown"
	}
	name := u.Hostname()
	if p := strings.TrimPrefix(u.Path, "/"); p != "" {
		name += "_" + strings.ReplaceAll(p, "/", "_")
	}
	return name
}
