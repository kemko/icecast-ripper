package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Format represents the output format for logs
type Format string

const (
	// JSON outputs logs in JSON format for machine readability
	JSON Format = "json"
	// Text outputs logs in a human-readable format
	Text Format = "text"
)

// Setup initializes the structured logger with the specified log level and format
func Setup(logLevel string, format ...Format) {
	level := parseLogLevel(logLevel)

	// Default to JSON format if not specified
	logFormat := JSON
	if len(format) > 0 {
		logFormat = format[0]
	}

	var handler slog.Handler

	switch logFormat {
	case Text:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	default: // JSON
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Debug("Logger initialized", "level", level.String(), "format", string(logFormat))
}

// parseLogLevel converts a string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
