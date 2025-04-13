package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/kemko/icecast-ripper/internal/config"
	"github.com/kemko/icecast-ripper/internal/rss"
)

// Server handles HTTP requests for the RSS feed and recordings
type Server struct {
	server         *http.Server
	rssGenerator   *rss.Generator
	recordingsPath string
}

// New creates a new HTTP server instance
func New(cfg *config.Config, rssGenerator *rss.Generator) *Server {
	mux := http.NewServeMux()

	// Get absolute path for recordings directory
	absRecordingsPath, err := filepath.Abs(cfg.RecordingsPath)
	if err != nil {
		slog.Error("Failed to get absolute path for recordings directory", "error", err)
		absRecordingsPath = cfg.RecordingsPath // Fallback to original path
	}

	s := &Server{
		rssGenerator:   rssGenerator,
		recordingsPath: absRecordingsPath,
	}

	// Set up routes
	mux.HandleFunc("GET /rss", s.handleRSS)

	// Serve static recordings
	slog.Info("Serving recordings from", "path", absRecordingsPath)
	fileServer := http.FileServer(http.Dir(absRecordingsPath))
	mux.Handle("GET /recordings/", http.StripPrefix("/recordings/", fileServer))

	// Configure server with sensible timeouts
	s.server = &http.Server{
		Addr:         cfg.BindAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// handleRSS generates and serves the RSS feed
func (s *Server) handleRSS(w http.ResponseWriter, _ *http.Request) {
	slog.Debug("Serving RSS feed")

	const maxItems = 50
	feedBytes, err := s.rssGenerator.GenerateFeed(maxItems)
	if err != nil {
		slog.Error("Failed to generate RSS feed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(feedBytes); err != nil {
		slog.Error("Failed to write RSS response", "error", err)
	}
}

// Start begins listening for HTTP requests
func (s *Server) Start() error {
	slog.Info("Starting HTTP server", "address", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	slog.Info("Stopping HTTP server")

	return s.server.Shutdown(ctx)
}
