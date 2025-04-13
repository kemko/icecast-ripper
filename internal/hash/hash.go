package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
