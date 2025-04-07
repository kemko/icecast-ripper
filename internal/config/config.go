package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration for the application.
// The values are read by viper from environment variables or a config file.
type Config struct {
	StreamURL      string        `mapstructure:"STREAM_URL"`
	CheckInterval  time.Duration `mapstructure:"CHECK_INTERVAL"`
	RecordingsPath string        `mapstructure:"RECORDINGS_PATH"`
	TempPath       string        `mapstructure:"TEMP_PATH"`
	DatabasePath   string        `mapstructure:"DATABASE_PATH"`
	ServerAddress  string        `mapstructure:"SERVER_ADDRESS"`
	RSSFeedURL     string        `mapstructure:"RSS_FEED_URL"` // Base URL for the RSS feed links
	LogLevel       string        `mapstructure:"LOG_LEVEL"`
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	viper.SetDefault("STREAM_URL", "")
	viper.SetDefault("CHECK_INTERVAL", "1m")
	viper.SetDefault("RECORDINGS_PATH", "./recordings")
	viper.SetDefault("TEMP_PATH", "./temp")
	viper.SetDefault("DATABASE_PATH", "./icecast-ripper.db")
	viper.SetDefault("SERVER_ADDRESS", ":8080")
	viper.SetDefault("RSS_FEED_URL", "http://localhost:8080/rss") // Example default
	viper.SetDefault("LOG_LEVEL", "info")

	var config Config
	// Bind environment variables to struct fields
	if err := viper.BindEnv("STREAM_URL"); err != nil {
		return nil, err
	}
	if err := viper.BindEnv("CHECK_INTERVAL"); err != nil {
		return nil, err
	}
	if err := viper.BindEnv("RECORDINGS_PATH"); err != nil {
		return nil, err
	}
	if err := viper.BindEnv("TEMP_PATH"); err != nil {
		return nil, err
	}
	if err := viper.BindEnv("DATABASE_PATH"); err != nil {
		return nil, err
	}
	if err := viper.BindEnv("SERVER_ADDRESS"); err != nil {
		return nil, err
	}
	if err := viper.BindEnv("RSS_FEED_URL"); err != nil {
		return nil, err
	}
	if err := viper.BindEnv("LOG_LEVEL"); err != nil {
		return nil, err
	}

	// Unmarshal the config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Ensure required fields are set
	if config.StreamURL == "" {
		// Consider returning an error or logging a fatal error if essential config is missing
		// return nil, errors.New("STREAM_URL environment variable is required")
	}

	return &config, nil
}
