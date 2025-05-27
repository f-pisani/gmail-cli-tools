package utils

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// InitLogger initializes slog with tint handler and sets it as the default logger
func InitLogger() {
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: "15:04:05",
		AddSource:  false,
	})

	logger := slog.New(handler)

	slog.SetDefault(logger)
}
