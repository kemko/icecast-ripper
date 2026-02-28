package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	StreamURL      string
	CheckInterval  time.Duration
	RecordingsPath string
	TempPath       string
	BindAddress    string
	PublicURL      string
	LogLevel       string
	RetentionDays  int // 0 = disabled (keep forever)
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		StreamURL:      os.Getenv("STREAM_URL"),
		RecordingsPath: getEnvOrDefault("RECORDINGS_PATH", "./recordings"),
		TempPath:       getEnvOrDefault("TEMP_PATH", "/tmp"),
		BindAddress:    getEnvOrDefault("BIND_ADDRESS", ":8080"),
		PublicURL:      getEnvOrDefault("PUBLIC_URL", "http://localhost:8080"),
		LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
	}

	interval, err := time.ParseDuration(getEnvOrDefault("CHECK_INTERVAL", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid CHECK_INTERVAL: %w", err)
	}
	cfg.CheckInterval = interval

	retentionDays, err := strconv.Atoi(getEnvOrDefault("RETENTION_DAYS", "90"))
	if err != nil {
		return nil, fmt.Errorf("invalid RETENTION_DAYS: %w", err)
	}
	cfg.RetentionDays = retentionDays

	if cfg.StreamURL == "" {
		return nil, fmt.Errorf("STREAM_URL is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
