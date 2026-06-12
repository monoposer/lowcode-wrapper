package db

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DSN string
}

func ConfigFromEnv() (Config, error) {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	}
	if dsn == "" {
		return Config{}, fmt.Errorf("DATABASE_URL or DATABASE_DSN is required for db store mode")
	}
	return Config{DSN: dsn}, nil
}
