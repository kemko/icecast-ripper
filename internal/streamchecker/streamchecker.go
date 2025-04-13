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
}

// Option represents a functional option for configuring the Checker
type Option func(*Checker)

// WithUserAgent sets a custom User-Agent header
func WithUserAgent(userAgent string) Option {
	return func(c *Checker) {
		c.userAgent = userAgent
	}
}

// WithTimeout sets a custom timeout for HTTP requests
func WithTimeout(timeout time.Duration) Option {
	return func(c *Checker) {
		c.client.Timeout = timeout
	}
}

// New creates a new stream checker with sensible defaults
func New(streamURL string, opts ...Option) *Checker {
	c := &Checker{
		streamURL: streamURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Apply any provided options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// IsLive checks if the stream is currently broadcasting
func (c *Checker) IsLive() (bool, error) {
	return c.IsLiveWithContext(context.Background())
}

// IsLiveWithContext checks if the stream is live using the provided context
func (c *Checker) IsLiveWithContext(ctx context.Context) (bool, error) {
	slog.Debug("Checking stream status", "url", c.streamURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.streamURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

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

// GetStreamURL returns the URL being monitored
func (c *Checker) GetStreamURL() string {
	return c.streamURL
}
