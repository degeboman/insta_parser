package logger

import (
	"log/slog"
	"os"
)

func NewLogger() *slog.Logger {
	options := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
