package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"lowcode-wrapper/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s up|down\n", os.Args[0])
		os.Exit(2)
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "up":
		err = migrate.Up(ctx, dsn)
	case "down":
		err = migrate.Down(ctx, dsn)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (use up or down)\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
}
