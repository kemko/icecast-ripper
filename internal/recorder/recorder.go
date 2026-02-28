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

var filenameSanitizer = strings.NewReplacer(
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

type Recorder struct {
	tempPath       string
	recordingsPath string
	client         *http.Client
	mu             sync.Mutex
	isRecording    bool
	cancelFunc     context.CancelFunc
	streamName     string
	userAgent      string
	log            *slog.Logger
}

func New(tempPath, recordingsPath, streamName, userAgent string, log *slog.Logger) (*Recorder, error) {
	for _, dir := range []string{tempPath, recordingsPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &Recorder{
		tempPath:       tempPath,
		recordingsPath: recordingsPath,
		streamName:     streamName,
		userAgent:      userAgent,
		log:            log,
		client:         &http.Client{Timeout: 0},
	}, nil
}

func (r *Recorder) RecordingsPath() string {
	return r.recordingsPath
}

func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRecording
}

func (r *Recorder) StartRecording(ctx context.Context, streamURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRecording {
		return errors.New("recording already in progress")
	}

	recordingCtx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel
	r.isRecording = true

	r.log.Info("Starting recording", "url", streamURL)
	go r.recordStream(recordingCtx, streamURL)

	return nil
}

func (r *Recorder) StopRecording() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRecording && r.cancelFunc != nil {
		r.log.Info("Stopping current recording")
		r.cancelFunc()
	}
}

func (r *Recorder) recordStream(ctx context.Context, streamURL string) {
	startTime := time.Now()
	var tempFilePath string
	var moveErr error

	defer func() {
		r.mu.Lock()
		r.isRecording = false
		r.cancelFunc = nil
		r.mu.Unlock()
		r.log.Info("Recording process finished")

		if tempFilePath == "" {
			return
		}
		if moveErr != nil {
			r.log.Warn("Temporary file preserved for inspection", "path", tempFilePath)
			return
		}
		if err := os.Remove(tempFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			r.log.Error("Failed to remove temporary file", "path", tempFilePath, "error", err)
		}
	}()

	tempFile, err := os.CreateTemp(r.tempPath, "recording-*.tmp")
	if err != nil {
		r.log.Error("Failed to create temporary file", "error", err)
		return
	}
	tempFilePath = tempFile.Name()

	bytesWritten, err := r.downloadStream(ctx, streamURL, tempFile)

	if closeErr := tempFile.Close(); closeErr != nil {
		r.log.Error("Failed to close temporary file", "error", closeErr)
		if err == nil {
			err = closeErr
		}
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			r.log.Info("Recording stopped via cancellation")
		} else {
			r.log.Error("Failed to download stream", "error", err)
			moveErr = err
			return
		}
	}

	if bytesWritten == 0 {
		r.log.Warn("No data written, discarding recording")
		return
	}

	finalPath := r.generateFinalPath(startTime)
	if moveErr = r.moveToFinalLocation(tempFilePath, finalPath); moveErr != nil {
		r.log.Error("Failed to move recording to final location", "error", moveErr)
		return
	}

	r.log.Info("Recording saved", "path", finalPath, "size", bytesWritten, "duration", time.Since(startTime))
}

func (r *Recorder) generateFinalPath(startTime time.Time) string {
	filename := fmt.Sprintf("%s_%s.mp3", r.streamName, startTime.Format("20060102_150405"))
	filename = sanitizeFilename(filename)
	return filepath.Join(r.recordingsPath, filename)
}

// Fallback to copy when rename fails (cross-filesystem)
func (r *Recorder) moveToFinalLocation(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	return copyFile(src, dst)
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
	defer resp.Body.Close() //nolint:errcheck // response body

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	r.log.Debug("Connected to stream, downloading", "url", streamURL)
	bytesWritten, err := io.Copy(writer, resp.Body)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return bytesWritten, ctx.Err()
		}

		if isNormalDisconnect(err) {
			r.log.Info("Stream disconnected normally", "bytesWritten", bytesWritten)
			return bytesWritten, nil
		}

		return bytesWritten, fmt.Errorf("failed during stream copy: %w", err)
	}

	r.log.Info("Stream download finished normally", "bytesWritten", bytesWritten)
	return bytesWritten, nil
}

func isNormalDisconnect(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF) ||
		strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), "broken pipe")
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close() //nolint:errcheck // read-only file

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		_ = destFile.Close()
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return destFile.Close()
}

func (r *Recorder) CleanOldRecordings(maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(r.recordingsPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read recordings directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	var deleted int

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".mp3") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			r.log.Warn("Failed to stat recording", "file", entry.Name(), "error", err)
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(r.recordingsPath, entry.Name())
			if err := os.Remove(path); err != nil {
				r.log.Warn("Failed to delete old recording", "file", entry.Name(), "error", err)
				continue
			}
			r.log.Info("Deleted old recording", "file", entry.Name(), "age", time.Since(info.ModTime()).Round(time.Hour))
			deleted++
		}
	}

	return deleted, nil
}

func sanitizeFilename(filename string) string {
	cleaned := filenameSanitizer.Replace(filename)
	return strings.Trim(cleaned, "_-. ")
}
