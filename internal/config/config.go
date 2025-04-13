package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config stores application configuration loaded from environment variables
type Config struct {
	StreamURL      string        `mapstructure:"STREAM_URL"`      // URL of the Icecast stream to record
	CheckInterval  time.Duration `mapstructure:"CHECK_INTERVAL"`  // How often to check if the stream is live
	RecordingsPath string        `mapstructure:"RECORDINGS_PATH"` // Where to store recordings
	TempPath       string        `mapstructure:"TEMP_PATH"`       // Where to store temporary files during recording
	BindAddress    string        `mapstructure:"BIND_ADDRESS"`    // HTTP server address:port
	PublicURL      string        `mapstructure:"PUBLIC_URL"`      // Public-facing URL for RSS feed links
	LogLevel       string        `mapstructure:"LOG_LEVEL"`       // Logging level (debug, info, warn, error)
}

// LoadConfig reads configuration from environment variables
func LoadConfig() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	// Set default values
	defaults := map[string]interface{}{
		"STREAM_URL":      "",
		"CHECK_INTERVAL":  "1m",
		"RECORDINGS_PATH": "./recordings",
		"TEMP_PATH":       "/tmp",
		"BIND_ADDRESS":    ":8080",
		"PUBLIC_URL":      "http://localhost:8080",
		"LOG_LEVEL":       "info",
	}

	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate required fields
	if config.StreamURL == "" {
		return nil, fmt.Errorf("STREAM_URL is required")
	}

	return &config, nil
}
