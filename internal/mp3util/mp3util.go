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

// GetDuration returns the actual duration of an MP3 file by analyzing its frames
func GetDuration(filePath string) (time.Duration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer file.Close()

	decoder := mp3.NewDecoder(file)
	var frame mp3.Frame
	var skipped int
	var totalSamples int
	sampleRate := 0

	// Process the frames to calculate the total duration
	for {
		if err := decoder.Decode(&frame, &skipped); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return 0, fmt.Errorf("failed to decode MP3 frame: %w", err)
		}

		if sampleRate == 0 {
			sampleRate = frame.Samples()
		}

		totalSamples += frame.Samples()
	}

	if totalSamples == 0 || sampleRate == 0 {
		return 0, errors.New("could not determine MP3 duration")
	}

	durationSeconds := float64(totalSamples) / float64(sampleRate)
	return time.Duration(durationSeconds * float64(time.Second)), nil
}
