package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GenerateGUID creates a unique identifier based on metadata
// Uses stream name, recording time and filename to create a deterministic GUID
func GenerateGUID(streamName string, recordedAt time.Time, filePath string) string {
	filename := filepath.Base(filePath)

	// Create a unique string combining metadata
	input := fmt.Sprintf("%s:%s:%s",
		streamName,
		recordedAt.UTC().Format(time.RFC3339),
		filename)

	// Hash the combined string
	hasher := sha256.New()
	hasher.Write([]byte(input))
	guid := hex.EncodeToString(hasher.Sum(nil))

	return guid
}

// GenerateFileHash creates a hash from file metadata instead of content
// This is faster than reading the entire file content
func GenerateFileHash(filePath string) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	streamName := filepath.Base(filepath.Dir(filePath))
	recordedAt := fileInfo.ModTime()

	return GenerateGUID(streamName, recordedAt, filePath), nil
}
