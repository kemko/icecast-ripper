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

// Server handles HTTP requests for the RSS feed and recordings.
type Server struct {
	server         *http.Server
	rssGenerator   *rss.Generator
	recordingsPath string
	serverAddress  string
}

// New creates a new HTTP server instance.
func New(cfg *config.Config, rssGenerator *rss.Generator) *Server {
	mux := http.NewServeMux()

	s := &Server{
		rssGenerator:   rssGenerator,
		recordingsPath: cfg.RecordingsPath,
		serverAddress:  cfg.ServerAddress,
	}

	// Handler for the RSS feed
	mux.HandleFunc("GET /rss", s.handleRSS)

	// Handler for serving static recording files
	// Ensure recordingsPath is absolute or relative to the working directory
	absRecordingsPath, err := filepath.Abs(cfg.RecordingsPath)
	if err != nil {
		slog.Error("Failed to get absolute path for recordings directory", "path", cfg.RecordingsPath, "error", err)
		// Decide how to handle this - maybe panic or return an error from New?
		// For now, log and continue, file serving might fail.
		absRecordingsPath = cfg.RecordingsPath // Fallback to original path
	}
	slog.Info("Serving recordings from", "path", absRecordingsPath)
	fileServer := http.FileServer(http.Dir(absRecordingsPath))
	// The path must end with a trailing slash for FileServer to work correctly with subdirectories
	mux.Handle("GET /recordings/", http.StripPrefix("/recordings/", fileServer))

	s.server = &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// handleRSS generates and serves the RSS feed.
func (s *Server) handleRSS(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Received request for RSS feed")
	// TODO: Consider adding caching headers (ETag, Last-Modified)
	// based on the latest recording timestamp or a hash of the feed content.

	// Generate feed (limit to a reasonable number, e.g., 50 latest items)
	maxItems := 50
	feedBytes, err := s.rssGenerator.GenerateFeed(maxItems)
	if err != nil {
		slog.Error("Failed to generate RSS feed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(feedBytes)
	if err != nil {
		slog.Error("Failed to write RSS feed response", "error", err)
	}
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	slog.Info("Starting HTTP server", "address", s.serverAddress)
	// Run the server in a separate goroutine so it doesn't block.
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server ListenAndServe error", "error", err)
			// Consider signaling failure to the main application thread if needed
		}
	}()
	return nil // Return immediately
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	slog.Info("Stopping HTTP server...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second) // Give 15 seconds to shutdown gracefully
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server graceful shutdown failed", "error", err)
		return err
	}
	slog.Info("HTTP server stopped.")
	return nil
}
