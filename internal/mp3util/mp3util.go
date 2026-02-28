package mp3util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tcolgate/mp3"
)

const maxFramesToSample = 100

func GetDuration(filePath string) (time.Duration, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat MP3 file: %w", err)
	}
	fileSize := fileInfo.Size()

	duration, err := getDurationByBitrateSampling(filePath, fileSize)
	if err == nil {
		return duration, nil
	}

	return getDurationByFileSize(fileSize), nil
}

func getDurationByBitrateSampling(filePath string, fileSize int64) (time.Duration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer file.Close() //nolint:errcheck // read-only file

	decoder := mp3.NewDecoder(file)
	var frame mp3.Frame
	var skipped int
	var totalBitrate int
	var frameCount int

	for frameCount < maxFramesToSample {
		if err := decoder.Decode(&frame, &skipped); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return 0, fmt.Errorf("failed to decode MP3 frame: %w", err)
		}

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

	averageBitrate := totalBitrate / frameCount
	if averageBitrate <= 0 {
		return 0, errors.New("invalid bitrate detected")
	}

	// Subtract ~10KB for ID3 tags and metadata
	metadataEstimate := int64(10 * 1024)
	adjustedSize := fileSize
	if fileSize > metadataEstimate {
		adjustedSize -= metadataEstimate
	}

	durationSeconds := float64(adjustedSize*8) / float64(averageBitrate)
	return time.Duration(durationSeconds * float64(time.Second)), nil
}

func getDurationByFileSize(fileSize int64) time.Duration {
	// 128kbps is the most common MP3 bitrate
	const assumedBitrate = 128 * 1024
	durationSeconds := float64(fileSize*8) / float64(assumedBitrate)
	return time.Duration(durationSeconds * float64(time.Second))
}
