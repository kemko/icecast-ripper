package streamchecker

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Checker struct {
	streamURL string
	client    *http.Client
}

func New(streamURL string) *Checker {
	return &Checker{
		streamURL: streamURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Checker) IsLive() (bool, error) {
	slog.Debug("Checking stream status", "url", c.streamURL)

	req, err := http.NewRequest(http.MethodGet, c.streamURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "icecast-ripper/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		slog.Debug("Connection to stream failed, considering not live", "error", err)
		return false, nil // Connection failures mean the stream is not available
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode == http.StatusOK {
		slog.Info("Stream is live", "status", resp.StatusCode)
		return true, nil
	}

	slog.Debug("Stream is not live", "status", resp.StatusCode)
	return false, nil
}

func (c *Checker) GetStreamURL() string {
	return c.streamURL
}
