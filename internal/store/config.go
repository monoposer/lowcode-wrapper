package store

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"lowcode-wrapper/internal/auth"

	"lowcode-wrapper/internal/store/file"
	"lowcode-wrapper/internal/store/postgres"
)

type Mode string

const (
	ModePostgres Mode = "postgres"
	ModeFile     Mode = "file"

	defaultDriversFile = "./drivers.yaml"
)

// Config selects where Foreign Server / Table metadata is loaded from.
// - postgres: Meta DB (Admin API / DATABASE_*)
// - file: declarative drivers.yaml
type Config struct {
	Mode Mode
	File file.Config
}

func LoadConfig() (Config, error) {
	cfg := Config{Mode: ModePostgres}

	if mode := strings.TrimSpace(os.Getenv("WRAPPER_STORE_MODE")); mode != "" {
		cfg.Mode = Mode(strings.ToLower(mode))
	}
	path := strings.TrimSpace(os.Getenv("WRAPPER_DRIVERS_FILE"))
	if path == "" {
		path = strings.TrimSpace(os.Getenv("WRAPPER_STORE_FILE")) // legacy alias
	}
	if path == "" {
		path = defaultDriversFile
	}
	cfg.File.Path = path // file path or directory of *.yaml / *.yml

	switch cfg.Mode {
	case ModePostgres, ModeFile:
	default:
		return Config{}, fmt.Errorf("unsupported store mode %q (use postgres or file)", cfg.Mode)
	}
	return cfg, nil
}

func postgresConfigFromEnv() postgres.Config {
	cfg := postgres.Config{}
	if v := envFirst("DATABASE_URL"); v != "" {
		cfg.DSN = v
		return cfg
	}
	if v := envFirst("DATABASE_HOST", "WRAPPER_STORE_PG_HOST"); v != "" {
		cfg.Host = v
	}
	if v := envFirst("DATABASE_PORT", "WRAPPER_STORE_PG_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}
	if v := envFirst("DATABASE_USER", "WRAPPER_STORE_PG_USER"); v != "" {
		cfg.Username = v
	}
	if v := envFirst("DATABASE_PASSWORD", "WRAPPER_STORE_PG_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := envFirst("DATABASE_NAME", "WRAPPER_STORE_PG_DATABASE"); v != "" {
		cfg.Database = v
	}
	if v := envFirst("DATABASE_SSLMODE", "WRAPPER_STORE_PG_SSLMODE"); v != "" {
		cfg.SSLMode = v
	}
	return cfg
}

func envFirst(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func New(vault *auth.Vault, cfg Config) (Store, error) {
	switch cfg.Mode {
	case ModeFile:
		return file.New(vault, cfg.File)
	case ModePostgres:
		return postgres.New(vault, postgresConfigFromEnv())
	default:
		return nil, fmt.Errorf("unsupported store mode %q", cfg.Mode)
	}
}

func NewFromEnv(vault *auth.Vault) (Store, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return New(vault, cfg)
}

func Ping(ctx context.Context, cfg Config) error {
	switch cfg.Mode {
	case ModeFile:
		return file.Ping(cfg.File)
	case ModePostgres:
		return postgres.Ping(ctx, postgresConfigFromEnv())
	default:
		return fmt.Errorf("unsupported store mode %q", cfg.Mode)
	}
}
