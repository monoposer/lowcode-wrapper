package logx

import (
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

// Init configures the process-wide slog logger from LOG_LEVEL and LOG_FORMAT.
// LOG_LEVEL: debug | info | warn | error (default info).
// LOG_FORMAT: json | text (default text).
func Init() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	addSource := level == slog.LevelDebug
	opts := &slog.HandlerOptions{Level: level, AddSource: addSource}

	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
	return defaultLogger
}

// L returns the configured default logger, or slog.Default() if Init was not called.
func L() *slog.Logger {
	if defaultLogger != nil {
		return defaultLogger
	}
	return slog.Default()
}

// Component returns a logger tagged with component=name.
func Component(name string) *slog.Logger {
	return L().With("component", name)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
