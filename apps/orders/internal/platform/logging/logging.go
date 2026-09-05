package logging

import (
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	configured := slog.LevelInfo
	switch level {
	case "debug":
		configured = slog.LevelDebug
	case "warn":
		configured = slog.LevelWarn
	case "error":
		configured = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: configured}))
}
