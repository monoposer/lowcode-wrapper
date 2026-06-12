package store_test

import (
	"testing"

	"github.com/monoposer/dataspan/internal/store"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DATASPAN_STORE_MODE", "")
	t.Setenv("DATASPAN_DRIVERS_FILE", "")

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

func TestLoadConfigDataspanEnv(t *testing.T) {
	t.Setenv("DATASPAN_STORE_MODE", "file")
	t.Setenv("DATASPAN_DRIVERS_FILE", "./config/drivers.yaml")

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

func TestLoadConfigRejectsPostgresMode(t *testing.T) {
	t.Setenv("DATASPAN_STORE_MODE", "postgres")
	_, err := store.LoadConfig()
	if err == nil {
		t.Fatal("expected error for unsupported store mode postgres")
	}
}
