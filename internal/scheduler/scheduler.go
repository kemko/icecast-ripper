package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kemko/icecast-ripper/internal/recorder"
	"github.com/kemko/icecast-ripper/internal/streamchecker"
)

// Scheduler manages the periodic checking of the stream and triggers recordings.
type Scheduler struct {
	interval      time.Duration
	checker       *streamchecker.Checker
	recorder      *recorder.Recorder
	stopChan      chan struct{}   // Channel to signal stopping the scheduler
	stopOnce      sync.Once       // Ensures Stop is only called once
	wg            sync.WaitGroup  // Waits for the run loop to finish
	parentContext context.Context // Base context for recordings
}

// New creates a new Scheduler instance.
func New(interval time.Duration, checker *streamchecker.Checker, recorder *recorder.Recorder) *Scheduler {
	return &Scheduler{
		interval: interval,
		checker:  checker,
		recorder: recorder,
		stopChan: make(chan struct{}),
	}
}

// Start begins the scheduling loop.
// It takes a parent context which will be used for recordings.
func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("Starting scheduler", "interval", s.interval.String())
	s.parentContext = ctx
	s.wg.Add(1)
	go s.run()
}

// Stop signals the scheduling loop to terminate gracefully.
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		slog.Info("Stopping scheduler...")
		close(s.stopChan) // Signal the run loop to stop
		// If a recording is in progress, stopping the scheduler doesn't automatically stop it.
		// The recorder needs to be stopped separately if immediate termination is desired.
		// s.recorder.StopRecording() // Uncomment if scheduler stop should also stop recording
		s.wg.Wait() // Wait for the run loop to exit
		slog.Info("Scheduler stopped.")
	})
}

// run is the main loop for the scheduler.
func (s *Scheduler) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		// Initial check immediately on start, then wait for ticker
		s.checkAndRecord()

		select {
		case <-ticker.C:
			// Check periodically based on the interval
			s.checkAndRecord()
		case <-s.stopChan:
			slog.Info("Scheduler run loop exiting.")
			return // Exit loop if stop signal received
		case <-s.parentContext.Done():
			slog.Info("Scheduler parent context cancelled, stopping.")
			// Ensure Stop is called to clean up resources if not already stopped
			s.Stop()
			return
		}
	}
}

// checkAndRecord performs the stream check and starts recording if necessary.
func (s *Scheduler) checkAndRecord() {
	// Do not check if a recording is already in progress
	if s.recorder.IsRecording() {
		slog.Debug("Skipping check, recording in progress.")
		return
	}

	slog.Debug("Scheduler checking stream status...")
	isLive, err := s.checker.IsLive()
	if err != nil {
		// Log the error but continue; maybe a temporary network issue
		slog.Warn("Error checking stream status", "error", err)
		return
	}

	if isLive {
		slog.Info("Stream is live, attempting to start recording.")
		// Use the parent context provided during Start for the recording
		err := s.recorder.StartRecording(s.parentContext, s.checker.GetStreamURL())
		if err != nil {
			// This likely means a recording was *just* started by another check, which is fine.
			slog.Warn("Failed to start recording (might already be in progress)", "error", err)
		} else {
			slog.Info("Recording started successfully by scheduler.")
			// Recording started, the check loop will skip until it finishes
		}
	} else {
		slog.Debug("Stream is not live, waiting for next interval.")
	}
}
