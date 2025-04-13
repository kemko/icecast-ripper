package recorder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Recorder handles recording streams
type Recorder struct {
	tempPath       string
	recordingsPath string
	client         *http.Client
	mu             sync.Mutex
	isRecording    bool
	cancelFunc     context.CancelFunc
	streamName     string
	userAgent      string
}

// Option represents a functional option for configuring the recorder
type Option func(*Recorder)

// WithUserAgent sets a custom User-Agent string for HTTP requests
func WithUserAgent(userAgent string) Option {
	return func(r *Recorder) {
		r.userAgent = userAgent
	}
}

// New creates a recorder instance
func New(tempPath, recordingsPath string, streamName string, opts ...Option) (*Recorder, error) {
	for _, dir := range []string{tempPath, recordingsPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	r := &Recorder{
		tempPath:       tempPath,
		recordingsPath: recordingsPath,
		streamName:     streamName,
		client:         &http.Client{Timeout: 0}, // No timeout for long-running downloads
	}

	// Apply any provided options
	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// IsRecording returns whether a recording is currently in progress
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRecording
}

// StartRecording begins recording a stream
func (r *Recorder) StartRecording(ctx context.Context, streamURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRecording {
		return errors.New("recording already in progress")
	}

	recordingCtx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel
	r.isRecording = true

	slog.Info("Starting recording", "url", streamURL)
	go r.recordStream(recordingCtx, streamURL)

	return nil
}

// StopRecording stops an in-progress recording
func (r *Recorder) StopRecording() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRecording && r.cancelFunc != nil {
		slog.Info("Stopping current recording")
		r.cancelFunc()
	}
}

func (r *Recorder) recordStream(ctx context.Context, streamURL string) {
	startTime := time.Now()
	var tempFilePath string
	var moveSuccessful bool

	defer func() {
		r.mu.Lock()
		r.isRecording = false
		r.cancelFunc = nil
		r.mu.Unlock()
		slog.Info("Recording process finished")

		if tempFilePath != "" && !moveSuccessful {
			slog.Warn("Temporary file preserved for inspection", "path", tempFilePath)
			return
		}

		if tempFilePath != "" {
			if err := cleanupTempFile(tempFilePath); err != nil {
				slog.Error("Failed to remove temporary file", "path", tempFilePath, "error", err)
			}
		}
	}()

	// Create temp file for recording
	tempFile, err := os.CreateTemp(r.tempPath, "recording-*.tmp")
	if err != nil {
		slog.Error("Failed to create temporary file", "error", err)
		return
	}
	tempFilePath = tempFile.Name()
	slog.Debug("Created temporary file", "path", tempFilePath)

	bytesWritten, err := r.downloadStream(ctx, streamURL, tempFile)

	if closeErr := tempFile.Close(); closeErr != nil {
		slog.Error("Failed to close temporary file", "error", closeErr)
		if err == nil {
			err = closeErr
		}
	}

	// Handle errors and early termination
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Info("Recording stopped via cancellation")
		} else {
			slog.Error("Failed to download stream", "error", err)
			return
		}
	}

	// Skip empty recordings
	if bytesWritten == 0 {
		slog.Warn("No data written, discarding recording")
		return
	}

	// Process successful recording
	finalPath := r.generateFinalPath(startTime)
	moveSuccessful = r.moveToFinalLocation(tempFilePath, finalPath)

	if moveSuccessful {
		duration := time.Since(startTime)
		slog.Info("Recording saved", "path", finalPath, "size", bytesWritten, "duration", duration)
	}
}

func (r *Recorder) generateFinalPath(startTime time.Time) string {
	finalFilename := fmt.Sprintf("%s_%s.mp3", r.streamName, startTime.Format("20060102_150405"))
	finalFilename = sanitizeFilename(finalFilename)
	return filepath.Join(r.recordingsPath, finalFilename)
}

func (r *Recorder) moveToFinalLocation(tempPath, finalPath string) bool {
	// Try rename first (fastest)
	if err := os.Rename(tempPath, finalPath); err == nil {
		return true
	}

	// Fallback to manual copy
	if err := copyFile(tempPath, finalPath); err != nil {
		slog.Error("Failed to move recording to final location", "error", err)
		return false
	}

	slog.Info("Recording copied successfully using fallback method")
	return true
}

func (r *Recorder) downloadStream(ctx context.Context, streamURL string, writer io.Writer) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to connect to stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	slog.Debug("Connected to stream, downloading", "url", streamURL)
	bytesWritten, err := io.Copy(writer, resp.Body)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Info("Stream download cancelled")
			return bytesWritten, ctx.Err()
		}

		// Handle common stream disconnections gracefully
		if isNormalDisconnect(err) {
			slog.Info("Stream disconnected normally", "bytesWritten", bytesWritten)
			return bytesWritten, nil
		}

		return bytesWritten, fmt.Errorf("failed during stream copy: %w", err)
	}

	slog.Info("Stream download finished normally", "bytesWritten", bytesWritten)
	return bytesWritten, nil
}

func isNormalDisconnect(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF) ||
		strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), "broken pipe")
}

func cleanupTempFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		slog.Debug("Cleaning up temporary file", "path", path)
		return os.Remove(path)
	}
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

func sanitizeFilename(filename string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "'",
		"<", "-",
		">", "-",
		"|", "-",
	)
	cleaned := replacer.Replace(filename)
	return strings.Trim(cleaned, "_-. ")
}
