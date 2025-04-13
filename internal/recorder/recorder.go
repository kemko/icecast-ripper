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
}

// New creates a recorder instance
func New(tempPath, recordingsPath string, streamName string) (*Recorder, error) {
	for _, dir := range []string{tempPath, recordingsPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &Recorder{
		tempPath:       tempPath,
		recordingsPath: recordingsPath,
		streamName:     streamName,
		client:         &http.Client{Timeout: 0}, // No timeout, rely on context cancellation
	}, nil
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

		// Only clean up temp file if it was successfully moved to final location
		if tempFilePath != "" && moveSuccessful {
			if _, err := os.Stat(tempFilePath); err == nil {
				slog.Debug("Cleaning up temporary file", "path", tempFilePath)
				if err := os.Remove(tempFilePath); err != nil {
					slog.Error("Failed to remove temporary file", "error", err)
				}
			}
		} else if tempFilePath != "" && !moveSuccessful {
			slog.Warn("Temporary file preserved for manual inspection", "path", tempFilePath)
		}
	}()

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

	// Handle context cancellation or download errors
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
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	finalFilename := fmt.Sprintf("recording_%s.mp3", startTime.Format("20060102_150405"))
	finalFilename = sanitizeFilename(finalFilename)
	finalPath := filepath.Join(r.recordingsPath, finalFilename)

	// Try rename first (fastest)
	if err := os.Rename(tempFilePath, finalPath); err != nil {
		slog.Warn("Failed to move recording with rename, trying copy fallback", "error", err)

		// Fallback to manual copy
		if err := copyFile(tempFilePath, finalPath); err != nil {
			slog.Error("Failed to move recording to final location", "error", err)
			return
		}

		// Copy successful, mark for cleanup
		moveSuccessful = true
		slog.Info("Recording copied successfully using fallback method", "path", finalPath)
	} else {
		moveSuccessful = true
	}

	slog.Info("Recording saved", "path", finalPath, "size", bytesWritten, "duration", duration)
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

func (r *Recorder) downloadStream(ctx context.Context, streamURL string, writer io.Writer) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "icecast-ripper/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, err
		}
		return 0, fmt.Errorf("failed to connect to stream: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close response body", "error", err)
		}
	}()

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
		if errors.Is(err, io.ErrUnexpectedEOF) ||
		   strings.Contains(err.Error(), "connection reset by peer") ||
		   strings.Contains(err.Error(), "broken pipe") {
			slog.Info("Stream disconnected normally", "bytesWritten", bytesWritten)
			return bytesWritten, nil
		}

		return bytesWritten, fmt.Errorf("failed during stream copy: %w", err)
	}

	slog.Info("Stream download finished normally", "bytesWritten", bytesWritten)
	return bytesWritten, nil
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
