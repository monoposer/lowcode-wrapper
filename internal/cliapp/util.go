package cliapp

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func writeOutput(path string, data []byte) error {
	if path == "" || path == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func exitError(component, msg string, err error) int {
	slog.Error(msg, "component", component, "err", err)
	return 1
}

func exitUsage(msg string) int {
	fmt.Fprintln(os.Stderr, msg)
	return 2
}

func programName() string {
	if base := strings.TrimSpace(os.Args[0]); base != "" {
		return base
	}
	return "dataspan"
}
