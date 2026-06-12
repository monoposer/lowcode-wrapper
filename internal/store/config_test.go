package store_test

import (
	"path/filepath"
	"testing"

	"github.com/monoposer/dataspan/internal/store"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("WRAPPER_STORE_MODE", "")
	t.Setenv("WRAPPER_DRIVERS_FILE", "")
	t.Setenv("WRAPPER_STORE_FILE", "")

	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != store.ModeDB {
		t.Fatalf("mode = %q, want db", cfg.Mode)
	}
	if cfg.File.Path != "./drivers.yaml" {
		t.Fatalf("drivers file = %q", cfg.File.Path)
	}
}

func TestLoadConfigLegacyPostgresMode(t *testing.T) {
	t.Setenv("WRAPPER_STORE_MODE", "postgres")
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != store.ModeDB {
		t.Fatalf("mode = %q, want db", cfg.Mode)
	}
}

func TestLoadConfigFileMode(t *testing.T) {
	t.Setenv("WRAPPER_STORE_MODE", "file")
	t.Setenv("WRAPPER_DRIVERS_FILE", "./config/drivers.yaml")

	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != store.ModeFile {
		t.Fatalf("mode = %q", cfg.Mode)
	}
	if cfg.File.Path != "./config/drivers.yaml" {
		t.Fatalf("path = %q", cfg.File.Path)
	}
}

func TestLoadConfigLegacyStoreFileEnv(t *testing.T) {
	t.Setenv("WRAPPER_STORE_MODE", "file")
	t.Setenv("WRAPPER_DRIVERS_FILE", "")
	t.Setenv("WRAPPER_STORE_FILE", "./legacy.yaml")

	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.File.Path != "./legacy.yaml" {
		t.Fatalf("path = %q", cfg.File.Path)
	}
}

func TestLoadConfigIgnoresMissingYAML(t *testing.T) {
	t.Setenv("WRAPPER_STORE_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != store.ModeDB {
		t.Fatalf("mode = %q", cfg.Mode)
	}
}
