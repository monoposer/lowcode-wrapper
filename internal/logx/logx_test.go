package logx

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"nope", slog.LevelInfo},
	}
	for _, tc := range tests {
		if got := parseLevel(tc.in); got != tc.want {
			t.Fatalf("parseLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
