package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kemko/icecast-ripper/internal/recorder"
	"github.com/kemko/icecast-ripper/internal/streamchecker"
)

type Scheduler struct {
	interval      time.Duration
	checker       *streamchecker.Checker
	recorder      *recorder.Recorder
	stopChan      chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup
	parentContext context.Context
}

func New(interval time.Duration, checker *streamchecker.Checker, recorder *recorder.Recorder) *Scheduler {
	return &Scheduler{
		interval: interval,
		checker:  checker,
		recorder: recorder,
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("Starting scheduler", "interval", s.interval.String())
	s.parentContext = ctx
	s.wg.Add(1)
	go s.run()
}

func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		slog.Info("Stopping scheduler...")
		close(s.stopChan)
		s.wg.Wait()
		slog.Info("Scheduler stopped.")
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
			slog.Info("Scheduler run loop exiting.")
			return
		case <-s.parentContext.Done():
			slog.Info("Parent context cancelled, stopping scheduler.")
			return
		}
	}
}

func (s *Scheduler) checkAndRecord() {
	if s.recorder.IsRecording() {
		slog.Debug("Recording in progress, skipping stream check.")
		return
	}

	slog.Debug("Checking stream status")
	isLive, err := s.checker.IsLive()
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
