package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type StreamChecker interface {
	IsLive(ctx context.Context) (bool, error)
	StreamURL() string
}

type Recorder interface {
	IsRecording() bool
	StartRecording(ctx context.Context, streamURL string) error
	StopRecording()
	CleanOldRecordings(maxAge time.Duration) (int, error)
}

type Scheduler struct {
	interval        time.Duration
	checker         StreamChecker
	recorder        Recorder
	log             *slog.Logger
	wg              sync.WaitGroup
	cancel          context.CancelFunc
	retentionMaxAge time.Duration
	lastCleanup     time.Time
}

func New(interval time.Duration, checker StreamChecker, recorder Recorder, retentionDays int, log *slog.Logger) *Scheduler {
	var maxAge time.Duration
	if retentionDays > 0 {
		maxAge = time.Duration(retentionDays) * 24 * time.Hour
	}

	return &Scheduler{
		interval:        interval,
		checker:         checker,
		recorder:        recorder,
		retentionMaxAge: maxAge,
		log:             log,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.log.Info("Starting scheduler", "interval", s.interval.String())
	ctx, s.cancel = context.WithCancel(ctx)
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
	s.log.Info("Scheduler stopped")
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.cleanupIfNeeded()
	s.checkAndRecord(ctx)

	for {
		select {
		case <-ticker.C:
			s.cleanupIfNeeded()
			s.checkAndRecord(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) cleanupIfNeeded() {
	if s.retentionMaxAge <= 0 || time.Since(s.lastCleanup) < 1*time.Hour {
		return
	}

	s.lastCleanup = time.Now()
	deleted, err := s.recorder.CleanOldRecordings(s.retentionMaxAge)
	if err != nil {
		s.log.Warn("Failed to clean old recordings", "error", err)
		return
	}
	if deleted > 0 {
		s.log.Info("Cleaned old recordings", "deleted", deleted)
	}
}

func (s *Scheduler) checkAndRecord(ctx context.Context) {
	if s.recorder.IsRecording() {
		s.log.Debug("Recording in progress, skipping stream check")
		return
	}

	isLive, err := s.checker.IsLive(ctx)
	if err != nil {
		s.log.Warn("Error checking stream status", "error", err)
		return
	}

	if isLive {
		s.log.Info("Stream is live, starting recording")
		if err := s.recorder.StartRecording(ctx, s.checker.StreamURL()); err != nil {
			s.log.Warn("Failed to start recording", "error", err)
		}
	} else {
		s.log.Debug("Stream is not live")
	}
}
