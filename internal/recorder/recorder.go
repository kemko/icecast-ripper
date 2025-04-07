package recorder

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kemko/icecast-ripper/internal/database"
	"github.com/kemko/icecast-ripper/internal/hash"
)

// Recorder handles downloading the stream and saving recordings.
type Recorder struct {
	tempPath       string
	recordingsPath string
	db             *database.DB
	client         *http.Client
	mu             sync.Mutex // Protects access to isRecording
	isRecording    bool
	cancelFunc     context.CancelFunc // To stop the current recording
}

// New creates a new Recorder instance.
func New(tempPath, recordingsPath string, db *database.DB) (*Recorder, error) {
	// Ensure temp and recordings directories exist
	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory %s: %w", tempPath, err)
	}
	if err := os.MkdirAll(recordingsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recordings directory %s: %w", recordingsPath, err)
	}

	return &Recorder{
		tempPath:       tempPath,
		recordingsPath: recordingsPath,
		db:             db,
		client: &http.Client{
			// Use a longer timeout for downloading the stream
			Timeout: 0, // No timeout for the download itself, rely on context cancellation
		},
	}, nil
}

// IsRecording returns true if a recording is currently in progress.
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRecording
}

// StartRecording starts downloading the stream to a temporary file.
// It runs asynchronously and returns immediately.
func (r *Recorder) StartRecording(ctx context.Context, streamURL string) error {
	r.mu.Lock()
	if r.isRecording {
		r.mu.Unlock()
		return fmt.Errorf("recording already in progress")
	}
	// Create a new context that can be cancelled to stop this specific recording
	recordingCtx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel
	r.isRecording = true
	r.mu.Unlock()

	slog.Info("Starting recording", "url", streamURL)

	go r.recordStream(recordingCtx, streamURL) // Run in a goroutine

	return nil
}

// StopRecording stops the current recording if one is in progress.
func (r *Recorder) StopRecording() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isRecording && r.cancelFunc != nil {
		slog.Info("Stopping current recording")
		r.cancelFunc() // Signal the recording goroutine to stop
		// The lock ensures that isRecording is set to false *after* the goroutine finishes
	}
}

