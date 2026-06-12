package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/monoposer/dataspan/internal/logx"
	"github.com/monoposer/dataspan/internal/migrate"
)

func main() {
	logx.Init()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s up|down\n", os.Args[0])
		os.Exit(2)
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()
	cmd := os.Args[1]
	var err error
	switch cmd {
	case "up":
		err = migrate.Up(ctx, dsn)
	case "down":
		err = migrate.Down(ctx, dsn)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (use up or down)\n", cmd)
		os.Exit(2)
	}
	if err != nil {
		slog.Error("migrate failed", "command", cmd, "err", err)
		os.Exit(1)
	}
	logx.Component("migrate").Info("migrate completed", "command", cmd)
}
