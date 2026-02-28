package streamchecker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Checker struct {
	streamURL string
	client    *http.Client
	userAgent string
	log       *slog.Logger
}

func New(streamURL, userAgent string, log *slog.Logger) *Checker {
	return &Checker{
		streamURL: streamURL,
		userAgent: userAgent,
		log:       log,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Checker) IsLive(ctx context.Context) (bool, error) {
	c.log.Debug("Checking stream status", "url", c.streamURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.streamURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Debug("Connection to stream failed, considering not live", "error", err)
		return false, nil
	}
	defer resp.Body.Close() //nolint:errcheck // response body

	if resp.StatusCode == http.StatusOK {
		c.log.Info("Stream is live", "status", resp.StatusCode)
		return true, nil
	}

	c.log.Debug("Stream is not live", "status", resp.StatusCode)
	return false, nil
}

func (c *Checker) StreamURL() string {
	return c.streamURL
}
