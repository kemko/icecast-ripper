// Package mp3util provides utilities for working with MP3 files
package mp3util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tcolgate/mp3"
)

// Maximum number of frames to sample for bitrate calculation
const maxFramesToSample = 100

// GetDuration returns the estimated duration of an MP3 file by sampling frames
// and using file size and bitrate for calculation to avoid processing very large files
func GetDuration(filePath string) (time.Duration, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat MP3 file: %w", err)
	}
	fileSize := fileInfo.Size()

	// First try to calculate duration by sampling bitrate
	duration, err := getDurationByBitrateSampling(filePath, fileSize)
	if err == nil {
		return duration, nil
	}

	// If bitrate sampling fails, fall back to size-based estimation
	return getDurationByFileSize(fileSize), nil
}

// getDurationByBitrateSampling calculates MP3 duration by sampling a limited number of frames
// to determine the average bitrate, then uses that with the file size to estimate duration
func getDurationByBitrateSampling(filePath string, fileSize int64) (time.Duration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Failed to close file: %v", err)
		}
	}(file)

	decoder := mp3.NewDecoder(file)
	var frame mp3.Frame
	var skipped int
	var totalBitrate int
	var frameCount int

	// Sample a limited number of frames to calculate average bitrate
	for frameCount < maxFramesToSample {
		if err := decoder.Decode(&frame, &skipped); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return 0, fmt.Errorf("failed to decode MP3 frame: %w", err)
		}

		// Get bitrate from the frame header in bits per second
		bitrate := int(frame.Header().BitRate())
		if bitrate <= 0 {
			continue
		}

		totalBitrate += bitrate
		frameCount++
	}

	if frameCount == 0 {
		return 0, errors.New("could not read any MP3 frames with a valid bitrate")
	}

	// Calculate average bitrate in bits per second
	averageBitrate := totalBitrate / frameCount

	if averageBitrate <= 0 {
		return 0, errors.New("invalid bitrate detected")
	}

	// Calculate duration: file size (bits) / bitrate (bits per second)
	// Subtract ~10KB for headers and metadata (conservative estimate)
	metadataEstimate := int64(10 * 1024)
	adjustedSize := fileSize
	if fileSize > metadataEstimate {
		adjustedSize -= metadataEstimate
	}

	durationSeconds := float64(adjustedSize*8) / float64(averageBitrate)
	return time.Duration(durationSeconds * float64(time.Second)), nil
}

// getDurationByFileSize provides a rough estimate of MP3 duration based only on file size
// using the assumption of ~128kbps average bitrate for most MP3 files
func getDurationByFileSize(fileSize int64) time.Duration {
	// Assume average bitrate of 128kbps (common for MP3)
	assumedBitrate := 128 * 1024 // bits per second

	// Calculate duration: file size (bits) / bitrate (bits per second)
	durationSeconds := float64(fileSize*8) / float64(assumedBitrate)
	return time.Duration(durationSeconds * float64(time.Second))
}
