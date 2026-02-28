package scheduler

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

type mockChecker struct {
	live bool
	url  string
}

func (m *mockChecker) IsLive(_ context.Context) (bool, error) { return m.live, nil }
func (m *mockChecker) StreamURL() string                      { return m.url }

type mockRecorder struct {
	mu            sync.Mutex
	recording     bool
	startedCount  int
	cleanupCalls  int
	lastCleanAge  time.Duration
	cleanupResult int
}

func (m *mockRecorder) IsRecording() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.recording
}

func (m *mockRecorder) StartRecording(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startedCount++
	return nil
}

func (m *mockRecorder) StopRecording() {}

func (m *mockRecorder) CleanOldRecordings(maxAge time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupCalls++
	m.lastCleanAge = maxAge
	return m.cleanupResult, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCleanupIfNeeded_Disabled(t *testing.T) {
	rec := &mockRecorder{}
	s := New(time.Minute, &mockChecker{}, rec, 0, testLogger())

	s.cleanupIfNeeded()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.cleanupCalls != 0 {
		t.Errorf("cleanup should not run when retention is disabled, got %d calls", rec.cleanupCalls)
	}
}

func TestCleanupIfNeeded_RunsOnFirstTick(t *testing.T) {
	rec := &mockRecorder{cleanupResult: 3}
	s := New(time.Minute, &mockChecker{}, rec, 90, testLogger())

	s.cleanupIfNeeded()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.cleanupCalls != 1 {
		t.Errorf("cleanupCalls = %d, want 1", rec.cleanupCalls)
	}
	expectedAge := 90 * 24 * time.Hour
	if rec.lastCleanAge != expectedAge {
		t.Errorf("maxAge = %v, want %v", rec.lastCleanAge, expectedAge)
	}
}

func TestCleanupIfNeeded_ThrottledToOncePerHour(t *testing.T) {
	rec := &mockRecorder{}
	s := New(time.Minute, &mockChecker{}, rec, 30, testLogger())

	s.cleanupIfNeeded()
	s.cleanupIfNeeded()
	s.cleanupIfNeeded()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.cleanupCalls != 1 {
		t.Errorf("cleanupCalls = %d, want 1 (should be throttled)", rec.cleanupCalls)
	}
}

func TestCleanupIfNeeded_RunsAgainAfterHour(t *testing.T) {
	rec := &mockRecorder{}
	s := New(time.Minute, &mockChecker{}, rec, 30, testLogger())

	s.cleanupIfNeeded()

	// Simulate an hour passing
	s.lastCleanup = time.Now().Add(-61 * time.Minute)

	s.cleanupIfNeeded()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.cleanupCalls != 2 {
		t.Errorf("cleanupCalls = %d, want 2", rec.cleanupCalls)
	}
}

func TestCheckAndRecord_StartsRecordingWhenLive(t *testing.T) {
	rec := &mockRecorder{}
	checker := &mockChecker{live: true, url: "http://stream.example.com/live"}
	s := New(time.Minute, checker, rec, 0, testLogger())

	s.checkAndRecord(context.Background())

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.startedCount != 1 {
		t.Errorf("startedCount = %d, want 1", rec.startedCount)
	}
}

func TestCheckAndRecord_SkipsWhenNotLive(t *testing.T) {
	rec := &mockRecorder{}
	checker := &mockChecker{live: false}
	s := New(time.Minute, checker, rec, 0, testLogger())

	s.checkAndRecord(context.Background())

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.startedCount != 0 {
		t.Errorf("startedCount = %d, want 0", rec.startedCount)
	}
}

func TestCheckAndRecord_SkipsWhenAlreadyRecording(t *testing.T) {
	rec := &mockRecorder{recording: true}
	checker := &mockChecker{live: true, url: "http://stream.example.com/live"}
	s := New(time.Minute, checker, rec, 0, testLogger())

	s.checkAndRecord(context.Background())

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.startedCount != 0 {
		t.Errorf("startedCount = %d, want 0 (should skip when already recording)", rec.startedCount)
	}
}

func TestScheduler_StartStop(t *testing.T) {
	rec := &mockRecorder{}
	checker := &mockChecker{live: false}
	s := New(50*time.Millisecond, checker, rec, 0, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	time.Sleep(150 * time.Millisecond)

	cancel()
	s.Stop()
}
