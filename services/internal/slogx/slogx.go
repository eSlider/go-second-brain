package slogx

import (
	"log/slog"
	"os"
)

// New returns a slog.Logger: text when MATRIX_DEBUG=true, JSON otherwise.
func New(debug bool) *slog.Logger {
	if debug {
		return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
