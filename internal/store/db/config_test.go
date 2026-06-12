package db

import (
	"testing"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5432/app?sslmode=disable")
	t.Setenv("DATABASE_DSN", "")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DSN != "postgresql://u:p@localhost:5432/app?sslmode=disable" {
		t.Fatalf("dsn = %q", cfg.DSN)
	}
}

func TestConfigFromEnvDSNFallback(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_DSN", "file:test.db")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DSN != "file:test.db" {
		t.Fatalf("dsn = %q", cfg.DSN)
	}
}

func TestConfigFromEnvMissing(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_DSN", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("expected error")
	}
}

func TestDialectorFor(t *testing.T) {
	cases := []struct {
		dsn    string
		wantOK bool
	}{
		{"postgresql://localhost/app", true},
		{"postgres://localhost/app", true},
		{"mysql://user:pass@tcp(localhost:3306)/app", true},
		{"file:test.db", true},
		{"meta.db", true},
		{"unknown://x", false},
	}
	for _, tc := range cases {
		_, err := dialectorFor(tc.dsn)
		if tc.wantOK && err != nil {
			t.Fatalf("dsn %q: %v", tc.dsn, err)
		}
		if !tc.wantOK && err == nil {
			t.Fatalf("dsn %q: expected error", tc.dsn)
		}
	}
}

func TestOpenSQLite(t *testing.T) {
	path := t.TempDir() + "/meta.db"
	gdb, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()
}

func TestConfigFromEnvTrims(t *testing.T) {
	t.Setenv("DATABASE_URL", "  postgresql://u@localhost/db  ")
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DSN != "postgresql://u@localhost/db" {
		t.Fatalf("dsn = %q", cfg.DSN)
	}
}
