package logger

import (
	"log/slog"
	"os"
)

// NewSlogLogger creates a new slog.Logger instance
func NewSlogLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(handler)
}
