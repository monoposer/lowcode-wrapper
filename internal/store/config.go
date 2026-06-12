package store

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/monoposer/dataspan/internal/auth"

	"github.com/monoposer/dataspan/internal/store/db"
	"github.com/monoposer/dataspan/internal/store/file"
)

type Mode string

const (
	ModeDB   Mode = "db"
	ModeFile Mode = "file"

	defaultDriversFile = "./drivers.yaml"
)

// Config selects where Foreign Server / Table metadata is loaded from.
// - db: Meta DB (DATABASE_URL / DATABASE_DSN) + Admin API
// - file: declarative drivers.yaml
type Config struct {
	Mode Mode
	File file.Config
}

func LoadConfig() (Config, error) {
	cfg := Config{Mode: ModeDB}

	if mode := strings.TrimSpace(os.Getenv("DATASPAN_STORE_MODE")); mode != "" {
		cfg.Mode = Mode(strings.ToLower(mode))
	}
	path := strings.TrimSpace(os.Getenv("DATASPAN_DRIVERS_FILE"))
	if path == "" {
		path = defaultDriversFile
	}
	cfg.File.Path = path // file path or directory of *.yaml / *.yml

	switch cfg.Mode {
	case ModeDB, ModeFile:
	default:
		return Config{}, fmt.Errorf("unsupported store mode %q (use db or file)", cfg.Mode)
	}
	return cfg, nil
}

func New(vault *auth.Vault, cfg Config) (Store, error) {
	switch cfg.Mode {
	case ModeFile:
		return file.New(vault, cfg.File)
	case ModeDB:
		dbCfg, err := db.ConfigFromEnv()
		if err != nil {
			return nil, err
		}
		return db.New(vault, dbCfg)
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
	case ModeDB:
		dbCfg, err := db.ConfigFromEnv()
		if err != nil {
			return err
		}
		return db.Ping(ctx, dbCfg)
	default:
		return fmt.Errorf("unsupported store mode %q", cfg.Mode)
	}
}
