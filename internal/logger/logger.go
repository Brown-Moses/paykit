package logger

import (
	"log/slog"
	"os"
)

// Init sets up structured JSON logging for the entire app.
// Call once at the top of main().
func Init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
