package logger

import (
	"log/slog"
	"os"
)

// Setup initializes the global slog logger based on the environment.
// In development, it uses a human-readable text handler with Debug level.
// In production, it uses a high-performance structured JSON handler with Info level.
func Setup(env string) *slog.Logger {
	var handler slog.Handler

	if env == "development" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
