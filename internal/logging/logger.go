package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func New(levelStr string, logFilePath string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var writer io.Writer = os.Stderr
	if logFilePath != "" {
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			slog.New(slog.NewTextHandler(os.Stderr, nil)).
				Warn("cannot open log file, stderr-only", "path", logFilePath, "error", err)
		} else {
			writer = io.MultiWriter(os.Stderr, f)
		}
	}

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return logger
}
