package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration for the application
type Config struct {
	StreamURL      string        `mapstructure:"STREAM_URL"`
	CheckInterval  time.Duration `mapstructure:"CHECK_INTERVAL"`
	RecordingsPath string        `mapstructure:"RECORDINGS_PATH"`
	TempPath       string        `mapstructure:"TEMP_PATH"`
	BindAddress    string        `mapstructure:"BIND_ADDRESS"`
	PublicUrl      string        `mapstructure:"PUBLIC_URL"`
	LogLevel       string        `mapstructure:"LOG_LEVEL"`
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
		return nil, err
	}

	return &config, nil
}
