package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the structured logger with the specified log level
func Setup(logLevel string) {
	level := parseLogLevel(logLevel)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Logger initialized", "level", level.String())
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
