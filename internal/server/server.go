package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

type FeedGenerator interface {
	GenerateFeed(maxItems int) ([]byte, error)
}

type Server struct {
	server       *http.Server
	feedGen      FeedGenerator
	log          *slog.Logger
}

func New(bindAddress, recordingsPath string, feedGen FeedGenerator, log *slog.Logger) *Server {
	mux := http.NewServeMux()

	absRecordingsPath, err := filepath.Abs(recordingsPath)
	if err != nil {
		log.Error("Failed to resolve recordings path", "error", err)
		absRecordingsPath = recordingsPath
	}

	s := &Server{
		feedGen: feedGen,
		log:     log,
	}

	mux.HandleFunc("GET /rss", s.handleRSS)
	mux.HandleFunc("GET /healthz", s.handleHealth)

	log.Info("Serving recordings", "path", absRecordingsPath)
	fileServer := http.FileServer(http.Dir(absRecordingsPath))
	mux.Handle("GET /recordings/", http.StripPrefix("/recordings/", fileServer))

	s.server = &http.Server{
		Addr:         bindAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) handleRSS(w http.ResponseWriter, _ *http.Request) {
	const maxItems = 50
	feedBytes, err := s.feedGen.GenerateFeed(maxItems)
	if err != nil {
		s.log.Error("Failed to generate RSS feed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	if _, err = w.Write(feedBytes); err != nil {
		s.log.Error("Failed to write RSS response", "error", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.server.Addr, err)
	}
	s.log.Info("HTTP server listening", "address", s.server.Addr)

	go func() {
		if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("Stopping HTTP server")
	return s.server.Shutdown(ctx)
}
