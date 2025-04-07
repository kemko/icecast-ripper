package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// GenerateGUID generates a unique identifier based on file metadata without reading file content.
// It uses recording metadata (stream name, recording start time, filename) to create a deterministic GUID.
func GenerateGUID(streamName string, recordedAt time.Time, filePath string) string {
	filename := filepath.Base(filePath)

	// Create a unique string combining stream name, recording time (rounded to seconds),
	// and filename for uniqueness
	input := fmt.Sprintf("%s:%s:%s",
		streamName,
		recordedAt.UTC().Format(time.RFC3339),
		filename)

	slog.Debug("Generating GUID", "input", input)

	// Hash the combined string to create a GUID
	hasher := sha256.New()
	hasher.Write([]byte(input))
	guid := hex.EncodeToString(hasher.Sum(nil))

	slog.Debug("Generated GUID", "file", filePath, "guid", guid)
	return guid
}

// GenerateFileHash remains for backward compatibility but is now built on GenerateGUID
// This uses file metadata from the path and mtime instead of reading file content
func GenerateFileHash(filePath string) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Extract stream name from directory structure if possible, otherwise use parent dir
	dir := filepath.Dir(filePath)
	streamName := filepath.Base(dir)

	// Use file modification time as a proxy for recording time
	recordedAt := fileInfo.ModTime()

	guid := GenerateGUID(streamName, recordedAt, filePath)

	slog.Debug("Generated hash using metadata", "file", filePath, "hash", guid)
	return guid, nil
}
