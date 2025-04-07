package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	// hashChunkSize defines how many bytes of the file to read for hashing.
	// 1MB should be sufficient for uniqueness in most cases while being fast.
	hashChunkSize = 1 * 1024 * 1024 // 1 MB
)

// GenerateFileHash generates a SHA256 hash based on the filename and the first `hashChunkSize` bytes of the file content.
// This avoids reading the entire file for large recordings.
func GenerateFileHash(filePath string) (string, error) {
	filename := filepath.Base(filePath)
	slog.Debug("Generating hash", "file", filePath, "chunkSize", hashChunkSize)

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s for hashing: %w", filePath, err)
	}
	defer file.Close()

	hasher := sha256.New()

	// Include filename in the hash
	if _, err := hasher.Write([]byte(filename)); err != nil {
		return "", fmt.Errorf("failed to write filename to hasher: %w", err)
	}

	// Read only the first part of the file
	chunk := make([]byte, hashChunkSize)
	bytesRead, err := file.Read(chunk)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file chunk %s: %w", filePath, err)
	}

	// Hash the chunk that was read
	if _, err := hasher.Write(chunk[:bytesRead]); err != nil {
		return "", fmt.Errorf("failed to write file chunk to hasher: %w", err)
	}

	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	slog.Debug("Generated hash", "file", filePath, "hash", hashString)
	return hashString, nil
}
