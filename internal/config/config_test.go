package config

import "testing"

func TestLoadConfig(t *testing.T) {
	t.Run("missing STREAM_URL returns error", func(t *testing.T) {
		t.Setenv("STREAM_URL", "")
		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for missing STREAM_URL")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		t.Setenv("STREAM_URL", "http://example.com/stream")
		t.Setenv("CHECK_INTERVAL", "5m")
		t.Setenv("BIND_ADDRESS", ":9090")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.StreamURL != "http://example.com/stream" {
			t.Errorf("StreamURL = %q, want %q", cfg.StreamURL, "http://example.com/stream")
		}
		if cfg.CheckInterval.String() != "5m0s" {
			t.Errorf("CheckInterval = %v, want 5m0s", cfg.CheckInterval)
		}
		if cfg.BindAddress != ":9090" {
			t.Errorf("BindAddress = %q, want %q", cfg.BindAddress, ":9090")
		}
	})

	t.Run("invalid CHECK_INTERVAL returns error", func(t *testing.T) {
		t.Setenv("STREAM_URL", "http://example.com/stream")
		t.Setenv("CHECK_INTERVAL", "not-a-duration")

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for invalid CHECK_INTERVAL")
		}
	})

	t.Run("defaults applied", func(t *testing.T) {
		t.Setenv("STREAM_URL", "http://example.com/stream")
		t.Setenv("CHECK_INTERVAL", "")
		t.Setenv("RECORDINGS_PATH", "")
		t.Setenv("BIND_ADDRESS", "")
		t.Setenv("RETENTION_DAYS", "")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.RecordingsPath != "./recordings" {
			t.Errorf("RecordingsPath = %q, want ./recordings", cfg.RecordingsPath)
		}
		if cfg.BindAddress != ":8080" {
			t.Errorf("BindAddress = %q, want :8080", cfg.BindAddress)
		}
		if cfg.CheckInterval.String() != "1m0s" {
			t.Errorf("CheckInterval = %v, want 1m0s", cfg.CheckInterval)
		}
		if cfg.RetentionDays != 90 {
			t.Errorf("RetentionDays = %d, want 90", cfg.RetentionDays)
		}
	})

	t.Run("retention disabled with zero", func(t *testing.T) {
		t.Setenv("STREAM_URL", "http://example.com/stream")
		t.Setenv("RETENTION_DAYS", "0")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.RetentionDays != 0 {
			t.Errorf("RetentionDays = %d, want 0", cfg.RetentionDays)
		}
	})

	t.Run("invalid RETENTION_DAYS returns error", func(t *testing.T) {
		t.Setenv("STREAM_URL", "http://example.com/stream")
		t.Setenv("RETENTION_DAYS", "abc")

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for invalid RETENTION_DAYS")
		}
	})
}
