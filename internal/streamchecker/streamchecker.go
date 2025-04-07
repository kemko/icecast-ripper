package streamchecker

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Checker checks if an Icecast stream is live.
type Checker struct {
	streamURL string
	client    *http.Client
}

// New creates a new Checker instance.
func New(streamURL string) *Checker {
	return &Checker{
		streamURL: streamURL,
		client: &http.Client{
			Timeout: 10 * time.Second, // Sensible timeout for checking a stream
		},
	}
}

// IsLive checks if the stream URL is currently broadcasting.
// It performs a GET request and checks for a 200 OK status.
// Note: Some Icecast servers might behave differently; this might need adjustment.
func (c *Checker) IsLive() (bool, error) {
	slog.Debug("Checking stream status", "url", c.streamURL)
	req, err := http.NewRequest("GET", c.streamURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request for stream check: %w", err)
	}
	// Set a user agent; some servers might require it.
	req.Header.Set("User-Agent", "icecast-ripper/1.0")
	// Icy-Metadata header can sometimes force the server to respond even if the stream is idle,
	// but we want to know if it's *actually* streaming audio data.
	// req.Header.Set("Icy-Metadata", "1")

	resp, err := c.client.Do(req)
	if err != nil {
		// Network errors (DNS lookup failure, connection refused) usually mean not live.
		slog.Warn("Failed to connect to stream URL during check", "url", c.streamURL, "error", err)
		return false, nil // Treat connection errors as 'not live'
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close response body", "error", err)
		}
	}()

	// A 200 OK status generally indicates the stream is live and broadcasting.
	if resp.StatusCode == http.StatusOK {
		slog.Info("Stream is live", "url", c.streamURL, "status", resp.StatusCode)
		return true, nil
	}

	slog.Info("Stream is not live", "url", c.streamURL, "status", resp.StatusCode)
	return false, nil
}

// GetStreamURL returns the URL being checked.
func (c *Checker) GetStreamURL() string {
	return c.streamURL
}
