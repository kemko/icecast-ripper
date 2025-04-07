package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the structured logger.
func Setup(logLevel string) {
	var level slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts) // Or slog.NewTextHandler
	slog.SetDefault(slog.New(handler))

	slog.Info("Logger initialized", "level", level.String())
}