// recordStream performs the actual download and processing.
func (r *Recorder) recordStream(ctx context.Context, streamURL string) {
	startTime := time.Now()
	var tempFilePath string

	// Defer setting isRecording to false and cleaning up cancelFunc
	defer func() {
		r.mu.Lock()
		r.isRecording = false
		r.cancelFunc = nil
		r.mu.Unlock()
		slog.Info("Recording process finished")
		// Clean up temp file if it still exists (e.g., due to early error)
		if tempFilePath != "" {
			if _, err := os.Stat(tempFilePath); err == nil {
				slog.Warn("Removing leftover temporary file", "path", tempFilePath)
				if err := os.Remove(tempFilePath); err != nil {
					slog.Error("Failed to remove temporary file", "path", tempFilePath, "error", err)
				}
			}
		}
	}()

	// Create temporary file
	tempFile, err := os.CreateTemp(r.tempPath, "recording-*.tmp")
	if err != nil {
		slog.Error("Failed to create temporary file", "error", err)
		return
	}
	tempFilePath = tempFile.Name()
	slog.Debug("Created temporary file", "path", tempFilePath)

	// Download stream
	bytesWritten, err := r.downloadStream(ctx, streamURL, tempFile)
	// Close the file explicitly *before* hashing and moving
	if closeErr := tempFile.Close(); closeErr != nil {
		slog.Error("Failed to close temporary file", "path", tempFilePath, "error", closeErr)
		// If download also failed, prioritize that error
		if err == nil {
			err = closeErr // Report closing error if download was ok
		}
	}

	if err != nil {
		// Check if the error was due to context cancellation (graceful stop)
		if ctx.Err() == context.Canceled {
			slog.Info("Recording stopped via cancellation.")
			// Proceed to process the partially downloaded file
		} else {
			slog.Error("Failed to download stream", "error", err)
			// No need to keep the temp file if download failed unexpectedly
			if err := os.Remove(tempFilePath); err != nil {
				slog.Error("Failed to remove temporary file after download error", "path", tempFilePath, "error", err)
			}
			tempFilePath = "" // Prevent deferred cleanup from trying again
			return
		}
	}

	// If no bytes were written (e.g., stream closed immediately or cancelled before data), discard.
	if bytesWritten == 0 {
		slog.Warn("No data written to temporary file, discarding.", "path", tempFilePath)
		if err := os.Remove(tempFilePath); err != nil {
			slog.Error("Failed to remove temporary file after no data written", "path", tempFilePath, "error", err)
		}
		tempFilePath = "" // Prevent deferred cleanup
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Generate hash
	fileHash, err := hash.GenerateFileHash(tempFilePath)
	if err != nil {
		slog.Error("Failed to generate file hash", "path", tempFilePath, "error", err)
		if err := os.Remove(tempFilePath); err != nil {
			slog.Error("Failed to remove temporary file after hash error", "path", tempFilePath, "error", err)
		}
		tempFilePath = "" // Prevent deferred cleanup
		return
	}

	// Generate final filename (e.g., based on timestamp)
	finalFilename := fmt.Sprintf("recording_%s.mp3", startTime.Format("20060102_150405")) // Assuming mp3, adjust if needed
	finalFilename = sanitizeFilename(finalFilename)                                       // Ensure filename is valid
	finalPath := filepath.Join(r.recordingsPath, finalFilename)

	// Move temporary file to final location
	if err := os.Rename(tempFilePath, finalPath); err != nil {
		slog.Error("Failed to move temporary file to final location", "temp", tempFilePath, "final", finalPath, "error", err)
		if err := os.Remove(tempFilePath); err != nil {
			slog.Error("Failed to remove temporary file after move error", "path", tempFilePath, "error", err)
		}
		tempFilePath = "" // Prevent deferred cleanup
		return
	}
	tempFilePath = "" // File moved, clear path to prevent deferred cleanup
	slog.Info("Recording saved", "path", finalPath, "size", bytesWritten, "duration", duration)

	// Add record to database
	_, err = r.db.AddRecordedFile(finalFilename, fileHash, bytesWritten, duration, startTime)
	if err != nil {
		slog.Error("Failed to add recording to database", "filename", finalFilename, "hash", fileHash, "error", err)
		// Consider how to handle this - maybe retry later? For now, just log.
	}
}

// downloadStream handles the HTTP GET request and writes the body to the writer.
func (r *Recorder) downloadStream(ctx context.Context, streamURL string, writer io.Writer) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "icecast-ripper/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		// Check if context was cancelled
		if ctx.Err() == context.Canceled {
			return 0, ctx.Err()
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

	slog.Debug("Connected to stream, starting download", "url", streamURL)
	// Copy the response body to the writer (temp file)
	// io.Copy will block until EOF or an error (including context cancellation via the request)
	bytesWritten, err := io.Copy(writer, resp.Body)
	if err != nil {
		// Check if the error is due to context cancellation
		if ctx.Err() == context.Canceled {
			slog.Info("Stream download cancelled.")
			return bytesWritten, ctx.Err() // Return context error
		}
		// Check for common stream disconnection errors (might need refinement)
		if err == io.ErrUnexpectedEOF || strings.Contains(err.Error(), "connection reset by peer") || strings.Contains(err.Error(), "broken pipe") {
			slog.Info("Stream disconnected gracefully (EOF or reset)", "bytes_written", bytesWritten)
			return bytesWritten, nil // Treat as normal end of stream
		}
		return bytesWritten, fmt.Errorf("failed during stream copy: %w", err)
	}

	slog.Info("Stream download finished (EOF)", "bytes_written", bytesWritten)
	return bytesWritten, nil
}

// sanitizeFilename removes potentially problematic characters from filenames.
func sanitizeFilename(filename string) string {
	// Replace common problematic characters
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
	// Trim leading/trailing underscores/hyphens/dots
	cleaned = strings.Trim(cleaned, "_-. ")
	// Limit length if necessary (though less common on modern filesystems)
	// const maxLen = 200
	// if len(cleaned) > maxLen {
	// 	cleaned = cleaned[:maxLen]
	// }
	return cleaned
}
