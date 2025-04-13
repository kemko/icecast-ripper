package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kemko/icecast-ripper/internal/recorder"
	"github.com/kemko/icecast-ripper/internal/streamchecker"
)

// Scheduler periodically checks if a stream is live and starts recording
type Scheduler struct {
	interval      time.Duration
	checker       *streamchecker.Checker
	recorder      *recorder.Recorder
	stopChan      chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup
	parentContext context.Context
}

// New creates a scheduler instance
func New(interval time.Duration, checker *streamchecker.Checker, recorder *recorder.Recorder) *Scheduler {
	return &Scheduler{
		interval: interval,
		checker:  checker,
		recorder: recorder,
		stopChan: make(chan struct{}),
	}
}

// Start begins the scheduling process
func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("Starting scheduler", "interval", s.interval.String())
	s.parentContext = ctx
	s.wg.Add(1)
	go s.run()
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		slog.Info("Stopping scheduler...")
		close(s.stopChan)
		s.wg.Wait()
		slog.Info("Scheduler stopped")
	})
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Initial check immediately on start
	s.checkAndRecord()

	for {
		select {
		case <-ticker.C:
			s.checkAndRecord()
		case <-s.stopChan:
			return
		case <-s.parentContext.Done():
			slog.Info("Parent context cancelled, stopping scheduler")
			return
		}
	}
}

func (s *Scheduler) checkAndRecord() {
	if s.recorder.IsRecording() {
		slog.Debug("Recording in progress, skipping stream check")
		return
	}

	isLive, err := s.checker.IsLiveWithContext(s.parentContext)
	if err != nil {
		slog.Warn("Error checking stream status", "error", err)
		return
	}

	if isLive {
		slog.Info("Stream is live, starting recording")
		if err := s.recorder.StartRecording(s.parentContext, s.checker.GetStreamURL()); err != nil {
			slog.Warn("Failed to start recording", "error", err)
		}
	} else {
		slog.Debug("Stream is not live")
	}
}
