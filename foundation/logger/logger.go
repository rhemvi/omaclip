// Package logger provides a structured logger built on top of log/slog.
package logger

import (
	"log/slog"
	"os"
)

func New(minLevel slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     minLevel,
		AddSource: true,
	})

	return slog.New(handler)
}
